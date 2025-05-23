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

package thrift

import (
	"errors"
	"io"
)

var errTransportInterrupted = errors.New("Transport Interrupted")

type Flusher interface {
	Flush() (err error)
}

// Encapsulates the I/O layer
type TTransport interface {
	io.ReadWriteCloser
	Flusher

	// Opens the transport for communication
	Open() error

	// Returns true if the transport is open
	IsOpen() bool

	WriteContext(ctx Context) error
}

// This is "enchanced" transport with extra capabilities. You need to use one of these
// to construct protocol.
// Notably, TSocket does not implement this interface, and it is always a mistake to use
// TSocket directly in protocol.
type TRichTransport interface {
	io.ReadWriter
	io.ByteReader
	io.ByteWriter
	io.StringWriter
	Flusher
}
