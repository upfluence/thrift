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

#ifndef T_ANNOTATED_H
#define T_ANNOTATED_H

#include <string>
#include <map>
#include "t_doc.h"

class t_type;
class t_const_value;

class t_structured_annotation {
public:
  t_structured_annotation() : type_(NULL), value_(NULL) {};
  t_type* type_;
  t_const_value* value_;

};

/**
 * Placeholder struct for returning the key and value of an annotation
 * during parsing.
 */
struct t_legacy_annotation {
  std::string key;
  std::string val;
};


class t_annotated : public t_doc {
public:
  void append_legacy_annotation(t_legacy_annotation* v) { legacy_annotations_[v->key] = v->val; }
  void append_structured_annotation(t_structured_annotation* v) { structured_annotations_.push_back(v); }

  void merge(t_annotated* other) {
    legacy_annotations_.insert(std::make_move_iterator(other->legacy_annotations_.begin()), std::make_move_iterator(other->legacy_annotations_.end()));
    structured_annotations_.insert(structured_annotations_.end(), std::make_move_iterator(other->structured_annotations_.begin()), std::make_move_iterator(other->structured_annotations_.end()));
  }

  const std::map<std::string, std::string>& legacy_annotations() { return legacy_annotations_; }

  const std::string& get_name() const { return name_; }

  bool has_legacy_annotation(std::string key) {
    return legacy_annotations_.find(key) != legacy_annotations_.end();
  }

  const std::string legacy_annotation_value(std::string key) {
    return legacy_annotations_.find(key)->second;
  }

  t_annotated() : name_("") {}
protected:
  t_annotated(std::string name) : name_(name) {}
  std::string name_;

private:
  std::map<std::string, std::string> legacy_annotations_;
  std::vector<t_structured_annotation*> structured_annotations_;
};

#endif
