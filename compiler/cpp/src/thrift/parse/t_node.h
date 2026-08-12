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

#ifndef T_NODE_H
#define T_NODE_H

#include <string>
#include <utility>
#include <vector>

/**
 * Represents the kind of a comment token as it appeared in the source file.
 */
enum t_comment_kind {
  COMMENT_LINE_SLASH, // // comment
  COMMENT_LINE_HASH,  // # comment
  COMMENT_BLOCK,      // /* ... */
  COMMENT_DOC,        // /** ... */
};

/**
 * A single comment token: its kind and raw text (excluding delimiters).
 */
struct t_comment {
  t_comment_kind kind;
  std::string    value;

  t_comment(t_comment_kind k, std::string v) : kind(k), value(std::move(v)) {}
};

/**
 * A source location: 1-based line and column.
 */
struct t_loc {
  int line;
  int col;

  t_loc() : line(0), col(0) {}
  t_loc(int l, int c) : line(l), col(c) {}
};

/**
 * Base class for every AST node.
 *
 * Carries:
 *   - source span (from / to)
 *   - leading comments (zero or more, in source order)
 *   - optional trailing comment (same line as the node's closing token)
 */
class t_node {
public:
  t_node() : has_trailing_comment_(false), trailing_comment_(COMMENT_LINE_SLASH, "") {}

  virtual ~t_node() = default;

  // ---- location ----

  void set_loc(int first_line, int first_col, int last_line, int last_col) {
    from_ = t_loc(first_line, first_col);
    to_   = t_loc(last_line,  last_col);
  }

  const t_loc& get_from() const { return from_; }
  const t_loc& get_to()   const { return to_;   }

  // ---- leading comments ----

  void add_leading_comment(t_comment_kind kind, std::string value) {
    leading_comments_.emplace_back(kind, std::move(value));
  }

  void set_leading_comments(std::vector<t_comment> comments) {
    leading_comments_ = std::move(comments);
  }

  const std::vector<t_comment>& get_leading_comments() const {
    return leading_comments_;
  }

  // ---- trailing comment ----

  void set_trailing_comment(t_comment_kind kind, std::string value) {
    trailing_comment_      = t_comment(kind, std::move(value));
    has_trailing_comment_  = true;
  }

  bool has_trailing_comment() const { return has_trailing_comment_; }

  const t_comment& get_trailing_comment() const { return trailing_comment_; }

private:
  t_loc from_;
  t_loc to_;

  std::vector<t_comment> leading_comments_;

  bool      has_trailing_comment_;
  t_comment trailing_comment_;
};

#endif
