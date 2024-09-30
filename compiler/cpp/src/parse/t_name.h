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

#ifndef T_NAME_H
#define T_NAME_H

#include "globals.h"

class t_name {
public:
  t_name(bool b) : is_legacy_(b) {}

  t_name(bool b, std::string ns, std::string n) : namespace_(ns), name_(n), is_legacy_(b) {}

  void set_namespace_(const std::string& n) { namespace_ = n; }
  void set_name(const std::string& n) { name_ = n; }

  const std::string& get_name() const { return name_; }
  const std::string& get_namespace() const { return namespace_; }
  bool is_legacy() const { return is_legacy_; }
private:
  std::string namespace_;
  std::string name_;
  bool is_legacy_;
};

class t_name_set {
public:
  void append(t_name* name) { names_.push_back(name); }

  const std::vector<t_name*>& get_names() { return names_; }
private:
  std::vector<t_name*> names_;
};

#endif
