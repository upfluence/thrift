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
	"fmt"
	"net"
	"testing"
	"time"

	thrift "github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/test/gen/onewaytest"
)

const TIMEOUT = time.Second

func FindAvailableTCPServerPort() net.Addr {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("Could not find available server port")
	}

	defer l.Close()

	return l.Addr()
}

type oneWayImpl struct{}

func (i *oneWayImpl) Hi(ctx thrift.Context, in int64, s string) (err error) {
	fmt.Println("Hi!")
	return
}

func (i *oneWayImpl) Emptyfunc(ctx thrift.Context) (err error) { return }

func (i *oneWayImpl) EchoInt(ctx thrift.Context, param int64) (r int64, err error) {
	return param, nil
}

func TestOneway(t *testing.T) {
	addr := FindAvailableTCPServerPort()

	serverTransport, err := thrift.NewTServerSocketTimeout(addr.String(), TIMEOUT)
	if err != nil {
		t.Fatal("Unable to create server socket", err)
	}

	processor := onewaytest.NewOneWayProcessor(&oneWayImpl{}, nil)
	server := thrift.NewTSimpleServer2(processor, serverTransport)

	go server.Serve()

	time.Sleep(10 * time.Millisecond)

	transport := thrift.NewTSocketFromAddrTimeout(addr, TIMEOUT)
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	client := onewaytest.NewOneWayClient(thrift.NewTSyncClient(transport, thrift.NewTBinaryProtocolFactoryDefault()))

	err = transport.Open()
	if err != nil {
		t.Fatal("Unable to open client socket", err)
	}

	defer transport.Close()

	_ = protocol

	err = client.Hi(defaultCtx, 1, "")
	if err != nil {
		t.Fatal("Unexpected error: ", err)
	}

	i, err := client.EchoInt(defaultCtx, 42)
	if err != nil {
		t.Fatal("Unexpected error: ", err)
	}

	if i != 42 {
		t.Fatal("Unexpected returned value: ", i)
	}

	server.Stop()
}
