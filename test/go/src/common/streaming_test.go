package common

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	thrift "github.com/upfluence/thrift/lib/go/thrift"
)

// tstr is a minimal TRequest/TResponse that wraps a single string field,
// used exclusively by the streaming integration tests.
type tstr struct {
	V string
}

func newTStr(v string) *tstr { return &tstr{V: v} }

func (s *tstr) GetResult() interface{} { return s }
func (s *tstr) GetError() error        { return nil }
func (s *tstr) String() string         { return s.V }

func (s *tstr) Write(prot thrift.TProtocol) error {
	return prot.WriteString(s.V)
}

func (s *tstr) Read(prot thrift.TProtocol) error {
	v, err := prot.ReadString()
	if err == nil {
		s.V = v
	}

	return err
}

// findFreePort finds an available TCP port on localhost.
func findFreePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}

	addr := l.Addr().String()
	l.Close()

	return addr
}

// startStreamingServer starts a TSimpleServer backed by the given processor
// on a random port and returns the address and a stop function.
func startStreamingServer(t *testing.T, p thrift.TProcessor) (addr string, stop func()) {
	t.Helper()

	addr = findFreePort(t)

	st, err := thrift.NewTServerSocket(addr)
	if err != nil {
		t.Fatalf("startStreamingServer: %v", err)
	}

	srv := thrift.NewTSimpleServer4(
		p,
		st,
		thrift.NewTTransportFactory(),
		thrift.NewTBinaryProtocolFactoryDefault(),
	)

	if err := srv.Listen(); err != nil {
		t.Fatalf("startStreamingServer Listen: %v", err)
	}

	go srv.AcceptLoop() //nolint:errcheck

	// Give the server a moment to start accepting.
	time.Sleep(10 * time.Millisecond)

	return addr, func() { srv.Stop() } //nolint:errcheck
}

// dialStreamingClient opens a TSyncClient connected to addr.
func dialStreamingClient(t *testing.T, addr string) (*thrift.TSyncClient, func()) {
	t.Helper()

	sock, err := thrift.NewTSocket(addr)
	if err != nil {
		t.Fatalf("dialStreamingClient NewTSocket: %v", err)
	}

	if err := sock.Open(); err != nil {
		t.Fatalf("dialStreamingClient Open: %v", err)
	}

	cl := thrift.NewTSyncClient(sock, thrift.NewTBinaryProtocolFactoryDefault())

	return cl, func() { sock.Close() } //nolint:errcheck
}

// --- server-streaming test ---

type serverStreamHandler struct {
	mu      sync.Mutex
	gotReq  string
	msgs    []string
	started chan struct{}
}

func (h *serverStreamHandler) Handle(_ thrift.Context, req thrift.TRequest, stream thrift.TOutboundStream) (thrift.TResponse, error) {
	ctx := context.Background()

	h.mu.Lock()
	h.gotReq = req.(*tstr).V
	h.mu.Unlock()

	close(h.started)

	go func() {
		stream.Send(ctx, newTStr("msg1")) //nolint:errcheck
		stream.Send(ctx, newTStr("msg2")) //nolint:errcheck
		stream.Send(ctx, newTStr("msg3")) //nolint:errcheck
		stream.Close()                    //nolint:errcheck
	}()

	return newTStr("ok"), nil
}

