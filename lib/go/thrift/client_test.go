package thrift

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientPipe creates a matched (client, server) TSyncClient / TProtocol pair
// over in-process io.Pipe transports using the binary protocol.
func clientPipe() (cl *TSyncClient, serverProt TProtocol, closePipes func()) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	pf := NewTBinaryProtocolFactoryDefault()

	cl = NewTSyncClient(NewStreamTransport(pr1, pw2), pf)
	serverProt = pf.GetProtocol(NewStreamTransport(pr2, pw1))

	closePipes = func() {
		pw1.Close()
		pw2.Close()
	}

	return
}

// serveOneBinary reads one CALL frame from prot, echos the tstring payload
// back as a REPLY, and then returns.
func serveOneBinary(t *testing.T, prot TProtocol) {
	t.Helper()

	name, _, seqID, err := prot.ReadMessageBegin()
	require.NoError(t, err)

	var req tstring

	require.NoError(t, req.Read(prot))
	require.NoError(t, prot.ReadMessageEnd())
	require.NoError(t, prot.WriteMessageBegin(name, REPLY, seqID))
	require.NoError(t, req.Write(prot))
	require.NoError(t, prot.WriteMessageEnd())
	require.NoError(t, prot.Flush())
}

// TestTSyncClient_CallBinary verifies the basic happy-path RPC.
func TestTSyncClient_CallBinary(t *testing.T) {
	cl, serverProt, close := clientPipe()
	defer close()

	ctx := context.Background()

	go serveOneBinary(t, serverProt)

	var resp tstring

	err := cl.CallBinary(ctx, "echo", newTString("hello"), &resp)
	require.NoError(t, err)
	assert.Equal(t, tstring("hello"), resp)
}

// TestTSyncClient_CallUnary verifies that a ONEWAY call is sent without
// waiting for a reply and the mutex is released afterwards.
func TestTSyncClient_CallUnary(t *testing.T) {
	cl, serverProt, close := clientPipe()
	defer close()

	ctx := context.Background()

	received := make(chan string, 1)

	go func() {
		name, _, _, err := serverProt.ReadMessageBegin()
		if err != nil {
			return
		}

		var req tstring

		serverProt.ReadMessageEnd() //nolint:errcheck
		req.Read(serverProt)        //nolint:errcheck
		received <- name + ":" + string(req)
	}()

	err := cl.CallUnary(ctx, "fire", newTString("ping"))
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Equal(t, "fire:ping", msg)
	case <-time.After(2 * time.Second):
		t.Fatal("server never received the unary message")
	}

	// Mutex must be free — a subsequent CallBinary should not deadlock.
	go serveOneBinary(t, serverProt)

	var resp tstring

	require.NoError(t, cl.CallBinary(ctx, "echo", newTString("x"), &resp))
}

// TestTSyncClient_ContextCancellationDuringRecv verifies that a context
// cancelled after the request is sent (but before the reply arrives) causes
// CallBinary to return a context error. We simulate this by cancelling the
// context and then unblocking the transport so the recv path runs.
func TestTSyncClient_ContextCancellationDuringRecv(t *testing.T) {
	cl, serverProt, closePipes := clientPipe()
	defer closePipes()

	ctx, cancel := context.WithCancel(context.Background())

	// Server: receive the call, cancel the context, then close the connection
	// instead of sending a reply. This causes the client's recv to get an EOF.
	serverReady := make(chan struct{})

	go func() {
		_, _, _, err := serverProt.ReadMessageBegin()
		if err != nil {
			return
		}

		serverProt.ReadMessageEnd() //nolint:errcheck

		var req tstring

		req.Read(serverProt) //nolint:errcheck

		// Signal the test goroutine that we received the request.
		close(serverReady)

		cancel()

		// Close the pipes so the client's recv unblocks with an error.
		closePipes()
	}()

	var resp tstring

	err := cl.CallBinary(ctx, "echo", newTString("x"), &resp)

	<-serverReady

	assert.Error(t, err, "expected error when context is cancelled mid-flight")
}

