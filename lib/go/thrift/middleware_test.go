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

type mockResponse struct {
	TResponse
}

func TestTMultiMiddlewareBinary(t *testing.T) {
	var buf bytes.Buffer

	_, err := TMultiMiddleware(
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

	err := TMultiMiddleware(
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
