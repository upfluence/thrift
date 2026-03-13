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
	"compress/zlib"
	"io"
	"log"
)

// TZlibTransportFactory is a factory for TZlibTransport instances
type TZlibTransportFactory struct {
	level   int
	factory TTransportFactory
}

// TZlibTransport is a TTransport implementation that makes use of zlib compression.
type TZlibTransport struct {
	reader    io.ReadCloser
	transport TTransport
	writer    *zlib.Writer
}

// GetTransport constructs a new instance of NewTZlibTransport
func (p *TZlibTransportFactory) GetTransport(trans TTransport) TTransport {
	if p.factory != nil {
		// wrap other factory
		trans = p.factory.GetTransport(trans)
	}
	transport, err := NewTZlibTransport(trans, p.level)
	if err != nil {
		return nil
	}
	return transport
}

// NewTZlibTransportFactory constructs a new instance of NewTZlibTransportFactory
func NewTZlibTransportFactory(level int) *TZlibTransportFactory {
	return &TZlibTransportFactory{level: level, factory: nil}
}

// NewTZlibTransportFactory constructs a new instance of TZlibTransportFactory
// as a wrapper over existing transport factory
func NewTZlibTransportFactoryWithFactory(level int, factory TTransportFactory) *TZlibTransportFactory {
	return &TZlibTransportFactory{level: level, factory: factory}
}

// NewTZlibTransport constructs a new instance of TZlibTransport
func NewTZlibTransport(trans TTransport, level int) (*TZlibTransport, error) {
	w, err := zlib.NewWriterLevel(trans, level)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &TZlibTransport{
		writer:    w,
		transport: trans,
	}, nil
}

// Close closes the reader and writer (flushing any unwritten data) and closes
// the underlying transport.
func (z *TZlibTransport) Close() error {
	if z.reader != nil {
		if err := z.reader.Close(); err != nil {
			return err
		}
	}
	if err := z.writer.Close(); err != nil {
		return err
	}
	return z.transport.Close()
}

// Flush flushes the writer and its underlying transport.
func (z *TZlibTransport) Flush() error {
	if err := z.writer.Flush(); err != nil {
		return err
	}
	return z.transport.Flush()
}

// IsOpen returns true if the transport is open
func (z *TZlibTransport) IsOpen() bool {
	return z.transport.IsOpen()
}

// Open opens the transport for communication
func (z *TZlibTransport) Open() error {
	return z.transport.Open()
}

func (z *TZlibTransport) Read(p []byte) (int, error) {
	if z.reader == nil {
		r, err := zlib.NewReader(z.transport)
		if err != nil {
			return 0, NewTTransportExceptionFromError(err)
		}
		z.reader = r
	}

	return z.reader.Read(p)
}

func (z *TZlibTransport) Write(p []byte) (int, error) {
	return z.writer.Write(p)
}

func (z *TZlibTransport) WriteContext(ctx Context) error {
	return z.transport.WriteContext(ctx)
}
