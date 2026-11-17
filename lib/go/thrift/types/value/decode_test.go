package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decodeKind int64

type decodeNested struct {
	ID int64 `thrift:"id,1,required"`
}

type decodeTarget struct {
	Name    string           `thrift:"name,1,required"`
	Count   *int16           `thrift:"count,2"`
	Enabled *bool            `thrift:"enabled,3"`
	Tags    []string         `thrift:"tags,4"`
	Attrs   map[string]int64 `thrift:"attrs,5"`
	Nested  *decodeNested    `thrift:"nested,6"`
	Kind    decodeKind       `thrift:"kind,7"`
	Data    []byte           `thrift:"data,8"`
	Ratio   float64          `thrift:"ratio,9"`
}

const (
	countField   = "count"
	enabledField = "enabled"
)

func structV(fields map[string]*Value) *Value {
	return &Value{StructValue: &StructValue{Fields: fields}}
}

func listV(vs ...*Value) *Value {
	return &Value{ListValue: &ListValue{Values: vs}}
}

func TestDecodeValue(t *testing.T) {
	var (
		count   = int16(3)
		enabled = true
	)

	for _, tt := range []struct {
		name    string
		have    *Value
		want    decodeTarget
		wantErr bool
	}{
		{
			name: "every kind",
			have: structV(map[string]*Value{
				"name":       EncodeStringValue("bar"),
				countField:   EncodeIntegerValue(3),
				enabledField: EncodeBoolValue(true),
				"tags":       listV(EncodeStringValue("a"), EncodeStringValue("b")),
				"attrs": {
					MapValue: &MapValue{
						Entries: []*MapEntry{
							{Key: EncodeStringValue("x"), Value: EncodeIntegerValue(1)},
						},
					},
				},
				"nested": structV(map[string]*Value{"id": EncodeIntegerValue(42)}),
				"kind":   EncodeIntegerValue(2),
				"data":   EncodeBinaryValue([]byte("raw")),
				"ratio":  EncodeDoubleValue(.5),
			}),
			want: decodeTarget{
				Name:    "bar",
				Count:   &count,
				Enabled: &enabled,
				Tags:    []string{"a", "b"},
				Attrs:   map[string]int64{"x": 1},
				Nested:  &decodeNested{ID: 42},
				Kind:    decodeKind(2),
				Data:    []byte("raw"),
				Ratio:   .5,
			},
		},
		{
			name: "integer constants",
			have: structV(map[string]*Value{
				enabledField: EncodeIntegerValue(1),
				"ratio":      EncodeIntegerValue(2),
				"data":       EncodeStringValue("raw"),
			}),
			want: decodeTarget{Enabled: &enabled, Ratio: 2, Data: []byte("raw")},
		},
		{
			name: "null value",
			have: structV(map[string]*Value{countField: EncodeNullValue()}),
		},
		{
			name:    "integer overflow",
			have:    structV(map[string]*Value{countField: EncodeIntegerValue(1 << 20)}),
			wantErr: true,
		},
		{
			name:    "bool out of range",
			have:    structV(map[string]*Value{enabledField: EncodeIntegerValue(2)}),
			wantErr: true,
		},
		{
			name:    "type mismatch",
			have:    structV(map[string]*Value{"name": EncodeIntegerValue(1)}),
			wantErr: true,
		},
		{
			name:    "unknown field",
			have:    structV(map[string]*Value{"other": EncodeIntegerValue(1)}),
			wantErr: true,
		},
		{
			name:    "not a struct",
			have:    EncodeStringValue("baz"),
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got decodeTarget

			err := DecodeValue(tt.have, &got)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeValueInvalidDestination(t *testing.T) {
	var target decodeTarget

	require.Error(t, DecodeValue(EncodeNullValue(), target))
	require.Error(t, DecodeValue(EncodeNullValue(), (*decodeTarget)(nil)))
}
