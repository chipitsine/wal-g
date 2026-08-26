//go:build brotli
// +build brotli

package brotli

import (
	"io"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/wal-g/wal-g/internal/ioextensions"
)

const (
	AlgorithmName = "brotli"
	FileExtension = "br"

	// DefaultLevel preserves the historical hardcoded quality.
	DefaultLevel = 3
)

// encoderPools holds one pool of writers per quality level. Building a brotli
// writer allocates its LZ77 hash table, and wal-g asks for a new writer for
// every tar part and every WAL segment, so without reuse that setup cost is
// paid once per uploaded member. Pools are keyed by level for the same reason
// as the zstd pools: a process can hold both the registry default and a
// configured Compressor.
var encoderPools sync.Map // int (level) -> *sync.Pool

func encoderPool(level int) *sync.Pool {
	if pool, ok := encoderPools.Load(level); ok {
		return pool.(*sync.Pool)
	}
	pool, _ := encoderPools.LoadOrStore(level, &sync.Pool{
		New: func() any {
			return brotli.NewWriterLevel(nil, level)
		},
	})
	return pool.(*sync.Pool)
}

// pooledWriter encodes a single stream and hands its writer back on Close.
// brotli.Writer.Reset keeps the underlying hasher (and its hash table)
// allocated, so returning writers to the pool avoids reallocating that state
// for every tar part.
type pooledWriter struct {
	*brotli.Writer
	pool   *sync.Pool
	once   sync.Once
	closed bool
}

func (writer *pooledWriter) Write(data []byte) (int, error) {
	if writer.closed {
		return 0, io.ErrClosedPipe
	}
	return writer.Writer.Write(data)
}

func (writer *pooledWriter) Flush() error {
	if writer.closed {
		return io.ErrClosedPipe
	}
	return writer.Writer.Flush()
}

func (writer *pooledWriter) Close() error {
	var err error
	writer.once.Do(func() {
		writer.closed = true
		err = writer.Writer.Close()
		// A writer that failed to close is left out of the pool rather than
		// recycled into the next backup member.
		if err == nil {
			writer.pool.Put(writer.Writer)
		}
	})
	return err
}

// Compressor writes brotli-compressed streams. A zero Level keeps the
// historical default quality, so an unconfigured Compressor behaves as before.
type Compressor struct {
	Level int
}

func (compressor Compressor) NewWriter(writer io.Writer) ioextensions.WriteFlushCloser {
	level := compressor.Level
	if level == 0 {
		level = DefaultLevel
	}
	pool := encoderPool(level)
	brotliWriter := pool.Get().(*brotli.Writer)
	brotliWriter.Reset(writer)

	return &pooledWriter{Writer: brotliWriter, pool: pool}
}

func (compressor Compressor) FileExtension() string {
	return FileExtension
}
