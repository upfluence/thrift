package thrift

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type streamBidiHandler struct {
	wg *sync.WaitGroup

	req string
}

func (sbh *streamBidiHandler) Handle(_ Context, req TRequest, sink TInboundStream, stream TOutboundStream) (TResponse, error) {
	sbh.req = string(*(req.(*tstring)))

	sbh.wg.Add(1)
	go func() {
		ctx := context.Background()

		defer sbh.wg.Done()
		defer stream.Close()
		defer sink.Close()

		for {
			var v tstring

			if err := sink.Receive(ctx, &v); err == io.EOF {
				return
			}

			v += "pong"

			if err := stream.Send(ctx, &v); err == io.EOF {
				return
			}
		}
	}()

	return newTString("resp"), nil
}

type streamServerHandler struct {
	wg *sync.WaitGroup

	req string
}

func (ssh *streamServerHandler) Handle(_ Context, req TRequest, stream TOutboundStream) (TResponse, error) {
	ssh.req = string(*(req.(*tstring)))

	ssh.wg.Add(1)
	go func() {
		ctx := context.Background()

		stream.Send(ctx, newTString("bar"))
		stream.Send(ctx, newTString("biz"))
		stream.Close()
		ssh.wg.Done()
	}()

	return newTString("resp"), nil
}

type streamClientHandler struct {
	wg *sync.WaitGroup

	req        string
	streamMsgs []string
}

func (sch *streamClientHandler) Handle(_ Context, req TRequest, stream TInboundStream) (TResponse, error) {
	sch.req = string(*(req.(*tstring)))

	sch.wg.Add(1)
	go func() {

		var v tstring

		ctx := context.Background()

		for {
			err := stream.Receive(ctx, &v)

			if err != nil {
				stream.Close()
				sch.wg.Done()
				return
			}

			sch.streamMsgs = append(sch.streamMsgs, string(v))
		}
	}()

	return newTString("resp"), nil
}

type tstring string

func newTString(v string) *tstring {
	s := tstring(v)

	return &s
}

func (s *tstring) GetError() error        { return nil }
func (s *tstring) GetResult() interface{} { return s }

func (s *tstring) String() string { return string(*s) }

func (s *tstring) Write(prot TProtocol) error {
	return prot.WriteString(string(*s))
}

func (s *tstring) Read(prot TProtocol) error {
	res, err := prot.ReadString()

	if err == nil {
		*s = tstring(res)
	}

	return err
}

func TestStreamClient(t *testing.T) {
	var (
		wg sync.WaitGroup

		pr1, pw1 = io.Pipe()
		pr2, pw2 = io.Pipe()

		ctx = context.Background()
		pf  = NewTBinaryProtocolFactoryDefault()
	)

	cl := NewTSyncClient(
		NewStreamTransport(pr1, pw2),
		NewTDebugProtocolFactory(pf, "client "),
	)

	h := streamClientHandler{wg: &wg}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"stream_client",
		NewTStreamClientProcessorFunction(
			p,
			"stream_client",
			func() TRequest {
				var s tstring
				return &s
			},
			&h,
		),
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		pf := NewTDebugProtocolFactory(pf, "server ")
		st := NewStreamTransport(pr2, pw1)

		ok, err := p.Process(ctx, pf.GetProtocol(st), pf.GetProtocol(st))
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	var resp tstring

	ostream, err := cl.StreamClient(ctx, "stream_client", newTString("foo"), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "resp", string(resp))

	err = ostream.Send(ctx, newTString("bar"))
	assert.NoError(t, err)

	err = ostream.Send(ctx, newTString("biz"))
	assert.NoError(t, err)

	err = ostream.Close()
	assert.NoError(t, err)

	assert.Equal(t, "foo", h.req)
	assert.Equal(t, []string{"bar", "biz"}, h.streamMsgs)

	wg.Wait()
}

func TestStreamServer(t *testing.T) {
	var (
		wg sync.WaitGroup

		pr1, pw1 = io.Pipe()
		pr2, pw2 = io.Pipe()

		ctx = context.Background()
		pf  = NewTBinaryProtocolFactoryDefault()
	)

	cl := NewTSyncClient(
		NewStreamTransport(pr1, pw2),
		NewTDebugProtocolFactory(pf, "client "),
	)

	h := streamServerHandler{wg: &wg}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"stream_server",
		NewTStreamServerProcessorFunction(
			p,
			"stream_server",
			func() TRequest {
				var s tstring
				return &s
			},
			&h,
		),
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		pf := NewTDebugProtocolFactory(pf, "server ")
		st := NewStreamTransport(pr2, pw1)

		ok, err := p.Process(ctx, pf.GetProtocol(st), pf.GetProtocol(st))
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	var resp tstring

	istream, err := cl.StreamServer(ctx, "stream_server", newTString("foo"), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "resp", string(resp))

	var (
		streamMsgs []string
		v          tstring
	)

	for {
		var done bool

		switch err := istream.Receive(ctx, &v); err {
		case io.EOF:
			done = true
		case nil:
			streamMsgs = append(streamMsgs, string(v))
		default:
			t.Errorf("Unexpected error: %v", err)
		}

		if done {
			break
		}
	}

	err = istream.Close()
	assert.NoError(t, err)

	assert.Equal(t, "foo", h.req)
	assert.Equal(t, []string{"bar", "biz"}, streamMsgs)

	wg.Wait()
}

func TestStreamBidi(t *testing.T) {
	var (
		wg sync.WaitGroup

		pr1, pw1 = io.Pipe()
		pr2, pw2 = io.Pipe()

		ctx = context.Background()
		pf  = NewTBinaryProtocolFactoryDefault()
	)

	cl := NewTSyncClient(
		NewStreamTransport(pr1, pw2),
		NewTDebugProtocolFactory(pf, "client "),
	)

	h := streamBidiHandler{wg: &wg}
	p := NewTStandardProcessor(nil)

	p.AddProcessor(
		"stream_bidi",
		NewTStreamBidiProcessorFunction(
			p,
			"stream_bidi",
			func() TRequest {
				var s tstring
				return &s
			},
			&h,
		),
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		pf := NewTDebugProtocolFactory(pf, "server ")
		st := NewStreamTransport(pr2, pw1)

		ok, err := p.Process(ctx, pf.GetProtocol(st), pf.GetProtocol(st))
		assert.NoError(t, err)
		assert.True(t, ok)
	}()

	var resp tstring

	istream, ostream, err := cl.StreamBidi(ctx, "stream_bidi", newTString("foo"), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "resp", string(resp))

	var (
		streamMsgs []string
		v          tstring
	)

	for i := 0; i < 5; i++ {
		var done bool
		v += "ping"
		ostream.Send(ctx, &v)

		switch err := istream.Receive(ctx, &v); err {
		case io.EOF:
			done = true
		case nil:
			streamMsgs = append(streamMsgs, string(v))
		default:
			t.Errorf("Unexpected error: %v", err)
		}

		if done {
			break
		}
	}

	err = ostream.Close()
	assert.NoError(t, err)

	err = istream.Close()
	assert.NoError(t, err)

	assert.Equal(t, "foo", h.req)
	assert.Equal(
		t,
		[]string{
			"pingpong",
			strings.Repeat("pingpong", 2),
			strings.Repeat("pingpong", 3),
			strings.Repeat("pingpong", 4),
			strings.Repeat("pingpong", 5),
		}, streamMsgs)

	wg.Wait()
}
