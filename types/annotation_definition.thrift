namespace * types.annotation_definition

include "types/core.thrift"
include "types/value_definition.thrift"

struct StructuredAnnotation {
  1: required core.Reference type;
  2: required value_definition.Value value;
}

struct Annotation {
  1: required string name;
  2: required list<StructuredAnnotation> structured_annotations;
  3: required map<string, string>        legacy_annotations;
}
