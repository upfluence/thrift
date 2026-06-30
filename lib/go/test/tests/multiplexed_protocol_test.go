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

package tests

import (
	"net"
	"testing"
	"time"

	thrift "github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/test/gen/multiplexedprotocoltest"
)

// multiplexedProcessor adapts TMultiplexedProcessor to the local TProcessor interface.
type multiplexedProcessor struct {
	*thrift.TMultiplexedProcessor
}

func (p *multiplexedProcessor) GetMiddlewares() []thrift.TMiddleware { return nil }

func (p *multiplexedProcessor) AddProcessor(_ string, _ thrift.TProcessorFunction) {}

func (p *multiplexedProcessor) Process(ctx thrift.Context, in, out thrift.TProtocol) (bool, thrift.TException) {
	return p.TMultiplexedProcessor.Process(ctx, in, out)
}

type firstImpl struct{}

func (f *firstImpl) ReturnOne(ctx thrift.Context) (r int64, err error) {
	return 1, nil
}

type secondImpl struct{}

func (s *secondImpl) ReturnTwo(ctx thrift.Context) (r int64, err error) {
	return 2, nil
}

func createTransport(addr net.Addr) (thrift.TTransport, error) {
	socket := thrift.NewTSocketFromAddrTimeout(addr, TIMEOUT)
	transport := thrift.NewTFramedTransport(socket)

	err := transport.Open()
	if err != nil {
		return nil, err
	}

	return transport, nil
}

// fixedProtocolFactory returns the same pre-built protocol on every GetProtocol call.
// Used to pass a TMultiplexedProtocol to NewTSyncClient without re-wrapping the transport.
type fixedProtocolFactory struct {
	protocol thrift.TProtocol
}

func (f *fixedProtocolFactory) GetProtocol(_ thrift.TTransport) thrift.TProtocol {
	return f.protocol
}

func newMultiplexedServer(t *testing.T, addr net.Addr) (*thrift.TSimpleServer, *thrift.TMultiplexedProcessor) {
	t.Helper()

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())

	serverTransport, err := thrift.NewTServerSocketTimeout(addr.String(), TIMEOUT)
	if err != nil {
		t.Fatal("Unable to create server socket", err)
	}

	mp := thrift.NewTMultiplexedProcessor()
	mp.RegisterProcessor("FirstService", multiplexedprotocoltest.NewFirstProcessor(&firstImpl{}, nil))
	mp.RegisterProcessor("SecondService", multiplexedprotocoltest.NewSecondProcessor(&secondImpl{}, nil))

	server := thrift.NewTSimpleServer4(&multiplexedProcessor{mp}, serverTransport, transportFactory, protocolFactory)

	return server, mp
}

func TestMultiplexedProtocolFirst(t *testing.T) {
	addr := FindAvailableTCPServerPort()
	server, _ := newMultiplexedServer(t, addr)

	defer server.Stop()

	go server.Serve()

	time.Sleep(10 * time.Millisecond)

	transport, err := createTransport(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer transport.Close()

	protocol := thrift.NewTMultiplexedProtocol(thrift.NewTBinaryProtocolTransport(transport), "FirstService")
	client := multiplexedprotocoltest.NewFirstClient(thrift.NewTSyncClient(transport, &fixedProtocolFactory{protocol}))

	ret, err := client.ReturnOne(defaultCtx)
	if err != nil {
		t.Fatal("Unable to call first server:", err)
	}

	if ret != 1 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}

func TestMultiplexedProtocolSecond(t *testing.T) {
	addr := FindAvailableTCPServerPort()
	server, _ := newMultiplexedServer(t, addr)

	defer server.Stop()

	go server.Serve()

	time.Sleep(10 * time.Millisecond)

	transport, err := createTransport(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer transport.Close()

	protocol := thrift.NewTMultiplexedProtocol(thrift.NewTBinaryProtocolTransport(transport), "SecondService")
	client := multiplexedprotocoltest.NewSecondClient(thrift.NewTSyncClient(transport, &fixedProtocolFactory{protocol}))

	ret, err := client.ReturnTwo(defaultCtx)
	if err != nil {
		t.Fatal("Unable to call second server:", err)
	}

	if ret != 2 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}

func TestMultiplexedProtocolLegacy(t *testing.T) {
	addr := FindAvailableTCPServerPort()
	server, mp := newMultiplexedServer(t, addr)

	defer server.Stop()

	go server.Serve()

	time.Sleep(10 * time.Millisecond)

	transport, err := createTransport(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer transport.Close()

	// expect error since default processor is not registered
	protocol := thrift.NewTBinaryProtocolTransport(transport)
	client := multiplexedprotocoltest.NewSecondClient(thrift.NewTSyncClient(transport, &fixedProtocolFactory{protocol}))

	_, err = client.ReturnTwo(defaultCtx)
	if err == nil {
		t.Fatal("Expecting error")
	}

	// register default processor and call again
	mp.RegisterDefault(multiplexedprotocoltest.NewSecondProcessor(&secondImpl{}, nil))

	transport2, err := createTransport(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer transport2.Close()

	protocol2 := thrift.NewTBinaryProtocolTransport(transport2)
	client2 := multiplexedprotocoltest.NewSecondClient(thrift.NewTSyncClient(transport2, &fixedProtocolFactory{protocol2}))

	ret, err := client2.ReturnTwo(defaultCtx)
	if err != nil {
		t.Fatal("Unable to call legacy server:", err)
	}

	if ret != 2 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}
