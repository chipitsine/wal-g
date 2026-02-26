//go:build brotli
// +build brotli

package brotli

import (
	"io"

	"github.com/andybalholm/brotli"
	"github.com/wal-g/wal-g/internal/compression/computils"
)

type Decompressor struct{}

func (decompressor Decompressor) Decompress(src io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(brotli.NewReader(computils.NewUntilEOFReader(src))), nil
}

func (decompressor Decompressor) FileExtension() string {
	return FileExtension
}
