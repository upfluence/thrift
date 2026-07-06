/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package main

import (
	"compress/zlib"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/test/go/gen/streamingtest"
)

var host = flag.String("host", "localhost", "Host to connect")
var port = flag.Int64("port", 9090, "Port number to connect")
var domainSocket = flag.String("domain-socket", "", "Unix domain socket path")
var transport = flag.String("transport", "buffered", "Transport: buffered, framed, zlib")
var protocol = flag.String("protocol", "binary", "Protocol: binary, compact, json, header")
var ssl = flag.Bool("ssl", false, "Use SSL")

func main() {
	flag.Parse()

	var protocolFactory thrift.TProtocolFactory

	switch *protocol {
	case "compact":
		protocolFactory = thrift.NewTCompactProtocolFactory()
	case "json":
		protocolFactory = thrift.NewTJSONProtocolFactory()
	case "header":
		protocolFactory = thrift.NewTHeaderProtocolFactory()
	default:
		protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	}

	var (
		trans thrift.TTransport
		err   error
	)

	hostPort := fmt.Sprintf("%s:%d", *host, *port)

	if *ssl {
		trans, err = thrift.NewTSSLSocket(hostPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	} else if *domainSocket != "" {
		trans, err = thrift.NewTSocket(*domainSocket)
	} else {
		trans, err = thrift.NewTSocket(hostPort)
	}

	if err != nil {
		log.Fatalf("Failed to create socket: %v", err)
	}

	switch *transport {
	case "framed":
		trans = thrift.NewTFramedTransport(trans)
	case "zlib":
		trans, err = thrift.NewTZlibTransport(trans, zlib.BestCompression)
		if err != nil {
			log.Fatalf("Failed to create zlib transport: %v", err)
		}
	case "buffered":
		fallthrough
	default:
		trans = thrift.NewTBufferedTransport(trans, 8192)
	}

	if err := trans.Open(); err != nil {
		log.Fatalf("Failed to open transport: %v", err)
	}

	defer trans.Close() //nolint:errcheck

	cl := streamingtest.NewStreamingTestClient(
		thrift.NewTSyncClient(trans, protocolFactory),
	)

	ctx := context.Background()

	testClientStream(ctx, cl)
	testServerStream(ctx, cl)
	testBidi(ctx, cl)

	log.Printf("All streaming tests passed")
}

// testClientStream sends 5 strings through the sink and closes it.
func testClientStream(ctx context.Context, cl *streamingtest.StreamingTestClient) {
	sink, err := cl.TestClientStream_(ctx, "go-client")
	if err != nil {
		log.Fatalf("testClientStream: TestClientStream_ failed: %v", err)
	}

	msgs := []string{"alpha", "beta", "gamma", "delta", "epsilon"}

	for _, m := range msgs {
		if err := sink.Send(ctx, m); err != nil {
			log.Fatalf("testClientStream: sink.Send(%q) failed: %v", m, err)
		}
	}

	if err := sink.Close(); err != nil {
		log.Fatalf("testClientStream: sink.Close failed: %v", err)
	}

	log.Printf("testClientStream: OK (sent %d messages)", len(msgs))
}

// testServerStream requests 3 messages with prefix "x" and verifies they
// arrive as "x_0", "x_1", "x_2".
func testServerStream(ctx context.Context, cl *streamingtest.StreamingTestClient) {
	const count = 3
	const prefix = "x"

	stream, err := cl.TestServerStream_(ctx, count, prefix)
	if err != nil {
		log.Fatalf("testServerStream: TestServerStream_ failed: %v", err)
	}

	var received []string

	for {
		msg, recvErr := stream.Receive(ctx)

		if recvErr == io.EOF {
			break
		}

		if recvErr != nil {
			log.Fatalf("testServerStream: stream.Receive failed: %v", recvErr)
		}

		received = append(received, msg)
	}

	if err := stream.Close(); err != nil {
		log.Fatalf("testServerStream: stream.Close failed: %v", err)
	}

	if len(received) != count {
		log.Fatalf("testServerStream: expected %d messages, got %d: %v", count, len(received), received)
	}

	for i, msg := range received {
		want := fmt.Sprintf("%s_%d", prefix, i)

		if msg != want {
			log.Fatalf("testServerStream: received[%d] = %q, want %q", i, msg, want)
		}
	}

	log.Printf("testServerStream: OK (received %d messages)", count)
}

// testBidi sends 3 pings and expects each one back uppercased.
func testBidi(ctx context.Context, cl *streamingtest.StreamingTestClient) {
	stream, sink, err := cl.TestBidi(ctx, "go-client")
	if err != nil {
		log.Fatalf("testBidi: TestBidi failed: %v", err)
	}

	pings := []string{"hello", "world", "foo"}

	for _, ping := range pings {
		if err := sink.Send(ctx, ping); err != nil {
			log.Fatalf("testBidi: sink.Send(%q) failed: %v", ping, err)
		}

		reply, recvErr := stream.Receive(ctx)
		if recvErr != nil {
			log.Fatalf("testBidi: stream.Receive failed: %v", recvErr)
		}

		want := strings.ToUpper(ping)

		if reply != want {
			log.Fatalf("testBidi: reply for %q = %q, want %q", ping, reply, want)
		}
	}

	if err := sink.Close(); err != nil {
		log.Fatalf("testBidi: sink.Close failed: %v", err)
	}

	if err := stream.Close(); err != nil {
		log.Fatalf("testBidi: stream.Close failed: %v", err)
	}

	log.Printf("testBidi: OK (%d ping/pong exchanges)", len(pings))
}
