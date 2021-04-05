package thrift

import (
	"errors"
	"io"
)

var ErrReadOnly = errors.New("Transport is read only")

type ReaderTransport struct {
	io.Reader
	io.Closer
}

func (r *ReaderTransport) Write(p []byte) (int, error) { return 0, ErrReadOnly }
func (r *ReaderTransport) Flush() error                { return ErrReadOnly }
func (r *ReaderTransport) Open() error                 { return nil }
func (r *ReaderTransport) IsOpen() bool                { return true }
func (r *ReaderTransport) WriteContext(Context) error  { return ErrReadOnly }

func WrapReader(r io.Reader) *ReaderTransport {
	c, ok := r.(io.Closer)

	if !ok {
		c = nopCloser{}
	}

	return &ReaderTransport{Reader: r, Closer: c}
}
