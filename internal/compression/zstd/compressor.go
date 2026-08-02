package zstd

import (
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/wal-g/wal-g/internal/ioextensions"
)

const (
	AlgorithmName = "zstd"
	FileExtension = "zst"
)

var encoderPools sync.Map // map[zstd.EncoderLevel]*sync.Pool

func getEncoderPool(level zstd.EncoderLevel) *sync.Pool {
	if pool, ok := encoderPools.Load(level); ok {
		return pool.(*sync.Pool)
	}
	pool := &sync.Pool{
		New: func() any {
			zw, err := zstd.NewWriter(io.Discard, zstd.WithEncoderLevel(level))
			if err != nil {
				panic(err)
			}
			return zw
		},
	}
	actual, _ := encoderPools.LoadOrStore(level, pool)
	return actual.(*sync.Pool)
}

type pooledEncoder struct {
	*zstd.Encoder
	pool   *sync.Pool
	closed bool
}

func (p *pooledEncoder) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	err := p.Encoder.Close()
	p.Reset(nil)
	p.pool.Put(p.Encoder)
	return err
}

// Compressor writes zstd-compressed streams. A zero Level keeps the historical
// default (zstd.SpeedDefault), so an unconfigured Compressor behaves as before.
type Compressor struct {
	Level zstd.EncoderLevel
}

func (compressor Compressor) NewWriter(writer io.Writer) ioextensions.WriteFlushCloser {
	level := compressor.Level
	if level == 0 { // level not set: preserve the previous default
		level = zstd.SpeedDefault
	}

	pool := getEncoderPool(level)
	enc := pool.Get().(*zstd.Encoder)
	enc.Reset(writer)

	return &pooledEncoder{
		Encoder: enc,
		pool:    pool,
	}
}

func (compressor Compressor) FileExtension() string {
	return FileExtension
}

// EncoderLevelFromName resolves a WALG_ZSTD_LEVEL value ("fastest", "default",
// "better", "best") to a zstd encoder level. The match ignores case; the
// returned bool is false when the name is not recognized.
func EncoderLevelFromName(name string) (zstd.EncoderLevel, bool) {
	ok, level := zstd.EncoderLevelFromString(name)
	return level, ok
}
