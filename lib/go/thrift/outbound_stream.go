package thrift

import "io"

type TOutboundStream interface {
	io.Closer

	Send(Context, TRequest) error
}

type tOutboundStream struct {
	tBaseStream

	messageType TMessageType
}

func newTClientOutboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tOutboundStream {
	return &tOutboundStream{
		tBaseStream: newTClientBaseStream(name, seqID, in, out, CLIENT_STREAM_GOAWAY, cl),
		messageType: CLIENT_STREAM_MESSAGE,
	}
}

func newTServerOutboundStream(name string, seqID int32, in, out TProtocol) *tOutboundStream {
	return &tOutboundStream{
		tBaseStream: newTServerBaseStream(name, seqID, in, out, SERVER_STREAM_GOAWAY),
		messageType: SERVER_STREAM_MESSAGE,
	}
}

func (s *tOutboundStream) readGoaway() {
	mt, err := s.readShell()

	if err != nil {
		s.close()
		return
	}

	if mt == s.goAwayType {
		s.writeGoAwayACK()
	}

	s.close()
}

func (s *tOutboundStream) ready() {
	s.readyOnce.Do(func() {
		close(s.readyc)
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

	if err := s.writeGoAway(); err != nil {
		s.close()
		return err
	}

	<-s.closec
	return nil
}

func (s *tOutboundStream) Send(ctx Context, req TRequest) error {
	return s.write(ctx, s.messageType, req)
}
