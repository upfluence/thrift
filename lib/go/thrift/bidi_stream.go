package thrift

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type tBidiStream struct {
	tBaseStream

	outboundCloseOnce sync.Once
	outboundClosec    chan struct{}

	inboundCloseOnce sync.Once
	inboundClosec    chan struct{}

	inboundMessageType   TMessageType
	inboundGoAwayType    TMessageType
	inboundGoAwayACKType TMessageType

	outboundMessageType   TMessageType
	outboundGoAwayType    TMessageType
	outboundGoAwayACKType TMessageType

	closingInbound  atomic.Bool
	closingOutbound atomic.Bool
	receivingc      chan struct{}

	writeMu sync.Mutex
}

func newTClientBidiStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tBidiStream {
	return &tBidiStream{
		tBaseStream:           newTClientBaseStream(name, seqID, in, out, 0, cl),
		outboundClosec:        make(chan struct{}),
		inboundClosec:         make(chan struct{}),
		inboundMessageType:    SERVER_STREAM_MESSAGE,
		inboundGoAwayType:     SERVER_STREAM_GOAWAY,
		inboundGoAwayACKType:  SERVER_STREAM_GOAWAY_ACK,
		outboundMessageType:   CLIENT_STREAM_MESSAGE,
		outboundGoAwayType:    CLIENT_STREAM_GOAWAY,
		outboundGoAwayACKType: CLIENT_STREAM_GOAWAY_ACK,
		receivingc:            make(chan struct{}, 1),
	}
}

func newTServerBidiStream(name string, seqID int32, in, out TProtocol) *tBidiStream {
	return &tBidiStream{
		tBaseStream:           newTServerBaseStream(name, seqID, in, out, 0),
		outboundClosec:        make(chan struct{}),
		inboundClosec:         make(chan struct{}),
		inboundMessageType:    CLIENT_STREAM_MESSAGE,
		inboundGoAwayType:     CLIENT_STREAM_GOAWAY,
		inboundGoAwayACKType:  CLIENT_STREAM_GOAWAY_ACK,
		outboundMessageType:   SERVER_STREAM_MESSAGE,
		outboundGoAwayType:    SERVER_STREAM_GOAWAY,
		outboundGoAwayACKType: SERVER_STREAM_GOAWAY_ACK,
		receivingc:            make(chan struct{}, 1),
	}
}

func (bs *tBidiStream) Close() error {
	bs.close()
	return nil
}

func (bs *tBidiStream) writeShell(mt TMessageType) error {
	bs.writeMu.Lock()
	defer bs.writeMu.Unlock()

	return bs.tBaseStream.writeShell(mt)
}

func (bs *tBidiStream) closeInbound() error {
	bs.inboundCloseOnce.Do(func() {
		close(bs.inboundClosec)

		select {
		case <-bs.outboundClosec:
		default:
			go bs.receive(bs.outboundClosec)
		}
	})

	select {
	case <-bs.outboundClosec:
		bs.close()
	default:
	}

	return io.EOF
}

func (bs *tBidiStream) closeOutbound() error {
	bs.outboundCloseOnce.Do(func() {
		close(bs.outboundClosec)
	})

	select {
	case <-bs.inboundClosec:
		bs.close()
	default:
	}

	return nil
}

func (bs *tBidiStream) processMessage(typeID TMessageType, err error) error {
	if err != nil {
		return err
	}

	defer bs.in.ReadMessageEnd()

	switch typeID {
	case bs.inboundGoAwayType:
		closing := bs.closingInbound.Swap(true)
		if !closing {
			if err := bs.writeShell(bs.inboundGoAwayACKType); err != nil {
				bs.abort()
				return err
			}
		}

		return bs.closeInbound()
	case bs.inboundGoAwayACKType:
		return bs.closeInbound()
	case bs.outboundGoAwayType:
		closing := bs.closingOutbound.Swap(true)
		if !closing {
			if err := bs.writeShell(bs.outboundGoAwayACKType); err != nil {
				bs.abort()
				return err
			}
		}

		return bs.closeOutbound()
	case bs.outboundGoAwayACKType:
		return bs.closeOutbound()
	case bs.inboundMessageType:
		return SkipDefaultDepth(bs.in, STRUCT)
	default:
		return fmt.Errorf("unexpected messaege type: %v", typeID)
	}
}

