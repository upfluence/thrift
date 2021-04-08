package thrift

import (
	"fmt"
	"reflect"
	"sync"
)

var defaultStructTypeRegistry = &structTypeRegistry{
	types: make(map[string]reflect.Type),
	defs:  make(map[string]StructDefinition),
}

type structTypeRegistry struct {
	sync.RWMutex

	defs  map[string]StructDefinition
	types map[string]reflect.Type
}

func (str *structTypeRegistry) registerStructType(rs RegistrableStruct) {
	t := reflect.TypeOf(rs)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	sd := rs.StructDefinition()
	n := sd.CanonicalName()

	str.Lock()
	defer str.Unlock()

	str.types[n] = t
	str.defs[n] = sd
}

func (str *structTypeRegistry) structType(n string) (reflect.Type, bool) {
	str.Lock()
	defer str.Unlock()

	t, ok := str.types[n]

	return t, ok
}

type StructDefinition struct {
	Namespace string
	Name      string
}

func (sd StructDefinition) CanonicalName() string {
	return fmt.Sprintf("%s.%s", sd.Namespace, sd.Name)
}

type RegistrableStruct interface {
	TStruct
	StructDefinition() StructDefinition
}

func RegisterStruct(rs RegistrableStruct) {
	defaultStructTypeRegistry.registerStructType(rs)
}

func StructType(n string) (reflect.Type, bool) {
	return defaultStructTypeRegistry.structType(n)
}
