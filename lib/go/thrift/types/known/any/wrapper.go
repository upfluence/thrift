package any

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift"
)

type RegistrableStruct interface {
	thrift.RegistrableStruct
	thrift.TStruct
}

func New(s RegistrableStruct) (*Any, error) {
	return NewWithEncoding(s, "", DefaultEncoding)
}

func NewWithEncoding(s RegistrableStruct, n string, e Encoding) (*Any, error) {
	var buf bytes.Buffer

	if err := e.Encoder(&buf).Encode(s); err != nil {
		return nil, err
	}

	return &Any{
		Type:  buildType(n, s.StructDefinition().CanonicalName()),
		Value: buf.Bytes(),
	}, nil
}

func (a *Any) Interface() (interface{}, error) {
	var et, tn, err = parseType(a.Type)

	if err != nil {
		return nil, err
	}

	t, ok := thrift.StructType(tn)

	if !ok {
		return nil, fmt.Errorf("Struct type %q not defined", tn)
	}

	v := reflect.New(t).Interface().(thrift.TStruct)

	encodingMu.RLock()
	e, ok := encodings[et]
	encodingMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("No Encoding matching type: %q", a.Type)
	}

	if err := e.Decoder(bytes.NewReader(a.Value)).Decode(v); err != nil {
		return nil, err
	}

	return v, nil
}

func buildType(et, tn string) string {
	if et != "" {
		et = "-" + et
	}

	return fmt.Sprintf("thrift%s/%s", et, tn)
}

func parseType(ct string) (string, string, error) {
	cts := strings.Split(ct, "/")

	if len(cts) != 2 || !strings.HasPrefix(cts[0], "thrift") {
		return "", "", fmt.Errorf("invalid format: %q", ct)
	}

	et := strings.TrimPrefix(cts[0], "thrift")

	if len(et) > 0 {
		if et[0] != '-' {
			return "", "", fmt.Errorf("invalid format: %q", ct)
		}

		et = et[1:]
	}

	return et, cts[1], nil
}
