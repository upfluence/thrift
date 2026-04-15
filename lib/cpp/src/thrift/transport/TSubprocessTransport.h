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

#include <cstddef>
#ifndef _THRIFT_TRANSPORT_TSUBPROCESSTRANSPORT_H_
#define _THRIFT_TRANSPORT_TSUBPROCESSTRANSPORT_H_ 1

#include <stddef.h>
#include <stdint.h>

#include <string>
#include <vector>

#include <thrift/transport/TTransport.h>

namespace apache {
namespace thrift {
namespace transport {

class TSubprocessTransport : public TTransport {
public:
  TSubprocessTransport(const std::string& command)
    : command_(command), readOffset_(0), open_(true) {}

  TSubprocessTransport(const std::string& command, const std::vector<std::string>& args)
    : command_(command), args_(args), readOffset_(0), open_(true) {}

  bool isOpen() { return open_; }

  void open() { open_ = true; }

  void close() { open_ = false; }

  uint32_t read_virt(uint8_t* buf, uint32_t len);

  void write_virt(const uint8_t* buf, uint32_t len);

  void flush();

  std::string getStderr() const;

protected:
  std::string command_;
  std::vector<std::string> args_;
  std::vector<uint8_t> writeBuffer_;
  std::vector<uint8_t> readBuffer_;
  std::vector<uint8_t> stderrBuffer_;
  size_t readOffset_;
  bool open_;
};
}
}
} // apache::thrift::transport

#endif // _THRIFT_TRANSPORT_TSUBPROCESSTRANSPORT_H_
