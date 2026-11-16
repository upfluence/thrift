package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift/types/program_definition"
)

func TestBuildGoScopeUsesGoNamespaceForIncludes(t *testing.T) {
	req := &GenerateCodeRequest{
		Program: &program_definition.ProgramDefinition{
			Namespaces: map[string]string{"*": "connectome.consumer"},
			Includes: []*program_definition.ProgramDefinition{
				{
					Namespaces: map[string]string{
						"*":  "connectome.internal.processor",
						"go": "connectome.system.processor",
					},
				},
			},
		},
		Options: map[string]string{"package_prefix": "github.com/upfluence/"},
	}

	got := BuildGoScope(req)

	require.Len(t, got.Includes, 1)
	assert.Equal(t, "connectome.internal.processor", got.Includes[0].Namespace)
	assert.Equal(t, "processor", got.Includes[0].PkgName)
	assert.Equal(t, "github.com/upfluence/connectome/system/processor", got.Includes[0].PkgPath())
}

func TestBuildGoScopeImportRewrites(t *testing.T) {
	const (
		identityOption = "import_rewrite:github.com/upfluence/identity"
		identityV2     = "github.com/upfluence/identity/v2"
	)

	for _, tt := range []struct {
		name        string
		haveOptions map[string]string
		want        map[string]string
	}{
		{
			name:        "no rewrite options",
			haveOptions: map[string]string{"package_prefix": "github.com/upfluence/"},
		},
		{
			name: "single rewrite option",
			haveOptions: map[string]string{
				"package_prefix": "github.com/upfluence/",
				identityOption:   identityV2,
			},
			want: map[string]string{"github.com/upfluence/identity": identityV2},
		},
		{
			name: "multiple rewrite options",
			haveOptions: map[string]string{
				identityOption: identityV2,
				"import_rewrite:github.com/upfluence/scope": "github.com/upfluence/scope/v3",
				"thrift_import": "github.com/upfluence/thrift/lib/go/thrift",
			},
			want: map[string]string{
				"github.com/upfluence/identity": identityV2,
				"github.com/upfluence/scope":    "github.com/upfluence/scope/v3",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := &GenerateCodeRequest{
				Program: &program_definition.ProgramDefinition{
					Namespaces: map[string]string{"*": "identity.types.user.v1.user"},
				},
				Options: tt.haveOptions,
			}

			got := BuildGoScope(req)

			assert.Equal(t, tt.want, got.ImportRewrites)
		})
	}
}

func TestBuildGoScopeAppliesImportRewritesToIncludes(t *testing.T) {
	req := &GenerateCodeRequest{
		Program: &program_definition.ProgramDefinition{
			Namespaces: map[string]string{"*": "identity.types.user.v1.user"},
			Includes: []*program_definition.ProgramDefinition{
				{Namespaces: map[string]string{"*": "identity.types.scope.v1.scope"}},
				{Namespaces: map[string]string{"*": "identityutil.types.util"}},
			},
		},
		Options: map[string]string{
			"package_prefix": "github.com/upfluence/",
			"import_rewrite:github.com/upfluence/identity": "github.com/upfluence/identity/v2",
		},
	}

	got := BuildGoScope(req)

	require.Len(t, got.Includes, 2)
	assert.Equal(
		t,
		[]string{
			"github.com/upfluence/identity/v2/types/scope/v1/scope",
			"github.com/upfluence/identityutil/types/util",
		},
		[]string{got.Includes[0].PkgPath(), got.Includes[1].PkgPath()},
	)
}

func TestBuildGoScopeUsesThriftImportForStdlibIncludes(t *testing.T) {
	req := &GenerateCodeRequest{
		Program: &program_definition.ProgramDefinition{
			Namespaces: map[string]string{"*": "identity.fetcher"},
			Includes: []*program_definition.ProgramDefinition{
				{
					Namespaces: map[string]string{"*": "types.known.timestamp"},
					Stdlib:     true,
				},
			},
		},
		Options: map[string]string{
			"package_prefix": "github.com/upfluence/",
			"thrift_import":  "github.com/upfluence/thrift/lib/go/thrift",
		},
	}

	got := BuildGoScope(req)

	require.Len(t, got.Includes, 1)
	assert.Equal(t, "thrift", got.ThriftPkg())
	assert.Equal(
		t,
		"github.com/upfluence/thrift/lib/go/thrift/types/known/timestamp",
		got.Includes[0].PkgPath(),
	)
}
