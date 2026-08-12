namespace * types.struct_definition

include "types/annotation_definition.thrift"
include "types/type_definition.thrift"
include "types/constant_definition.thrift"

enum Requiredness {
  Unknown  = 0,
  Optional = 1,
  Required = 2,
  Default  = 3,
}

struct FieldDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: required i32                                         id;
  3: required type_definition.TypeDefinition&             type;
  4: required Requiredness                                requiredness;
  5: optional constant_definition.ConstantValueDefinition default_value;
  6: optional bool                                        reference;
}

enum StructKind {
  Unknown   = 0,
  Struct    = 1,
  Exception = 2,
  Union     = 3,
}

struct StructDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: required StructKind                                  kind;
  3: required list<FieldDefinition>                       fields;
}
