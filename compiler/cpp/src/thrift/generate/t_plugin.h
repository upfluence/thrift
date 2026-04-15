#include <sstream>
#include "thrift/logging.h"
#include "thrift/parse/t_program.h"
#include "thrift/parse/t_base_type.h"
#include "thrift/parse/t_list.h"
#include "thrift/parse/t_map.h"
#include "thrift/parse/t_set.h"
#include "thrift/parse/t_function.h"
#include "thrift/parse/t_typedef.h"
#include "thrift/parse/t_enum.h"
#include "thrift/generate/t_generator_registry.h"
#include "thrift/transport/TSubprocessTransport.h"
#include <thrift/types/Plugin.h>
#include <thrift/types/plugin_types.h>
#include <thrift/types/program_definition_types.h>
#include <thrift/types/enum_definition_types.h>
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
  static ::types::core::Reference build_reference(const t_type* type) {
    ::types::core::Reference ref;
    ref.__set_namespace_(type->get_program()->get_namespace("*"));
    ref.__set_name(type->get_name());
    return ref;
  }

  static ::types::type_definition::TypeDefinition build_type(const t_type* type) {
    ::types::type_definition::TypeDefinition t;
    type = type->get_true_type();

    if (type->is_base_type()) {
      const t_base_type* bt = static_cast<const t_base_type*>(type);
      ::types::type_definition::ScalarType::type st = ::types::type_definition::ScalarType::Unknown;
      switch (bt->get_base()) {
        case t_base_type::TYPE_STRING:
          st = bt->is_binary() ? ::types::type_definition::ScalarType::Binary
                               : ::types::type_definition::ScalarType::String;
          break;
        case t_base_type::TYPE_BOOL:   st = ::types::type_definition::ScalarType::Bool;   break;
        case t_base_type::TYPE_I16:    st = ::types::type_definition::ScalarType::I16;    break;
        case t_base_type::TYPE_I32:    st = ::types::type_definition::ScalarType::I32;    break;
        case t_base_type::TYPE_I64:    st = ::types::type_definition::ScalarType::I64;    break;
        case t_base_type::TYPE_DOUBLE: st = ::types::type_definition::ScalarType::Double; break;
        default: break;
      }
      t.__set_scalar_type(st);
    } else if (type->is_list()) {
      ::types::type_definition::ListTypeDefinition lt;
      lt.__set_element_type(std::make_shared<::types::type_definition::TypeDefinition>(
          build_type(static_cast<const t_list*>(type)->get_elem_type())));
      t.__set_list_type(lt);
    } else if (type->is_map()) {
      const t_map* mt = static_cast<const t_map*>(type);
      ::types::type_definition::MapTypeDefinition mpt;
      mpt.__set_key_type(std::make_shared<::types::type_definition::TypeDefinition>(build_type(mt->get_key_type())));
      mpt.__set_value_type(std::make_shared<::types::type_definition::TypeDefinition>(build_type(mt->get_val_type())));
      t.__set_map_type(mpt);
    } else if (type->is_set()) {
      ::types::type_definition::SetTypeDefinition st;
      st.__set_element_type(std::make_shared<::types::type_definition::TypeDefinition>(
          build_type(static_cast<const t_set*>(type)->get_elem_type())));
      t.__set_set_type(st);
    } else {
      t.__set_reference_type(build_reference(type));
    }
    return t;
  }

  static ::types::annotation_definition::AnnotationDefinition build_annotation(const t_annotated* node) {
    ::types::annotation_definition::AnnotationDefinition ann;
    ann.__set_name(node->get_name());
    ann.__set_legacy_annotations(node->legacy_annotations());
    ann.__set_structured_annotations({});
    return ann;
  }

  static ::types::struct_definition::StructDefinition build_struct(const t_struct* s) {
    ::types::struct_definition::StructDefinition sd;
    ::types::struct_definition::StructKind::type kind = ::types::struct_definition::StructKind::Struct;
    if (s->is_xception())   kind = ::types::struct_definition::StructKind::Exception;
    else if (s->is_union()) kind = ::types::struct_definition::StructKind::Union;
    sd.__set_kind(kind);
    sd.__set_annotation(build_annotation(s));

    std::vector<::types::struct_definition::FieldDefinition> fields;
    for (const t_field* f : s->get_members()) {
      ::types::struct_definition::FieldDefinition field;
      field.__set_id(f->get_key());
      field.__set_type(std::make_shared<::types::type_definition::TypeDefinition>(build_type(f->get_type())));
      field.__set_annotation(build_annotation(f));
      ::types::struct_definition::Requiredness::type req = ::types::struct_definition::Requiredness::Unknown;
      if (f->get_req() == t_field::T_REQUIRED)      req = ::types::struct_definition::Requiredness::Required;
      else if (f->get_req() == t_field::T_OPTIONAL) req = ::types::struct_definition::Requiredness::Optional;
      field.__set_requiredness(req);
      fields.push_back(field);
    }
    sd.__set_fields(fields);
    return sd;
  }

  static ::types::service_definition::ServiceDefinition build_service(const t_service* svc) {
    ::types::service_definition::ServiceDefinition sd;
    sd.__set_annotation(build_annotation(svc));
    if (svc->get_extends()) {
      sd.__set_extends_(build_reference(svc->get_extends()));
    }
    std::vector<::types::service_definition::FunctionDefinition> fns;
    for (const t_function* f : svc->get_functions()) {
      ::types::service_definition::FunctionDefinition fd;
      fd.__set_annotation(build_annotation(f));
      fd.__set_return_type(build_type(f->get_returntype()));
      fd.__set_oneway_(f->is_oneway());
      std::vector<::types::struct_definition::FieldDefinition> args;
      for (const t_field* a : f->get_arglist()->get_members()) {
        ::types::struct_definition::FieldDefinition field;
        field.__set_id(a->get_key());
        field.__set_type(std::make_shared<::types::type_definition::TypeDefinition>(build_type(a->get_type())));
        field.__set_annotation(build_annotation(a));
        field.__set_requiredness(::types::struct_definition::Requiredness::Required);
        args.push_back(field);
      }
      fd.__set_arguments(args);
      std::vector<::types::core::Reference> excs;
      for (const t_field* e : f->get_xceptions()->get_members()) {
        excs.push_back(build_reference(e->get_type()->get_true_type()));
      }
      fd.__set_exceptions(excs);
      fns.push_back(fd);
    }
    sd.__set_functions(fns);
    return sd;
  }

  static ::types::enum_definition::EnumDefinition build_enum(const t_enum* e) {
    ::types::enum_definition::EnumDefinition ed;
    ed.__set_annotation(build_annotation(e));

    std::vector<::types::enum_definition::EnumValueDefinition> values;
    for (const t_enum_value* ev : e->get_constants()) {
      ::types::enum_definition::EnumValueDefinition evd;
      evd.__set_annotation(build_annotation(ev));
      evd.__set_id(ev->get_value());
      values.push_back(evd);
    }
    ed.__set_values(values);
    return ed;
  }

  static ::types::program_definition::ProgramDefinition build_program_definition(const t_program* p) {
    ::types::program_definition::ProgramDefinition pd;
    pd.__set_name(p->get_name());
    pd.__set_path(p->get_path());
    pd.__set_namespaces(p->get_all_namespaces());

    std::vector<::types::program_definition::ProgramDefinition> includes;
    for (const t_program* inc : p->get_includes()) {
      includes.push_back(build_program_definition(inc));
    }
    pd.__set_includes(includes);

    std::map<std::string, ::types::struct_definition::StructDefinition> types;
    for (const t_struct* s : p->get_objects()) {
      types[s->get_name()] = build_struct(s);
    }
    pd.__set_structs(types);

    std::map<std::string, ::types::service_definition::ServiceDefinition> services;
    for (const t_service* svc : p->get_services()) {
      services[svc->get_name()] = build_service(svc);
    }
    pd.__set_services(services);

    pd.__set_constants({});

    std::map<std::string, ::types::type_definition::TypeDefinition> typedefs;
    for (const t_typedef* td : p->get_typedefs()) {
      typedefs[td->get_symbolic()] = build_type(td->get_type());
    }
    pd.__set_typedefs(typedefs);

    std::map<std::string, ::types::enum_definition::EnumDefinition> enums;
    for (const t_enum* e : p->get_enums()) {
      enums[e->get_name()] = build_enum(e);
    }
    pd.__set_enums(enums);
    return pd;
  }

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
    req->__set_program(build_program_definition(generator_->get_program()));

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
