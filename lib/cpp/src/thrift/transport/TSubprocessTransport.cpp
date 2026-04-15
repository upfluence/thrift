/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

#include "thrift/transport/TSubprocessTransport.h"

#include <errno.h>
#include <string.h>

#ifndef _WIN32
#include <sys/select.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#endif

namespace apache {
namespace thrift {
namespace transport {

uint32_t TSubprocessTransport::read_virt(uint8_t* buf, uint32_t len) {
  if (!open_) {
    throw TTransportException(TTransportException::NOT_OPEN,
                              "TSubprocessTransport::read() not open");
  }

  if (readOffset_ >= readBuffer_.size()) {
    return 0;
  }

  uint32_t available = static_cast<uint32_t>(readBuffer_.size() - readOffset_);
  uint32_t toCopy = (len < available) ? len : available;
  if (toCopy > 0) {
    memcpy(buf, &readBuffer_[readOffset_], toCopy);
    readOffset_ += toCopy;
  }
  return toCopy;
}

void TSubprocessTransport::write_virt(const uint8_t* buf, uint32_t len) {
  if (!open_) {
    throw TTransportException(TTransportException::NOT_OPEN,
                              "TSubprocessTransport::write() not open");
  }

  if (len == 0) {
    return;
  }

  size_t offset = writeBuffer_.size();
  writeBuffer_.resize(offset + len);
  memcpy(&writeBuffer_[offset], buf, len);
}

void TSubprocessTransport::flush() {
  if (!open_) {
    throw TTransportException(TTransportException::NOT_OPEN,
                              "TSubprocessTransport::flush() not open");
  }

#ifdef _WIN32
  throw TTransportException(TTransportException::NOT_OPEN,
                            "TSubprocessTransport::flush() not supported on Windows");
#else
  int stdinPipe[2];
  int stdoutPipe[2];
  int stderrPipe[2];
  if (pipe(stdinPipe) != 0) {
    int errno_copy = errno;
    throw TTransportException(TTransportException::UNKNOWN,
                              "TSubprocessTransport::flush() pipe(stdin) failed",
                              errno_copy);
  }
  if (pipe(stdoutPipe) != 0) {
    int errno_copy = errno;
    ::close(stdinPipe[0]);
    ::close(stdinPipe[1]);
    throw TTransportException(TTransportException::UNKNOWN,
                              "TSubprocessTransport::flush() pipe(stdout) failed",
                              errno_copy);
  }
  if (pipe(stderrPipe) != 0) {
    int errno_copy = errno;
    ::close(stdinPipe[0]);
    ::close(stdinPipe[1]);
    ::close(stdoutPipe[0]);
    ::close(stdoutPipe[1]);
    throw TTransportException(TTransportException::UNKNOWN,
                              "TSubprocessTransport::flush() pipe(stderr) failed",
                              errno_copy);
  }

  pid_t pid = fork();
  if (pid < 0) {
    int errno_copy = errno;
    ::close(stdinPipe[0]);
    ::close(stdinPipe[1]);
    ::close(stdoutPipe[0]);
    ::close(stdoutPipe[1]);
    ::close(stderrPipe[0]);
    ::close(stderrPipe[1]);
    throw TTransportException(TTransportException::UNKNOWN,
                              "TSubprocessTransport::flush() fork failed",
                              errno_copy);
  }

  if (pid == 0) {
    dup2(stdinPipe[0], STDIN_FILENO);
    dup2(stdoutPipe[1], STDOUT_FILENO);
    dup2(stderrPipe[1], STDERR_FILENO);
    ::close(stdinPipe[0]);
    ::close(stdinPipe[1]);
    ::close(stdoutPipe[0]);
    ::close(stdoutPipe[1]);
    ::close(stderrPipe[0]);
    ::close(stderrPipe[1]);

    std::vector<char*> argv;
    argv.reserve(args_.size() + 2);
    argv.push_back(const_cast<char*>(command_.c_str()));
    for (std::vector<std::string>::const_iterator it = args_.begin(); it != args_.end(); ++it) {
      argv.push_back(const_cast<char*>(it->c_str()));
    }
    argv.push_back(NULL);

    execvp(command_.c_str(), &argv[0]);
    _exit(127);
  }

  ::close(stdinPipe[0]);
  ::close(stdoutPipe[1]);
  ::close(stderrPipe[1]);

  size_t writeOffset = 0;
  while (writeOffset < writeBuffer_.size()) {
    ssize_t written = ::write(stdinPipe[1],
                              &writeBuffer_[writeOffset],
                              writeBuffer_.size() - writeOffset);
    if (written < 0) {
      int errno_copy = errno;
      ::close(stdinPipe[1]);
      ::close(stdoutPipe[0]);
      ::close(stderrPipe[0]);
      throw TTransportException(TTransportException::UNKNOWN,
                                "TSubprocessTransport::flush() write failed",
                                errno_copy);
    }
    writeOffset += static_cast<size_t>(written);
  }
  ::close(stdinPipe[1]);

  readBuffer_.clear();
  readOffset_ = 0;
  stderrBuffer_.clear();

  bool stdoutOpen = true;
  bool stderrOpen = true;
  uint8_t temp[4096];
  int maxFd = stdoutPipe[0] > stderrPipe[0] ? stdoutPipe[0] : stderrPipe[0];
  while (stdoutOpen || stderrOpen) {
    fd_set readfds;
    FD_ZERO(&readfds);
    if (stdoutOpen) {
      FD_SET(stdoutPipe[0], &readfds);
    }
    if (stderrOpen) {
      FD_SET(stderrPipe[0], &readfds);
    }

    int ready = select(maxFd + 1, &readfds, NULL, NULL, NULL);
    if (ready < 0) {
      if (errno == EINTR) {
        continue;
      }
      int errno_copy = errno;
      if (stdoutOpen) {
        ::close(stdoutPipe[0]);
      }
      if (stderrOpen) {
        ::close(stderrPipe[0]);
      }
      throw TTransportException(TTransportException::UNKNOWN,
                                "TSubprocessTransport::flush() select failed",
                                errno_copy);
    }

    if (stdoutOpen && FD_ISSET(stdoutPipe[0], &readfds)) {
      ssize_t readCount = ::read(stdoutPipe[0], temp, sizeof(temp));
      if (readCount == 0) {
        ::close(stdoutPipe[0]);
        stdoutOpen = false;
      } else if (readCount < 0) {
        if (errno != EINTR) {
          int errno_copy = errno;
          ::close(stdoutPipe[0]);
          if (stderrOpen) {
            ::close(stderrPipe[0]);
          }
          throw TTransportException(TTransportException::UNKNOWN,
                                    "TSubprocessTransport::flush() read failed",
                                    errno_copy);
        }
      } else {
        size_t offset = readBuffer_.size();
        readBuffer_.resize(offset + static_cast<size_t>(readCount));
        memcpy(&readBuffer_[offset], temp, static_cast<size_t>(readCount));
      }
    }

    if (stderrOpen && FD_ISSET(stderrPipe[0], &readfds)) {
      ssize_t readCount = ::read(stderrPipe[0], temp, sizeof(temp));
      if (readCount == 0) {
        ::close(stderrPipe[0]);
        stderrOpen = false;
      } else if (readCount < 0) {
        if (errno != EINTR) {
          int errno_copy = errno;
          ::close(stderrPipe[0]);
          if (stdoutOpen) {
            ::close(stdoutPipe[0]);
          }
          throw TTransportException(TTransportException::UNKNOWN,
                                    "TSubprocessTransport::flush() read(stderr) failed",
                                    errno_copy);
        }
      } else {
        size_t offset = stderrBuffer_.size();
        stderrBuffer_.resize(offset + static_cast<size_t>(readCount));
        memcpy(&stderrBuffer_[offset], temp, static_cast<size_t>(readCount));
      }
    }
  }

  int status = 0;
  waitpid(pid, &status, 0);
  writeBuffer_.clear();
#endif
}

std::string TSubprocessTransport::getStderr() const {
  return std::string(stderrBuffer_.begin(), stderrBuffer_.end());
}
}
}
} // apache::thrift::transport
