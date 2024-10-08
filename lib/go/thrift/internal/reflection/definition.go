package reflection

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	defaultCanonicalNameExtractor = &canonicalNameExtractors{
		es: []CanonicalNameExtractor{
			CanonicalNameExtractorFunc(func(ns string, def AnnotatedDefinition) []string {
				return []string{fmt.Sprintf("%s.%s", ns, def.Name)}
			}),
		},
	}

	defaultStructTypeRegistry = &structTypeRegistry{
		types: make(map[string]reflect.Type),
		ext:   defaultCanonicalNameExtractor,
	}

	defaultServiceRegistry = &serviceRegistry{
		defs: make(map[string]ServiceDefinition),
		ext:  defaultCanonicalNameExtractor,
	}
)

type CanonicalNameExtractor interface {
	Extract(string, AnnotatedDefinition) []string
}

type CanonicalNameExtractorFunc func(string, AnnotatedDefinition) []string

func (fn CanonicalNameExtractorFunc) Extract(ns string, def AnnotatedDefinition) []string {
	return fn(ns, def)
}

type canonicalNameExtractors struct {
	es []CanonicalNameExtractor
	mu sync.RWMutex
}

func (fns *canonicalNameExtractors) registerExtractor(e CanonicalNameExtractor) {
	fns.mu.Lock()
	defer fns.mu.Unlock()

	fns.es = append(fns.es, e)
}

func (fns *canonicalNameExtractors) Extract(ns string, def AnnotatedDefinition) []string {
	var res []string

	fns.mu.RLock()
	defer fns.mu.RUnlock()

	for _, e := range fns.es {
		res = append(res, e.Extract(ns, def)...)
	}

	return res
}

func RegisterCanonicalNameExtractor(e CanonicalNameExtractor) {
	defaultCanonicalNameExtractor.registerExtractor(e)
}

func ExtractCanonicalNames(ns string, def AnnotatedDefinition) []string {
	return defaultCanonicalNameExtractor.Extract(ns, def)
}

type serviceRegistry struct {
	mu sync.RWMutex

	ext  CanonicalNameExtractor
	defs map[string]ServiceDefinition
}

func (str *serviceRegistry) registerService(sd ServiceDefinition) {
	ns := str.ext.Extract(sd.Namespace, sd.AnnotatedDefinition)

	str.mu.Lock()
	defer str.mu.Unlock()

	for _, n := range ns {
		str.defs[n] = sd
	}
}

type structTypeRegistry struct {
	mu sync.RWMutex

	ext CanonicalNameExtractor

	types map[string]reflect.Type
}

func (str *structTypeRegistry) registerStructType(rs RegistrableStruct) {
	t := reflect.TypeOf(rs)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	sd := rs.StructDefinition()
	ns := str.ext.Extract(sd.Namespace, sd.AnnotatedDefinition)

	str.mu.Lock()
	defer str.mu.Unlock()

	for _, n := range ns {
		str.types[n] = t
	}
}

func (str *structTypeRegistry) structType(n string) (reflect.Type, bool) {
	str.mu.Lock()
	defer str.mu.Unlock()

	t, ok := str.types[n]

	return t, ok
}

type AnnotatedDefinition struct {
	Name string

	LegacyAnnotations     map[string]string
	StructuredAnnotations []RegistrableStruct
}

type FieldDefinition struct {
	AnnotatedDefinition
}

type StructDefinition struct {
	AnnotatedDefinition

	Namespace string

	// Deprecated: Use the Name in AnnotatedDefinition, this is here for
	// backward compatibility for previously generated code.
	Name string

	IsException bool
	IsUnion     bool

	Fields []FieldDefinition
}

func (sd StructDefinition) name() string {
	if sd.AnnotatedDefinition.Name != "" {
		return sd.AnnotatedDefinition.Name
	}

	return sd.Name
}

func (sd StructDefinition) CanonicalName() string {
	return fmt.Sprintf("%s.%s", sd.Namespace, sd.name())
}

type FunctionDefinition struct {
	AnnotatedDefinition

	IsOneway bool

	Args   StructDefinition
	Result *StructDefinition
}

type ServiceDefinition struct {
	AnnotatedDefinition

	Namespace string

	Functions []FunctionDefinition
}

func (sd ServiceDefinition) CanonicalName() string {
	return fmt.Sprintf("%s.%s", sd.Namespace, sd.Name)
}

type RegistrableStruct interface {
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
