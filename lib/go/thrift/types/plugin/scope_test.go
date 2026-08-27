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
