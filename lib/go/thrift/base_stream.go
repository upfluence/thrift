package thrift

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultStreamHandshakeTimeout bounds how long a stream waits, once it
	// started to close, for the other side to answer: nothing of what is left
	// on the connection can be trusted after it, so the stream is aborted, and
	// its connection with it. A value of zero or less disables it.
	DefaultStreamHandshakeTimeout = 5 * time.Second

	// DefaultStreamIdleTimeout aborts the streams that neither received nor
	// sent a message for that long. A value of zero or less, the default,
	// disables it, as a quiet stream is a legitimate one.
	DefaultStreamIdleTimeout time.Duration = 0
)

// ReadContexter is implemented by the transports that are told, before each
// message of a stream is read, the context of the read: it is a start of a
// message, so what was left of the previous one is forgotten. It is called
// instead of WriteContext, which belongs to the writes of a message.
type ReadContexter interface {
	ReadContext(ctx Context) error
}

// ReadContext hands the context of a read to a transport.
func ReadContext(t TTransport, ctx Context) error {
	if rc, ok := t.(ReadContexter); ok {
		return rc.ReadContext(ctx)
	}

	return t.WriteContext(ctx)
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

	// aborted is set when the stream did not end with the go away
	// handshake, but because of an error: what the other side sent, or is
	// going to send, is then left on the connection.
	aborted atomic.Bool

	// abortCtx is cancelled once the stream is aborted, which releases the
	// reads of the stream that are waiting on the transport.
	abortCtx    context.Context
	abortCancel context.CancelFunc

	// onAbort is called when the stream is aborted, to drop the connection
	// that can not be reused.
	onAbort func()

	handshakeTimeout time.Duration
	idleTimeout      time.Duration

	// lastActivity is the time of the last message read or written, as the
	// time elapsed since epoch.
	epoch        time.Time
	lastActivity atomic.Int64

	// watch is the goroutine, started once, that sleeps until the first
	// deadline of the stream, the idle one or the one of a handshake, with one
	// timer. wakec tells it a deadline was added.
	watchOnce    sync.Once
	wakec        chan struct{}
	handshakesMu sync.Mutex
	handshakes   []handshake

	// noMoreData is set once a go away was sent or received: nothing but the
	// messages of the closing handshake is written after it, the other side
	// reads no more of them.
	noMoreData atomic.Bool

	readyOnce sync.Once
	readyc    chan struct{}

	// writec and readc hold a slot for the whole time a message is written or
	// read: a stream is written and read by several goroutines (the caller,
	// and the one waiting for the go away), and the transport and the
	// protocols it goes through only support one message at a time in each
	// direction. Reading and writing still go on at the same time.
	writec chan struct{}
	readc  chan struct{}

	// A stream that is closed can still be in the middle of writing a message,
	// which the next one to use the connection has to wait for.
	writingMu sync.Mutex
	writing   int
	quiet     []func()
}

func newTServerBaseStream(name string, seqID int32, in, out TProtocol, goAwayType TMessageType) tBaseStream {
	return newTBaseStream(
		name,
		seqID,
		in,
		out,
		goAwayType,
		DefaultStreamHandshakeTimeout,
		DefaultStreamIdleTimeout,
		nil,
		nil,
	)
}

func newTClientBaseStream(name string, seqID int32, in, out TProtocol, goAwayType TMessageType, cl *TSyncClient) tBaseStream {
	var unlockOnce sync.Once

	return newTBaseStream(
		name,
		seqID,
		in,
		out,
		goAwayType,
		cl.handshakeTimeout,
		cl.idleTimeout,
		func() { in.Transport().Close() },
		func() { unlockOnce.Do(func() { cl.mu.Unlock() }) },
	)
}

