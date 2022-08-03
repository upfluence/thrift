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
	"fmt"

	"github.com/upfluence/errors"
)

// A processor is a generic object which operates upon an input stream and
// writes to some output stream.
type TProcessor interface {
	Process(ctx Context, in, out TProtocol) (bool, TException)
	GetMiddlewares() []TMiddleware
	AddProcessor(string, TProcessorFunction)
}

type TStandardProcessor struct {
	ProcessorMap map[string]TProcessorFunction
	Middlewares  []TMiddleware
}

func NewTStandardProcessor(ms []TMiddleware) *TStandardProcessor {
	return &TStandardProcessor{
		Middlewares:  ms,
		ProcessorMap: make(map[string]TProcessorFunction),
	}
}

func (p *TStandardProcessor) GetMiddlewares() []TMiddleware {
	return p.Middlewares
}

func (p *TStandardProcessor) AddProcessor(fname string, fn TProcessorFunction) {
	p.ProcessorMap[fname] = fn
}

func (p *TStandardProcessor) Process(ctx Context, in, out TProtocol) (bool, TException) {
	name, _, seqID, err := in.ReadMessageBegin()

	if err != nil {
		return false, err
	}

	if processor, ok := p.ProcessorMap[name]; ok {
		return processor.Process(ctx, seqID, in, out)
	}

	in.Skip(STRUCT)
	in.ReadMessageEnd()
	x5 := NewTApplicationException(UNKNOWN_METHOD, "Unknown function "+name)
	out.WriteMessageBegin(name, EXCEPTION, seqID)
	x5.Write(out)
	out.WriteMessageEnd()
	out.Flush()
	return false, x5
}

type TProcessorFunction interface {
	Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException)
}

type TBaseProcessorFunction struct {
	fname      string
	argBuilder func() TRequest
	middleware TStreamingMiddleware
}

func NewTBaseProcessorFunction(p TProcessor, fname string, builder func() TRequest) *TBaseProcessorFunction {
	return &TBaseProcessorFunction{
		fname:      fname,
		argBuilder: builder,
		middleware: WrapMiddlewares(p.GetMiddlewares()),
	}
}

func (p *TBaseProcessorFunction) readRequest(in TProtocol) (TRequest, error) {
	args := p.argBuilder()

	if err := args.Read(in); err != nil {
		in.ReadMessageEnd()
		return nil, err
	}

	in.ReadMessageEnd()

	return args, nil
}

func (p *TBaseProcessorFunction) writeResponse(out TProtocol, seqID int32, res TResponse, err error) (bool, error) {
	if err != nil {
		tid := INTERNAL_ERROR

		if errors.IsTimeout(err) {
			tid = INTERNAL_TIME_OUT_ERROR
		}

		rerr := p.writeException(
			out,
			seqID,
			int32(tid),
			fmt.Sprintf("Internal error processing : %s: %s", p.fname, err.Error()),
		)

		return rerr == nil, err
	}

	return true, p.writeReply(out, seqID, res)
}

type protocolWriter interface {
	Write(TProtocol) error
}

func (p *TBaseProcessorFunction) write(out TProtocol, seqID int32, mType TMessageType, x protocolWriter) error {
	err := out.WriteMessageBegin(p.fname, mType, seqID)

	if err2 := x.Write(out); err == nil && err2 != nil {
		err = err2
	}

	if err2 := out.WriteMessageEnd(); err == nil && err2 != nil {
		err = err2
	}

	if err2 := out.Flush(); err == nil && err2 != nil {
		err = err2
	}

	return err
}

func (p *TBaseProcessorFunction) writeException(out TProtocol, seqID, tID int32, msg string) error {
	return p.write(out, seqID, EXCEPTION, NewTApplicationException(tID, msg))
}

func (p *TBaseProcessorFunction) writeReply(out TProtocol, seqID int32, resp TResponse) error {
	return p.write(out, seqID, REPLY, resp)
}

type TBinaryHandler interface {
	Handle(Context, TRequest) (TResponse, error)
}

type TBinaryProcessorFunction struct {
	*TBaseProcessorFunction
	handler TBinaryHandler
}

func NewTBinaryProcessorFunction(p TProcessor, fname string, builder func() TRequest, handler TBinaryHandler) *TBinaryProcessorFunction {
	return &TBinaryProcessorFunction{
		TBaseProcessorFunction: NewTBaseProcessorFunction(p, fname, builder),
		handler:                handler,
	}
}

func (p *TBinaryProcessorFunction) Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException) {
	var args, err = p.readRequest(in)

	if err != nil {
		p.writeException(out, seqID, PROTOCOL_ERROR, err.Error())
		return false, err
	}

	res, err := p.middleware.HandleBinaryRequest(
		ctx,
		p.fname,
		seqID,
		args,
		func(ctx Context, req TRequest) (TResponse, error) {
			return p.handler.Handle(ctx, req)
		},
	)

	return p.writeResponse(out, seqID, res, err)
}

