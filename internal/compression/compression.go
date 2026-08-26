package compression

import (
	"io"

	"github.com/wal-g/wal-g/internal/ioextensions"
)

type Compressor interface {
	NewWriter(writer io.Writer) ioextensions.WriteFlushCloser
	FileExtension() string
}

// LevelConfigurators resolves a compression method name to a function that
// builds a Compressor from a level string (e.g. WALG_BROTLI_LEVEL,
// WALG_ZSTD_LEVEL). Algorithms compiled behind a build tag (e.g. brotli)
// register themselves here from their _enabled.go init(), so callers that are
// always built (like configure.go) don't need to import build-tagged packages
// directly.
var LevelConfigurators = map[string]func(level string) (Compressor, error){}

type Decompressor interface {
	Decompress(src io.Reader) (io.ReadCloser, error)
	FileExtension() string
}

func GetDecompressorByCompressor(compressor Compressor) Decompressor {
	return FindDecompressor(compressor.FileExtension())
}

func FindDecompressor(fileExtension string) Decompressor {
	// cut the leading '.' (e.g. ".lz4" => "lz4")
	if len(fileExtension) > 0 && fileExtension[0] == '.' {
		fileExtension = fileExtension[1:]
	}

	for _, decompressor := range Decompressors {
		if decompressor.FileExtension() == fileExtension {
			return decompressor
		}
	}
	return nil
}
