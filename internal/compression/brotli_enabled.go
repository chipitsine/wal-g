//go:build brotli && !windows && !(darwin && arm64)
// +build brotli
// +build !windows
// +build !darwin !arm64

package compression

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/wal-g/wal-g/internal/compression/brotli"
)

func init() {
	Decompressors = append(Decompressors, brotli.Decompressor{})
	Compressors[brotli.AlgorithmName] = brotli.Compressor{}
	CompressingAlgorithms = append(CompressingAlgorithms, brotli.AlgorithmName)
	LevelConfigurators[brotli.AlgorithmName] = func(levelName string) (Compressor, error) {
		level, err := strconv.Atoi(levelName)
		if err != nil || level < 0 || level > 11 {
			return nil, errors.Errorf("Unknown brotli level: '%s', must be an integer between 0 and 11", levelName)
		}
		return brotli.Compressor{Level: level}, nil
	}
}
