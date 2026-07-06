package common

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	thrift "github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/test/go/gen/streamingtest"
)

// findFreeStreamingAddr returns a free TCP address on localhost.
func findFreeStreamingAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("findFreeStreamingAddr: %v", err)
	}

	addr := l.Addr().String()
	l.Close()

	return addr
}

// startStreamingTestServer starts a TSimpleServer backed by SimpleStreamingHandler
// on a free port using the binary protocol and plain transport.
// It returns the listening address and a stop function.
func startStreamingTestServer(t *testing.T) (addr string, stop func()) {
	t.Helper()

	handler := &SimpleStreamingHandler{}
	processor := streamingtest.NewStreamingTestProcessor(handler, nil)

	addr = findFreeStreamingAddr(t)

	serverTransport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		t.Fatalf("NewTServerSocket: %v", err)
	}

	server := thrift.NewTSimpleServer4(
		processor,
		serverTransport,
		thrift.NewTTransportFactory(),
		thrift.NewTBinaryProtocolFactoryDefault(),
	)

	if err := server.Listen(); err != nil {
		t.Fatalf("server.Listen: %v", err)
	}

	go server.AcceptLoop() //nolint:errcheck

	// Give the server a moment to start.
	time.Sleep(10 * time.Millisecond)

	return addr, func() { server.Stop() } //nolint:errcheck
}

// dialStreamingTestClient opens a StreamingTestClient connected to addr.
func dialStreamingTestClient(t *testing.T, addr string) (*streamingtest.StreamingTestClient, func()) {
	t.Helper()

	sock, err := thrift.NewTSocket(addr)
	if err != nil {
		t.Fatalf("NewTSocket: %v", err)
	}

	if err := sock.Open(); err != nil {
		t.Fatalf("socket.Open: %v", err)
	}

	cl := thrift.NewTSyncClient(sock, thrift.NewTBinaryProtocolFactoryDefault())

	return streamingtest.NewStreamingTestClient(cl), func() { sock.Close() } //nolint:errcheck
}

// TestStreamingClientStream verifies the client-side streaming (sink) path:
// the client sends N strings through the sink and closes it without error.
func TestStreamingClientStream(t *testing.T) {
	addr, stop := startStreamingTestServer(t)
	defer stop()

	client, closeClient := dialStreamingTestClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	const nmsgs = 5

	sink, err := client.TestClientStream_(ctx, "test-client-stream")
	if err != nil {
		t.Fatalf("TestClientStream_: %v", err)
	}

	for i := 0; i < nmsgs; i++ {
		if sendErr := sink.Send(ctx, fmt.Sprintf("msg%d", i)); sendErr != nil {
			t.Fatalf("sink.Send[%d]: %v", i, sendErr)
		}
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("sink.Close: %v", err)
	}
}

// TestStreamingServerStream verifies the server-side streaming (stream) path:
// the server pushes N strings back and the client reads them all.
func TestStreamingServerStream(t *testing.T) {
	addr, stop := startStreamingTestServer(t)
	defer stop()

	client, closeClient := dialStreamingTestClient(t, addr)
	defer closeClient()

	ctx := context.Background()
	const count = 4
	const prefix = "hello"

	stream, err := client.TestServerStream_(ctx, count, prefix)
	if err != nil {
		t.Fatalf("TestServerStream_: %v", err)
	}

	var received []string

	for {
		msg, recvErr := stream.Receive(ctx)
		if recvErr == io.EOF {
			break
		}

		if recvErr != nil {
			t.Fatalf("stream.Receive: %v", recvErr)
		}

		received = append(received, msg)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close: %v", err)
	}

	if len(received) != count {
		t.Fatalf("want %d messages, got %d: %v", count, len(received), received)
	}

	for i, msg := range received {
		want := fmt.Sprintf("%s_%d", prefix, i)
		if msg != want {
			t.Errorf("received[%d]: want %q, got %q", i, want, msg)
		}
	}
}

// TestStreamingBidi verifies the bidirectional streaming path:
// the client sends strings and the server echoes them uppercased.
func TestStreamingBidi(t *testing.T) {
	addr, stop := startStreamingTestServer(t)
	defer stop()

	client, closeClient := dialStreamingTestClient(t, addr)
	defer closeClient()

	ctx := context.Background()

	stream, sink, err := client.TestBidi(ctx, "bidi-label")
	if err != nil {
		t.Fatalf("TestBidi: %v", err)
	}

	pings := []string{"hello", "world", "foo"}

	for _, ping := range pings {
		if sendErr := sink.Send(ctx, ping); sendErr != nil {
			t.Fatalf("sink.Send(%q): %v", ping, sendErr)
		}

		reply, recvErr := stream.Receive(ctx)
		if recvErr != nil {
			t.Fatalf("stream.Receive: %v", recvErr)
		}

		if reply != strings.ToUpper(ping) {
			t.Errorf("bidi reply: want %q, got %q", strings.ToUpper(ping), reply)
		}
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("sink.Close: %v", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("stream.Close: %v", err)
	}
}
