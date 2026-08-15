package crypto

import "io"

// OpenPGPPlaintextBufferSize batches compressed data before OpenPGP packet encoding.
const OpenPGPPlaintextBufferSize = 256 << 10

// Crypter is responsible for making cryptographical pipeline parts when needed
type Crypter interface {
	Name() string
	Encrypt(writer io.Writer) (io.WriteCloser, error)
	Decrypt(reader io.Reader) (io.Reader, error)
}
