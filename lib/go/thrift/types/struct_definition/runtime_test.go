package struct_definition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation/text"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/struct_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/value"
)

func annotation(name string) *annotation_definition.AnnotationDefinition {
	return &annotation_definition.AnnotationDefinition{Name: name}
}

func TestRuntime(t *testing.T) {
	namespace := text.Namespace

	sd := struct_definition.StructDefinition{
		Annotation: &annotation_definition.AnnotationDefinition{
			Name: "Ownership",
			StructuredAnnotations: []*annotation_definition.StructuredAnnotationDefinition{
				{
					Type: &core.Reference{Namespace_: &namespace, Name: "Format"},
					Value: &value.Value{
						StructValue: &value.StructValue{
							Fields: map[string]*value.Value{
								"pattern": value.EncodeStringValue("{type}:{entity_id}"),
							},
						},
					},
				},
			},
		},
		Kind: struct_definition.StructKind_Struct,
		Fields: []*struct_definition.FieldDefinition{
			{Annotation: annotation("type"), ID: 1, Requiredness: struct_definition.Requiredness_Required},
			{Annotation: annotation("entity_id"), ID: 2, Requiredness: struct_definition.Requiredness_Unknown},
			{Annotation: annotation("note"), ID: 3, Requiredness: struct_definition.Requiredness_Optional},
		},
	}

	got, err := sd.Runtime("base.ownership")

	require.NoError(t, err)
	assert.Equal(
		t,
		thrift.StructDefinition{
			AnnotatedDefinition: thrift.AnnotatedDefinition{
				Name: "Ownership",
				StructuredAnnotations: []thrift.RegistrableStruct{
					&text.Format{Pattern: "{type}:{entity_id}"},
				},
			},
			Namespace: "base.ownership",
			Fields: []thrift.FieldDefinition{
				{
					AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "type"},
					Requiredness:        thrift.RequirednessRequired,
				},
				{AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "entity_id"}},
				{
					AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "note"},
					Requiredness:        thrift.RequirednessOptional,
				},
			},
		},
		got,
	)
}

func TestRuntimeKind(t *testing.T) {
	for _, tt := range []struct {
		name string
		have struct_definition.StructKind
		want thrift.StructDefinition
	}{
		{name: "struct", have: struct_definition.StructKind_Struct},
		{name: "exception", have: struct_definition.StructKind_Exception, want: thrift.StructDefinition{IsException: true}},
		{name: "union", have: struct_definition.StructKind_Union, want: thrift.StructDefinition{IsUnion: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&struct_definition.StructDefinition{Kind: tt.have}).Runtime("")

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
