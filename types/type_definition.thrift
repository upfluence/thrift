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

struct ListType {
  1: required Type& element_type;
}

struct MapType {
  1: required Type& key_type;
  2: required Type& value_type;
}

struct SetType {
  1: required Type& element_type;
}

union Type {
  1: ScalarType scalar_type;
  2: ListType list_type;
  3: MapType map_type;
  4: SetType set_type;
  8: core.Reference reference_type;
}
