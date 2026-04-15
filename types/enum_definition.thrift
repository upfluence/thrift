namespace * types.enum_definition

include "types/annotation_definition.thrift"

struct EnumValueDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;

  2: required i32 id;
}

struct EnumDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: required list<EnumValueDefinition> values;
}
