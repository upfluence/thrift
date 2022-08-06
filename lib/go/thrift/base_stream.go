package thrift

import (
	"context"
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

func (bs *tBaseStream) write(ctx Context, typeID TMessageType, req TStruct) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-bs.closec:
		return io.EOF
	case <-bs.readyc:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-bs.closec:
		return io.EOF
	default:
	}

	if err := bs.out.Transport().WriteContext(ctx); err != nil {
		return err
	}

	if err := send(ctx, bs.out, bs.seqID, bs.name, req, typeID); err != nil {
		bs.close()
		return parseStreamingError(err)
	}

	return nil
}

func (bs *tBaseStream) writeShell(mt TMessageType) error {
	return bs.write(context.Background(), mt, nil)
}
func (bs *tBaseStream) readMessageBegin(ctx Context) (TMessageType, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-bs.closec:
		return 0, io.EOF
	case <-bs.readyc:
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-bs.closec:
		return 0, io.EOF
	default:
	}

	if err := bs.in.Transport().WriteContext(ctx); err != nil {
		return 0, err
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

	return typeID, nil
}

func (bs *tBaseStream) readShell() (TMessageType, error) {
	var typeID, err = bs.readMessageBegin(context.Background())

	if err != nil {
		return 0, err
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
