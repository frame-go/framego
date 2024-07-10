package crypto

import (
	"crypto/rand"
	"io"
)

func RandomBytes(data []byte) error {
	_, err := io.ReadFull(rand.Reader, data)
	return err
}
