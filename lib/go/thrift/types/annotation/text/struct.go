package text

import (
	"database/sql/driver"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/upfluence/errors"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/internal/fstring"
)

// FieldKind is the kind of a field referenced by a Format pattern.
type FieldKind int

const (
	// IntegerField is an i8, i16, i32 or i64 field, its value is an int64.
	IntegerField FieldKind = iota + 1
	// StringField is a string field, its value is a string.
	StringField
	// EnumField is an enum field, its value is an int64.
	EnumField
	// StructField is a struct field, its value is a map[string]any holding
	// the values of the struct. Its text form, defined by its own Format,
	// is embedded in the pattern.
	StructField
)

// FieldType is the type of a field referenced by a Format pattern.
type FieldType struct {
	Kind FieldKind
	// Bits is the size of an integer field: 8, 16, 32 or 64.
	Bits int
	// Enum is the codec of an enum field.
	Enum *EnumCodec
	// Struct is the codec of a struct field, nil when the struct has no
	// Format annotation.
	Struct *StructCodec
}

// structField is an integer, string or enum field of the pattern, once the
// struct fields are inlined.
type structField struct {
	// name is the path of the field joined by dots, e.g. "entity.id".
	name string
	path []string
	typ  FieldType
}

// StructCodec encodes the values of a struct as text, following its Format
// annotation.
type StructCodec struct {
	name string
	// pattern has the patterns of the struct fields inlined.
	pattern *fstring.Pattern
	// fields are in pattern order.
	fields []structField
}

// CompileStruct builds the codec of the struct described by def. fields are
// the types of the fields, keyed by name: a field missing from fields has an
// unsupported type. Only the types of the fields referenced by the pattern
// are required.
func CompileStruct(def thrift.StructDefinition, fields map[string]FieldType) (*StructCodec, error) {
	c, err := compileStruct(def, fields)

	if err != nil {
		return nil, errors.Wrapf(err, "struct %s", def.CanonicalName())
	}

	return c, nil
}

// formatPattern returns the compiled Format pattern of def, nil when def has
// no Format annotation.
func formatPattern(def thrift.StructDefinition) (*fstring.Pattern, error) {
	format, err := findAnnotation[*Format](def.AnnotatedDefinition)

	if err != nil {
		return nil, err
	}

	if format == nil {
		return nil, nil
	}

	if def.IsUnion || def.IsException {
		return nil, invalidDefinition("Format can only be set on a struct")
	}

	p, err := fstring.Compile(format.Pattern)

	if err != nil {
		return nil, invalidDefinition("%v", err)
	}

	return p, nil
}

func compileStruct(def thrift.StructDefinition, fields map[string]FieldType) (*StructCodec, error) {
	p, err := formatPattern(def)

	if err != nil {
		return nil, err
	}

	if p == nil {
		return nil, invalidDefinition("no Format annotation")
	}

	referenced := p.Fields()

	for _, f := range def.Fields {
		if f.Requiredness != thrift.RequirednessOptional && !slices.Contains(referenced, f.Name) {
			return nil, invalidDefinition("pattern does not reference field %q", f.Name)
		}
	}

	leaves := make(map[string]FieldType, len(referenced))

	for _, name := range referenced {
		if p, err = inlineField(p, def, name, fields, leaves); err != nil {
			return nil, err
		}
	}

	codec := StructCodec{name: def.AnnotatedDefinition.Name, pattern: p}

	for _, name := range p.Fields() {
		codec.fields = append(
			codec.fields,
			structField{name: name, path: strings.Split(name, "."), typ: leaves[name]},
		)
	}

	if err := codec.validateSeparators(); err != nil {
		return nil, err
	}

	return &codec, nil
}

// inlineField checks the field name referenced by p and adds its type to
// leaves. The pattern of a struct field is inlined in p, and its fields added
// to leaves.
func inlineField(p *fstring.Pattern, def thrift.StructDefinition, name string, fields, leaves map[string]FieldType) (*fstring.Pattern, error) {
	i := slices.IndexFunc(def.Fields, func(f thrift.FieldDefinition) bool { return f.Name == name })

	if i == -1 {
		return nil, invalidDefinition("pattern references unknown field %q", name)
	}

	if def.Fields[i].Requiredness == thrift.RequirednessOptional {
		return nil, invalidDefinition("field %q must not be optional", name)
	}

	typ, ok := fields[name]

	switch {
	case !ok:
		return nil, invalidDefinition("field %q has an unsupported type", name)
	case typ.Kind != StructField:
		leaves[name] = typ

		return p, nil
	case typ.Struct == nil:
		return nil, invalidDefinition("field %q is a struct without Format annotation", name)
	}

	for _, f := range typ.Struct.fields {
		leaves[name+"."+f.name] = f.typ
	}

	return p.Inline(name, typ.Struct.pattern)
}

