package ioextensions

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type testWriteCloser struct {
	io.Writer
	closed bool
}

func (w *testWriteCloser) Close() error {
	w.closed = true
	return nil
}

func TestBufferedWriteCloserFlushesBeforeClose(t *testing.T) {
	var output bytes.Buffer
	underlying := &testWriteCloser{Writer: &output}
	writer := NewBufferedWriteCloser(underlying, 16)

	_, err := writer.Write([]byte("buffered data"))
	require.NoError(t, err)
	require.Empty(t, output.String())

	require.NoError(t, writer.Close())
	require.Equal(t, "buffered data", output.String())
	require.True(t, underlying.closed)
}

func TestBufferedWriteCloserDoesNotCloseAfterFlushFailure(t *testing.T) {
	underlying := &testWriteCloser{Writer: errorWriter{}}
	writer := NewBufferedWriteCloser(underlying, 16)

	_, err := writer.Write([]byte("buffered data"))
	require.NoError(t, err)

	err = writer.Close()
	require.Error(t, err)
	require.True(t, errors.Is(err, errWrite))
	require.False(t, underlying.closed)
}

var errWrite = errors.New("write error")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errWrite }
