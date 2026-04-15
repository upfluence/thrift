namespace * types.value_definition

struct NullValueDefinition {}

struct ListValueDefinition {
  1: required list<ValueDefinition> values;
}

struct MapEntryDefinition {
  1: required ValueDefinition& key;
  2: required ValueDefinition& value;
}

struct MapValueDefinition {
  1: required list<MapEntryDefinition> entries;
}

union ValueDefinition {
  1: NullValueDefinition  null_value;
  2: string               string_value;
  3: binary               binary_value;
  4: i64                  integer_value;
  5: double               double_value;
  6: bool                 bool_value;
  7: ListValueDefinition  list_value;
  8: MapValueDefinition   map_value;
  9: StructValueDefinition& struct_value;
}

struct StructValueDefinition {
  1: optional map<string, ValueDefinition> fields;
}
