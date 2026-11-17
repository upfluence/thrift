package fstring

import (
	"slices"
	"strconv"
	"strings"

	"github.com/upfluence/errors"
)

var (
	ErrInvalidPattern = errors.New("fstring: invalid pattern")
	ErrMismatch       = errors.New("fstring: text does not match the pattern")
	ErrAmbiguousValue = errors.New("fstring: value contains the next literal")
)

// Segment is a part of a Pattern: either a literal or a field placeholder.
type Segment struct {
	Literal string
	Field   string
}

// Pattern is a compiled pattern following a subset of the python format
// string syntax: "{name}" is a field placeholder, "{{" and "}}" are the
// escaped braces, everything else is literal. Format specs and conversions
// ("{name:spec}", "{name!r}") are not supported.
//
// A pattern can be printed and scanned back: a field holds the text up to the
// first occurrence of the literal following it, the last field holds the rest
// of the text. Consequently, two fields must be separated by a literal and
// printing a value containing the literal following it fails.
type Pattern struct {
	pattern  string
	segments []Segment
	fields   []string
}

// Compile compiles the pattern. Every field must appear once and be a valid
// identifier.
func Compile(pattern string) (*Pattern, error) {
	var (
		p       = Pattern{pattern: pattern}
		literal strings.Builder
		fields  = make(map[string]struct{})

		rest = pattern
	)

	for rest != "" {
		switch {
		case strings.HasPrefix(rest, "{{"), strings.HasPrefix(rest, "}}"):
			literal.WriteByte(rest[0])

			rest = rest[2:]
		case rest[0] == '}':
			return nil, invalidPattern(pattern, "single } must be escaped")
		case rest[0] == '{':
			name, tail, ok := strings.Cut(rest[1:], "}")

			if !ok {
				return nil, invalidPattern(pattern, "unclosed {")
			}

			if !isIdentifier(name) {
				return nil, invalidPattern(
					pattern,
					"invalid field name "+strconv.Quote(name),
				)
			}

			if _, ok := fields[name]; ok {
				return nil, invalidPattern(
					pattern,
					"field "+strconv.Quote(name)+" is repeated",
				)
			}

			if literal.Len() > 0 {
				p.segments = append(p.segments, Segment{Literal: literal.String()})
				literal.Reset()
			} else if len(p.segments) > 0 {
				return nil, invalidPattern(
					pattern,
					"fields must be separated by a literal",
				)
			}

			fields[name] = struct{}{}
			p.segments = append(p.segments, Segment{Field: name})
			p.fields = append(p.fields, name)
			rest = tail
		default:
			i := strings.IndexAny(rest, "{}")

			if i == -1 {
				i = len(rest)
			}

			literal.WriteString(rest[:i])
			rest = rest[i:]
		}
	}

	if literal.Len() > 0 {
		p.segments = append(p.segments, Segment{Literal: literal.String()})
	}

	return &p, nil
}

func invalidPattern(pattern, reason string) error {
	return errors.Wrapf(ErrInvalidPattern, "%q: %s", pattern, reason)
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		switch {
		case r == '_', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}

	return true
}

// String returns the source of the pattern.
func (p *Pattern) String() string {
	return p.pattern
}

// Fields returns the field names, in pattern order.
func (p *Pattern) Fields() []string {
	return append([]string(nil), p.fields...)
}

// Segments returns the literals and fields of the pattern, in pattern order.
// Two fields are never adjacent.
func (p *Pattern) Segments() []Segment {
	return append([]Segment(nil), p.segments...)
}

// Scan returns the text of every field of s, in pattern order.
func (p *Pattern) Scan(s string) ([]string, error) {
	var (
		values = make([]string, 0, len(p.fields))

		rest = s
	)

	for i, seg := range p.segments {
		if seg.Field == "" {
			var ok bool

			if rest, ok = strings.CutPrefix(rest, seg.Literal); !ok {
				return nil, p.mismatch(s)
			}

			continue
		}

		if i == len(p.segments)-1 {
			values = append(values, rest)
			rest = ""

			continue
		}

		j := strings.Index(rest, p.segments[i+1].Literal)

		if j == -1 {
			return nil, p.mismatch(s)
		}

		values = append(values, rest[:j])
		rest = rest[j:]
	}

	if rest != "" {
		return nil, p.mismatch(s)
	}

	return values, nil
}

func (p *Pattern) mismatch(s string) error {
	return errors.Wrapf(ErrMismatch, "%q does not match %q", s, p.pattern)
}

// Append appends the pattern printed with the text of every field, in
// pattern order, to b.
func (p *Pattern) Append(b []byte, values []string) ([]byte, error) {
	if len(values) != len(p.fields) {
		return nil, errors.Newf(
			"fstring: %d values given for the %d fields of %q",
			len(values),
			len(p.fields),
			p.pattern,
		)
	}

	var field int

	for i, seg := range p.segments {
		if seg.Field == "" {
			b = append(b, seg.Literal...)

			continue
		}

		v := values[field]
		field++

		if i == len(p.segments)-1 {
			b = append(b, v...)

			continue
		}

		// The value is scanned back up to the first occurrence of the next
		// literal, which must be the one following the value.
		if next := p.segments[i+1].Literal; strings.Index(v+next, next) != len(v) {
			return nil, errors.Wrapf(
				ErrAmbiguousValue,
				"field %q: %q contains %q",
				seg.Field,
				v,
				next,
			)
		}

		b = append(b, v...)
	}

	return b, nil
}

// Inline returns a copy of p where the placeholder of field is replaced by
// the segments of inner, whose fields are renamed "<field>.<name>". The
// literals surrounding the placeholder are merged with the ones of inner.
func (p *Pattern) Inline(field string, inner *Pattern) (*Pattern, error) {
	if !slices.Contains(p.fields, field) {
		return nil, errors.Newf("fstring: %q has no field %q", p.pattern, field)
	}

	res := Pattern{pattern: p.pattern}

	appendSegment := func(s Segment) {
		if s.Field != "" {
			res.segments = append(res.segments, s)
			res.fields = append(res.fields, s.Field)

			return
		}

		if n := len(res.segments); n > 0 && res.segments[n-1].Field == "" {
			res.segments[n-1].Literal += s.Literal

			return
		}

		res.segments = append(res.segments, s)
	}

	for _, s := range p.segments {
		if s.Field != field {
			appendSegment(s)

			continue
		}

		for _, is := range inner.segments {
			if is.Field != "" {
				is.Field = field + "." + is.Field
			}

			appendSegment(is)
		}
	}

	return &res, nil
}
