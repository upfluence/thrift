package any

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/upfluence/thrift/lib/go/thrift"
)

var (
	DefaultEncoding = &TEncoding{ProtocolFactory: thrift.NewTJSONProtocolFactory()}
	JSONEncoding    = &StdEncoding{
		NewEncoder: func(w io.Writer) StdEncoder { return json.NewEncoder(w) },
		NewDecoder: func(r io.Reader) StdDecoder { return json.NewDecoder(r) },
	}

	encodingMu sync.RWMutex
	encodings  = map[string]Encoding{"": DefaultEncoding, "json": JSONEncoding}
)

type Encoding interface {
	Encoder(io.Writer) Encoder
	Decoder(io.Reader) Decoder
}

type Encoder interface {
	Encode(thrift.TStruct) error
}

type Decoder interface {
	Decode(thrift.TStruct) error
}

func RegisterEncoding(n string, e Encoding) {
	encodingMu.Lock()
	defer encodingMu.Unlock()

	encodings[n] = e
}

type TEncoding struct {
	ProtocolFactory thrift.TProtocolFactory
}

func (te *TEncoding) Encoder(w io.Writer) Encoder {
	return &protocol{
		TProtocol: te.ProtocolFactory.GetProtocol(thrift.WrapWriter(w)),
	}
}

func (te *TEncoding) Decoder(r io.Reader) Decoder {
	return &protocol{
		TProtocol: te.ProtocolFactory.GetProtocol(thrift.WrapReader(r)),
	}
}

type protocol struct {
	thrift.TProtocol
}

func (p *protocol) Encode(s thrift.TStruct) error {
	if err := s.Write(p); err != nil {
		return err
	}

	return p.Flush()
}
func (p *protocol) Decode(s thrift.TStruct) error { return s.Read(p) }

type StdEncoder interface {
	Encode(interface{}) error
}

type StdDecoder interface {
	Decode(interface{}) error
}

type StdEncoding struct {
	NewEncoder func(io.Writer) StdEncoder
	NewDecoder func(io.Reader) StdDecoder
}

func (se *StdEncoding) Encoder(w io.Writer) Encoder {
	return &stdEncoder{e: se.NewEncoder(w)}
}

func (se *StdEncoding) Decoder(r io.Reader) Decoder {
	return &stdDecoder{d: se.NewDecoder(r)}
}

type stdEncoder struct {
	e StdEncoder
}

func (se *stdEncoder) Encode(v thrift.TStruct) error { return se.e.Encode(v) }

type stdDecoder struct {
	d StdDecoder
}

func (sd *stdDecoder) Decode(v thrift.TStruct) error { return sd.d.Decode(v) }