// validateSeparators ensures every enum and integer value can be scanned
// back: the literal following such a field can not occur in its text form.
// String values are checked when formatted.
func (c *StructCodec) validateSeparators() error {
	var (
		field int
		segs  = c.pattern.Segments()
	)

	for i, seg := range segs {
		if seg.Field == "" {
			continue
		}

		f := c.fields[field]
		field++

		if i == len(segs)-1 {
			continue
		}

		sep := segs[i+1].Literal

		switch f.typ.Kind {
		case IntegerField:
			if strings.ContainsAny(sep[:1], "+-0123456789") {
				return invalidDefinition(
					"integer field %q can not be followed by %q",
					f.name,
					sep,
				)
			}
		case EnumField:
			for _, v := range f.typ.Enum.values {
				for _, t := range v.texts {
					if strings.Index(t+sep, sep) != len(t) {
						return invalidDefinition(
							"text %q of %s.%s conflicts with the separator %q",
							t,
							f.typ.Enum.name,
							v.name,
							sep,
						)
					}
				}
			}
		}
	}

	return nil
}

// Format returns the text form of the struct holding the values, keyed by
// field name: an int64 for integer and enum fields, a string for string
// fields and a map[string]any for struct fields.
func (c *StructCodec) Format(values map[string]any) (string, error) {
	vs := make([]any, len(c.fields))

	for i, f := range c.fields {
		v, ok := lookupValue(values, f.path)

		if !ok {
			return "", errors.Wrapf(ErrInvalidValue, "%s: missing field %q", c.name, f.name)
		}

		vs[i] = v
	}

	b, err := c.appendValues(nil, vs)

	return string(b), err
}

func lookupValue(values map[string]any, path []string) (any, bool) {
	v, ok := values[path[0]]

	if !ok || len(path) == 1 {
		return v, ok
	}

	nested, ok := v.(map[string]any)

	if !ok {
		return nil, false
	}

	return lookupValue(nested, path[1:])
}

// Parse returns the values of the struct of text form s, keyed by field name:
// an int64 for integer and enum fields, a string for string fields and a
// map[string]any for struct fields.
func (c *StructCodec) Parse(s string) (map[string]any, error) {
	vs, err := c.parseValues(s)

	if err != nil {
		return nil, err
	}

	values := make(map[string]any, len(vs))

	for i, f := range c.fields {
		m := values

		for _, p := range f.path[:len(f.path)-1] {
			nested, ok := m[p].(map[string]any)

			if !ok {
				nested = make(map[string]any)
				m[p] = nested
			}

			m = nested
		}

		m[f.path[len(f.path)-1]] = vs[i]
	}

	return values, nil
}

func (c *StructCodec) appendValues(b []byte, vs []any) ([]byte, error) {
	texts := make([]string, len(vs))

	for i, f := range c.fields {
		t, err := f.format(vs[i])

		if err != nil {
			return nil, errors.Wrapf(err, "%s.%s", c.name, f.name)
		}

		texts[i] = t
	}

	b, err := c.pattern.Append(b, texts)

	if err != nil {
		return nil, errors.Wrapf(ErrInvalidValue, "%s: %v", c.name, err)
	}

	return b, nil
}

func (c *StructCodec) parseValues(s string) ([]any, error) {
	texts, err := c.pattern.Scan(s)

	if err != nil {
		return nil, errors.Wrapf(ErrInvalidFormat, "%s: %v", c.name, err)
	}

	vs := make([]any, len(texts))

	for i, f := range c.fields {
		v, err := f.parse(texts[i])

		if err != nil {
			return nil, errors.Wrapf(err, "%s.%s", c.name, f.name)
		}

		vs[i] = v
	}

	return vs, nil
}

func (f structField) format(v any) (string, error) {
	switch f.typ.Kind {
	case StringField:
		if s, ok := v.(string); ok {
			return s, nil
		}
	case IntegerField:
		if i, ok := v.(int64); ok {
			if !fitsBits(i, f.typ.Bits) {
				return "", errors.Wrapf(ErrInvalidValue, "%d overflows i%d", i, f.typ.Bits)
			}

			return strconv.FormatInt(i, 10), nil
		}
	case EnumField:
		if i, ok := v.(int64); ok {
			return f.typ.Enum.Format(i)
		}
	}

	return "", errors.Wrapf(ErrInvalidValue, "unexpected %T value", v)
}

