package text

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/internal/fstring"
	"github.com/upfluence/thrift/lib/go/thrift/internal/wordcase"
)

// The cases are shared with the ruby library.
const casesPath = "../../../../../../test/TextTestCases.json"

type testCases struct {
	Words []struct {
		Have string   `json:"have"`
		Want []string `json:"want"`
	} `json:"words"`
	Cases []struct {
		Have string `json:"have"`
		Case string `json:"case"`
		Want string `json:"want"`
	} `json:"cases"`
	Patterns []struct {
		Pattern string   `json:"pattern"`
		Fields  []string `json:"fields"`
		Error   bool     `json:"error"`
		Scans   []struct {
			Have  string   `json:"have"`
			Want  []string `json:"want"`
			Error bool     `json:"error"`
		} `json:"scans"`
		Formats []struct {
			Have  []string `json:"have"`
			Want  string   `json:"want"`
			Error bool     `json:"error"`
		} `json:"formats"`
	} `json:"patterns"`
	Enums   []enumCase   `json:"enums"`
	Structs []structCase `json:"structs"`
}

type enumCase struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format *struct {
		Format            json.RawMessage `json:"format"`
		TrimPrefixes      []string        `json:"trim_prefixes"`
		TrimSuffixes      []string        `json:"trim_suffixes"`
		AcceptThriftNames bool            `json:"accept_thrift_names"`
	} `json:"format"`
	Values []struct {
		Name  string  `json:"name"`
		Value int64   `json:"value"`
		Text  *string `json:"text"`
	} `json:"values"`
	Want []struct {
		Value int64    `json:"value"`
		Texts []string `json:"texts"`
	} `json:"want"`
	Error bool `json:"error"`
}

type structCase struct {
	Name    string  `json:"name"`
	Pattern *string `json:"pattern"`
	Union   bool    `json:"union"`
	Fields  []struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		Enum         string `json:"enum"`
		Struct       string `json:"struct"`
		Requiredness string `json:"requiredness"`
	} `json:"fields"`
	Formats []struct {
		Have  map[string]any `json:"have"`
		Want  string         `json:"want"`
		Error bool           `json:"error"`
	} `json:"formats"`
	Scans []struct {
		Have  string         `json:"have"`
		Want  map[string]any `json:"want"`
		Error bool           `json:"error"`
	} `json:"scans"`
	Error bool `json:"error"`
}

func loadCases(t *testing.T) testCases {
	t.Helper()

	b, err := os.ReadFile(casesPath)
	require.NoError(t, err)

	var (
		tc  testCases
		dec = json.NewDecoder(bytes.NewReader(b))
	)

	dec.UseNumber()
	require.NoError(t, dec.Decode(&tc))

	return tc
}

func TestSplitWords(t *testing.T) {
	for _, tt := range loadCases(t).Words {
		t.Run(tt.Have, func(t *testing.T) {
			got := wordcase.SplitWords(tt.Have)

			if len(tt.Want) == 0 {
				assert.Empty(t, got)

				return
			}

			assert.Equal(t, tt.Want, got)
		})
	}
}

func TestCaser(t *testing.T) {
	for _, tt := range loadCases(t).Cases {
		t.Run(tt.Have+"/"+tt.Case, func(t *testing.T) {
			c, err := CaseFromString(tt.Case)
			require.NoError(t, err)

			caser := wordcase.NewCaser(caseFormats[c])

			assert.Equal(t, tt.Want, caser.String(tt.Have))
		})
	}
}

func TestPattern(t *testing.T) {
	for _, tt := range loadCases(t).Patterns {
		t.Run(tt.Pattern, func(t *testing.T) {
			p, err := fstring.Compile(tt.Pattern)

			if tt.Error {
				require.ErrorIs(t, err, fstring.ErrInvalidPattern)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.Fields, append([]string{}, p.Fields()...))

			for _, sc := range tt.Scans {
				got, err := p.Scan(sc.Have)

				if sc.Error {
					require.ErrorIs(t, err, fstring.ErrMismatch, sc.Have)

					continue
				}

				require.NoError(t, err, sc.Have)
				assert.Equal(t, sc.Want, append([]string{}, got...), sc.Have)
			}

			for _, fc := range tt.Formats {
				got, err := p.Append(nil, fc.Have)

				if fc.Error {
					require.ErrorIs(t, err, fstring.ErrAmbiguousValue, fc.Have)

					continue
				}

				require.NoError(t, err, fc.Have)
				assert.Equal(t, fc.Want, string(got), fc.Have)
			}
		})
	}
}

func (ec enumCase) definition(t *testing.T) thrift.EnumDefinition {
	t.Helper()

	def := thrift.EnumDefinition{
		Namespace:           "texttest",
		AnnotatedDefinition: thrift.AnnotatedDefinition{Name: ec.Name},
	}

	if f := ec.Format; f != nil {
		ef := EnumFormat{
			TrimPrefixes:      f.TrimPrefixes,
			TrimSuffixes:      f.TrimSuffixes,
			AcceptThriftNames: thrift.BoolPtr(f.AcceptThriftNames),
		}

		if len(f.Format) > 0 {
			var name string

			if json.Unmarshal(f.Format, &name) == nil {
				c, err := CaseFromString(name)
				require.NoError(t, err)

				ef.Format = c
			} else {
				require.NoError(t, json.Unmarshal(f.Format, &ef.Format))
			}
		}

		def.StructuredAnnotations = []thrift.RegistrableStruct{&ef}
	}

	for _, v := range ec.Values {
		vd := thrift.EnumValueDefinition{
			AnnotatedDefinition: thrift.AnnotatedDefinition{Name: v.Name},
			Value:               v.Value,
		}

		if v.Text != nil {
			vd.StructuredAnnotations = []thrift.RegistrableStruct{&Name{Value: *v.Text}}
		}

		def.Values = append(def.Values, vd)
	}

	return def
}

