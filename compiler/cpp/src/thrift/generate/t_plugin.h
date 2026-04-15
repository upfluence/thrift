#include <sstream>
#include "thrift/logging.h"
#include "thrift/parse/t_program.h"
#include "thrift/generate/t_generator_registry.h"
#include "thrift/transport/TSubprocessTransport.h"
#include <thrift/types/Plugin.h>
#include <thrift/types/plugin_types.h>
#include <thrift/protocol/TBinaryProtocol.h>
#include <thrift/transport/TBufferTransports.h>

using std::ostream;

class t_plugin {
public:
  t_plugin(t_generator* generator, std::string cmd, std::string out_path): generator_(generator), cmd_(cmd), out_path_(out_path), out_path_is_absolute_(false) {}

  void set_out_path(std::string out_path, bool out_path_is_absolute) {
    out_path_ = out_path;
    out_path_is_absolute_ = out_path_is_absolute;
    // Ensure that it ends with a trailing '/' (or '\' for windows machines)
    char c = out_path_.at(out_path_.size() - 1);
    if (!(c == '/' || c == '\\')) {
      out_path_.push_back('/');
    }
  }

  void execute() {
    types::plugin::GenerateCodeResponse* resp = new types::plugin::GenerateCodeResponse();

    build_client()->generate_code(*resp, build_request());
    log_stderr();

    std::map<std::string, std::string>::iterator files_iter;

    MKDIR(get_out_dir().c_str());

    for (files_iter = resp->files.begin(); files_iter != resp->files.end(); files_iter++) {
      pverbose("Adding file %s\n", (get_out_dir() + "/" + files_iter->first).c_str());
      ofstream_with_content_based_conditional_update file;

      file.open(get_out_dir() + "/" + files_iter->first);
      file << files_iter->second;

      file.close();
    }
  }
private:
  types::plugin::PluginClient* build_client() {
    subprocess_transport_.reset(new apache::thrift::transport::TSubprocessTransport(cmd_));
    std::shared_ptr<apache::thrift::protocol::TBinaryProtocol> binaryProtocol(new apache::thrift::protocol::TBinaryProtocol(subprocess_transport_));

    return new types::plugin::PluginClient(binaryProtocol);
  }

  void log_stderr() const {
    if (!subprocess_transport_) {
      return;
    }

    std::string stderr_output = subprocess_transport_->getStderr();
    if (stderr_output.empty()) {
      return;
    }

    size_t start = 0;
    while (start <= stderr_output.size()) {
      size_t end = stderr_output.find('\n', start);
      if (end == std::string::npos) {
        end = stderr_output.size();
      }

      if (end > start) {
        pverbose("[plugin %s]: %s\n", cmd_.c_str(), stderr_output.substr(start, end - start).c_str());
      }

      if (end == stderr_output.size()) {
        break;
      }
      start = end + 1;
    }
  }

  const types::plugin::GenerateCodeRequest& build_request() {
    types::plugin::GenerateCodeRequest* req = new types::plugin::GenerateCodeRequest();

    req->__set_language(generator_->get_language());
    req->__set_options(generator_->get_parsed_options());
    req->__set_program(generator_->get_program()->get_thrift_program());

    return *req;
  }

  std::string get_out_dir() const {
    if (out_path_is_absolute_) {
      return out_path_;
    }

    return generator_->get_out_dir() + out_path_;
  }

  t_generator* generator_;
  std::string cmd_;
  std::string out_path_;
  bool out_path_is_absolute_;
  std::shared_ptr<apache::thrift::transport::TSubprocessTransport> subprocess_transport_;
};
