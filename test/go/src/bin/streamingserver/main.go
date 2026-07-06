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
	"crypto/tls"
	"flag"
	"fmt"
	"log"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/test/go/gen/streamingtest"
	"github.com/upfluence/thrift/test/go/src/common"
)

var host = flag.String("host", "localhost", "Host to listen on")
var port = flag.Int64("port", 9090, "Port number to listen on")
var domainSocket = flag.String("domain-socket", "", "Unix domain socket path")
var transport = flag.String("transport", "buffered", "Transport: buffered, framed, zlib")
var protocol = flag.String("protocol", "binary", "Protocol: binary, compact, json, header")
var ssl = flag.Bool("ssl", false, "Use SSL")
var certPath = flag.String("certPath", "keys", "Directory containing SSL certificates")

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
		serverTransport thrift.TServerTransport
		err             error
	)

	hostPort := fmt.Sprintf("%s:%d", *host, *port)

	if *ssl {
		cfg := new(tls.Config)

		cert, loadErr := tls.LoadX509KeyPair(*certPath+"/server.crt", *certPath+"/server.key")
		if loadErr != nil {
			log.Fatalf("Failed to load SSL certs: %v", loadErr)
		}

		cfg.Certificates = append(cfg.Certificates, cert)
		serverTransport, err = thrift.NewTSSLServerSocket(hostPort, cfg)
	} else if *domainSocket != "" {
		serverTransport, err = thrift.NewTServerSocket(*domainSocket)
	} else {
		serverTransport, err = thrift.NewTServerSocket(hostPort)
	}

	if err != nil {
		log.Fatalf("Failed to create server transport: %v", err)
	}

	var transportFactory thrift.TTransportFactory

	switch *transport {
	case "framed":
		transportFactory = thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
	case "zlib":
		transportFactory, err = thrift.NewTZlibTransportFactory(zlib.BestCompression), nil
	case "buffered":
		fallthrough
	default:
		transportFactory = thrift.NewTBufferedTransportFactory(8192)
	}

	processor := streamingtest.NewStreamingTestProcessor(&common.SimpleStreamingHandler{}, nil)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)

	if listenErr := server.Listen(); listenErr != nil {
		log.Fatalf("Failed to listen: %v", listenErr)
	}

	log.Printf("Starting streaming server on %s (transport=%s protocol=%s ssl=%v)",
		hostPort, *transport, *protocol, *ssl)

	server.Serve()
}
