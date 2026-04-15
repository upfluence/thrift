namespace * types.program_definition

include "types/enum_definition.thrift"
include "types/type_definition.thrift"
include "types/struct_definition.thrift"
include "types/service_definition.thrift"
include "types/value.thrift"

struct ProgramDefinition {
  1: required string name;
  2: required string path;
  3: optional string doc;

  4: required map<string, string> namespaces;
  5: required list<ProgramDefinition> includes;

  6: required map<string, struct_definition.StructDefinition> structs;
  7: required map<string, service_definition.ServiceDefinition> services;
  8: required map<string, value.Value> constants;
  9: required map<string, type_definition.TypeDefinition> typedefs;
  10: required map<string, enum_definition.EnumDefinition> enums;
}
