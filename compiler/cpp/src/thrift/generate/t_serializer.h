#ifndef T_SERIALIZER_H
#define T_SERIALIZER_H

#include "thrift/parse/t_annotated.h"
#include "thrift/parse/t_base_type.h"
#include "thrift/parse/t_const.h"
#include "thrift/parse/t_const_value.h"
#include "thrift/parse/t_enum.h"
#include "thrift/parse/t_function.h"
#include "thrift/parse/t_list.h"
#include "thrift/parse/t_map.h"
#include "thrift/parse/t_program.h"
#include "thrift/parse/t_service.h"
#include "thrift/parse/t_set.h"
#include "thrift/parse/t_struct.h"
#include "thrift/parse/t_typedef.h"
#include "thrift/parse/t_type.h"

#include <thrift/types/annotation_definition_types.h>
#include <thrift/types/constant_definition_types.h>
#include <thrift/types/core_types.h>
#include <thrift/types/enum_definition_types.h>
#include <thrift/types/program_definition_types.h>
#include <thrift/types/service_definition_types.h>
#include <thrift/types/struct_definition_types.h>
#include <thrift/types/type_definition_types.h>

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
      case t_base_type::TYPE_I8:     st = ::types::type_definition::ScalarType::I8;     break;
      case t_base_type::TYPE_I16:    st = ::types::type_definition::ScalarType::I16;    break;
      case t_base_type::TYPE_I32:    st = ::types::type_definition::ScalarType::I32;    break;
      case t_base_type::TYPE_I64:    st = ::types::type_definition::ScalarType::I64;    break;
      case t_base_type::TYPE_DOUBLE: st = ::types::type_definition::ScalarType::Double; break;
      case t_base_type::TYPE_VOID:   st = ::types::type_definition::ScalarType::Void;   break;
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
    else if (f->get_req() == t_field::T_OPTIONAL)  req = ::types::struct_definition::Requiredness::Optional;

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

    if (f->get_return()->get_sink() != NULL) {
      fd.__set_sink_type(build_type(f->get_return()->get_sink()));
    }

    if (f->get_return()->get_stream() != NULL) {
      fd.__set_stream_type(build_type(f->get_return()->get_stream()));
    }

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

static ::types::constant_definition::ConstantValueDefinition build_constant_value(
    const t_const_value* cv, const t_type* declared_type) {
  ::types::constant_definition::ConstantValueDefinition out;
  const t_type* true_type = declared_type->get_true_type();

  switch (cv->get_type()) {
    case t_const_value::CV_INTEGER:
      if (true_type->is_base_type()
          && static_cast<const t_base_type*>(true_type)->get_base() == t_base_type::TYPE_BOOL) {
        out.__set_bool_value(cv->get_integer() != 0);
      } else {
        out.__set_integer_value(cv->get_integer());
      }
      break;
    case t_const_value::CV_DOUBLE:
      out.__set_double_value(cv->get_double());
      break;
    case t_const_value::CV_STRING:
      out.__set_string_value(cv->get_string());
      break;
    case t_const_value::CV_IDENTIFIER: {
      ::types::core::Reference ref;
      std::string id = cv->get_identifier();
      size_t dot = id.rfind('.');

      if (dot != std::string::npos) {
        ref.__set_namespace_(id.substr(0, dot));
        ref.__set_name(id.substr(dot + 1));
      } else {
        ref.__set_name(id);
      }

      out.__set_reference(ref);
      break;
    }
    case t_const_value::CV_LIST: {
      ::types::constant_definition::ListConstantValueDefinition lv;
      std::vector<::types::constant_definition::ConstantValueDefinition> elems;
      const t_type* elem_type = true_type->is_list()
          ? static_cast<const t_list*>(true_type)->get_elem_type()
          : declared_type;

      for (const t_const_value* elem : cv->get_list()) {
        elems.push_back(build_constant_value(elem, elem_type));
      }

      lv.__set_values(elems);
      out.__set_list_value(lv);
      break;
    }
    case t_const_value::CV_MAP: {
      ::types::constant_definition::MapConstantValueDefinition mv;
      std::vector<::types::constant_definition::MapConstantValueDefinitionEntry> entries;
      const t_type* key_type = declared_type;
      const t_type* val_type = declared_type;

      if (true_type->is_map()) {
        const t_map* tm = static_cast<const t_map*>(true_type);
        key_type = tm->get_key_type();
        val_type = tm->get_val_type();
      }

      for (const auto& kv : cv->get_map()) {
        ::types::constant_definition::MapConstantValueDefinitionEntry entry;
        entry.__set_key(std::make_shared<::types::constant_definition::ConstantValueDefinition>(
            build_constant_value(kv.first, key_type)));
        entry.__set_value(std::make_shared<::types::constant_definition::ConstantValueDefinition>(
            build_constant_value(kv.second, val_type)));
        entries.push_back(entry);
      }

      mv.__set_entries(entries);
      out.__set_map_value(mv);
      break;
    }
    default:
      break;
  }

  return out;
}

static ::types::constant_definition::ConstantDefinition build_constant_definition(const t_const* c) {
  ::types::constant_definition::ConstantDefinition cd;
  cd.__set_annotation(build_annotation(c));
  cd.__set_value(build_constant_value(c->get_value(), c->get_type()));
  return cd;
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

  std::map<std::string, ::types::struct_definition::StructDefinition> structs;

  for (const t_struct* s : p->get_objects()) {
    structs[s->get_name()] = build_struct(s);
  }

  pd.__set_structs(structs);

  std::map<std::string, ::types::service_definition::ServiceDefinition> services;

  for (const t_service* svc : p->get_services()) {
    services[svc->get_name()] = build_service(svc);
  }

  pd.__set_services(services);

  std::map<std::string, ::types::constant_definition::ConstantDefinition> constants;

  for (const t_const* c : p->get_consts()) {
    constants[c->get_name()] = build_constant_definition(c);
  }

  pd.__set_constants(constants);

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

#endif
