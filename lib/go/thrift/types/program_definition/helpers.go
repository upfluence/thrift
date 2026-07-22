package program_definition

import (
	"path/filepath"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/goscope"
	"github.com/upfluence/thrift/lib/go/thrift/types/type_definition"
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
func GoImportPath(p *ProgramDefinition, gs goscope.GoScope) string {
	if p.Stdlib {
		return gs.ThriftPkg + "/" + GoPackagePath(p)
	}

	return gs.ImportPkgPrefix + GoPackagePath(p)
}

// GoZeroValue returns the Go zero value expression for a TypeDefinition.
func GoZeroValue(t *type_definition.TypeDefinition, required bool) string {
	if t == nil {
		return "nil"
	}

	switch td := t.Interface().(type) {
	case *core.Reference:
		return "nil"
	case *type_definition.ScalarType:
		if !required {
			return "nil"
		}

		switch *td {
		case type_definition.ScalarType_String:
			return `""`
		case type_definition.ScalarType_Binary:
			return "nil"
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
		return "nil"
	}

	return "nil"
}


// references using the provided program scope and GoScope. If required is false,
// reference types are prefixed with "*" (optional pointer semantics).
func GoType(t *type_definition.TypeDefinition, gs goscope.GoScope, required bool) string {
	if t == nil {
		return "interface{}"
	}

	switch td := t.Interface().(type) {
	case *core.Reference:
		name := goCamelize(td.Name)

		if gs.LocalPkg == "" || (td.IsSetNamespace_() && td.GetNamespace_() != gs.LocalPkg) {
			for _, inc := range gs.Includes {
				if inc.Namespace == td.GetNamespace_() {
					return "*" + inc.PkgPath() + inc.PkgName + "." + name
				}
			}
		}

		return "*" + name
	case *type_definition.ScalarType:
		ptr := ""

		if !required {
			ptr = "*"
		}

		switch *td {
		case type_definition.ScalarType_String:
			return ptr + "string"
		case type_definition.ScalarType_Binary:
			return "[]byte"
		case type_definition.ScalarType_Bool:
			return ptr + "bool"
		case type_definition.ScalarType_I8:
			return ptr + "int8"
		case type_definition.ScalarType_I16:
			return ptr + "int16"
		case type_definition.ScalarType_I32:
			return ptr + "int32"
		case type_definition.ScalarType_I64:
			return ptr + "int64"
		case type_definition.ScalarType_Double:
			return ptr + "float64"
		case type_definition.ScalarType_Void:
			return ""
		}
	case *type_definition.ListTypeDefinition:
		return "[]" + GoType(td.ElementType, gs, true)
	case *type_definition.SetTypeDefinition:
		return "[]" + GoType(td.ElementType, gs, true)
	case *type_definition.MapTypeDefinition:
		return "map[" + GoType(td.KeyType, gs, true) + "]" + GoType(td.ValueType, gs, true)
	}

	return "interface{}"
}

// goCamelize converts a snake_case or kebab-case identifier to CamelCase.
func goCamelize(s string) string {
	var result []byte
	upper := true

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '_' || c == '-' {
			upper = true
			continue
		}

		if upper && c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}

		result = append(result, c)
		upper = false
	}

	return string(result)
}
