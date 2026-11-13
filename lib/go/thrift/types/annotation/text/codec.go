// Package text holds the text annotations, see types/annotation/text.thrift,
// and encodes the annotated structs and enums as their text form. It accepts
// and produces exactly the same texts as Thrift::Types::Annotation::Text of
// the ruby library.
package text

import (
	"reflect"
	"sync"

	"github.com/upfluence/errors"

	"github.com/upfluence/thrift/lib/go/thrift"
)

var (
	// ErrInvalidDefinition is returned when the text annotations of a type
	// are invalid, or when the type has none.
	ErrInvalidDefinition = errors.New("text: invalid definition")
	// ErrInvalidFormat is returned when a text can not be decoded.
	ErrInvalidFormat = errors.New("text: invalid format")
	// ErrInvalidValue is returned when a value has no text form.
	ErrInvalidValue = errors.New("text: invalid value")
)

func invalidDefinition(msg string, args ...any) error {
	return errors.Wrapf(ErrInvalidDefinition, msg, args...)
}

// findAnnotation returns the structured annotation of type T of def, nil if
// def has none.
func findAnnotation[T thrift.RegistrableStruct](def thrift.AnnotatedDefinition) (T, error) {
	var zero T

	switch as := thrift.FindAnnotations[T](def); len(as) {
	case 0:
		return zero, nil
	case 1:
		return as[0], nil
	default:
		return zero, invalidDefinition(
			"%s is set twice",
			as[0].StructDefinition().CanonicalName(),
		)
	}
}

type cacheEntry[T any] struct {
	v   T
	err error
}

// cache memoizes the codecs by type. Concurrent misses compile the codec
// several times, which is harmless.
type cache[T any] struct {
	m sync.Map
}

func (c *cache[T]) load(t reflect.Type, fn func() (T, error)) (T, error) {
	if e, ok := c.m.Load(t); ok {
		ce := e.(cacheEntry[T])

		return ce.v, ce.err
	}

	v, err := fn()

	c.m.Store(t, cacheEntry[T]{v: v, err: err})

	return v, err
}

func scanText(src any, set func(string) error) error {
	switch v := src.(type) {
	case string:
		return set(v)
	case []byte:
		return set(string(v))
	}

	return errors.Wrapf(ErrInvalidFormat, "can not scan %T", src)
}

// Validate reports whether the text annotations of v are valid: v must be a
// generated enum value, or a pointer to a generated struct annotated with
// Format.
func Validate(v any) error {
	t := reflect.TypeOf(v)

	if t == nil {
		return invalidDefinition("can not validate nil")
	}

	if t.Implements(enumDefinitionType) {
		_, err := enumCodecOf(t)

		return err
	}

	if t.Kind() == reflect.Pointer && t.Implements(structDefinitionType) {
		_, err := structCodecOf(t.Elem())

		return err
	}

	return invalidDefinition("%s is not a thrift enum or struct pointer", t)
}
