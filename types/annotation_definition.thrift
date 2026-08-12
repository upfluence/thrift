namespace * types.annotation_definition

include "types/core.thrift"
include "types/value.thrift"

enum CommentKind {
  LineSlash = 1, // // comment
  LineHash  = 2, // # comment
  Block     = 3, // /* ... */
  Doc       = 4, // /** ... */
}

struct Comment {
  1: required CommentKind kind;
  2: required string      value;
}

struct NodeLocation {
  1: required i32 line;
  2: required i32 col;
}

struct StructuredAnnotationDefinition {
  1: required core.Reference type;
  2: required value.Value value;
}

struct AnnotationDefinition {
  1: required string name;
  2: required list<StructuredAnnotationDefinition> structured_annotations;
  3: required map<string, string>                  legacy_annotations;
  4: optional NodeLocation                         from_loc;
  5: optional NodeLocation                         to_loc;
  6: optional list<Comment>                        leading_comments;
  7: optional Comment                              trailing_comment;
}
