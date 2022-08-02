package thrift

import (
	"fmt"
	"io"
	"sync"
)

type TInboundStream interface {
	io.Closer

	Receive(Context, TRequest) error
}

type TOutboundStream interface {
	io.Closer

	Send(Context, TRequest) error
}

type tClientInboundStream struct {
	tBaseStream
}

func newTClientInboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tClientInboundStream {
	var unlockOnce sync.Once

	return &tClientInboundStream{
		tBaseStream: tBaseStream{
			name:          name,
			goAwayType:    SERVER_STREAM_GOAWAY,
			goAwayACKType: SERVER_STREAM_GOAWAY_ACK,
			out:           out,
			in:            in,
			seqID:         seqID,
			closec:        make(chan struct{}),
			closerFunc:    func() { unlockOnce.Do(func() { cl.mu.Unlock() }) },
		},
	}
}

func (s *tClientInboundStream) Receive(ctx Context, req TRequest) error {
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

	defer s.in.ReadMessageEnd()

	if seqID != s.seqID {
		s.close()
		return fmt.Errorf("invalid sequence ID, expected: %d", s.seqID)
	}

	switch typeID {
	case SERVER_STREAM_MESSAGE:
	case SERVER_STREAM_GOAWAY:
		s.writeGoAwayACK()
		s.goAwayOnce.Do(func() {})
		s.close()
		return io.EOF
	default:
		return fmt.Errorf("unexpected messaege type: %v", typeID)
	}

	return req.Read(s.in)
}

type tClientOutboundStream struct {
	tBaseStream
}

func newTClientOutboundStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tClientOutboundStream {
	var unlockOnce sync.Once

	cos := tClientOutboundStream{
		tBaseStream: tBaseStream{
			name:          name,
			goAwayType:    CLIENT_STREAM_GOAWAY,
			goAwayACKType: CLIENT_STREAM_GOAWAY_ACK,
			out:           out,
			in:            in,
			seqID:         seqID,
			closec:        make(chan struct{}),
			closerFunc:    func() { unlockOnce.Do(func() { cl.mu.Unlock() }) },
		},
	}

	go cos.readGoaway()

	return &cos
}

func (s *tClientOutboundStream) Close() error {
	select {
	case <-s.closec:
		return nil
	default:
	}

	if err := s.writeGoAway(); err != nil {
		return err
	}

	<-s.closec

	s.close()
	return nil
}

func (s *tClientOutboundStream) Send(ctx Context, req TRequest) error {
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

	if err := send(ctx, s.out, s.seqID, s.name, req, CLIENT_STREAM_MESSAGE); err != nil {
		s.close()
		return parseStreamingError(err)
	}

	return nil
}

type tServerOutboundStream struct {
	tBaseStream

	readyOnce sync.Once
	readyc    chan struct{}
}

func newTServerOutboundStream(name string, seqID int32, in, out TProtocol) *tServerOutboundStream {
	return &tServerOutboundStream{
		tBaseStream: tBaseStream{
			name:          name,
			goAwayType:    SERVER_STREAM_GOAWAY,
			goAwayACKType: SERVER_STREAM_GOAWAY_ACK,
			in:            in,
			out:           out,
			seqID:         seqID,
			closec:        make(chan struct{}),
		},
		readyc: make(chan struct{}, 1),
	}
}

func (s *tServerOutboundStream) ready() {
	s.readyOnce.Do(func() {
		s.readyc <- struct{}{}
		go s.readGoaway()
	})
}

func (s *tServerOutboundStream) Close() error {
	select {
	case <-s.closec:
		return nil
	default:
	}

	if err := s.writeGoAway(); err != nil {
		return err
	}

	<-s.closec

	s.close()
	return nil
}

func (s *tServerOutboundStream) Send(ctx Context, req TRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	case <-s.readyc:
	}

	defer func() { s.readyc <- struct{}{} }()

	if !s.out.Transport().IsOpen() {
		s.close()
		return io.EOF
	}

	if err := send(ctx, s.out, s.seqID, s.name, req, SERVER_STREAM_MESSAGE); err != nil {
		s.close()
		return parseStreamingError(err)
	}

	return nil
}

