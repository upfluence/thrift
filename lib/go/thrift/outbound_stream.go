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
		return err
	}

	<-s.closec
	return nil
}

func (s *tOutboundStream) Send(ctx Context, req TRequest) error {
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

	if !s.out.Transport().IsOpen() {
		s.close()
		return io.EOF
	}

	if err := send(ctx, s.out, s.seqID, s.name, req, s.messageType); err != nil {
		s.close()
		return parseStreamingError(err)
	}

	return nil
}
