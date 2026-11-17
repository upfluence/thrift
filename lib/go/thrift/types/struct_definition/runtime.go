package struct_definition

import (
	"fmt"

	"github.com/upfluence/thrift/lib/go/thrift"
)

var requirednesses = map[Requiredness]thrift.Requiredness{
	Requiredness_Required: thrift.RequirednessRequired,
	Requiredness_Optional: thrift.RequirednessOptional,
}

// Runtime returns the struct as read by the libraries at runtime, see
// annotation_definition.AnnotationDefinition.Runtime. namespace is the
// namespace of the program defining the struct.
func (sd *StructDefinition) Runtime(namespace string) (thrift.StructDefinition, error) {
	ad, err := sd.GetAnnotation().Runtime()

	if err != nil {
		return thrift.StructDefinition{}, fmt.Errorf("struct %s: %w", sd.GetAnnotation().GetName(), err)
	}

	res := thrift.StructDefinition{
		AnnotatedDefinition: ad,
		Namespace:           namespace,
		IsException:         sd.GetKind() == StructKind_Exception,
		IsUnion:             sd.GetKind() == StructKind_Union,
	}

	for _, f := range sd.GetFields() {
		fad, err := f.GetAnnotation().Runtime()

		if err != nil {
			return thrift.StructDefinition{}, fmt.Errorf(
				"struct %s: field %s: %w",
				sd.GetAnnotation().GetName(),
				f.GetAnnotation().GetName(),
				err,
			)
		}

		res.Fields = append(
			res.Fields,
			thrift.FieldDefinition{
				AnnotatedDefinition: fad,
				Requiredness:        requirednesses[f.GetRequiredness()],
			},
		)
	}

	return res, nil
}
