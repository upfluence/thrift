package value

import (
	"encoding/json"
	"reflect"
	"testing"
)

type exampleStruct struct {
	Foo int32
	Bar int64
	Buz float64
	Baz uint32
}

func TestEncodeValue(t *testing.T) {
	for _, tt := range []struct {
		in interface{}

		wantErr   bool
		wantValue string
	}{
		{
			wantValue: "{\"null_value\":{}}",
		},
		{
			in:        map[string]int{"foo": 1},
			wantValue: "{\"map_value\":{\"entries\":[{\"key\":{\"string_value\":\"foo\"},\"value\":{\"integer_value\":1}}]}}",
		},
		{
			in:        [][]byte{[]byte("foo")},
			wantValue: "{\"list_value\":{\"values\":[{\"binary_value\":\"Zm9v\"}]}}",
		},
		{
			in:        &exampleStruct{Bar: 5},
			wantValue: "{\"struct_value\":{\"fields\":{\"Bar\":{\"integer_value\":5},\"Baz\":{\"integer_value\":0},\"Buz\":{\"double_value\":0},\"Foo\":{\"integer_value\":0}}}}",
		},
	} {
		v, err := EncodeValue(tt.in)

		if tt.wantErr != (err != nil) {
			t.Errorf("unexpected error outcome: %v", err)
		}

		if err != nil {
			continue
		}

		buf, err := json.Marshal(v)

		if err != nil {
			t.Fatalf("unexpected json marshaling issue: %v", err)
		}

		if res := string(buf); res != tt.wantValue {
			t.Errorf("Unexpected value: %q [want: %q]", res, tt.wantValue)
		}
	}
}

func TestToValue(t *testing.T) {
	for _, tt := range []struct {
		in string

		want interface{}
	}{
		{
			in: "{\"null_value\":{}}",
		},
		{
			in:   "{\"map_value\":{\"entries\":[{\"key\":{\"string_value\":\"foo\"},\"value\":{\"integer_value\":1}}]}}",
			want: map[interface{}]interface{}{"foo": int64(1)},
		},
		{
			in:   "{\"list_value\":{\"values\":[{\"binary_value\":\"Zm9v\"}]}}",
			want: []interface{}{[]byte("foo")},
		},
		{
			in: "{\"struct_value\":{\"fields\":{\"Bar\":{\"integer_value\":5},\"Baz\":{\"integer_value\":0},\"Buz\":{\"double_value\":0},\"Foo\":{\"integer_value\":0}}}}",
			want: map[string]interface{}{
				"Bar": int64(5),
				"Baz": int64(0),
				"Buz": .0,
				"Foo": int64(0),
			},
		},
	} {
		var v Value

		if err := json.Unmarshal([]byte(tt.in), &v); err != nil {
			t.Fatalf("unexpected json marshaling issue: %v", err)
		}

		if res := v.ToInterface(); !reflect.DeepEqual(tt.want, res) {
			t.Errorf("Unexpected value: %q [want: %q]", res, tt.want)
		}
	}
}
