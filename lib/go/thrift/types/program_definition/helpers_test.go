package program_definition

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/gocodegen"
	"github.com/upfluence/thrift/lib/go/thrift/types/type_definition"
)

func TestGoPackageName(t *testing.T) {
	for _, tt := range []struct {
		name        string
		haveProgram *ProgramDefinition
		want        string
	}{
		{
			name: "go namespace takes priority over wildcard",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"go": "github.com/foo/bar.baz",
					"*":  "foo.bar.baz",
				},
				Path: "some/other.thrift",
			},
			want: "baz",
		},
		{
			name: "wildcard namespace used when no go namespace",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"*": "foo.bar.qux",
				},
				Path: "some/other.thrift",
			},
			want: "qux",
		},
		{
			name: "falls back to filename stem when no namespace",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{},
				Path:       "some/my_service.thrift",
			},
			want: "my_service",
		},
		{
			name: "single-segment go namespace",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"go": "mypkg",
				},
				Path: "other.thrift",
			},
			want: "mypkg",
		},
		{
			name: "empty go namespace falls through to wildcard",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"go": "",
					"*":  "types.service_definition",
				},
				Path: "types/service_definition.thrift",
			},
			want: "service_definition",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := GoPackageName(tt.haveProgram)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoPackagePath(t *testing.T) {
	for _, tt := range []struct {
		name        string
		haveProgram *ProgramDefinition
		want        string
	}{
		{
			name: "wildcard namespace dots converted to path separators",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"*": "types.program_definition",
				},
				Path: "some/other.thrift",
			},
			want: "types/program_definition",
		},
		{
			name: "falls back to filename stem when no namespace",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{},
				Path:       "some/my_service.thrift",
			},
			want: "my_service",
		},
		{
			name: "go namespace takes priority over wildcard",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"go": "foo.bar",
					"*":  "baz.qux",
				},
				Path: "other.thrift",
			},
			want: "foo/bar",
		},
		{
			name: "empty go namespace falls through to wildcard",
			haveProgram: &ProgramDefinition{
				Namespaces: map[string]string{
					"go": "",
					"*":  "types.core",
				},
				Path: "types/core.thrift",
			},
			want: "types/core",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := GoPackagePath(tt.haveProgram)

			assert.Equal(t, tt.want, got)
		})
	}
}

func scalarTypeDef(s type_definition.ScalarType) *type_definition.TypeDefinition {
	return &type_definition.TypeDefinition{ScalarType: &s}
}

func refTypeDef(namespace, name string) *type_definition.TypeDefinition {
	ref := core.NewReference()
	ref.Name = name

	if namespace != "" {
		ref.SetNamespace_(namespace)
	}

	return &type_definition.TypeDefinition{ReferenceType: ref}
}

func listTypeDef(elem *type_definition.TypeDefinition) *type_definition.TypeDefinition {
	return &type_definition.TypeDefinition{
		ListType: &type_definition.ListTypeDefinition{ElementType: elem},
	}
}

func mapTypeDef(key, val *type_definition.TypeDefinition) *type_definition.TypeDefinition {
	return &type_definition.TypeDefinition{
		MapType: &type_definition.MapTypeDefinition{KeyType: key, ValueType: val},
	}
}

func buildGoScope(localPkg string, includes ...gocodegen.Include) gocodegen.Scope {
	gs := gocodegen.Scope{
		ThriftPkg:       "thrift",
		ImportPkgPrefix: "github.com/upfluence/",
		LocalPkg:        localPkg,
	}

	incs := make([]gocodegen.Include, 0, len(includes))

	for _, inc := range includes {
		incs = append(incs, gs.NewInclude(inc.Namespace, inc.PkgName, inc.Stdlib))
	}

	gs.Includes = incs

	return gs
}

