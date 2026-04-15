namespace * types.plugin

include "types/program_definition.thrift"

struct GenerateCodeRequest {
  1: required program_definition.ProgramDefinition program;
  2: required string language;
  3: required map<string, string> options;
}

struct GenerateCodeResponse {
  1: required map<string, binary> files;
}

service Plugin {
  GenerateCodeResponse generate_code(1: GenerateCodeRequest req)
}