func (f structField) parse(s string) (any, error) {
	switch f.typ.Kind {
	case StringField:
		return s, nil
	case IntegerField:
		i, err := strconv.ParseInt(s, 10, f.typ.Bits)

		if err != nil {
			return nil, errors.Wrapf(ErrInvalidFormat, "%q is not an i%d", s, f.typ.Bits)
		}

		return i, nil
	case EnumField:
		return f.typ.Enum.Parse(s)
	}

	return nil, invalidDefinition("unknown field kind %d", f.typ.Kind)
}

func fitsBits(i int64, bits int) bool {
	if bits >= 64 {
		return true
	}

	limit := int64(1) << (bits - 1)

	return i >= -limit && i < limit
}

type structPtr[T any] interface {
	*T
	thrift.RegistrableStruct
}

var structDefinitionType = reflect.TypeFor[thrift.RegistrableStruct]()

// reflectCodec binds a StructCodec to a generated struct type.
type reflectCodec struct {
	codec *StructCodec
	// index is the index path of the go field of every codec field.
	index [][]int
}

var structCodecs cache[*reflectCodec]

// structCodecOf returns the codec of the generated struct type t.
func structCodecOf(t reflect.Type) (*reflectCodec, error) {
	return reflectCodecOf(t, make(map[reflect.Type]bool))
}

// reflectCodecOf returns the codec of t, visiting holds the struct types
// being compiled to reject the structs embedding themselves.
func reflectCodecOf(t reflect.Type, visiting map[reflect.Type]bool) (*reflectCodec, error) {
	return structCodecs.load(t, func() (*reflectCodec, error) {
		if visiting[t] {
			return nil, invalidDefinition("%s embeds itself", t)
		}

		visiting[t] = true
		defer delete(visiting, t)

		return compileReflectCodec(t, visiting)
	})
}

func compileReflectCodec(t reflect.Type, visiting map[reflect.Type]bool) (*reflectCodec, error) {
	if t.Kind() != reflect.Struct || !reflect.PointerTo(t).Implements(structDefinitionType) {
		return nil, invalidDefinition("%s is not a thrift struct", t)
	}

	def := reflect.New(t).Interface().(thrift.RegistrableStruct).StructDefinition()
	p, err := formatPattern(def)

	if err != nil || p == nil {
		// CompileStruct reports the error.
		_, err := CompileStruct(def, nil)

		return nil, err
	}

	var (
		fields  = make(map[string]FieldType)
		indexes = make(map[string]int)
		nested  = make(map[string]*reflectCodec)
		goNames = make(map[string]int, t.NumField())
	)

	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("thrift"), ","); name != "" {
			goNames[name] = i
		}
	}

	for _, name := range p.Fields() {
		i, ok := goNames[name]

		if !ok {
			continue
		}

		typ, rc, ok, err := goFieldType(t.Field(i).Type, visiting)

		if err != nil {
			return nil, errors.Wrapf(err, "struct %s: field %q", def.CanonicalName(), name)
		}

		if ok {
			fields[name] = typ
			indexes[name] = i
			nested[name] = rc
		}
	}

	codec, err := CompileStruct(def, fields)

	if err != nil {
		return nil, err
	}

	rc := reflectCodec{codec: codec}

	for _, f := range codec.fields {
		index := []int{indexes[f.path[0]]}

		if n := nested[f.path[0]]; n != nil {
			index = append(index, n.indexOf(strings.Join(f.path[1:], "."))...)
		}

		rc.index = append(rc.index, index)
	}

	return &rc, nil
}

func (rc *reflectCodec) indexOf(name string) []int {
	for i, f := range rc.codec.fields {
		if f.name == name {
			return rc.index[i]
		}
	}

	return nil
}

// goFieldType returns the text type of a go field of type t, and the codec
// of a struct field.
func goFieldType(t reflect.Type, visiting map[reflect.Type]bool) (FieldType, *reflectCodec, bool, error) {
	if t.Implements(enumDefinitionType) {
		c, err := enumCodecOf(t)

		if err != nil {
			return FieldType{}, nil, false, err
		}

		return FieldType{Kind: EnumField, Enum: c}, nil, true, nil
	}

	switch t.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Enums generated before EnumDefinition existed must not be
		// mistaken for integers.
		if _, ok := t.MethodByName("HumanizedString"); ok {
			return FieldType{}, nil, false, invalidDefinition(
				"enum %s has no EnumDefinition method, regenerate it",
				t,
			)
		}

		return FieldType{Kind: IntegerField, Bits: t.Bits()}, nil, true, nil
	case reflect.String:
		return FieldType{Kind: StringField}, nil, true, nil
	case reflect.Pointer:
		return goStructFieldType(t.Elem(), visiting)
	case reflect.Struct:
		return goStructFieldType(t, visiting)
	}

	return FieldType{}, nil, false, nil
}

