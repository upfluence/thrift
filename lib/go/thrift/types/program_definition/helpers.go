package program_definition

import (
	"path/filepath"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/gocodegen"
	"github.com/upfluence/thrift/lib/go/thrift/types/type_definition"
)

const (
	goAny = "interface{}"
	goNil = "nil"
)

func GoPackageName(p *ProgramDefinition) string {
	return goParseValue(
		p,
		func(key string) string {
			ks := strings.Split(key, ".")

			return ks[len(ks)-1]
		},
	)
}

func GoPackagePath(p *ProgramDefinition) string {
	return goParseValue(
		p,
		func(key string) string {
			return strings.ReplaceAll(key, ".", string(filepath.Separator))
		},
	)
}

func goParseValue(p *ProgramDefinition, fn func(key string) string) string {
	for _, key := range []string{"go", "*"} {
		v := p.Namespaces[key]

		if v == "" {
			continue
		}

		if res := fn(v); res != "" {
			return res
		}
	}

	return strings.TrimSuffix(filepath.Base(p.Path), ".thrift")
}

// GoImportPath returns the fully-qualified Go import path for a program,
// using gs to determine the correct prefix.
func GoImportPath(p *ProgramDefinition, gs gocodegen.Scope) string {
	if p.Stdlib {
		return gs.ThriftImport() + "/" + GoPackagePath(p)
	}

	return gs.ImportPkgPrefix + GoPackagePath(p)
}

// GoZeroValue returns the Go zero value expression for a TypeDefinition.
func GoZeroValue(p *ProgramDefinition, t *type_definition.TypeDefinition, required bool) string {
	if t == nil {
		return goNil
	}

	switch td := t.Interface().(type) {
	case *core.Reference:
		definition := resolveReference(p, td)

		if definition == nil {
			return goNil
		}

		if definition.enum {
			if required {
				return "0"
			}

			return goNil
		}

		if definition.typedef != nil {
			return GoZeroValue(definition.program, definition.typedef, required)
		}

		return goNil
	case *type_definition.ScalarType:
		if !required {
			return goNil
		}

		switch *td {
		case type_definition.ScalarType_String:
			return `""`
		case type_definition.ScalarType_Binary:
			return goNil
		case type_definition.ScalarType_Bool:
			return "false"
		case type_definition.ScalarType_I8,
			type_definition.ScalarType_I16,
			type_definition.ScalarType_I32,
			type_definition.ScalarType_I64,
			type_definition.ScalarType_Double:
			return "0"
		case type_definition.ScalarType_Void:
			return ""
		}
	case *type_definition.ListTypeDefinition,
		*type_definition.SetTypeDefinition,
		*type_definition.MapTypeDefinition:
		return goNil
	}

	return goNil
}

// GoType returns the Go type expression for a TypeDefinition, resolving named
// types against the current program and its includes.
func GoType(p *ProgramDefinition, t *type_definition.TypeDefinition, gs gocodegen.Scope, required bool) string {
	if t == nil {
		return goAny
	}

	switch td := t.Interface().(type) {
	case *core.Reference:
		definition := resolveReference(p, td)
		pointer := definition == nil || (!definition.enum && definition.typedef == nil) || !required
		pkgPath := ""

		if definition != nil {
			pkgPath = GoPackagePath(definition.program)
		}

		return goReferenceType(td, pkgPath, gs, pointer)
	case *type_definition.ScalarType:
		return goScalarType(*td, required)
	case *type_definition.ListTypeDefinition:
		return "[]" + GoType(p, td.ElementType, gs, true)
	case *type_definition.SetTypeDefinition:
		return "[]" + GoType(p, td.ElementType, gs, true)
	case *type_definition.MapTypeDefinition:
		return "map[" + GoType(p, td.KeyType, gs, true) + "]" + GoType(p, td.ValueType, gs, true)
	}

	return goAny
}

func goScalarType(t type_definition.ScalarType, required bool) string {
	pointer := ""

	if !required {
		pointer = "*"
	}

	switch t {
	case type_definition.ScalarType_String:
		return pointer + "string"
	case type_definition.ScalarType_Binary:
		return "[]byte"
	case type_definition.ScalarType_Bool:
		return pointer + "bool"
	case type_definition.ScalarType_I8:
		return pointer + "int8"
	case type_definition.ScalarType_I16:
		return pointer + "int16"
	case type_definition.ScalarType_I32:
		return pointer + "int32"
	case type_definition.ScalarType_I64:
		return pointer + "int64"
	case type_definition.ScalarType_Double:
		return pointer + "float64"
	case type_definition.ScalarType_Void:
		return ""
	}

	return goAny
}

type referenceDefinition struct {
	program *ProgramDefinition
	typedef *type_definition.TypeDefinition
	enum    bool
}

func resolveReference(p *ProgramDefinition, ref *core.Reference) *referenceDefinition {
	if p == nil {
		return nil
	}

	namespace := ref.GetNamespace_()

	if namespace == "" || hasNamespace(p, namespace) {
		if _, ok := p.Structs[ref.Name]; ok {
			return &referenceDefinition{program: p}
		}

		if _, ok := p.Enums[ref.Name]; ok {
			return &referenceDefinition{program: p, enum: true}
		}

		if td, ok := p.Typedefs[ref.Name]; ok {
			return &referenceDefinition{program: p, typedef: td}
		}
	}

	for _, include := range p.Includes {
		if definition := resolveReference(include, ref); definition != nil {
			return definition
		}
	}

	return nil
}

func hasNamespace(p *ProgramDefinition, namespace string) bool {
	for _, candidate := range p.Namespaces {
		if candidate == namespace {
			return true
		}
	}

	return false
}

func goReferenceType(ref *core.Reference, pkgPath string, gs gocodegen.Scope, pointer bool) string {
	name := gocodegen.Publicize(ref.Name)

	if gs.LocalPkg == "" || (ref.IsSetNamespace_() && ref.GetNamespace_() != gs.LocalPkg) {
		for _, inc := range gs.Includes {
			incPkgPath := inc.GoPkgPath

			if incPkgPath == "" {
				incPkgPath = strings.ReplaceAll(inc.Namespace, ".", "/")
			}

			if (pkgPath != "" && incPkgPath == pkgPath) ||
				(pkgPath == "" && inc.Namespace == ref.GetNamespace_()) {
				name = inc.PkgName + "." + name

				break
			}
		}
	}

	if pointer {
		return "*" + name
	}

	return name
}
