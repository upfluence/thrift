package thrift

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type middleware struct {
	v int
	w io.Writer
}

func (m *middleware) HandleBinaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	fmt.Fprintf(m.w, "[b %d in]", m.v)
	resp, err := next(ctx, req)
	fmt.Fprintf(m.w, "[b %d out]", m.v)

	return resp, err
}

func (m *middleware) HandleUnaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) error) error {
	fmt.Fprintf(m.w, "[u %d in]", m.v)
	err := next(ctx, req)
	fmt.Fprintf(m.w, "[u %d out]", m.v)

	return err
}

// streamingMiddleware extends middleware with all three streaming hooks.
type streamingMiddleware struct {
	v int
	w io.Writer
}

func (m *streamingMiddleware) HandleBinaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	return next(ctx, req)
}

func (m *streamingMiddleware) HandleUnaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) error) error {
	return next(ctx, req)
}

func (m *streamingMiddleware) HandleInboundStream(ctx Context, mth string, seqID int32, req TRequest, s TInboundStream, next func(Context, TRequest, TInboundStream) (TResponse, error)) (TResponse, error) {
	fmt.Fprintf(m.w, "[i %d in]", m.v)
	resp, err := next(ctx, req, s)
	fmt.Fprintf(m.w, "[i %d out]", m.v)

	return resp, err
}

func (m *streamingMiddleware) HandleOutboundStream(ctx Context, mth string, seqID int32, req TRequest, s TOutboundStream, next func(Context, TRequest, TOutboundStream) (TResponse, error)) (TResponse, error) {
	fmt.Fprintf(m.w, "[o %d in]", m.v)
	resp, err := next(ctx, req, s)
	fmt.Fprintf(m.w, "[o %d out]", m.v)

	return resp, err
}

func (m *streamingMiddleware) HandleBidiStream(ctx Context, mth string, seqID int32, req TRequest, is TInboundStream, os TOutboundStream, next func(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)) (TResponse, error) {
	fmt.Fprintf(m.w, "[d %d in]", m.v)
	resp, err := next(ctx, req, is, os)
	fmt.Fprintf(m.w, "[d %d out]", m.v)

	return resp, err
}

type mockResponse struct {
	TResponse
}

func TestTMultiMiddlewareBinary(t *testing.T) {
	var buf bytes.Buffer

	_, err := WrapMiddlewares(
		[]TMiddleware{
			&middleware{v: 1, w: &buf},
			&middleware{v: 2, w: &buf},
			&middleware{v: 3, w: &buf},
		},
	).HandleBinaryRequest(
		context.Background(),
		"",
		0,
		mockResponse{},
		func(ctx Context, req TRequest) (TResponse, error) {
			return req.(TResponse), nil
		},
	)

	assert.Nil(t, err)
	assert.Equal(
		t,
		"[b 1 in][b 2 in][b 3 in][b 3 out][b 2 out][b 1 out]",
		buf.String(),
	)
}

func TestTMultiMiddlewaryUnary(t *testing.T) {
	var buf bytes.Buffer

	err := WrapMiddlewares(
		[]TMiddleware{
			&middleware{v: 1, w: &buf},
			&middleware{v: 2, w: &buf},
			&middleware{v: 3, w: &buf},
		},
	).HandleUnaryRequest(
		context.Background(),
		"",
		0,
		mockResponse{},
		func(Context, TRequest) error { return nil },
	)

	assert.Nil(t, err)
	assert.Equal(
		t,
		"[u 1 in][u 2 in][u 3 in][u 3 out][u 2 out][u 1 out]",
		buf.String(),
	)
}

func TestTMultiMiddlewareInboundStream(t *testing.T) {
	var buf bytes.Buffer

	_, err := WrapMiddlewares(
		[]TMiddleware{
			&streamingMiddleware{v: 1, w: &buf},
			&streamingMiddleware{v: 2, w: &buf},
			&streamingMiddleware{v: 3, w: &buf},
		},
	).HandleInboundStream(
		context.Background(),
		"",
		0,
		mockResponse{},
		nil,
		func(ctx Context, req TRequest, s TInboundStream) (TResponse, error) {
			return req.(TResponse), nil
		},
	)

	assert.Nil(t, err)
	assert.Equal(
		t,
		"[i 1 in][i 2 in][i 3 in][i 3 out][i 2 out][i 1 out]",
		buf.String(),
	)
}

func TestTMultiMiddlewareOutboundStream(t *testing.T) {
	var buf bytes.Buffer

	_, err := WrapMiddlewares(
		[]TMiddleware{
			&streamingMiddleware{v: 1, w: &buf},
			&streamingMiddleware{v: 2, w: &buf},
			&streamingMiddleware{v: 3, w: &buf},
		},
	).HandleOutboundStream(
		context.Background(),
		"",
		0,
		mockResponse{},
		nil,
		func(ctx Context, req TRequest, s TOutboundStream) (TResponse, error) {
			return req.(TResponse), nil
		},
	)

	assert.Nil(t, err)
	assert.Equal(
		t,
		"[o 1 in][o 2 in][o 3 in][o 3 out][o 2 out][o 1 out]",
		buf.String(),
	)
}

func TestTMultiMiddlewareBidiStream(t *testing.T) {
	var buf bytes.Buffer

	_, err := WrapMiddlewares(
		[]TMiddleware{
			&streamingMiddleware{v: 1, w: &buf},
			&streamingMiddleware{v: 2, w: &buf},
			&streamingMiddleware{v: 3, w: &buf},
		},
	).HandleBidiStream(
		context.Background(),
		"",
		0,
		mockResponse{},
		nil,
		nil,
		func(ctx Context, req TRequest, is TInboundStream, os TOutboundStream) (TResponse, error) {
			return req.(TResponse), nil
		},
	)

	assert.Nil(t, err)
	assert.Equal(
		t,
		"[d 1 in][d 2 in][d 3 in][d 3 out][d 2 out][d 1 out]",
		buf.String(),
	)
}
