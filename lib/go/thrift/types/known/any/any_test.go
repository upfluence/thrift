package any

import (
	"reflect"
	"testing"
	"time"

	"github.com/upfluence/thrift/lib/go/thrift/types/known/duration"
	"github.com/upfluence/thrift/lib/go/thrift/types/known/timestamp"
)

func TestNew(t *testing.T) {
	for _, tt := range []struct {
		name string

		in RegistrableStruct

		wantType  string
		wantValue string
	}{
		{
			name:      "duration",
			in:        duration.New(time.Minute),
			wantType:  "thrift/types.known.duration.Duration",
			wantValue: "{\"1\":{\"i64\":60},\"2\":{\"i32\":0}}",
		},
		{
			name:      "timestamp",
			in:        timestamp.New(time.Unix(123, 0)),
			wantType:  "thrift/types.known.timestamp.Timestamp",
			wantValue: "{\"1\":{\"i64\":123},\"2\":{\"i32\":0}}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(tt.in)

			if err != nil {
				t.Fatalf("Cant build any: %v", err)
			}

			if a.Type != tt.wantType {
				t.Errorf("Invalid type: %q [want: %q]", a.Type, tt.wantType)
			}

			if v := string(a.Value); v != tt.wantValue {
				t.Errorf("Invalid type: %q [want: %q]", v, tt.wantValue)
			}
		})
	}
}

func TestNewJSON(t *testing.T) {
	for _, tt := range []struct {
		name string

		in RegistrableStruct

		wantType  string
		wantValue string
	}{
		{
			name:      "duration",
			in:        duration.New(time.Minute),
			wantType:  "thrift-json/types.known.duration.Duration",
			wantValue: "{\"seconds\":60,\"nanos\":0}\n",
		},
		{
			name:      "timestamp",
			in:        timestamp.New(time.Unix(123, 0)),
			wantType:  "thrift-json/types.known.timestamp.Timestamp",
			wantValue: "{\"seconds\":123,\"nanos\":0}\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewWithEncoding(tt.in, "json", JSONEncoding)

			if err != nil {
				t.Fatalf("Cant build any: %v", err)
			}

			if a.Type != tt.wantType {
				t.Errorf("Invalid type: %q [want: %q]", a.Type, tt.wantType)
			}

			if v := string(a.Value); v != tt.wantValue {
				t.Errorf("Invalid type: %q [want: %q]", v, tt.wantValue)
			}
		})
	}
}

func TestInterface(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   *Any
		want interface{}
	}{
		{
			name: "duration",
			in: &Any{
				Type:  "thrift/types.known.duration.Duration",
				Value: []byte("{\"1\":{\"i64\":60},\"2\":{\"i32\":0}}"),
			},
			want: duration.New(time.Minute),
		},
		{
			name: "duration/json",
			in: &Any{
				Type:  "thrift-json/types.known.duration.Duration",
				Value: []byte("{\"seconds\":60,\"nanos\":0}\n"),
			},
			want: duration.New(time.Minute),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.in.Interface()

			if err != nil {
				t.Fatalf("Cant build any: %v", err)
			}

			if !reflect.DeepEqual(tt.want, v) {
				t.Errorf("Invalid type: %+v [want: %+v]", v, tt.want)
			}
		})
	}
}
