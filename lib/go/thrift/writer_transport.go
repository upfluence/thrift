package thrift

import (
	"errors"
	"io"
)

var ErrWriteOnly = errors.New("Transport is write only")

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

type nopFlusher struct{}

func (nopFlusher) Flush() error { return nil }

type WriterTransport struct {
	io.Writer
	io.Closer
	Flusher
}

func (*WriterTransport) WriteContext(Context) error { return nil }
func (*WriterTransport) Open() error                { return nil }
func (*WriterTransport) IsOpen() bool               { return true }
func (*WriterTransport) Read([]byte) (int, error)   { return 0, ErrWriteOnly }

func WrapWriter(w io.Writer) *WriterTransport {
	c, ok := w.(io.Closer)

	if !ok {
		c = nopCloser{}
	}

	f, ok := w.(Flusher)

	if !ok {
		f = nopFlusher{}
	}

	return &WriterTransport{Writer: w, Closer: c, Flusher: f}
}
