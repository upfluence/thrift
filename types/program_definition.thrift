namespace * types.program_definition

include "types/type_definition.thrift"
include "types/struct_definition.thrift"
include "types/service_definition.thrift"
include "types/value_definition.thrift"

struct ProgramDefinition {
  1: required string name;
  2: required string path;
  3: optional string doc;

  4: required map<string, string> namespaces;
  5: required list<ProgramDefinition> includes;

  6: required map<string, struct_definition.StructDefinition> types;
  7: required map<string, service_definition.ServiceDefinition> services;
  8: required map<string, value_definition.Value> constants;
}