func (bs *tBidiStream) receiveOnce() error {
	ctx, done := bs.withAbort(context.Background())

	defer done()

	typeID, err := bs.readMessageBegin(ctx)

	if err := bs.processMessage(typeID, err); err != nil && err != io.EOF {
		bs.abort()
		return err
	}

	return nil
}

func (bs *tBidiStream) receive(ch chan struct{}) error {
	select {
	case bs.receivingc <- struct{}{}:
	case <-ch:
		return nil
	case <-bs.closec:
		return nil
	}

	defer func() { <-bs.receivingc }()

	for {
		if err := bs.receiveOnce(); err != nil {
			return err
		}

		select {
		case <-ch:
			return nil
		case <-bs.closec:
			return nil
		default:
		}
	}
}

type tOutboundBidiStream struct {
	*tBidiStream
}

func (s *tOutboundBidiStream) Close() error {
	select {
	case <-s.outboundClosec:
		return nil
	case <-s.closec:
		return nil
	case <-s.readyc:
	}

	// Whoever sets it first closes the half: this side with its go away, or the
	// receiver acknowledging the one of the other side. A go away sent after
	// the one of the other side was acknowledged is left on the connection.
	if s.closingOutbound.Swap(true) {
		select {
		case <-s.outboundClosec:
		case <-s.closec:
		}

		return nil
	}

	s.startHandshake(s.outboundClosec)

	go s.receive(s.outboundClosec)

	if err := s.writeShell(s.outboundGoAwayType); err != nil && err != io.EOF {
		s.abort()
		return err
	}

	select {
	case <-s.outboundClosec:
	case <-s.closec:
	}

	return nil
}

func (s *tOutboundBidiStream) Send(ctx Context, req TRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.outboundClosec:
		return io.EOF
	case <-s.closec:
		return io.EOF
	case <-s.readyc:
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.closingOutbound.Load() {
		return io.EOF
	}

	if err := s.write(ctx, s.outboundMessageType, req); err != nil {
		s.abort()
		return parseStreamingError(err)
	}

	return nil
}

type tInboundBidiStream struct {
	*tBidiStream
}

func (s *tInboundBidiStream) Close() error {
	select {
	case <-s.inboundClosec:
		return nil
	case <-s.closec:
		return nil
	case <-s.readyc:
	}

	if s.closingInbound.Swap(true) {
		select {
		case <-s.inboundClosec:
		case <-s.closec:
		}

		return nil
	}

	s.startHandshake(s.inboundClosec)

	go s.receive(s.inboundClosec)

	if err := s.writeShell(s.inboundGoAwayType); err != nil && err != io.EOF {
		s.abort()
		return err
	}

	select {
	case <-s.inboundClosec:
	case <-s.closec:
	}

	return nil
}

func (s *tInboundBidiStream) Receive(ctx Context, req TRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	case <-s.inboundClosec:
		return io.EOF
	case <-s.readyc:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	case <-s.inboundClosec:
		return io.EOF
	case s.receivingc <- struct{}{}:
	}

	defer func() { <-s.receivingc }()

	ctx, done := s.withAbort(ctx)

	defer done()

	for {
		typeID, err := s.readMessageBegin(ctx)

		if typeID == s.inboundMessageType {
			defer s.in.ReadMessageEnd()
			return req.Read(s.in)
		}

		if err := s.processMessage(typeID, err); err != nil {
			if err != io.EOF {
				s.abort()
			}
			return err
		}
	}
}
