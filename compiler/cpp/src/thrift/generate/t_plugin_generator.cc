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

#include <string>
#include <fstream>
#include <iostream>
#include <vector>
#include <map>
#include <list>

#include <stdlib.h>
#include <sys/stat.h>
#include <sstream>
#include "t_generator.h"
#include "platform.h"
#include "transport/TTransportUtils.h"
#include <thrift/protocol/TBase64Utils.h>

using std::map;
using std::ofstream;
using std::ostringstream;
using std::pair;
using std::string;
using std::stringstream;
using std::vector;

static const string endl = "\n"; // avoid ostream << std::endl flushes

/**
 * Graphviz code generator
 */
class t_plugin_generator : public t_generator {
public:
  t_plugin_generator(t_program* program,
                 const std::map<std::string, std::string>& parsed_options,
                 const std::string& option_string)
    : t_generator(program) {
    (void)parsed_options;
    (void)option_string;
    out_dir_base_ = "gen-plugin";
  }

  /**
   * Init and end of generator
   */
  void init_generator();
  void close_generator();

  void generate_program();

  void generate_typedef(t_typedef*) {};
  void generate_enum(t_enum*) {};
  void generate_const(t_const*) {};
  void generate_struct(t_struct*) {};
  void generate_service(t_service*) {};
private:
  std::ofstream f_out_;
};

void t_plugin_generator::generate_program() {
  using namespace apache::thrift::transport;
  using namespace apache::thrift::protocol;

  init_generator();

  TMemoryBuffer* buffer = new TMemoryBuffer();
  boost::shared_ptr<TTransport> trans(buffer);
  TBinaryProtocol protocol(trans);

  program_->get_serializable().write(&protocol);

  uint8_t* buf;
  uint32_t size;
  uint8_t b[4];

  buffer->getBuffer(&buf, &size);


  while (size >= 3) {
    // Encode 3 bytes at a time
    base64_encode(buf, 3, b);
    f_out_.write((char *)b, 4);
    buf += 3;
    size -= 3;
  }
  if (size > 0) { // Handle remainder
    base64_encode(buf, size, b);
    f_out_.write((char *)b, size + 1);
  }

 f_out_ << endl;

  close_generator();
}

/**
 * Init generator:
 * - Create output directory and open file for writting.
 */
void t_plugin_generator::init_generator() {
  MKDIR(get_out_dir().c_str());
  string fname = get_out_dir() + program_->get_name() + ".thrift.bin";
  f_out_.open(fname.c_str());
}

/**
 * Closes generator:
 * - Closes file.
 */
void t_plugin_generator::close_generator() {
  f_out_.close();
}

THRIFT_REGISTER_GENERATOR(
    plugin,
    "Plugin",
    "    protocol:  Which protocol used to encode the program.\n")
