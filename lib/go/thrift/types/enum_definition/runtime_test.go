package enum_definition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation/text"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/core"
	"github.com/upfluence/thrift/lib/go/thrift/types/enum_definition"
	"github.com/upfluence/thrift/lib/go/thrift/types/value"
)

func TestRuntime(t *testing.T) {
	namespace := text.Namespace

	ed := enum_definition.EnumDefinition{
		Annotation: &annotation_definition.AnnotationDefinition{Name: "Type"},
		Values: []*enum_definition.EnumValueDefinition{
			{Annotation: &annotation_definition.AnnotationDefinition{Name: "Company"}},
			{
				Annotation: &annotation_definition.AnnotationDefinition{
					Name: "GoogleOAuth",
					StructuredAnnotations: []*annotation_definition.StructuredAnnotationDefinition{
						{
							Type: &core.Reference{Namespace_: &namespace, Name: "Name"},
							Value: &value.Value{
								StructValue: &value.StructValue{
									Fields: map[string]*value.Value{"value": value.EncodeStringValue("gmail")},
								},
							},
						},
					},
				},
				ID: 3,
			},
		},
	}

	got, err := ed.Runtime("base.ownership")

	require.NoError(t, err)
	assert.Equal(
		t,
		thrift.EnumDefinition{
			AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "Type"},
			Namespace:           "base.ownership",
			Values: []thrift.EnumValueDefinition{
				{AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "Company"}},
				{
					AnnotatedDefinition: thrift.AnnotatedDefinition{
						Name:                  "GoogleOAuth",
						StructuredAnnotations: []thrift.RegistrableStruct{&text.Name{Value: "gmail"}},
					},
					Value: 3,
				},
			},
		},
		got,
	)

	c, err := text.CompileEnum(got)
	require.NoError(t, err)

	s, err := c.Format(3)
	require.NoError(t, err)
	assert.Equal(t, "gmail", s)
}