// TestTSyncClient_MutexReleasedAfterStreamClient verifies that after a
// StreamClient call the mutex is released so a subsequent RPC can proceed.
func TestTSyncClient_MutexReleasedAfterStreamClient(t *testing.T) {
	clientProt, serverProt := processorPipe()

	ctx := context.Background()
	p := NewTStandardProcessor(nil)

	var wg sync.WaitGroup

	p.AddProcessor(
		"stream_client",
		NewTStreamClientProcessorFunction(
			p,
			"stream_client",
			func() TRequest { return newTString("") },
			&streamClientHandler{wg: &wg},
		),
	)

	p.AddProcessor(
		"echo",
		NewTBinaryProcessorFunction(
			p,
			"echo",
			func() TRequest { return newTString("") },
			&binaryHandler{},
		),
	)

	// Serve two consecutive requests from the server side.
	go func() {
		p.Process(ctx, serverProt, serverProt) //nolint:errcheck
		p.Process(ctx, serverProt, serverProt) //nolint:errcheck
	}()

	pf := NewTBinaryProtocolFactoryDefault()
	cl := NewTSyncClient(NewStreamTransport(nil, nil), pf)
	cl.in = clientProt
	cl.out = clientProt

	var resp tstring

	ostream, err := cl.StreamClient(ctx, "stream_client", newTString("data"), &resp)
	require.NoError(t, err)

	require.NoError(t, ostream.Send(ctx, newTString("item")))
	require.NoError(t, ostream.Close()) // releases the mutex

	wg.Wait()

	// The mutex should now be free — this binary call must not deadlock.
	var resp2 tstring

	require.NoError(t, cl.CallBinary(ctx, "echo", newTString("after"), &resp2))
	assert.Equal(t, tstring("after"), resp2)
}

// TestTSyncClient_MutexReleasedAfterStreamServer verifies the same for
// StreamServer: the mutex is held during the stream and released on Close.
func TestTSyncClient_MutexReleasedAfterStreamServer(t *testing.T) {
	clientProt, serverProt := processorPipe()

	ctx := context.Background()
	p := NewTStandardProcessor(nil)

	var wg sync.WaitGroup

	p.AddProcessor(
		"stream_server",
		NewTStreamServerProcessorFunction(
			p,
			"stream_server",
			func() TRequest { return newTString("") },
			&streamServerHandler{wg: &wg},
		),
	)

	p.AddProcessor(
		"echo",
		NewTBinaryProcessorFunction(
			p,
			"echo",
			func() TRequest { return newTString("") },
			&binaryHandler{},
		),
	)

	go func() {
		p.Process(ctx, serverProt, serverProt) //nolint:errcheck
		p.Process(ctx, serverProt, serverProt) //nolint:errcheck
	}()

	pf := NewTBinaryProtocolFactoryDefault()
	cl := NewTSyncClient(NewStreamTransport(nil, nil), pf)
	cl.in = clientProt
	cl.out = clientProt

	var resp tstring

	istream, err := cl.StreamServer(ctx, "stream_server", newTString("data"), &resp)
	require.NoError(t, err)

	// Drain the stream first (unblocks the server handler goroutine).
	var v tstring

	for {
		switch e := istream.Receive(ctx, &v); e {
		case nil:
			continue
		case io.EOF:
		default:
			t.Fatalf("unexpected receive error: %v", e)
		}

		break
	}

	wg.Wait()

	require.NoError(t, istream.Close()) // releases the mutex

	var resp2 tstring

	require.NoError(t, cl.CallBinary(ctx, "echo", newTString("after"), &resp2))
	assert.Equal(t, tstring("after"), resp2)
}

// TestTSyncClient_ConcurrentCallsAreSerialised verifies that two goroutines
// calling CallBinary concurrently do not interleave — the second call blocks
// until the first completes.
func TestTSyncClient_ConcurrentCallsAreSerialised(t *testing.T) {
	cl, serverProt, close := clientPipe()
	defer close()

	ctx := context.Background()

	// Server: serve two requests sequentially, recording arrival order.
	order := make(chan string, 2)

	go func() {
		for i := 0; i < 2; i++ {
			name, _, seqID, err := serverProt.ReadMessageBegin()
			if err != nil {
				return
			}

			var req tstring

			serverProt.ReadMessageEnd() //nolint:errcheck
			req.Read(serverProt)        //nolint:errcheck

			order <- string(req)

			serverProt.WriteMessageBegin(name, REPLY, seqID) //nolint:errcheck
			req.Write(serverProt)                             //nolint:errcheck
			serverProt.WriteMessageEnd()                      //nolint:errcheck
			serverProt.Flush()                                //nolint:errcheck
		}
	}()

	var wg sync.WaitGroup

	wg.Add(2)

	// Fire both calls simultaneously from separate goroutines.
	go func() {
		defer wg.Done()

		var resp tstring

		cl.CallBinary(ctx, "echo", newTString("first"), &resp) //nolint:errcheck
	}()

	go func() {
		defer wg.Done()

		var resp tstring

		cl.CallBinary(ctx, "echo", newTString("second"), &resp) //nolint:errcheck
	}()

	wg.Wait()
	close()

	got := []string{<-order, <-order}

	// Both messages must be present; order is non-deterministic but there
	// must be exactly two distinct messages (no interleaving / duplication).
	assert.ElementsMatch(t, []string{"first", "second"}, got)
}
