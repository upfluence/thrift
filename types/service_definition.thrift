namespace * types.service_definition

include "types/core.thrift"
include "types/annotation_definition.thrift"
include "types/type_definition.thrift"
include "types/struct_definition.thrift"

struct FunctionDefinition {
  1: required annotation_definition.Annotation annotation;

  2: required list<struct_definition.Field> arguments;
  3: required type_definition.Type return_type;
  4: required list<core.Reference> exceptions;
  5: required bool oneway_;
}

struct ServiceDefinition {
  1: required annotation_definition.Annotation annotation;
  2: optional core.Reference extends_;

  3: required list<FunctionDefinition> functions;
}
