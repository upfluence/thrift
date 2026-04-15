namespace * types.service

include "types/core.thrift"
include "types/annotation.thrift"
include "types/type.thrift"

struct FunctionDefinition {
  1: required annotation.Annotation annotation;

  2: required list<type.Field> arguments;
  3: required type.Type return_type;
  4: required list<core.Reference> exceptions;
  5: required bool oneway_;
}

struct ServiceDefinition {
  1: required annotation.Annotation annotation;
  2: optional core.Reference extends_;

  3: required list<Function> functions;
}