func TestStreamingServerIntegration(t *testing.T) {
	p := thrift.NewTStandardProcessor(nil)
	h := &serverStreamHandler{started: make(chan struct{})}

	p.AddProcessor(
		"srv_stream",
		thrift.NewTStreamServerProcessorFunction(
			p,
			"srv_stream",
			func() thrift.TRequest { return &tstr{} },
			h,
		),
	)

	addr, stop := startStreamingServer(t, p)
	defer stop()

	cl, closeClient := dialStreamingClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	var resp tstr

	istream, err := cl.StreamServer(ctx, "srv_stream", newTStr("hello"), &resp)
	if err != nil {
		t.Fatalf("StreamServer: %v", err)
	}

	if resp.V != "ok" {
		t.Errorf("initial response: want %q got %q", "ok", resp.V)
	}

	<-h.started

	var received []string
	var v tstr

	for {
		switch e := istream.Receive(ctx, &v); e {
		case nil:
			received = append(received, v.V)
			continue
		case io.EOF:
		default:
			t.Fatalf("Receive: %v", e)
		}

		break
	}

	if err := istream.Close(); err != nil {
		t.Errorf("istream.Close: %v", err)
	}

	h.mu.Lock()
	gotReq := h.gotReq
	h.mu.Unlock()

	if gotReq != "hello" {
		t.Errorf("handler got req %q, want %q", gotReq, "hello")
	}

	if len(received) != 3 || received[0] != "msg1" || received[1] != "msg2" || received[2] != "msg3" {
		t.Errorf("received messages: %v", received)
	}
}

// --- client-streaming test ---

type clientStreamHandler struct {
	mu     sync.Mutex
	gotReq string
	msgs   []string
	done   chan struct{}
}

func (h *clientStreamHandler) Handle(_ thrift.Context, req thrift.TRequest, sink thrift.TInboundStream) (thrift.TResponse, error) {
	ctx := context.Background()

	h.mu.Lock()
	h.gotReq = req.(*tstr).V
	h.mu.Unlock()

	go func() {
		defer close(h.done)

		var v tstr

		for {
			if err := sink.Receive(ctx, &v); err == io.EOF {
				sink.Close() //nolint:errcheck
				return
			}

			h.mu.Lock()
			h.msgs = append(h.msgs, v.V)
			h.mu.Unlock()
		}
	}()

	return newTStr("ack"), nil
}

func TestStreamingClientIntegration(t *testing.T) {
	p := thrift.NewTStandardProcessor(nil)
	h := &clientStreamHandler{done: make(chan struct{})}

	p.AddProcessor(
		"cli_stream",
		thrift.NewTStreamClientProcessorFunction(
			p,
			"cli_stream",
			func() thrift.TRequest { return &tstr{} },
			h,
		),
	)

	addr, stop := startStreamingServer(t, p)
	defer stop()

	cl, closeClient := dialStreamingClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	var resp tstr

	ostream, err := cl.StreamClient(ctx, "cli_stream", newTStr("start"), &resp)
	if err != nil {
		t.Fatalf("StreamClient: %v", err)
	}

	if resp.V != "ack" {
		t.Errorf("initial response: want %q got %q", "ack", resp.V)
	}

	for _, m := range []string{"a", "b", "c"} {
		if err := ostream.Send(ctx, newTStr(m)); err != nil {
			t.Fatalf("Send %q: %v", m, err)
		}
	}

	if err := ostream.Close(); err != nil {
		t.Fatalf("ostream.Close: %v", err)
	}

	<-h.done

	h.mu.Lock()
	gotReq := h.gotReq
	msgs := append([]string(nil), h.msgs...)
	h.mu.Unlock()

	if gotReq != "start" {
		t.Errorf("handler got req %q, want %q", gotReq, "start")
	}

	if len(msgs) != 3 || msgs[0] != "a" || msgs[1] != "b" || msgs[2] != "c" {
		t.Errorf("handler received: %v", msgs)
	}
}

// --- bidirectional streaming test ---

type bidiStreamHandler struct {
	mu     sync.Mutex
	gotReq string
	done   chan struct{}
}

func (h *bidiStreamHandler) Handle(_ thrift.Context, req thrift.TRequest, sink thrift.TInboundStream, stream thrift.TOutboundStream) (thrift.TResponse, error) {
	ctx := context.Background()

	h.mu.Lock()
	h.gotReq = req.(*tstr).V
	h.mu.Unlock()

	go func() {
		defer close(h.done)
		defer stream.Close() //nolint:errcheck
		defer sink.Close()   //nolint:errcheck

		var v tstr

		for {
			if err := sink.Receive(ctx, &v); err == io.EOF {
				return
			}

			pong := v.V + "-pong"
			stream.Send(ctx, newTStr(pong)) //nolint:errcheck
		}
	}()

	return newTStr("ready"), nil
}

