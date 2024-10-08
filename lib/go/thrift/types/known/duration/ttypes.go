// Autogenerated by Thrift Compiler (2.5.4-upfluence)
// DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING

package duration

import (
	"bytes"
	"fmt"
	"github.com/upfluence/thrift/lib/go/thrift"
)

// (needed to ensure safety because of naive import list construction.)
var _ = thrift.ZERO
var _ = fmt.Printf
var _ = bytes.Equal
var GoUnusedProtection__ int

// Attributes:
//   - Seconds
//   - Nanos
type Duration struct {
	Seconds int64 `thrift:"seconds,1,required" json:"seconds"`
	Nanos   int32 `thrift:"nanos,2,required" json:"nanos"`
}

func NewDuration() *Duration {
	return &Duration{}
}

var durationStructDefinition = thrift.StructDefinition{
	Namespace: Namespace,
	AnnotatedDefinition: thrift.AnnotatedDefinition{
		Name:                  "Duration",
		LegacyAnnotations:     map[string]string{},
		StructuredAnnotations: []thrift.RegistrableStruct{},
	},
	Fields: []thrift.FieldDefinition{
		{
			AnnotatedDefinition: thrift.AnnotatedDefinition{
				Name:                  "seconds",
				LegacyAnnotations:     map[string]string{},
				StructuredAnnotations: []thrift.RegistrableStruct{},
			},
		},

		{
			AnnotatedDefinition: thrift.AnnotatedDefinition{
				Name:                  "nanos",
				LegacyAnnotations:     map[string]string{},
				StructuredAnnotations: []thrift.RegistrableStruct{},
			},
		},
	},
}

func (p *Duration) StructDefinition() thrift.StructDefinition {
	return durationStructDefinition
}

func (p *Duration) GetSeconds() int64 {
	return p.Seconds
}

func (p *Duration) SetSeconds(v int64) {
	p.Seconds = v
}

func (p *Duration) GetNanos() int32 {
	return p.Nanos
}

func (p *Duration) SetNanos(v int32) {
	p.Nanos = v
}
func (p *Duration) Read(iprot thrift.TProtocol) error {
	if _, err := iprot.ReadStructBegin(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read error: ", p), err)
	}

	var issetSeconds bool = false
	var issetNanos bool = false

	for {
		_, fieldTypeId, fieldId, err := iprot.ReadFieldBegin()
		if err != nil {
			return thrift.PrependError(fmt.Sprintf("%T field %d read error: ", p, fieldId), err)
		}
		if fieldTypeId == thrift.STOP {
			break
		}
		switch fieldId {
		case 1:
			if err := p.ReadField1(iprot); err != nil {
				return err
			}
			issetSeconds = true
		case 2:
			if err := p.ReadField2(iprot); err != nil {
				return err
			}
			issetNanos = true
		default:
			if err := iprot.Skip(fieldTypeId); err != nil {
				return err
			}
		}
		if err := iprot.ReadFieldEnd(); err != nil {
			return err
		}
	}
	if err := iprot.ReadStructEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read struct end error: ", p), err)
	}
	if !issetSeconds {
		return thrift.NewTProtocolExceptionWithType(thrift.INVALID_DATA, fmt.Errorf("Required field Seconds is not set"))
	}
	if !issetNanos {
		return thrift.NewTProtocolExceptionWithType(thrift.INVALID_DATA, fmt.Errorf("Required field Nanos is not set"))
	}
	return nil
}

func (p *Duration) ReadField1(iprot thrift.TProtocol) error {
	if v, err := iprot.ReadI64(); err != nil {
		return thrift.PrependError("error reading field 1: ", err)
	} else {
		p.Seconds = v
	}
	return nil
}

func (p *Duration) ReadField2(iprot thrift.TProtocol) error {
	if v, err := iprot.ReadI32(); err != nil {
		return thrift.PrependError("error reading field 2: ", err)
	} else {
		p.Nanos = v
	}
	return nil
}

func (p *Duration) Write(oprot thrift.TProtocol) error {
	if err := oprot.WriteStructBegin("Duration"); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write struct begin error: ", p), err)
	}
	if err := p.writeField1(oprot); err != nil {
		return err
	}
	if err := p.writeField2(oprot); err != nil {
		return err
	}
	if err := oprot.WriteFieldStop(); err != nil {
		return thrift.PrependError("write field stop error: ", err)
	}
	if err := oprot.WriteStructEnd(); err != nil {
		return thrift.PrependError("write struct stop error: ", err)
	}
	return nil
}

func (p *Duration) writeField1(oprot thrift.TProtocol) (err error) {
	if err := oprot.WriteFieldBegin("seconds", thrift.I64, 1); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field begin error 1:seconds: ", p), err)
	}
	if err := oprot.WriteI64(int64(p.Seconds)); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T.seconds (1) field write error: ", p), err)
	}
	if err := oprot.WriteFieldEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field end error 1:seconds: ", p), err)
	}
	return err
}

func (p *Duration) writeField2(oprot thrift.TProtocol) (err error) {
	if err := oprot.WriteFieldBegin("nanos", thrift.I32, 2); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field begin error 2:nanos: ", p), err)
	}
	if err := oprot.WriteI32(int32(p.Nanos)); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T.nanos (2) field write error: ", p), err)
	}
	if err := oprot.WriteFieldEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field end error 2:nanos: ", p), err)
	}
	return err
}

func (p *Duration) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"Duration({seconds: %v, nanos: %v})",
		p.GetSeconds(),
		p.GetNanos(),
	)
}
