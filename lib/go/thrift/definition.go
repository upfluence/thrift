package thrift

import (
	"reflect"

	"github.com/upfluence/thrift/lib/go/thrift/internal/reflection"
)

type AnnotatedDefinition = reflection.AnnotatedDefinition
type FieldDefinition = reflection.FieldDefinition
type StructDefinition = reflection.StructDefinition
type FunctionDefinition = reflection.FunctionDefinition
type ServiceDefinition = reflection.ServiceDefinition
type RegistrableStruct = reflection.RegistrableStruct

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
