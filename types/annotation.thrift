namespace * types.annotation

include "types/core.thrift"
include "types/value.thrift"

struct StructuredAnnotation {
  1: required core.Reference type;
  2: required value.Value value;
}

struct Annotation {
  1: required string name;
  2: required list<StructuredAnnotation> structured_annotations;
  3: required map<string, string>        legacy_annotations;
}
