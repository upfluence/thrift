namespace * types.struct
namespace go types.tstruct

include "types/type.thrift"

enum Requiredness {
  Unknown = 0,
  Optional = 1,
  Required = 2,
}

struct Field {
  1: required annotation.Annotation annotation;
  2: required i32    id;
  3: required Type&   type;
  4: required Requiredness requiredness;
}

enum StructKind {
  Unknown = 0,
  Struct = 1,
  Exception = 2,
  Union = 3,
}

struct StructDefinition {
  1: required annotation.Annotation annotation;
  2: required StructKind  kind;
  3: required list<Field> fields;
}
