package thrift

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silentPeerStream returns a server stream whose peer reads nothing and
// writes nothing: what it sends is discarded, and nothing ever arrives.
func silentPeerStream(t *testing.T) *tOutboundStream {
	t.Helper()

	pr, pw := io.Pipe()

	t.Cleanup(func() {
		pr.Close()
		pw.Close()
	})

	var (
		pf = NewTBinaryProtocolFactoryDefault()
		tr = NewStreamTransport(pr, discardWriteCloser{})
	)

	return newTServerOutboundStream("stream", 1, pf.GetProtocol(tr), pf.GetProtocol(tr))
}

type discardWriteCloser struct{}

func (discardWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriteCloser) Close() error                { return nil }

func setStreamTimeouts(s *tBaseStream, handshake, idle time.Duration) {
	s.handshakeTimeout, s.idleTimeout = handshake, idle
}

func closedWithin(s *tBaseStream, d time.Duration) bool {
	select {
	case <-s.closec:
		return true
	case <-time.After(d):
		return false
	}
}

func TestStreamHandshakeTimeout(t *testing.T) {
	var (
		s    = silentPeerStream(t)
		done = make(chan error, 1)
	)

	setStreamTimeouts(&s.tBaseStream, 100*time.Millisecond, 0)

	s.ready()

	go func() { done <- s.Close() }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("the close of a stream whose go away is never acknowledged did not return")
	}

	assert.True(t, s.aborted.Load())
}

func TestStreamHandshakeTimeoutDisabled(t *testing.T) {
	s := silentPeerStream(t)

	setStreamTimeouts(&s.tBaseStream, 0, 0)

	s.ready()

	done := make(chan struct{})

	go func() {
		defer close(done)

		s.Close() //nolint:errcheck
	}()

	assert.False(t, closedWithin(&s.tBaseStream, 300*time.Millisecond))

	s.abort()

	<-done
}

// ctxTransport is a stream transport whose pending read is released when the
// context it was given is done, as the ones of uthrift are.
type ctxTransport struct {
	*StreamTransport

	pr  *io.PipeReader
	ctx Context
}

func (t *ctxTransport) WriteContext(ctx Context) error {
	t.ctx = ctx

	return nil
}

func (t *ctxTransport) Read(p []byte) (int, error) {
	stop := context.AfterFunc(t.ctx, func() { t.pr.CloseWithError(t.ctx.Err()) })

	defer stop()

	return t.StreamTransport.Read(p)
}

func TestStreamIdleTimeout(t *testing.T) {
	s := silentPeerStream(t)

	setStreamTimeouts(&s.tBaseStream, 5*time.Second, 100*time.Millisecond)

	s.ready()

	require.True(t, closedWithin(&s.tBaseStream, 2*time.Second), "a quiet stream was not aborted")
	assert.True(t, s.aborted.Load())
}

func TestStreamIdleTimeoutResetByTraffic(t *testing.T) {
	s := silentPeerStream(t)

	setStreamTimeouts(&s.tBaseStream, 5*time.Second, 300*time.Millisecond)

	s.ready()

	for range 5 {
		time.Sleep(100 * time.Millisecond)

		require.NoError(t, s.Send(defaultCtx, newTString("foo")))
	}

	assert.False(t, s.aborted.Load(), "a stream sending messages was aborted")
	assert.True(t, closedWithin(&s.tBaseStream, 2*time.Second), "a quiet stream was not aborted")
}

func TestStreamAbortReleasesPendingRead(t *testing.T) {
	var (
		pf     = NewTBinaryProtocolFactoryDefault()
		pr, pw = io.Pipe()
		tr     = &ctxTransport{StreamTransport: NewStreamTransport(pr, discardWriteCloser{}), pr: pr}
		s      = newTServerInboundStream("stream", 1, pf.GetProtocol(tr), pf.GetProtocol(tr))
	)

	t.Cleanup(func() { pw.Close() })

	setStreamTimeouts(&s.tBaseStream, 5*time.Second, 0)

	s.ready()

	errc := make(chan error, 1)

	go func() { errc <- s.Receive(defaultCtx, newTString("")) }()

	time.Sleep(50 * time.Millisecond)

	s.abort()

	select {
	case err := <-errc:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("the abort did not release the pending read")
	}
}
