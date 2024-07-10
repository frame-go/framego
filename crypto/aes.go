package crypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/pkg/errors"
)

// AESGCMEncrypt encrypts data by AES in GCM mode with random nonce header
// `data` can be bytes in any length
// `key` should be a slice with length 16 / 24 / 32
// Result = (Nonce [12] byte + EncryptedData [] byte + Tag [16] byte)
func AESGCMEncrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes_encrypt_new_cipher_failed")
	}
	crypter, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "aes_encrypt_new_gcm_failed")
	}
	nonceSize := crypter.NonceSize()
	bufferSize := nonceSize + len(data) + crypter.Overhead()
	buffer := make([]byte, nonceSize, bufferSize)
	RandomBytes(buffer)
	buffer = crypter.Seal(buffer, buffer, data, nil)
	return buffer, nil
}

// AESGCMDecrypt decrypts data encrypted by AESGCMEncrypt
// Return errors if the data can not be decrypted or can not pass integrity verification
func AESGCMDecrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes_decrypt_new_cipher_failed")
	}
	crypter, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "aes_decrypt_new_gcm_failed")
	}
	nonceSize := crypter.NonceSize()
	return crypter.Open(nil, data[:nonceSize], data[nonceSize:], nil)
}
