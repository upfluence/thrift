package program_definition

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
