package thrift

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

type TInboundStream interface {
	io.Closer

	Receive(Context, TRequest) error
}

type tInboundStream struct {
	tBaseStream

	messageType TMessageType

	// acked is closed once the acknowledgement of the go away this side sent
	// was read, by Close or by a Receive going on at the same time.
	acked   chan struct{}
	ackOnce sync.Once
}

func (s *tInboundStream) ack() {
	s.ackOnce.Do(func() { close(s.acked) })
}

func newTClientInboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tInboundStream {
	return &tInboundStream{
		tBaseStream: newTClientBaseStream(name, seqID, in, out, SERVER_STREAM_GOAWAY, cl),
		messageType: SERVER_STREAM_MESSAGE,
		acked:       make(chan struct{}),
	}
}

func newTServerInboundStream(name string, seqID int32, in, out TProtocol) *tInboundStream {
	return &tInboundStream{
		tBaseStream: newTServerBaseStream(name, seqID, in, out, CLIENT_STREAM_GOAWAY),
		messageType: CLIENT_STREAM_MESSAGE,
		acked:       make(chan struct{}),
	}
}

func (s *tInboundStream) Close() error {
	defer s.close()

	var err error

	s.goAwayOnce.Do(func() {
		s.startHandshake(s.acked)

		err = s.writeGoAway()

		if err != nil {
			return
		}

		err = s.readGoAwayACK()
	})

	if err != nil {
		s.markAborted()
	}

	return nil
}

// readGoAwayACK waits for the acknowledgement of the go away that was sent.
// The other side may have started closing the stream too, or may still be
// sending messages: they are skipped, and a go away is acknowledged, so that
// nothing of the stream is left to be read as the reply of the next call. A
// Receive going on at the same time may read the acknowledgement first.
func (s *tInboundStream) readGoAwayACK() error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go func() {
		select {
		case <-s.acked:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		select {
		case <-s.acked:
			return nil
		default:
		}

		var mt TMessageType

		_, err := s.readMessage(ctx, func(t TMessageType) error {
			mt = t

			if t == s.messageType {
				if err := SkipDefaultDepth(s.in, STRUCT); err != nil {
					return err
				}
			}

			if err := s.in.ReadMessageEnd(); err != nil {
				return err
			}

			// The stream is over before the slot is given back: a Receive
			// waiting for it would otherwise start a read nothing answers.
			if t == s.goAwayACKType {
				s.ack()
				s.close()
			}

			return nil
		})

		if err != nil {
			select {
			case <-s.acked:
				return nil
			default:
			}

			if errors.Is(err, io.EOF) {
				// the stream was closed by a Receive reading the go away
				return nil
			}

			return err
		}

		switch mt {
		case s.goAwayACKType:
			s.ack()

			return nil
		case s.goAwayType:
			if err := s.writeGoAwayACK(); err != nil {
				return err
			}
		case s.messageType:
		default:
			return fmt.Errorf("invalid go away ack")
		}
	}
}

func (s *tInboundStream) Receive(ctx Context, req TRequest) error {
	begun, err := s.readMessage(ctx, func(typeID TMessageType) error {
		switch typeID {
		case s.messageType:
			defer s.in.ReadMessageEnd()
			return req.Read(s.in)
		case s.goAwayType:
			s.in.ReadMessageEnd()
			s.writeGoAwayACK()
			s.goAwayOnce.Do(func() {})
			s.close()
			return io.EOF
		case s.goAwayACKType:
			s.in.ReadMessageEnd()
			s.ack()
			s.close()
			return io.EOF
		default:
			s.in.ReadMessageEnd()
			return fmt.Errorf("unexpected messaege type: %v", typeID)
		}
	})

	if !begun {
		return parseStreamingError(err)
	}

	return err
}
