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

/**
 * Noop generator.
 *
 * Produces no output files. Its sole purpose is to drive the plugin
 * machinery: the compiler parses the input, builds the AST, then hands
 * the serialized ProgramDefinition to every --plugin binary.
 *
 * Usage:
 *   thrift -gen noop --plugin key=/path/to/plugin file.thrift
 */

#include "thrift/generate/t_generator.h"

class t_noop_generator : public t_generator {
public:
  t_noop_generator(
      t_program* program,
      const std::map<std::string, std::string>& parsed_options,
      const std::string& option_string)
    : t_generator(program, "noop", parsed_options) {
    (void)option_string;
    out_dir_base_ = "gen-noop";
  }

  // All virtual methods are intentional no-ops.
  void generate_typedef(t_typedef*)  override {}
  void generate_enum(t_enum*)        override {}
  void generate_struct(t_struct*)    override {}
  void generate_service(t_service*)  override {}

  // Accept streaming programs without filtering so the plugin receives
  // the full, unmodified AST.
  bool support_streaming() const override { return true; }
};

THRIFT_REGISTER_GENERATOR(noop,
  "Noop (plugin-only driver, produces no output files)",
  "")
