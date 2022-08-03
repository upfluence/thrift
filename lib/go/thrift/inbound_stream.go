package thrift

import (
	"fmt"
	"io"
)

type TInboundStream interface {
	io.Closer

	Receive(Context, TRequest) error
}

type tInboundStream struct {
	tBaseStream

	messageType TMessageType
}

func newTClientInboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tInboundStream {
	return &tInboundStream{
		tBaseStream: newTClientBaseStream(name, seqID, in, out, SERVER_STREAM_GOAWAY, cl),
		messageType: SERVER_STREAM_MESSAGE,
	}
}

func newTServerInboundStream(name string, seqID int32, in, out TProtocol) *tInboundStream {
	return &tInboundStream{
		tBaseStream: newTServerBaseStream(name, seqID, in, out, CLIENT_STREAM_GOAWAY),
		messageType: CLIENT_STREAM_MESSAGE,
	}
}

func (s *tInboundStream) Close() error {
	defer s.close()

	var err error

	s.goAwayOnce.Do(func() {
		err = s.writeGoAway()

		if err != nil {
			return
		}

		err = s.readGoAwayACK()
	})

	return nil
}

func (s *tInboundStream) readGoAwayACK() error {
	mt, err := s.readShell()

	if err != nil {
		return err
	}

	if mt != s.goAwayACKType {
		return fmt.Errorf("invalid go away ack")
	}

	return nil
}

func (s *tInboundStream) Receive(ctx Context, req TRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	case <-s.readyc:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	default:
	}

	s.in.Transport().WriteContext(ctx)

	_, typeID, seqID, err := s.in.ReadMessageBegin()

	if err != nil {
		s.close()
		return parseStreamingError(err)
	}

	if seqID != s.seqID {
		s.in.ReadMessageEnd()
		s.close()
		return fmt.Errorf("invalid sequence ID, expected: %d", s.seqID)
	}

	switch typeID {
	case s.messageType:
	case s.goAwayType:
		s.in.ReadMessageEnd()
		s.writeGoAwayACK()
		s.goAwayOnce.Do(func() {})
		s.close()
		return io.EOF
	default:
		s.in.ReadMessageEnd()
		return fmt.Errorf("unexpected messaege type: %v", typeID)
	}

	defer s.in.ReadMessageEnd()
	return req.Read(s.in)
}