type TUnaryHandler interface {
	Handle(Context, TRequest) error
}

type TUnaryProcessorFunction struct {
	*TBaseProcessorFunction
	handler TUnaryHandler
}

func NewTUnaryProcessorFunction(p TProcessor, fname string, builder func() TRequest, handler TUnaryHandler) *TUnaryProcessorFunction {
	return &TUnaryProcessorFunction{
		TBaseProcessorFunction: NewTBaseProcessorFunction(p, fname, builder),
		handler:                handler,
	}
}

func (p *TUnaryProcessorFunction) Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException) {
	var args, err = p.readRequest(in)

	if err != nil {
		return false, err
	}

	return true, p.middleware.HandleUnaryRequest(
		ctx,
		p.fname,
		seqID,
		args,
		func(ctx Context, req TRequest) error {
			return p.handler.Handle(ctx, req)
		},
	)
}

type TStreamServerHandler interface {
	Handle(Context, TRequest, TOutboundStream) (TResponse, error)
}

type TStreamServerProcessorFunction struct {
	*TBaseProcessorFunction
	handler TStreamServerHandler
}

func NewTStreamServerProcessorFunction(p TProcessor, fname string, builder func() TRequest, handler TStreamServerHandler) *TStreamServerProcessorFunction {
	return &TStreamServerProcessorFunction{
		TBaseProcessorFunction: NewTBaseProcessorFunction(p, fname, builder),
		handler:                handler,
	}
}

func (p *TStreamServerProcessorFunction) Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException) {
	var args, err = p.readRequest(in)

	if err != nil {
		return false, err
	}

	stream := newTServerOutboundStream(p.fname, seqID, in, out)

	res, err := p.middleware.HandleOutboundStream(
		ctx,
		p.fname,
		seqID,
		args,
		stream,
		func(ctx Context, req TRequest, s TOutboundStream) (TResponse, error) {
			return p.handler.Handle(ctx, req, s)
		},
	)

	ok, err := p.writeResponse(out, seqID, res, err)

	stream.ready()

	defer stream.Close()

	if !ok || err != nil {
		return ok, err
	}

	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-stream.closec:
		return true, nil
	}
}

type TStreamClientHandler interface {
	Handle(Context, TRequest, TInboundStream) (TResponse, error)
}

type TStreamClientProcessorFunction struct {
	*TBaseProcessorFunction
	handler TStreamClientHandler
}

func NewTStreamClientProcessorFunction(p TProcessor, fname string, builder func() TRequest, handler TStreamClientHandler) *TStreamClientProcessorFunction {
	return &TStreamClientProcessorFunction{
		TBaseProcessorFunction: NewTBaseProcessorFunction(p, fname, builder),
		handler:                handler,
	}
}

func (p *TStreamClientProcessorFunction) Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException) {
	var args, err = p.readRequest(in)

	if err != nil {
		return false, err
	}

	stream := newTServerInboundStream(p.fname, seqID, in, out)

	res, err := p.middleware.HandleInboundStream(
		ctx,
		p.fname,
		seqID,
		args,
		stream,
		func(ctx Context, req TRequest, s TInboundStream) (TResponse, error) {
			return p.handler.Handle(ctx, req, s)
		},
	)
	ok, err := p.writeResponse(out, seqID, res, err)

	stream.ready()

	defer stream.Close()

	if !ok || err != nil {
		return ok, err
	}

	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-stream.closec:
		return true, nil
	}
}

type TStreamBidiHandler interface {
	Handle(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)
}

type TStreamBidiProcessorFunction struct {
	*TBaseProcessorFunction
	handler TStreamBidiHandler
}

func NewTStreamBidiProcessorFunction(p TProcessor, fname string, builder func() TRequest, handler TStreamBidiHandler) *TStreamBidiProcessorFunction {
	return &TStreamBidiProcessorFunction{
		TBaseProcessorFunction: NewTBaseProcessorFunction(p, fname, builder),
		handler:                handler,
	}
}

func (p *TStreamBidiProcessorFunction) Process(ctx Context, seqID int32, in, out TProtocol) (bool, TException) {
	var args, err = p.readRequest(in)

	if err != nil {
		return false, err
	}

	bidiStream := newTServerBidiStream(p.fname, seqID, in, out)

	res, err := p.middleware.HandleBidiStream(
		ctx,
		p.fname,
		seqID,
		args,
		&tInboundBidiStream{tBidiStream: bidiStream},
		&tOutboundBidiStream{tBidiStream: bidiStream},
		func(ctx Context, req TRequest, is TInboundStream, os TOutboundStream) (TResponse, error) {
			return p.handler.Handle(ctx, req, is, os)
		},
	)
	ok, err := p.writeResponse(out, seqID, res, err)

	bidiStream.ready()

	defer bidiStream.Close()

	if !ok || err != nil {
		return ok, err
	}

	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-bidiStream.closec:
		return true, nil
	}
}
