namespace * types.struct_definition

include "types/annotation_definition.thrift"
include "types/type_definition.thrift"

enum Requiredness {
  Unknown = 0,
  Optional = 1,
  Required = 2,
}

struct FieldDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: required i32                    id;
  3: required type_definition.TypeDefinition&  type;
  4: required Requiredness            requiredness;
}

enum StructKind {
  Unknown = 0,
  Struct = 1,
  Exception = 2,
  Union = 3,
}

struct StructDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: required StructKind              kind;
  3: required list<FieldDefinition>   fields;
}