func newTBaseStream(
	name string,
	seqID int32,
	in, out TProtocol,
	goAwayType TMessageType,
	handshakeTimeout, idleTimeout time.Duration,
	onAbort, closerFunc func(),
) tBaseStream {
	abortCtx, abortCancel := context.WithCancel(context.Background())

	return tBaseStream{
		abortCtx:         abortCtx,
		abortCancel:      abortCancel,
		epoch:            time.Now(),
		wakec:            make(chan struct{}, 1),
		handshakeTimeout: handshakeTimeout,
		idleTimeout:      idleTimeout,
		onAbort:          onAbort,
		name:             name,
		goAwayType:       goAwayType,
		goAwayACKType:    goAwayType + 1,
		in:               in,
		out:              out,
		seqID:            seqID,
		closec:           make(chan struct{}),
		closerFunc:       closerFunc,
		readyc:           make(chan struct{}),
		writec:           make(chan struct{}, 1),
		readc:            make(chan struct{}, 1),
	}
}

// ready opens the stream: its idle timeout starts to run.
func (s *tBaseStream) ready() {
	s.readyOnce.Do(func() {
		close(s.readyc)
		s.startIdle()
	})
}

// acquire takes the slot of c, unless the context is done or the stream
// closed first.
func (bs *tBaseStream) acquire(ctx Context, c chan struct{}) error {
	select {
	case c <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-bs.closec:
		return io.EOF
	}
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

	bs.beginWrite()

	defer bs.endWrite()

	if err := bs.acquire(ctx, bs.writec); err != nil {
		return err
	}

	defer func() { <-bs.writec }()

	// The stream may have been closed while waiting for the slot, by the
	// message acknowledging its go away: nothing is to be written after it.
	select {
	case <-bs.closec:
		return io.EOF
	default:
	}

	if bs.noMoreData.Load() && typeID != bs.goAwayType && typeID != bs.goAwayACKType {
		return io.EOF
	}

	if err := bs.out.Transport().WriteContext(ctx); err != nil {
		return err
	}

	if err := send(ctx, bs.out, bs.seqID, bs.name, req, typeID); err != nil {
		bs.abort()
		return parseStreamingError(err)
	}

	bs.touch()

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

	if err := ReadContext(bs.in.Transport(), ctx); err != nil {
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

// readMessage reads one message: fn is given its type and has to read what is
// left of it, which is done before another message can be read. It tells
// whether the message was begun, in which case the error is the one of fn.
func (bs *tBaseStream) readMessage(ctx Context, fn func(TMessageType) error) (bool, error) {
	ctx, done := bs.withAbort(ctx)

	defer done()

	if err := bs.acquire(ctx, bs.readc); err != nil {
		return false, err
	}

	defer func() { <-bs.readc }()

	typeID, err := bs.readMessageBegin(ctx)

	if err != nil {
		return false, err
	}

	bs.touch()

	return true, fn(typeID)
}

// withAbort makes ctx done once the stream is aborted too, which is what
// releases a read of the stream waiting on its transport. The context has to
// be kept for the whole message: the transport reads what is left of it with
// the one it was given.
func (bs *tBaseStream) withAbort(ctx Context) (Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(bs.abortCtx, cancel)

	return ctx, func() {
		stop()
		cancel()
	}
}

func (bs *tBaseStream) readShell() (TMessageType, error) {
	var typeID TMessageType

	_, err := bs.readMessage(bs.abortCtx, func(mt TMessageType) error {
		typeID = mt

		return bs.in.ReadMessageEnd()
	})

	if err != nil {
		return 0, err
	}

	return typeID, nil
}

func (bs *tBaseStream) writeGoAway() error {
	return bs.writeShell(bs.goAwayType)
}

func (bs *tBaseStream) writeGoAwayACK() error {
	return bs.writeShell(bs.goAwayACKType)
}

// markAborted records that the stream ended because of an error, and drops
// its connection: it can not be trusted anymore.
func (bs *tBaseStream) markAborted() {
	if bs.aborted.Swap(true) {
		return
	}

	if bs.abortCancel != nil {
		bs.abortCancel()
	}

	if bs.onAbort != nil {
		bs.onAbort()
	}
}

// abort closes the stream that ended because of an error.
func (bs *tBaseStream) abort() {
	bs.markAborted()
	bs.close()
}

// handshake is a go away that was sent: the stream is aborted if done is not
// closed by the deadline.
type handshake struct {
	deadline time.Time
	done     <-chan struct{}
}

// startHandshake aborts the stream if done is not closed once the timeout
// elapsed: it is called when a go away is sent, which the other side has to
// acknowledge.
func (bs *tBaseStream) startHandshake(done <-chan struct{}) {
	d := bs.handshakeTimeout

	if d <= 0 {
		return
	}

	bs.handshakesMu.Lock()
	bs.handshakes = append(bs.handshakes, handshake{deadline: time.Now().Add(d), done: done})
	bs.handshakesMu.Unlock()

	bs.watchOnce.Do(func() { go bs.watch() })

	select {
	case bs.wakec <- struct{}{}:
	default:
	}
}

// startIdle aborts the stream once it did not receive nor send a message for
// the idle timeout.
func (bs *tBaseStream) startIdle() {
	if bs.idleTimeout <= 0 {
		return
	}

	bs.touch()

	bs.watchOnce.Do(func() { go bs.watch() })
}

// nextDeadline returns the first deadline of the stream, the zero time if
// there is none.
func (bs *tBaseStream) nextDeadline() time.Time {
	var next time.Time

	if bs.idleTimeout > 0 {
		next = bs.epoch.Add(time.Duration(bs.lastActivity.Load()) + bs.idleTimeout)
	}

	bs.handshakesMu.Lock()
	defer bs.handshakesMu.Unlock()

	pending := bs.handshakes[:0]

	for _, h := range bs.handshakes {
		select {
		case <-h.done:
			continue
		default:
		}

		pending = append(pending, h)

		if next.IsZero() || h.deadline.Before(next) {
			next = h.deadline
		}
	}

	bs.handshakes = pending

	return next
}

// watch aborts the stream when a deadline passes. It sleeps until the first
// one, and goes back to sleep if it moved in the meantime: the stream was used,
// or a handshake was acknowledged.
func (bs *tBaseStream) watch() {
	t := time.NewTimer(time.Hour)

	defer t.Stop()

	for {
		next := bs.nextDeadline()

		switch {
		case next.IsZero():
			t.Stop()
		case !time.Now().Before(next):
			bs.abort()

			return
		default:
			t.Reset(time.Until(next))
		}

		select {
		case <-bs.closec:
			return
		case <-bs.wakec:
		case <-t.C:
		}
	}
}

func (bs *tBaseStream) touch() {
	bs.lastActivity.Store(int64(time.Since(bs.epoch)))
}

func (bs *tBaseStream) close() {
	if fn := bs.closerFunc; fn != nil {
		bs.whenQuiet(fn)
	}

	bs.closeOnce.Do(func() { close(bs.closec) })
}

func (bs *tBaseStream) beginWrite() {
	bs.writingMu.Lock()
	bs.writing++
	bs.writingMu.Unlock()
}

func (bs *tBaseStream) endWrite() {
	bs.writingMu.Lock()

	bs.writing--

	var fns []func()

	if bs.writing == 0 {
		fns, bs.quiet = bs.quiet, nil
	}

	bs.writingMu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

// whenQuiet runs fn once no message is being written anymore.
func (bs *tBaseStream) whenQuiet(fn func()) {
	bs.writingMu.Lock()

	if bs.writing > 0 {
		bs.quiet = append(bs.quiet, fn)
		bs.writingMu.Unlock()

		return
	}

	bs.writingMu.Unlock()

	fn()
}

// quiesce waits for the messages being written to be written.
// It gives up after the handshake timeout, and aborts the stream: a message
// that is never written leaves the connection in an unknown state.
func (bs *tBaseStream) quiesce() {
	done := make(chan struct{})

	bs.whenQuiet(func() { close(done) })

	d := bs.handshakeTimeout

	if d <= 0 {
		<-done

		return
	}

	t := time.NewTimer(d)

	defer t.Stop()

	select {
	case <-done:
	case <-t.C:
		bs.markAborted()
	}
}

func parseStreamingError(err error) error {
	if terr, ok := err.(TTransportException); ok && terr.TypeId() == END_OF_FILE {
		return io.EOF
	}

	return err
}
