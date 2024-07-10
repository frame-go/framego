package crypto

import (
	"crypto/hmac"
	"crypto/sha256"

	"golang.org/x/crypto/sha3"
)

// Sha2Sum256 calculates Sha2-256 hash
func Sha2Sum256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// Sha3Sum256 calculates Sha3-256 hash
func Sha3Sum256(data []byte) []byte {
	hash := sha3.Sum256(data)
	return hash[:]
}

// HmacSha2Sum256 calculates HMAC-SHA2-256 signature
func HmacSha2Sum256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// HmacSha3Sum256 calculates HMAC-SHA3-256 signature
func HmacSha3Sum256(data, key []byte) []byte {
	h := hmac.New(sha3.New256, key)
	h.Write(data)
	return h.Sum(nil)
}
