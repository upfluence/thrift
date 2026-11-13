package thrift

import (
	"reflect"

	"github.com/upfluence/thrift/lib/go/thrift/internal/reflection"
)

type AnnotatedDefinition = reflection.AnnotatedDefinition
type FieldDefinition = reflection.FieldDefinition
type Requiredness = reflection.Requiredness
type StructDefinition = reflection.StructDefinition
type EnumValueDefinition = reflection.EnumValueDefinition
type EnumDefinition = reflection.EnumDefinition
type FunctionDefinition = reflection.FunctionDefinition
type ServiceDefinition = reflection.ServiceDefinition
type RegistrableStruct = reflection.RegistrableStruct

const (
	RequirednessDefault  = reflection.RequirednessDefault
	RequirednessRequired = reflection.RequirednessRequired
	RequirednessOptional = reflection.RequirednessOptional
)

func RegisterService(def ServiceDefinition) {
	reflection.RegisterService(def)
}

func GetServiceDefinition(n string) (ServiceDefinition, bool) {
	return reflection.GetServiceDefinition(n)
}

func RegisterStruct(rs RegistrableStruct) {
	reflection.RegisterStruct(rs)
}

func StructType(n string) (reflect.Type, bool) {
	return reflection.StructType(n)
}

func ExtractCanonicalNames(ns string, def AnnotatedDefinition) []string {
	return reflection.ExtractCanonicalNames(ns, def)
}

// FindAnnotations returns the structured annotations of def of type T, in
// declaration order.
func FindAnnotations[T RegistrableStruct](def AnnotatedDefinition) []T {
	var res []T

	for _, a := range def.StructuredAnnotations {
		if v, ok := a.(T); ok {
			res = append(res, v)
		}
	}

	return res
}