func goStructFieldType(t reflect.Type, visiting map[reflect.Type]bool) (FieldType, *reflectCodec, bool, error) {
	if t.Kind() != reflect.Struct || !reflect.PointerTo(t).Implements(structDefinitionType) {
		return FieldType{}, nil, false, nil
	}

	def := reflect.New(t).Interface().(thrift.RegistrableStruct).StructDefinition()

	if len(thrift.FindAnnotations[*Format](def.AnnotatedDefinition)) == 0 {
		return FieldType{Kind: StructField}, nil, true, nil
	}

	rc, err := reflectCodecOf(t, visiting)

	if err != nil {
		return FieldType{}, nil, false, err
	}

	return FieldType{Kind: StructField, Struct: rc.codec}, rc, true, nil
}

// goField returns the go field at index of rv, allocating the nil struct
// pointers when alloc is set.
func goField(rv reflect.Value, index []int, alloc bool) (reflect.Value, bool) {
	for i, idx := range index {
		rv = rv.Field(idx)

		if i == len(index)-1 || rv.Kind() != reflect.Pointer {
			continue
		}

		if rv.IsNil() {
			if !alloc {
				return reflect.Value{}, false
			}

			rv.Set(reflect.New(rv.Type().Elem()))
		}

		rv = rv.Elem()
	}

	return rv, true
}

func appendStruct(b []byte, rv reflect.Value) ([]byte, error) {
	rc, err := structCodecOf(rv.Type())

	if err != nil {
		return nil, err
	}

	vs := make([]any, len(rc.index))

	for i, index := range rc.index {
		f, ok := goField(rv, index, false)

		if !ok {
			return nil, errors.Wrapf(
				ErrInvalidValue,
				"%s: missing field %q",
				rc.codec.name,
				rc.codec.fields[i].name,
			)
		}

		if f.Kind() == reflect.String {
			vs[i] = f.String()
		} else {
			vs[i] = f.Int()
		}
	}

	return rc.codec.appendValues(b, vs)
}

func unmarshalStruct(s string, rv reflect.Value) error {
	rc, err := structCodecOf(rv.Type())

	if err != nil {
		return err
	}

	vs, err := rc.codec.parseValues(s)

	if err != nil {
		return err
	}

	for i, index := range rc.index {
		f, _ := goField(rv, index, true)

		switch v := vs[i].(type) {
		case string:
			f.SetString(v)
		case int64:
			f.SetInt(v)
		}
	}

	return nil
}

// MarshalStruct returns the text form of v, a pointer to a generated struct
// annotated with Format.
func MarshalStruct(v thrift.RegistrableStruct) (string, error) {
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return "", errors.Wrapf(ErrInvalidValue, "%T is not a non nil struct pointer", v)
	}

	b, err := appendStruct(nil, rv.Elem())

	return string(b), err
}

// UnmarshalStruct returns the struct of text form s.
func UnmarshalStruct[T any, PT structPtr[T]](s string) (*T, error) {
	var v T

	if err := unmarshalStruct(s, reflect.ValueOf(&v).Elem()); err != nil {
		return nil, err
	}

	return &v, nil
}

// Struct encodes the wrapped struct as its text form in JSON, SQL or flags.
// A pointer to T must be a generated struct annotated with Format, otherwise
// every method fails with ErrInvalidDefinition. Use sql.Null[Struct[T]] for
// nullable columns.
type Struct[T any] struct {
	V T
}

// AppendText implements encoding.TextAppender.
func (s Struct[T]) AppendText(b []byte) ([]byte, error) {
	return appendStruct(b, reflect.ValueOf(&s.V).Elem())
}

// MarshalText implements encoding.TextMarshaler.
func (s Struct[T]) MarshalText() ([]byte, error) {
	return s.AppendText(nil)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Struct[T]) UnmarshalText(b []byte) error {
	return s.Set(string(b))
}

// String implements flag.Value, it returns an empty string if the value has
// no text form.
func (s Struct[T]) String() string {
	b, _ := s.AppendText(nil)

	return string(b)
}

// Set implements flag.Value.
func (s *Struct[T]) Set(v string) error {
	var t T

	if err := unmarshalStruct(v, reflect.ValueOf(&t).Elem()); err != nil {
		return err
	}

	s.V = t

	return nil
}

// Value implements driver.Valuer.
func (s Struct[T]) Value() (driver.Value, error) {
	b, err := s.AppendText(nil)

	if err != nil {
		return nil, err
	}

	return string(b), nil
}

// Scan implements sql.Scanner.
func (s *Struct[T]) Scan(src any) error {
	return scanText(src, s.Set)
}
