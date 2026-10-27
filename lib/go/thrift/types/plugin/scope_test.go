package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/program_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/type_definition"
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

func TestBuildGoScopeDedupsHomonymousIncludes(t *testing.T) {
	ref := core.NewReference()
	ref.Name = "Post"
	ref.SetNamespace_("base.post")

	program := &program_definition.ProgramDefinition{
		Namespaces: map[string]string{"*": "influencer.apis.post.fetcher.v2.post_fetcher"},
		Includes: []*program_definition.ProgramDefinition{
			{Namespaces: map[string]string{"*": "influencer.types.post.v2.post"}},
			{Namespaces: map[string]string{"*": "base.post"}},
			{Namespaces: map[string]string{"*": "foo.thrift"}},
			{Namespaces: map[string]string{"*": "foo.error"}},
		},
	}

	got := BuildGoScope(&GenerateCodeRequest{
		Program: program,
		Options: map[string]string{"package_prefix": "github.com/upfluence/"},
	})

	require.Len(t, got.Includes, 4)
	assert.Equal(t, "post", got.Includes[0].PkgName)
	assert.Equal(t, "post0", got.Includes[1].PkgName)
	assert.Equal(t, "thrift0", got.Includes[2].PkgName)
	assert.Equal(t, "error0", got.Includes[3].PkgName)
	assert.Equal(
		t,
		"*post0.Post",
		program_definition.GoType(
			program,
			&type_definition.TypeDefinition{ReferenceType: ref},
			got,
			true,
		),
	)
}
