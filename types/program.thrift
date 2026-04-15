namespace * types.program

include "types/type.thrift"
include "types/service.thrift"
include "types/value.thrift"

struct ProgramDefinition {
  1: required string name;
  2: required string path;
  3: optional string doc;

  4: required map<string, string> namespaces;
  5: required list<ProgramDefinition> includes;

  6: required map<string, struct.StructDefinition> types;
  7: required map<string, service.ServiceDefinition> services;
  8: required map<string, value.ValueDefinition> constants;
}
