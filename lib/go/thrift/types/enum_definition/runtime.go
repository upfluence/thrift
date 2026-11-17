package enum_definition

import (
	"fmt"

	"github.com/upfluence/thrift/lib/go/thrift"
)

// Runtime returns the enum as read by the libraries at runtime, see
// annotation_definition.AnnotationDefinition.Runtime. namespace is the
// namespace of the program defining the enum.
func (ed *EnumDefinition) Runtime(namespace string) (thrift.EnumDefinition, error) {
	ad, err := ed.GetAnnotation().Runtime()

	if err != nil {
		return thrift.EnumDefinition{}, fmt.Errorf("enum %s: %w", ed.GetAnnotation().GetName(), err)
	}

	res := thrift.EnumDefinition{AnnotatedDefinition: ad, Namespace: namespace}

	for _, v := range ed.GetValues() {
		vad, err := v.GetAnnotation().Runtime()

		if err != nil {
			return thrift.EnumDefinition{}, fmt.Errorf(
				"enum %s: value %s: %w",
				ed.GetAnnotation().GetName(),
				v.GetAnnotation().GetName(),
				err,
			)
		}

		res.Values = append(
			res.Values,
			thrift.EnumValueDefinition{AnnotatedDefinition: vad, Value: int64(v.GetID())},
		)
	}

	return res, nil
}