func TestStreamingBidiIntegration(t *testing.T) {
	p := thrift.NewTStandardProcessor(nil)
	h := &bidiStreamHandler{done: make(chan struct{})}

	p.AddProcessor(
		"bidi_stream",
		thrift.NewTStreamBidiProcessorFunction(
			p,
			"bidi_stream",
			func() thrift.TRequest { return &tstr{} },
			h,
		),
	)

	addr, stop := startStreamingServer(t, p)
	defer stop()

	cl, closeClient := dialStreamingClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	var resp tstr

	istream, ostream, err := cl.StreamBidi(ctx, "bidi_stream", newTStr("begin"), &resp)
	if err != nil {
		t.Fatalf("StreamBidi: %v", err)
	}

	if resp.V != "ready" {
		t.Errorf("initial response: want %q got %q", "ready", resp.V)
	}

	pings := []string{"ping1", "ping2", "ping3"}
	var replies []string

	for _, ping := range pings {
		if err := ostream.Send(ctx, newTStr(ping)); err != nil {
			t.Fatalf("Send %q: %v", ping, err)
		}

		var v tstr

		if err := istream.Receive(ctx, &v); err != nil {
			t.Fatalf("Receive: %v", err)
		}

		replies = append(replies, v.V)
	}

	if err := ostream.Close(); err != nil {
		t.Fatalf("ostream.Close: %v", err)
	}

	if err := istream.Close(); err != nil {
		t.Fatalf("istream.Close: %v", err)
	}

	<-h.done

	expected := []string{"ping1-pong", "ping2-pong", "ping3-pong"}

	for i, want := range expected {
		if i >= len(replies) || replies[i] != want {
			t.Errorf("reply[%d]: want %q got %v", i, want, replies)
		}
	}

	h.mu.Lock()
	gotReq := h.gotReq
	h.mu.Unlock()

	if gotReq != "begin" {
		t.Errorf("handler got req %q, want %q", gotReq, "begin")
	}
}

// TestStreamingConnectionDrop verifies that when the server closes the
// connection mid-stream the client's Receive returns an error.
func TestStreamingConnectionDrop(t *testing.T) {
	p := thrift.NewTStandardProcessor(nil)
	dropConn := make(chan struct{})

	p.AddProcessor(
		"drop_stream",
		thrift.NewTStreamServerProcessorFunction(
			p,
			"drop_stream",
			func() thrift.TRequest { return &tstr{} },
			&dropServerStreamHandler{drop: dropConn},
		),
	)

	addr, stop := startStreamingServer(t, p)
	defer stop()

	cl, closeClient := dialStreamingClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	var resp tstr

	istream, err := cl.StreamServer(ctx, "drop_stream", newTStr("go"), &resp)
	if err != nil {
		t.Fatalf("StreamServer: %v", err)
	}

	// Tell the server handler to drop the connection.
	close(dropConn)

	// Receive should return a non-nil error once the connection is dropped.
	var v tstr

	err = istream.Receive(ctx, &v)
	if err == nil {
		t.Error("expected error after connection drop, got nil")
	}

	istream.Close() //nolint:errcheck
}

// dropServerStreamHandler sends the initial response, then closes the
// underlying transport to simulate a connection drop.
type dropServerStreamHandler struct {
	drop chan struct{}
}

func (h *dropServerStreamHandler) Handle(_ thrift.Context, _ thrift.TRequest, stream thrift.TOutboundStream) (thrift.TResponse, error) {
	// Wait for the test to signal us, then close the stream abruptly.
	go func() {
		<-h.drop
		stream.Close() //nolint:errcheck
	}()

	return newTStr("ok"), nil
}