type tServerInboundStream struct {
	tBaseStream

	readyOnce sync.Once
	readyc    chan struct{}
}

func newTServerInboundStream(name string, seqID int32, in, out TProtocol) *tServerInboundStream {
	return &tServerInboundStream{
		tBaseStream: tBaseStream{
			name:          name,
			goAwayType:    CLIENT_STREAM_GOAWAY,
			goAwayACKType: CLIENT_STREAM_GOAWAY_ACK,
			in:            in,
			out:           out,
			seqID:         seqID,
			closec:        make(chan struct{}),
		},
		readyc: make(chan struct{}, 1),
	}
}

func (s *tServerInboundStream) ready() {
	s.readyOnce.Do(func() { s.readyc <- struct{}{} })
}

func (s *tServerInboundStream) Receive(ctx Context, req TRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closec:
		return io.EOF
	case <-s.readyc:
	}

	defer func() { s.readyc <- struct{}{} }()

	s.in.Transport().WriteContext(ctx)

	_, typeID, seqID, err := s.in.ReadMessageBegin()

	if err != nil {
		s.close()
		return parseStreamingError(err)
	}

	defer s.in.ReadMessageEnd()

	if seqID != s.seqID {
		s.close()
		return fmt.Errorf("invalid sequence ID, expected: %d", s.seqID)
	}

	switch typeID {
	case CLIENT_STREAM_MESSAGE:
	case CLIENT_STREAM_GOAWAY:
		s.writeGoAwayACK()
		s.goAwayOnce.Do(func() {})
		s.close()
		return io.EOF
	default:
		return fmt.Errorf("unexpected messaege type: %v", typeID)
	}

	return req.Read(s.in)
}

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
}

