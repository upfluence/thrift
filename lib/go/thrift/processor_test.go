package thrift

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// binaryHandler is a TBinaryHandler that echoes its request.
type binaryHandler struct {
	called bool
	retErr error
}

func (h *binaryHandler) Handle(_ Context, req TRequest) (TResponse, error) {
	h.called = true

	if h.retErr != nil {
		return nil, h.retErr
	}

	return req.(TResponse), nil
}

// testUnaryHandler is a TUnaryHandler.
type testUnaryHandler struct {
	called bool
	retErr error
}

func (h *testUnaryHandler) Handle(_ Context, _ TRequest) error {
	h.called = true
	return h.retErr
}

// processorPipe builds a symmetric in-process pipe pair wired with the binary
// protocol. The caller gets (clientProt, serverProt) where writes on one side
// are reads on the other.
func processorPipe() (clientProt, serverProt TProtocol) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	pf := NewTBinaryProtocolFactoryDefault()

	clientProt = pf.GetProtocol(NewStreamTransport(pr1, pw2))
	serverProt = pf.GetProtocol(NewStreamTransport(pr2, pw1))

	return
}

// rawCall writes a vanilla CALL frame carrying a tstring payload.
func rawCall(t *testing.T, prot TProtocol, name string, seqID int32, arg string) {
	t.Helper()

	require.NoError(t, prot.WriteMessageBegin(name, CALL, seqID))
	require.NoError(t, newTString(arg).Write(prot))
	require.NoError(t, prot.WriteMessageEnd())
	require.NoError(t, prot.Flush())
}

// TestTStandardProcessor_DispatchKnown verifies that a registered processor
// function is invoked and its REPLY frame reaches the caller.
func TestTStandardProcessor_DispatchKnown(t *testing.T) {
	clientProt, serverProt := processorPipe()

	h := &binaryHandler{}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"echo",
		NewTBinaryProcessorFunction(
			p,
			"echo",
			func() TRequest { return newTString("") },
			h,
		),
	)

	ctx := context.Background()

	go func() {
		ok, err := p.Process(ctx, serverProt, serverProt)
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	rawCall(t, clientProt, "echo", 1, "hello")

	name, typeID, _, err := clientProt.ReadMessageBegin()
	require.NoError(t, err)
	assert.Equal(t, "echo", name)
	assert.Equal(t, REPLY, typeID)

	var result tstring

	require.NoError(t, result.Read(clientProt))
	require.NoError(t, clientProt.ReadMessageEnd())

	assert.Equal(t, tstring("hello"), result)
	assert.True(t, h.called)
}

// TestTStandardProcessor_DispatchUnknown verifies that an unregistered method
// produces an EXCEPTION frame with UNKNOWN_METHOD type id.
func TestTStandardProcessor_DispatchUnknown(t *testing.T) {
	clientProt, serverProt := processorPipe()

	p := NewTStandardProcessor(nil)
	ctx := context.Background()

	go func() {
		ok, _ := p.Process(ctx, serverProt, serverProt)
		assert.False(t, ok)
	}()

	require.NoError(t, clientProt.WriteMessageBegin("no_such_method", CALL, 1))
	require.NoError(t, clientProt.WriteStructBegin("no_such_method_args"))
	require.NoError(t, clientProt.WriteFieldStop())
	require.NoError(t, clientProt.WriteStructEnd())
	require.NoError(t, clientProt.WriteMessageEnd())
	require.NoError(t, clientProt.Flush())

	name, typeID, _, err := clientProt.ReadMessageBegin()
	require.NoError(t, err)
	assert.Equal(t, "no_such_method", name)
	assert.Equal(t, EXCEPTION, typeID)

	var ex tApplicationException

	require.NoError(t, ex.Read(clientProt))
	require.NoError(t, clientProt.ReadMessageEnd())
	assert.Equal(t, int32(UNKNOWN_METHOD), ex.TypeId())
}

// TestTBinaryProcessorFunction_HandlerError verifies that a handler error
// produces an EXCEPTION frame with INTERNAL_ERROR type id.
func TestTBinaryProcessorFunction_HandlerError(t *testing.T) {
	clientProt, serverProt := processorPipe()

	h := &binaryHandler{retErr: errors.New("boom")}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"echo",
		NewTBinaryProcessorFunction(p, "echo", func() TRequest { return newTString("") }, h),
	)

	ctx := context.Background()

	go p.Process(ctx, serverProt, serverProt) //nolint:errcheck

	rawCall(t, clientProt, "echo", 1, "req")

	_, typeID, _, err := clientProt.ReadMessageBegin()
	require.NoError(t, err)
	assert.Equal(t, EXCEPTION, typeID)

	var ex tApplicationException

	require.NoError(t, ex.Read(clientProt))
	require.NoError(t, clientProt.ReadMessageEnd())
	assert.Equal(t, int32(INTERNAL_ERROR), ex.TypeId())
}

// TestTUnaryProcessorFunction_NoReply verifies that a one-way call is handled
// (handler called) and that no reply frame is written to the output protocol.
func TestTUnaryProcessorFunction_NoReply(t *testing.T) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	pf := NewTBinaryProtocolFactoryDefault()
	clientProt := pf.GetProtocol(NewStreamTransport(pr1, pw2))
	serverProt := pf.GetProtocol(NewStreamTransport(pr2, pw1))

	h := &testUnaryHandler{}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"ping",
		NewTUnaryProcessorFunction(p, "ping", func() TRequest { return newTString("") }, h),
	)

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		p.Process(ctx, serverProt, serverProt) //nolint:errcheck
		close(done)
	}()

	require.NoError(t, clientProt.WriteMessageBegin("ping", ONEWAY, 1))
	require.NoError(t, newTString("hi").Write(clientProt))
	require.NoError(t, clientProt.WriteMessageEnd())
	require.NoError(t, clientProt.Flush())

	<-done
	assert.True(t, h.called)

	// Close the write end of the server→client pipe so the client's
	// ReadMessageBegin returns an error, confirming no REPLY was written.
	pw1.Close()

	_, _, _, err := clientProt.ReadMessageBegin()
	assert.Error(t, err, "expected error: no REPLY should have been written for a unary call")
}

