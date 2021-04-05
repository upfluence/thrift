package value

import (
	"fmt"
	"reflect"
	"unicode/utf8"
)

func EncodeValue(v interface{}) (*Value, error) {
	switch v := v.(type) {
	case nil:
		return EncodeNullValue(), nil
	case bool:
		return EncodeBoolValue(v), nil
	case int:
		return EncodeIntegerValue(int64(v)), nil
	case int32:
		return EncodeIntegerValue(int64(v)), nil
	case int64:
		return EncodeIntegerValue(int64(v)), nil
	case uint:
		return EncodeIntegerValue(int64(v)), nil
	case uint32:
		return EncodeIntegerValue(int64(v)), nil
	case uint64:
		return EncodeIntegerValue(int64(v)), nil
	case float32:
		return EncodeDoubleValue(float64(v)), nil
	case float64:
		return EncodeDoubleValue(float64(v)), nil
	case string:
		if !utf8.ValidString(v) {
			return nil, fmt.Errorf("invalid UTF-8 in string: %q", v)
		}

		return EncodeStringValue(v), nil
	case []byte:
		return EncodeBinaryValue(v), nil
	case map[string]interface{}:
		sv, err := EncodeStructValue(v)

		if err != nil {
			return nil, err
		}

		return &Value{StructValue: sv}, nil
	case []interface{}:
		lv, err := EncodeListValue(v)

		if err != nil {
			return nil, err
		}

		return &Value{ListValue: lv}, nil
	case map[interface{}]interface{}:
		mv, err := EncodeMapValue(v)

		if err != nil {
			return nil, err
		}

		return &Value{MapValue: mv}, nil
	}

	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		vs := make([]interface{}, rv.Len())

		for i := 0; i < rv.Len(); i++ {
			vs[i] = rv.Index(i).Interface()
		}

		return EncodeValue(vs)
	case reflect.Map:
		ks := rv.MapKeys()
		vs := make(map[interface{}]interface{}, len(ks))

		for _, k := range ks {
			vs[k.Interface()] = rv.MapIndex(k).Interface()
		}

		return EncodeValue(vs)
	case reflect.Struct:
		tv := rv.Type()
		vs := make(map[string]interface{}, tv.NumField())

		for i := 0; i < tv.NumField(); i++ {
			sf := tv.Field(i)

			vs[sf.Name] = rv.FieldByName(sf.Name).Interface()
		}

		return EncodeValue(vs)
	case reflect.Ptr:
		if !rv.IsNil() {
			return EncodeValue(reflect.Indirect(rv).Interface())
		}
	}

	return nil, fmt.Errorf("invalid type: %T", v)
}

func (v *Value) ToInterface() interface{} {
	switch vv := v.Interface().(type) {
	case *NullValue:
		return nil
	case *ListValue:
		return vv.ToSlice()
	case *MapValue:
		return vv.ToMap()
	case *StructValue:
		return vv.ToMap()
	case *string:
		return *vv
	case *[]byte:
		return *vv
	case *int64:
		return *vv
	case *float64:
		return *vv
	default:
		return vv
	}
}

func EncodeNullValue() *Value            { return &Value{NullValue: &NullValue{}} }
func EncodeBoolValue(b bool) *Value      { return &Value{BoolValue: &b} }
func EncodeStringValue(s string) *Value  { return &Value{StringValue: &s} }
func EncodeBinaryValue(b []byte) *Value  { return &Value{BinaryValue: b} }
func EncodeIntegerValue(i int64) *Value  { return &Value{IntegerValue: &i} }
func EncodeDoubleValue(f float64) *Value { return &Value{DoubleValue: &f} }