func TestCompileEnum(t *testing.T) {
	for _, tt := range loadCases(t).Enums {
		t.Run(tt.ID, func(t *testing.T) {
			c, err := CompileEnum(tt.definition(t))

			if tt.Error {
				require.ErrorIs(t, err, ErrInvalidDefinition)

				return
			}

			require.NoError(t, err)

			for _, w := range tt.Want {
				got, err := c.Format(w.Value)

				require.NoError(t, err)
				assert.Equal(t, w.Texts[0], got)

				for _, text := range w.Texts {
					v, err := c.Parse(text)

					require.NoError(t, err)
					assert.Equal(t, w.Value, v, text)
				}
			}

			_, err = c.Parse("unknown")
			require.ErrorIs(t, err, ErrInvalidFormat)

			_, err = c.Format(42)
			require.ErrorIs(t, err, ErrInvalidValue)
		})
	}
}

var integerBits = map[string]int{"i8": 8, "i16": 16, "i32": 32, "i64": 64}

func (sc structCase) build(t *testing.T, enums map[string]*EnumCodec, structs map[string]*StructCodec) (thrift.StructDefinition, map[string]FieldType) {
	t.Helper()

	var (
		def = thrift.StructDefinition{
			Namespace:           "texttest",
			AnnotatedDefinition: thrift.AnnotatedDefinition{Name: sc.Name},
			IsUnion:             sc.Union,
		}
		fields = make(map[string]FieldType)
	)

	if sc.Pattern != nil {
		def.StructuredAnnotations = []thrift.RegistrableStruct{&Format{Pattern: *sc.Pattern}}
	}

	for _, f := range sc.Fields {
		fd := thrift.FieldDefinition{AnnotatedDefinition: thrift.AnnotatedDefinition{Name: f.Name}}

		switch f.Requiredness {
		case "required":
			fd.Requiredness = thrift.RequirednessRequired
		case "optional":
			fd.Requiredness = thrift.RequirednessOptional
		}

		def.Fields = append(def.Fields, fd)

		switch {
		case f.Type == "string":
			fields[f.Name] = FieldType{Kind: StringField}
		case f.Type == "enum":
			require.Contains(t, enums, f.Enum)
			fields[f.Name] = FieldType{Kind: EnumField, Enum: enums[f.Enum]}
		case f.Type == "struct":
			fields[f.Name] = FieldType{Kind: StructField, Struct: structs[f.Struct]}
		case integerBits[f.Type] > 0:
			fields[f.Name] = FieldType{Kind: IntegerField, Bits: integerBits[f.Type]}
		}
	}

	return def, fields
}

func values(t *testing.T, vs map[string]any) map[string]any {
	t.Helper()

	res := make(map[string]any, len(vs))

	for k, v := range vs {
		switch vv := v.(type) {
		case string:
			res[k] = vv
		case json.Number:
			i, err := vv.Int64()
			require.NoError(t, err)

			res[k] = i
		case map[string]any:
			res[k] = values(t, vv)
		default:
			require.Failf(t, "unexpected value", "%T", v)
		}
	}

	return res
}

func TestCompileStruct(t *testing.T) {
	var (
		tc    = loadCases(t)
		enums = make(map[string]*EnumCodec)
	)

	for _, ec := range tc.Enums {
		if !ec.Error {
			c, err := CompileEnum(ec.definition(t))
			require.NoError(t, err)

			enums[ec.ID] = c
		}
	}

	structs := make(map[string]*StructCodec)

	for _, tt := range tc.Structs {
		t.Run(tt.Name, func(t *testing.T) {
			def, fields := tt.build(t, enums, structs)
			c, err := CompileStruct(def, fields)

			if tt.Error {
				require.ErrorIs(t, err, ErrInvalidDefinition)

				return
			}

			require.NoError(t, err)

			structs[tt.Name] = c

			for _, fc := range tt.Formats {
				got, err := c.Format(values(t, fc.Have))

				if fc.Error {
					require.ErrorIs(t, err, ErrInvalidValue, fc.Have)

					continue
				}

				require.NoError(t, err, fc.Have)
				assert.Equal(t, fc.Want, got)
			}

			for _, sc := range tt.Scans {
				got, err := c.Parse(sc.Have)

				if sc.Error {
					require.ErrorIs(t, err, ErrInvalidFormat, sc.Have)

					continue
				}

				require.NoError(t, err, sc.Have)
				assert.Equal(t, values(t, sc.Want), got, sc.Have)
			}
		})
	}
}

func TestReferencedFields(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    []thrift.RegistrableStruct
		want    []string
		wantErr error
	}{
		{name: "no Format"},
		{name: "pattern", have: []thrift.RegistrableStruct{&Format{Pattern: "{a}:{b}"}}, want: []string{"a", "b"}},
		{
			name:    "invalid pattern",
			have:    []thrift.RegistrableStruct{&Format{Pattern: "{a}{b}"}},
			wantErr: ErrInvalidDefinition,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReferencedFields(
				thrift.StructDefinition{
					AnnotatedDefinition: thrift.AnnotatedDefinition{Name: "S", StructuredAnnotations: tt.have},
				},
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