// TestTStreamServerProcessorFunction exercises server-streaming: the handler
// pushes two messages and closes; the client reads both then gets io.EOF.
func TestTStreamServerProcessorFunction(t *testing.T) {
	clientProt, serverProt := processorPipe()

	ctx := context.Background()
	p := NewTStandardProcessor(nil)

	var wg sync.WaitGroup

	h := &streamServerHandler{wg: &wg}
	p.AddProcessor(
		"stream_server",
		NewTStreamServerProcessorFunction(
			p,
			"stream_server",
			func() TRequest { return newTString("") },
			h,
		),
	)

	processorDone := make(chan struct{})

	go func() {
		defer close(processorDone)

		ok, err := p.Process(ctx, serverProt, serverProt)
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	pf := NewTBinaryProtocolFactoryDefault()
	cl := NewTSyncClient(NewStreamTransport(nil, nil), pf)
	cl.in = clientProt
	cl.out = clientProt

	var resp tstring

	istream, err := cl.StreamServer(ctx, "stream_server", newTString("foo"), &resp)
	require.NoError(t, err)
	assert.Equal(t, tstring("resp"), resp)

	// Read until EOF — this also unblocks the server goroutine's sends.
	var (
		msgs []string
		v    tstring
	)

	for {
		switch recvErr := istream.Receive(ctx, &v); recvErr {
		case nil:
			msgs = append(msgs, string(v))
			continue
		case io.EOF:
		default:
			t.Fatalf("unexpected receive error: %v", recvErr)
		}

		break
	}

	require.NoError(t, istream.Close())

	<-processorDone
	wg.Wait()

	assert.Equal(t, "foo", h.req)
	assert.Equal(t, []string{"bar", "biz"}, msgs)
}

// TestTStreamClientProcessorFunction exercises client-streaming: the client
// sends two messages then closes; the server collects them all.
func TestTStreamClientProcessorFunction(t *testing.T) {
	clientProt, serverProt := processorPipe()

	ctx := context.Background()
	p := NewTStandardProcessor(nil)

	var wg sync.WaitGroup

	h := &streamClientHandler{wg: &wg}
	p.AddProcessor(
		"stream_client",
		NewTStreamClientProcessorFunction(
			p,
			"stream_client",
			func() TRequest { return newTString("") },
			h,
		),
	)

	processorDone := make(chan struct{})

	go func() {
		defer close(processorDone)

		ok, err := p.Process(ctx, serverProt, serverProt)
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	pf := NewTBinaryProtocolFactoryDefault()
	cl := NewTSyncClient(NewStreamTransport(nil, nil), pf)
	cl.in = clientProt
	cl.out = clientProt

	var resp tstring

	ostream, err := cl.StreamClient(ctx, "stream_client", newTString("foo"), &resp)
	require.NoError(t, err)
	assert.Equal(t, tstring("resp"), resp)

	require.NoError(t, ostream.Send(ctx, newTString("msg1")))
	require.NoError(t, ostream.Send(ctx, newTString("msg2")))
	require.NoError(t, ostream.Close())

	<-processorDone
	wg.Wait()

	assert.Equal(t, "foo", h.req)
	assert.Equal(t, []string{"msg1", "msg2"}, h.streamMsgs)
}

// TestTBinaryProcessorFunction_MiddlewareIntegration verifies that the
// HandleBinaryRequest middleware hook is called when a binary processor
// function processes a request.
func TestTBinaryProcessorFunction_MiddlewareIntegration(t *testing.T) {
	clientProt, serverProt := processorPipe()

	called := false
	mw := &funcMiddleware{
		handleBinary: func(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
			called = true
			return next(ctx, req)
		},
	}

	p := NewTStandardProcessor([]TMiddleware{mw})

	p.AddProcessor(
		"echo",
		NewTBinaryProcessorFunction(
			p,
			"echo",
			func() TRequest { return newTString("") },
			&binaryHandler{},
		),
	)

	ctx := context.Background()

	go p.Process(ctx, serverProt, serverProt) //nolint:errcheck

	rawCall(t, clientProt, "echo", 1, "x")

	_, typeID, _, err := clientProt.ReadMessageBegin()
	require.NoError(t, err)
	assert.Equal(t, REPLY, typeID)

	var result tstring

	require.NoError(t, result.Read(clientProt))
	require.NoError(t, clientProt.ReadMessageEnd())

	assert.True(t, called, "middleware HandleBinaryRequest was not called")
}

// funcMiddleware is a TMiddleware backed by function fields for easy testing.
type funcMiddleware struct {
	handleBinary func(Context, string, int32, TRequest, func(Context, TRequest) (TResponse, error)) (TResponse, error)
	handleUnary  func(Context, string, int32, TRequest, func(Context, TRequest) error) error
}

func (m *funcMiddleware) HandleBinaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	if m.handleBinary != nil {
		return m.handleBinary(ctx, mth, seqID, req, next)
	}

	return next(ctx, req)
}

func (m *funcMiddleware) HandleUnaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) error) error {
	if m.handleUnary != nil {
		return m.handleUnary(ctx, mth, seqID, req, next)
	}

	return next(ctx, req)
}
