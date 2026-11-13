package text

import (
	"database/sql/driver"
	"reflect"
	"slices"
	"strings"

	"github.com/upfluence/errors"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/internal/wordcase"
)

var caseFormats = map[Case]wordcase.Case{
	Case_SnakeCase:          wordcase.SnakeCase,
	Case_KebabCase:          wordcase.KebabCase,
	Case_ScreamingSnakeCase: wordcase.ScreamingSnakeCase,
	Case_CamelCase:          wordcase.CamelCase,
	Case_PascalCase:         wordcase.PascalCase,
	Case_LowerCase:          wordcase.LowerCase,
	Case_Identity:           wordcase.IdentityCase,
}

type enumValue struct {
	name  string
	value int64
	// texts are the accepted text forms, the canonical one first.
	texts []string
}

// EnumCodec encodes the values of an enum as text, following its Name and
// EnumFormat annotations.
type EnumCodec struct {
	name   string
	values []enumValue

	texts     map[int64]string
	fromTexts map[string]int64
}

// CompileEnum builds the codec of the enum described by def.
func CompileEnum(def thrift.EnumDefinition) (*EnumCodec, error) {
	c, err := compileEnum(def)

	if err != nil {
		return nil, errors.Wrapf(err, "enum %s", def.CanonicalName())
	}

	return c, nil
}

func compileEnum(def thrift.EnumDefinition) (*EnumCodec, error) {
	format, err := findAnnotation[*EnumFormat](def.AnnotatedDefinition)

	if err != nil {
		return nil, err
	}

	if format == nil {
		format = NewEnumFormat()
	}

	c, ok := caseFormats[format.Format]

	if !ok {
		return nil, invalidDefinition("EnumFormat.format %d is unknown", format.Format)
	}

	var (
		caser = wordcase.NewCaser(c, wordcase.GoInitialisms...)
		codec = EnumCodec{
			name:      def.Name,
			texts:     make(map[int64]string, len(def.Values)),
			fromTexts: make(map[string]int64, len(def.Values)),
		}
		owners = make(map[string]string, len(def.Values))
	)

	for _, v := range def.Values {
		if other, ok := codec.texts[v.Value]; ok {
			return nil, invalidDefinition(
				"values %s and %s share the id %d",
				owners[other],
				v.Name,
				v.Value,
			)
		}

		text, err := valueText(v, format, caser)

		if err != nil {
			return nil, err
		}

		if other, ok := owners[text]; ok {
			return nil, invalidDefinition(
				"values %s and %s share the text %q",
				other,
				v.Name,
				text,
			)
		}

		owners[text] = v.Name
		codec.texts[v.Value] = text
		codec.values = append(
			codec.values,
			enumValue{name: v.Name, value: v.Value, texts: []string{text}},
		)
	}

	for i, v := range codec.values {
		if format.GetAcceptThriftNames() {
			for _, alias := range []string{v.name, def.Name + "_" + v.name} {
				if other, ok := owners[alias]; ok && other != v.name {
					return nil, invalidDefinition(
						"the text %q of value %s is the name of value %s",
						alias,
						other,
						v.name,
					)
				}

				if !slices.Contains(v.texts, alias) {
					codec.values[i].texts = append(codec.values[i].texts, alias)
				}
			}
		}

		for _, t := range codec.values[i].texts {
			codec.fromTexts[t] = v.value
		}
	}

	return &codec, nil
}

func valueText(v thrift.EnumValueDefinition, format *EnumFormat, caser wordcase.Caser) (string, error) {
	name, err := findAnnotation[*Name](v.AnnotatedDefinition)

	if err != nil {
		return "", errors.Wrapf(err, "value %s", v.Name)
	}

	if name != nil {
		if name.Value == "" {
			return "", invalidDefinition("value %s has an empty text", v.Name)
		}

		return name.Value, nil
	}

	trimmed := v.Name

	for _, p := range format.TrimPrefixes {
		if t, ok := strings.CutPrefix(trimmed, p); ok {
			trimmed = t

			break
		}
	}

	for _, s := range format.TrimSuffixes {
		if t, ok := strings.CutSuffix(trimmed, s); ok {
			trimmed = t

			break
		}
	}

	if trimmed == "" {
		return "", invalidDefinition("value %s is empty once trimmed", v.Name)
	}

	return caser.String(trimmed), nil
}

// Format returns the text form of the value v.
func (c *EnumCodec) Format(v int64) (string, error) {
	t, ok := c.texts[v]

	if !ok {
		return "", errors.Wrapf(ErrInvalidValue, "%s(%d)", c.name, v)
	}

	return t, nil
}

// Parse returns the value of the text form s.
func (c *EnumCodec) Parse(s string) (int64, error) {
	v, ok := c.fromTexts[s]

	if !ok {
		return 0, errors.Wrapf(ErrInvalidFormat, "invalid %s %q", c.name, s)
	}

	return v, nil
}

type enum interface {
	~int64
	EnumDefinition() thrift.EnumDefinition
}

type enumDefinition interface {
	EnumDefinition() thrift.EnumDefinition
}

var enumDefinitionType = reflect.TypeFor[enumDefinition]()

var enumCodecs cache[*EnumCodec]

// enumCodecOf returns the codec of the generated enum type t.
func enumCodecOf(t reflect.Type) (*EnumCodec, error) {
	return enumCodecs.load(t, func() (*EnumCodec, error) {
		if t.Kind() != reflect.Int64 || !t.Implements(enumDefinitionType) {
			return nil, invalidDefinition("%s is not a thrift enum", t)
		}

		return CompileEnum(
			reflect.Zero(t).Interface().(enumDefinition).EnumDefinition(),
		)
	})
}

// MarshalEnum returns the text form of v.
func MarshalEnum[E enum](v E) (string, error) {
	c, err := enumCodecOf(reflect.TypeFor[E]())

	if err != nil {
		return "", err
	}

	return c.Format(int64(v))
}

// UnmarshalEnum returns the value of the text form s.
func UnmarshalEnum[E enum](s string) (E, error) {
	c, err := enumCodecOf(reflect.TypeFor[E]())

	if err != nil {
		return 0, err
	}

	v, err := c.Parse(s)

	return E(v), err
}

// Enum encodes the wrapped enum value as its text form in JSON, SQL or
// flags. Use sql.Null[Enum[E]] for nullable columns.
type Enum[E enum] struct {
	V E
}

// AppendText implements encoding.TextAppender.
func (e Enum[E]) AppendText(b []byte) ([]byte, error) {
	s, err := MarshalEnum(e.V)

	if err != nil {
		return nil, err
	}

	return append(b, s...), nil
}

// MarshalText implements encoding.TextMarshaler.
func (e Enum[E]) MarshalText() ([]byte, error) {
	return e.AppendText(nil)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (e *Enum[E]) UnmarshalText(b []byte) error {
	return e.Set(string(b))
}

// String implements flag.Value, it returns an empty string if the value has
// no text form.
func (e Enum[E]) String() string {
	s, _ := MarshalEnum(e.V)

	return s
}

// Set implements flag.Value.
func (e *Enum[E]) Set(s string) error {
	v, err := UnmarshalEnum[E](s)

	if err != nil {
		return err
	}

	e.V = v

	return nil
}

// Value implements driver.Valuer.
func (e Enum[E]) Value() (driver.Value, error) {
	return MarshalEnum(e.V)
}

// Scan implements sql.Scanner.
func (e *Enum[E]) Scan(src any) error {
	return scanText(src, e.Set)
}
