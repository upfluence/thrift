package common

import (
	"context"
	"fmt"
	"io"
	"strings"

	thrift "github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/test/go/gen/streamingtest"
)

// SimpleStreamingHandler is a deterministic StreamingTestHandler used by the
// cross-language streaming integration tests.
type SimpleStreamingHandler struct{}

// TestClientStream_ spawns a goroutine that drains the sink. Returns nil
// immediately so the server can send the initial REPLY.
func (h *SimpleStreamingHandler) TestClientStream_(
	_ thrift.Context,
	_ string,
	sink streamingtest.StreamingTestTestClientStreamSinkInboundStream,
) error {
	ctx := context.Background()

	go func() {
		defer sink.Close() //nolint:errcheck

		for {
			_, err := sink.Receive(ctx)
			if err == io.EOF {
				return
			}
		}
	}()

	return nil
}

// TestServerStream_ spawns a goroutine that pushes `count` strings of the
// form "{prefix}_{i}" into the outbound stream and then closes it.  It returns
// immediately so that the server can send the initial REPLY before the goroutine
// starts sending stream messages.
func (h *SimpleStreamingHandler) TestServerStream_(
	_ thrift.Context,
	count int32,
	prefix string,
	stream streamingtest.StreamingTestTestServerStreamStreamOutboundStream,
) error {
	ctx := context.Background()

	go func() {
		defer stream.Close() //nolint:errcheck

		for i := int32(0); i < count; i++ {
			if err := stream.Send(ctx, fmt.Sprintf("%s_%d", prefix, i)); err != nil {
				return
			}
		}
	}()

	return nil
}

// TestBidi spawns a goroutine that echoes each string received from the client
// back uppercased.  It returns immediately so the server can send the initial
// REPLY.  The stream and sink I/O happen after ready() is called.
func (h *SimpleStreamingHandler) TestBidi(
	_ thrift.Context,
	_ string,
	stream streamingtest.StreamingTestTestBidiStreamOutboundStream,
	sink streamingtest.StreamingTestTestBidiSinkInboundStream,
) error {
	ctx := context.Background()

	go func() {
		defer stream.Close() //nolint:errcheck
		defer sink.Close()   //nolint:errcheck

		for {
			msg, err := sink.Receive(ctx)
			if err == io.EOF {
				return
			}

			if err != nil {
				return
			}

			if sendErr := stream.Send(ctx, strings.ToUpper(msg)); sendErr != nil {
				return
			}
		}
	}()

	return nil
}

