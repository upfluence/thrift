package thrift

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	defaultStructTypeRegistry = &structTypeRegistry{
		types: make(map[string]reflect.Type),
		defs:  make(map[string]StructDefinition),
	}

	defaultServiceRegistry = &serviceRegistry{
		defs: make(map[string]ServiceDefinition),
	}
)

type serviceRegistry struct {
	mu sync.RWMutex

	defs map[string]ServiceDefinition
}

func (str *serviceRegistry) registerService(sd ServiceDefinition) {
	str.mu.Lock()
	defer str.mu.Unlock()

	str.defs[sd.CanonicalName()] = sd
}

type structTypeRegistry struct {
	mu sync.RWMutex

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

	str.mu.Lock()
	defer str.mu.Unlock()

	str.types[n] = t
	str.defs[n] = sd
}

func (str *structTypeRegistry) structType(n string) (reflect.Type, bool) {
	str.mu.Lock()
	defer str.mu.Unlock()

	t, ok := str.types[n]

	return t, ok
}

type AnnotatedDefintion struct {
	Name string

	LegacyAnnotations     map[string]string
	SturcturedAnnotations []RegistrableStruct
}

type FieldDefinition struct {
	AnnotatedDefintion
}

type StructDefinition struct {
	AnnotatedDefintion

	Namespace string

	IsException bool
	IsUnion     bool

	Fields []FieldDefinition
}

func (sd StructDefinition) CanonicalName() string {
	return fmt.Sprintf("%s.%s", sd.Namespace, sd.Name)
}

type FunctionDefinition struct {
	AnnotatedDefintion

	IsOneway bool

	Args   StructDefinition
	Result *StructDefinition
}

type ServiceDefinition struct {
	AnnotatedDefintion

	Namespace string

	Functions []FunctionDefinition
}

func (sd ServiceDefinition) CanonicalName() string {
	return fmt.Sprintf("%s.%s", sd.Namespace, sd.Name)
}

type RegistrableStruct interface {
	TStruct
	StructDefinition() StructDefinition
}

func RegisterService(def ServiceDefinition) {
	defaultServiceRegistry.registerService(def)
}

func GetServiceDefinition(n string) (ServiceDefinition, bool) {
	def, ok := defaultServiceRegistry.defs[n]

	return def, ok
}

func RegisterStruct(rs RegistrableStruct) {
	defaultStructTypeRegistry.registerStructType(rs)
}

func StructType(n string) (reflect.Type, bool) {
	return defaultStructTypeRegistry.structType(n)
}
