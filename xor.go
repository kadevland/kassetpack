package kassetpack

import "io"

// XORReader is a decorator that decrypts data on the fly using XOR.
type XORReader struct {
	reader io.Reader
	key    []byte
	pos    int
}

// NewXORReader wraps an existing reader.
func NewXORReader(reader io.Reader, key []byte) *XORReader {
	return &XORReader{
		reader: reader,
		key:    key,
	}
}

// Read implements the io.Reader interface.
func (r *XORReader) Read(buffer []byte) (int, error) {
	// Read encrypted bytes from the underlying reader
	n, err := r.reader.Read(buffer)

	// Decrypt the bytes that were just read
	for i := 0; i < n; i++ {
		buffer[i] ^= r.key[(r.pos+i)%len(r.key)]
	}

	// Update our position in the key
	r.pos += n

	return n, err
}
