package thrift

import (
	"errors"
	"io"
	"sync"
)

type TOutboundStream interface {
	io.Closer

	Send(Context, TRequest) error
}

type tOutboundStream struct {
	tBaseStream

	messageType TMessageType

	// closingMu orders the go away of this side and the one of the other
	// side. goAwaySent is set once this side sends its go away, which the other
	// side is then going to acknowledge, and peerGoAway once it read the one of
	// the other side: only the first of the two is sent, the other side having
	// finished with the stream by the time a second one would arrive.
	closingMu  sync.Mutex
	goAwaySent bool
	peerGoAway bool

	// goAwayWritten is closed once the go away of this side was written, or
	// could not be: the acknowledgement of the one of the other side comes
	// after it, so that the other side, which is done once its own go away is
	// acknowledged, still reads it.
	goAwayWritten     chan struct{}
	goAwayWrittenOnce sync.Once
}

func (s *tOutboundStream) written() {
	s.goAwayWrittenOnce.Do(func() { close(s.goAwayWritten) })
}

func newTClientOutboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tOutboundStream {
	return &tOutboundStream{
		tBaseStream: newTClientBaseStream(name, seqID, in, out, CLIENT_STREAM_GOAWAY, cl),
		messageType: CLIENT_STREAM_MESSAGE,

		goAwayWritten: make(chan struct{}),
	}
}

func newTServerOutboundStream(name string, seqID int32, in, out TProtocol) *tOutboundStream {
	return &tOutboundStream{
		tBaseStream: newTServerBaseStream(name, seqID, in, out, SERVER_STREAM_GOAWAY),
		messageType: SERVER_STREAM_MESSAGE,

		goAwayWritten: make(chan struct{}),
	}
}

// readGoaway reads what closes the stream: the go away of the other side,
// which is acknowledged, and the acknowledgement of the go away this side
// sent. When both sides close at the same time, both are read, so that none is
// left on the connection to be read as the next request.
func (s *tOutboundStream) readGoaway() {
	defer s.close()

	for {
		mt, err := s.readShell()

		if err != nil {
			s.markAborted()

			return
		}

		switch mt {
		case s.goAwayType:
			s.noMoreData.Store(true)

			s.closingMu.Lock()
			s.peerGoAway = true
			crossing := s.goAwaySent
			s.closingMu.Unlock()

			if crossing {
				select {
				case <-s.goAwayWritten:
				case <-s.closec:
				}
			}

			s.writeGoAwayACK()

			if !crossing {
				return
			}
		case s.goAwayACKType:
			return
		default:
			s.markAborted()

			return
		}
	}
}

func (s *tOutboundStream) ready() {
	s.readyOnce.Do(func() {
		close(s.readyc)
		s.startIdle()
		go s.readGoaway()
	})
}

func (s *tOutboundStream) Close() error {
	select {
	case <-s.readyc:
	case <-s.closec:
		return nil
	}

	select {
	case <-s.closec:
		return nil
	default:
	}

	s.closingMu.Lock()

	// A second Close, the one of the handler and the one of the processor, has
	// nothing left to send.
	if s.peerGoAway || s.goAwaySent {
		s.closingMu.Unlock()

		<-s.closec

		return nil
	}

	s.goAwaySent = true
	s.closingMu.Unlock()

	s.startHandshake(s.closec)

	s.noMoreData.Store(true)

	err := s.writeGoAway()

	s.written()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		s.abort()

		return err
	}

	<-s.closec
	return nil
}

func (s *tOutboundStream) Send(ctx Context, req TRequest) error {
	return s.write(ctx, s.messageType, req)
}
