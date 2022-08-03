package thrift

import (
	"fmt"
	"io"
	"sync"
)

type tBaseStream struct {
	out TProtocol
	in  TProtocol

	goAwayType    TMessageType
	goAwayACKType TMessageType

	goAwayOnce sync.Once
	closerFunc func()

	name  string
	seqID int32

	closeOnce sync.Once
	closec    chan struct{}

	readyOnce sync.Once
	readyc    chan struct{}
}

func newTServerBaseStream(name string, seqID int32, in, out TProtocol, goAwayType TMessageType) tBaseStream {
	return tBaseStream{
		name:          name,
		goAwayType:    goAwayType,
		goAwayACKType: goAwayType + 1,
		in:            in,
		out:           out,
		seqID:         seqID,
		closec:        make(chan struct{}),
		readyc:        make(chan struct{}),
	}
}

func newTClientBaseStream(name string, seqID int32, in, out TProtocol, goAwayType TMessageType, cl *TSyncClient) tBaseStream {
	var unlockOnce sync.Once

	return tBaseStream{
		name:          name,
		goAwayType:    goAwayType,
		goAwayACKType: goAwayType + 1,
		in:            in,
		out:           out,
		seqID:         seqID,
		closec:        make(chan struct{}),
		closerFunc:    func() { unlockOnce.Do(func() { cl.mu.Unlock() }) },
		readyc:        make(chan struct{}),
	}
}

func (s *tBaseStream) ready() {
	s.readyOnce.Do(func() { close(s.readyc) })
}

func (bs *tBaseStream) writeShell(mt TMessageType) error {
	if !bs.out.Transport().IsOpen() {
		return io.EOF
	}

	select {
	case <-bs.closec:
		return io.EOF
	default:
	}

	if err := bs.out.WriteMessageBegin(bs.name, mt, bs.seqID); err != nil {
		return err
	}

	if err := bs.out.WriteMessageEnd(); err != nil {
		return err
	}

	return bs.out.Flush()
}

func (bs *tBaseStream) readShell() (TMessageType, error) {
	if !bs.out.Transport().IsOpen() {
		return 0, io.EOF
	}

	select {
	case <-bs.closec:
		return 0, io.EOF
	default:
	}

	name, typeID, seqID, err := bs.in.ReadMessageBegin()

	if err != nil {
		return 0, err
	}

	if name != bs.name {
		return 0, fmt.Errorf("invalid method name, expected: %q", bs.name)
	}

	if seqID != bs.seqID {
		return 0, fmt.Errorf("invalid sequence ID, expected: %d", bs.seqID)
	}

	return typeID, bs.in.ReadMessageEnd()
}

func (bs *tBaseStream) writeGoAway() error {
	return bs.writeShell(bs.goAwayType)
}

func (bs *tBaseStream) writeGoAwayACK() error {
	return bs.writeShell(bs.goAwayACKType)
}

func (bs *tBaseStream) close() {
	if fn := bs.closerFunc; fn != nil {
		fn()
	}

	bs.closeOnce.Do(func() { close(bs.closec) })
}

func parseStreamingError(err error) error {
	if terr, ok := err.(TTransportException); ok && terr.TypeId() == END_OF_FILE {
		return io.EOF
	}

	return err
}
