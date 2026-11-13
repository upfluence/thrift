package annotation_definition

import (
	"fmt"
	"reflect"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/value"
)

// Runtime returns the annotations as read by the libraries at runtime. The
// structured annotations are decoded into their generated types, found in
// the struct registry: the annotations whose type is not linked in the
// binary are skipped.
func (ad *AnnotationDefinition) Runtime() (thrift.AnnotatedDefinition, error) {
	if ad == nil {
		return thrift.AnnotatedDefinition{}, nil
	}

	res := thrift.AnnotatedDefinition{
		Name:              ad.GetName(),
		LegacyAnnotations: ad.GetLegacyAnnotations(),
	}

	for _, sa := range ad.GetStructuredAnnotations() {
		name := sa.GetType().GetNamespace_() + "." + sa.GetType().GetName()
		t, ok := thrift.StructType(name)

		if !ok {
			continue
		}

		a, ok := reflect.TypeAssert[thrift.RegistrableStruct](reflect.New(t))

		if !ok {
			continue
		}

		if err := value.DecodeValue(sa.GetValue(), a); err != nil {
			return thrift.AnnotatedDefinition{}, fmt.Errorf("annotation %s: %w", name, err)
		}

		res.StructuredAnnotations = append(res.StructuredAnnotations, a)
	}

	return res, nil
}
