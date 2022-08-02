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

#ifndef T_RETURN_H
#define T_RETURN_H

#include "t_type.h"

class t_return {
public:
  t_return(t_type* return_,
             bool oneway)
    : oneway_(oneway),
      returntype_(return_),
      stream_(NULL),
      sink_(NULL) {
    if (oneway_ && (!returntype_->is_void())) {
      pwarning(1, "Oneway methods should return void.\n");
    }
  }

  ~t_return() {}

  t_type* get_returntype() const { return returntype_; }
  t_type* get_sink() const  { return sink_; }
  t_type* get_stream() const { return stream_; }

  bool is_oneway() const { return oneway_; }
  bool is_streaming() const { return sink_ != NULL || stream_ != NULL; }

  void set_stream(t_type* type_) {
    stream_ = type_;

    if (oneway_) {
      pwarning(1, "Oneway methods should not have streams.\n");
    }
  }

  void set_sink(t_type* type_) {
    sink_ = type_;

    if (oneway_) {
      pwarning(1, "Oneway methods should not have sinks.\n");
    }
  }

private:
  bool oneway_;
  t_type* returntype_;
  t_type* stream_;
  t_type* sink_;
};

#endif
