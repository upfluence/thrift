namespace * types.value

struct NullValue {}

struct ListValue {
  1: required list<Value> values;
}

struct MapEntry {
  1: required Value key;
  2: required Value value;
}

struct MapValue {
  1: required list<MapEntry> entries;
}

struct StructValue {
  1: required map<string,Value> fields;
}

union Value {
  1: NullValue   null_value;
  2: string      string_value;
  3: binary      binary_value;
  4: i64         integer_value;
  5: double      double_value;
  6: bool        bool_value;
  7: ListValue   list_value;
  8: MapValue    map_value;
  9: StructValue struct_value;
}