func TestGoType(t *testing.T) {
	for _, tc := range []struct {
		name     string
		haveType *type_definition.TypeDefinition
		haveGs   gocodegen.Scope
		haveReq  bool
		want     string
	}{
		{
			name:    "nil type",
			haveGs:  buildGoScope("sigma.holder"),
			haveReq: true,
			want:    "interface{}",
		},
		{
			name:     "required string scalar",
			haveType: scalarTypeDef(type_definition.ScalarType_String),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  true,
			want:     "string",
		},
		{
			name:     "optional string scalar",
			haveType: scalarTypeDef(type_definition.ScalarType_String),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  false,
			want:     "*string",
		},
		{
			name:     "required i64 scalar",
			haveType: scalarTypeDef(type_definition.ScalarType_I64),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  true,
			want:     "int64",
		},
		{
			name:     "binary scalar always bare",
			haveType: scalarTypeDef(type_definition.ScalarType_Binary),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  false,
			want:     "[]byte",
		},
		{
			name:     "void scalar returns empty string",
			haveType: scalarTypeDef(type_definition.ScalarType_Void),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  true,
			want:     "",
		},
		{
			name:     "local reference always pointer",
			haveType: refTypeDef("", "Holder"),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  true,
			want:     "*Holder",
		},
		{
			name:     "foreign reference resolves with import prefix",
			haveType: refTypeDef("base.page", "Page"),
			haveGs: buildGoScope("sigma.holder",
				gocodegen.Include{Namespace: "base.page", PkgName: "page"},
			),
			haveReq: true,
			want:    "*page.Page",
		},
		{
			name:     "reference namespace matches localPkg returns bare name",
			haveType: refTypeDef("sigma.holder", "Holder"),
			haveGs: buildGoScope("sigma.holder",
				gocodegen.Include{Namespace: "sigma.holder", PkgName: "holder"},
			),
			haveReq: true,
			want:    "*Holder",
		},
		{
			name:     "empty localPkg always resolves prefix even for matching namespace",
			haveType: refTypeDef("sigma.holder", "Holder"),
			haveGs: buildGoScope("",
				gocodegen.Include{Namespace: "sigma.holder", PkgName: "holder"},
			),
			haveReq: true,
			want:    "*holder.Holder",
		},
		{
			name:     "stdlib reference resolves with thrift pkg prefix",
			haveType: refTypeDef("types.core", "Exception"),
			haveGs: buildGoScope("sigma.holder",
				gocodegen.Include{Namespace: "types.core", PkgName: "core", Stdlib: true},
			),
			haveReq: true,
			want:    "*core.Exception",
		},
		{
			name:     "list of required scalars",
			haveType: listTypeDef(scalarTypeDef(type_definition.ScalarType_I32)),
			haveGs:   buildGoScope("sigma.holder"),
			haveReq:  true,
			want:     "[]int32",
		},
		{
			name: "map with string key and reference value",
			haveType: mapTypeDef(
				scalarTypeDef(type_definition.ScalarType_String),
				refTypeDef("", "Holder"),
			),
			haveGs:  buildGoScope("sigma.holder"),
			haveReq: true,
			want:    "map[string]*Holder",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := GoType(tc.haveType, tc.haveGs, tc.haveReq)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGoZeroValue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		haveType *type_definition.TypeDefinition
		haveReq  bool
		want     string
	}{
		{
			name:    "nil type",
			haveReq: true,
			want:    "nil",
		},
		{
			name:     "required string",
			haveType: scalarTypeDef(type_definition.ScalarType_String),
			haveReq:  true,
			want:     `""`,
		},
		{
			name:     "optional string",
			haveType: scalarTypeDef(type_definition.ScalarType_String),
			haveReq:  false,
			want:     "nil",
		},
		{
			name:     "required bool",
			haveType: scalarTypeDef(type_definition.ScalarType_Bool),
			haveReq:  true,
			want:     "false",
		},
		{
			name:     "required i32",
			haveType: scalarTypeDef(type_definition.ScalarType_I32),
			haveReq:  true,
			want:     "0",
		},
		{
			name:     "required i64",
			haveType: scalarTypeDef(type_definition.ScalarType_I64),
			haveReq:  true,
			want:     "0",
		},
		{
			name:     "required double",
			haveType: scalarTypeDef(type_definition.ScalarType_Double),
			haveReq:  true,
			want:     "0",
		},
		{
			name:     "binary always nil",
			haveType: scalarTypeDef(type_definition.ScalarType_Binary),
			haveReq:  true,
			want:     "nil",
		},
		{
			name:     "void returns empty",
			haveType: scalarTypeDef(type_definition.ScalarType_Void),
			haveReq:  true,
			want:     "",
		},
		{
			name:     "reference always nil",
			haveType: refTypeDef("", "Holder"),
			haveReq:  true,
			want:     "nil",
		},
		{
			name:     "list always nil",
			haveType: listTypeDef(scalarTypeDef(type_definition.ScalarType_String)),
			haveReq:  true,
			want:     "nil",
		},
		{
			name: "map always nil",
			haveType: mapTypeDef(
				scalarTypeDef(type_definition.ScalarType_String),
				refTypeDef("", "Holder"),
			),
			haveReq: true,
			want:    "nil",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := GoZeroValue(tc.haveType, tc.haveReq)

			assert.Equal(t, tc.want, got)
		})
	}
}
