namespace * types.annotation_definition

include "types/core.thrift"
include "types/value.thrift"

struct StructuredAnnotationDefinition {
  1: required core.Reference type;
  2: required value.Value value;
}

struct AnnotationDefinition {
  1: required string name;
  2: required list<StructuredAnnotationDefinition> structured_annotations;
  3: required map<string, string>                  legacy_annotations;
}
