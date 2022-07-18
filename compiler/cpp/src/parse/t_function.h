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

#ifndef T_FUNCTION_H
#define T_FUNCTION_H

#include <string>
#include "t_return.h"
#include "t_type.h"
#include "t_struct.h"
#include "t_annotated.h"

/**
 * Representation of a function. Key parts are return type, function name,
 * optional modifiers, and an argument list, which is implemented as a thrift
 * struct.
 *
 */
class t_function : public t_annotated {
public:
  t_function(t_type* returntype_,
             std::string name,
             t_struct* arglist)
    : t_annotated(name),
      arglist_(arglist) {
    return_ = new t_return(returntype_, false);
  }

  t_function(t_return* return_,
             std::string name,
             t_struct* arglist,
             t_struct* xceptions)
    : t_annotated(name),
      return_(return_),
      arglist_(arglist),
      xceptions_(xceptions) {
    if (return_->is_oneway() && !xceptions_->get_members().empty()) {
      throw std::string("Oneway methods can't throw exceptions.");
    }
  }

  ~t_function() {}

  t_return* get_return() const { return return_; }
  t_type* get_returntype() const { return return_->get_returntype(); }

  t_struct* get_arglist() const { return arglist_; }

  t_struct* get_xceptions() const { return xceptions_; }

  bool is_oneway() const { return return_->is_oneway(); }

private:
  t_return* return_;
  std::string name_;
  t_struct* arglist_;
  t_struct* xceptions_;
};

#endif
