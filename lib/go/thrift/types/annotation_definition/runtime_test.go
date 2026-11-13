package annotation_definition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation/text"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/value"
)

const typeName = "Type"

func structuredAnnotation(namespace, name string, fields map[string]*value.Value) *annotation_definition.StructuredAnnotationDefinition {
	return &annotation_definition.StructuredAnnotationDefinition{
		Type:  &core.Reference{Namespace_: &namespace, Name: name},
		Value: &value.Value{StructValue: &value.StructValue{Fields: fields}},
	}
}

func TestRuntime(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    *annotation_definition.AnnotationDefinition
		want    thrift.AnnotatedDefinition
		wantErr bool
	}{
		{name: "nil definition"},
		{
			name: "decoded annotations",
			have: &annotation_definition.AnnotationDefinition{
				Name:              typeName,
				LegacyAnnotations: map[string]string{"go.tag": "x"},
				StructuredAnnotations: []*annotation_definition.StructuredAnnotationDefinition{
					structuredAnnotation(
						text.Namespace,
						"EnumFormat",
						map[string]*value.Value{
							"format":              value.EncodeIntegerValue(int64(text.Case_KebabCase)),
							"trim_prefixes":       {ListValue: &value.ListValue{Values: []*value.Value{value.EncodeStringValue("Media")}}},
							"accept_thrift_names": value.EncodeIntegerValue(1),
						},
					),
					structuredAnnotation("unknown.namespace", "Foo", nil),
				},
			},
			want: thrift.AnnotatedDefinition{
				Name:              typeName,
				LegacyAnnotations: map[string]string{"go.tag": "x"},
				StructuredAnnotations: []thrift.RegistrableStruct{
					&text.EnumFormat{
						Format:            text.Case_KebabCase,
						TrimPrefixes:      []string{"Media"},
						AcceptThriftNames: thrift.BoolPtr(true),
					},
				},
			},
		},
		{
			name: "invalid value",
			have: &annotation_definition.AnnotationDefinition{
				Name: typeName,
				StructuredAnnotations: []*annotation_definition.StructuredAnnotationDefinition{
					structuredAnnotation(
						text.Namespace,
						"Format",
						map[string]*value.Value{"pattern": value.EncodeIntegerValue(1)},
					),
				},
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.have.Runtime()

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
