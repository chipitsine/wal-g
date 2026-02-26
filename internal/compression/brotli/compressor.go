//go:build brotli
// +build brotli

package brotli

import (
	"io"

	"github.com/andybalholm/brotli"
	"github.com/wal-g/wal-g/internal/ioextensions"
)

const (
	AlgorithmName = "brotli"
	FileExtension = "br"
)

type Compressor struct{}

func (compressor Compressor) NewWriter(writer io.Writer) ioextensions.WriteFlushCloser {
	return brotli.NewWriterLevel(writer, 3)
}

func (compressor Compressor) FileExtension() string {
	return FileExtension
}