func (bs *tBaseStream) writeShell(mt TMessageType) error {
	if !bs.out.Transport().IsOpen() {
		return io.EOF
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

func (bs *tBaseStream) readGoAwayACK() error {
	mt, err := bs.readShell()

	if err != nil {
		return err
	}

	if mt != bs.goAwayACKType {
		return fmt.Errorf("invalid go away ack")
	}

	return nil
}

func (bs *tBaseStream) writeGoAway() error {
	return bs.writeShell(bs.goAwayType)
}

func (bs *tBaseStream) writeGoAwayACK() error {
	return bs.writeShell(bs.goAwayACKType)
}

func (bs *tBaseStream) readGoaway() {
	mt, err := bs.readShell()

	if err != nil {
		bs.close()
		return
	}

	switch mt {
	case bs.goAwayType:
		bs.writeGoAwayACK()
		bs.close()
	case bs.goAwayACKType:
		bs.close()
	}

	bs.close()
}

func (bs *tBaseStream) Close() error {
	defer bs.close()

	var err error

	bs.goAwayOnce.Do(func() {
		err = bs.writeGoAway()

		if err != nil {
			return
		}

		err = bs.readGoAwayACK()
	})

	return nil
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

type tBidiStream struct {
	*tBaseStream

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

	readyOnce sync.Once
	readyc    chan struct{}

	closingInbound  bool
	closingOutbound bool
	receivingc      chan struct{}

	writeMu sync.Mutex
}

func newTClientBidiStream(name string, seqID int32, in, out TProtocol, cl *TSyncClient) *tBidiStream {
	var (
		unlockOnce sync.Once

		readyc = make(chan struct{})
	)

	close(readyc)

	return &tBidiStream{
		tBaseStream: &tBaseStream{
			name:       name,
			in:         in,
			out:        out,
			seqID:      seqID,
			closec:     make(chan struct{}),
			closerFunc: func() { unlockOnce.Do(func() { cl.mu.Unlock() }) },
		},
		outboundClosec:        make(chan struct{}),
		inboundClosec:         make(chan struct{}),
		inboundMessageType:    SERVER_STREAM_MESSAGE,
		inboundGoAwayType:     SERVER_STREAM_GOAWAY,
		inboundGoAwayACKType:  SERVER_STREAM_GOAWAY_ACK,
		outboundMessageType:   CLIENT_STREAM_MESSAGE,
		outboundGoAwayType:    CLIENT_STREAM_GOAWAY,
		outboundGoAwayACKType: CLIENT_STREAM_GOAWAY_ACK,
		readyc:                readyc,
		receivingc:            make(chan struct{}, 1),
	}
}

func newTServerBidiStream(name string, seqID int32, in, out TProtocol) *tBidiStream {
	return &tBidiStream{
		tBaseStream: &tBaseStream{
			name:   name,
			in:     in,
			out:    out,
			seqID:  seqID,
			closec: make(chan struct{}),
		},
		outboundClosec:        make(chan struct{}),
		inboundClosec:         make(chan struct{}),
		inboundMessageType:    CLIENT_STREAM_MESSAGE,
		inboundGoAwayType:     CLIENT_STREAM_GOAWAY,
		inboundGoAwayACKType:  CLIENT_STREAM_GOAWAY_ACK,
		outboundMessageType:   SERVER_STREAM_MESSAGE,
		outboundGoAwayType:    SERVER_STREAM_GOAWAY,
		outboundGoAwayACKType: SERVER_STREAM_GOAWAY_ACK,
		readyc:                make(chan struct{}),
		receivingc:            make(chan struct{}, 1),
	}
}

func (bs *tBidiStream) ready() {
	bs.readyOnce.Do(func() { close(bs.readyc) })
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

func (bs *tBidiStream) processMessage(name string, typeID TMessageType, seqID int32, err error) error {
	if err != nil {
		return parseStreamingError(err)
	}

	defer bs.in.ReadMessageEnd()

	if seqID != bs.seqID {
		return fmt.Errorf("invalid sequence ID, expected: %d", bs.seqID)
	}

	switch typeID {
	case bs.inboundGoAwayType:
		if !bs.closingInbound {
			if err := bs.writeShell(bs.inboundGoAwayACKType); err != nil {
				bs.close()
				return err
			}
		}

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
	case bs.inboundGoAwayACKType:
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
	case bs.outboundGoAwayType:
		if !bs.closingOutbound {
			if err := bs.writeShell(bs.outboundGoAwayACKType); err != nil {
				bs.close()
				return err
			}
		}

		bs.outboundCloseOnce.Do(func() {
			close(bs.outboundClosec)
		})

		select {
		case <-bs.inboundClosec:
			bs.close()
		default:
		}

		return nil
	case bs.outboundGoAwayACKType:
		bs.outboundCloseOnce.Do(func() {
			close(bs.outboundClosec)
		})

		select {
		case <-bs.inboundClosec:
			bs.close()
		default:
		}

		return nil
	default:
		return fmt.Errorf("unexpected messaege type: %v", typeID)
	}
}

func (bs *tBidiStream) receiveOnce() error {
	name, typeID, seqID, err := bs.in.ReadMessageBegin()

	if err := bs.processMessage(name, typeID, seqID, err); err != nil && err != io.EOF {
		bs.close()
		return err
	}

	return nil
}

func (bs *tBidiStream) receive(ch chan struct{}) error {
	select {
	case bs.receivingc <- struct{}{}:
	case <-ch:
		bs.close()
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
	defer func() {
	}()

	select {
	case <-s.outboundClosec:
		return nil
	case <-s.closec:
		return nil
	default:
	}

	go s.receive(s.outboundClosec)

	s.closingOutbound = true

	if err := s.writeShell(s.outboundGoAwayType); err != nil {
		s.close()
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

	if !s.out.Transport().IsOpen() {
		s.close()
		return io.EOF
	}

	if err := send(ctx, s.out, s.seqID, s.name, req, s.outboundMessageType); err != nil {
		s.close()
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
	default:
	}

	go s.receive(s.inboundClosec)

	s.closingInbound = true

	if err := s.writeShell(s.inboundGoAwayType); err != nil && err != io.EOF {
		s.close()
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
	case s.receivingc <- struct{}{}:
	}

	defer func() { <-s.receivingc }()

	for {
		name, typeID, seqID, err := s.in.ReadMessageBegin()

		if typeID == s.inboundMessageType {
			defer s.in.ReadMessageEnd()
			return req.Read(s.in)
		}

		if err := s.processMessage(name, typeID, seqID, err); err != nil {
			if err != io.EOF {
				s.close()
			}
			return err
		}
	}
}
