namespace * types.constant_definition

include "types/core.thrift"
include "types/annotation_definition.thrift"

struct ListConstantValueDefinition {
  1: required list<ConstantValueDefinition> values;
}

struct MapConstantValueDefinitionEntry {
  1: required ConstantValueDefinition& key;
  2: required ConstantValueDefinition& value;
}

struct MapConstantValueDefinition {
  1: required list<MapConstantValueDefinitionEntry> entries;
}

union ConstantValueDefinition {
  1: core.Reference reference;
  2: i64               integer_value;
  3: double            double_value;
  4: bool              bool_value;
  5: string            string_value;
  6: ListConstantValueDefinition     list_value;
  7: MapConstantValueDefinition      map_value;
}

struct ConstantDefinition {
  1: required annotation_definition.AnnotationDefinition annotation;
  2: optional ConstantValueDefinition value;
}
