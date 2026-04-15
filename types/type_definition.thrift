namespace * types.type_definition

include "types/core.thrift"
include "types/annotation_definition.thrift"

enum ScalarType {
  Unknown = 0,
  String = 1,
  Binary = 2,
  Bool = 3,
  I16 = 4,
  I32 = 5,
  I64 = 6,
  Double = 7,
}

struct ListTypeDefinition {
  1: required TypeDefinition& element_type;
}

struct MapTypeDefinition {
  1: required TypeDefinition& key_type;
  2: required TypeDefinition& value_type;
}

struct SetTypeDefinition {
  1: required TypeDefinition& element_type;
}

union TypeDefinition {
  1: ScalarType         scalar_type;
  2: ListTypeDefinition list_type;
  3: MapTypeDefinition  map_type;
  4: SetTypeDefinition  set_type;
  8: core.Reference     reference_type;
}
