package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSha2Sum256(t *testing.T) {
	data := []string{
		"This is a test",
		"",
	}
	sha256Expectation := []string{
		"c7be1ed902fb8dd4d48997c6452f5d7e509fbcdbe2808b16bcf4edce4c07d14e",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	for i := range data {
		result := Sha2Sum256([]byte(data[i]))
		strSum := hex.EncodeToString(result[:])
		assert.Equal(t, sha256Expectation[i], strSum)
	}
}

func TestSha3Sum256(t *testing.T) {
	data := []string{
		"This is a test",
		"",
	}
	sha256Expectation := []string{
		"3c3b66edcfe51f5b15bf372f61e25710ffc1ad3c0e3c60d832b42053a96772cf",
		"a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
	}

	for i := range data {
		result := Sha3Sum256([]byte(data[i]))
		strSum := hex.EncodeToString(result[:])
		assert.Equal(t, sha256Expectation[i], strSum)
	}
}

func TestHmacSha2Sum256(t *testing.T) {
	data := []string{
		"This is a test",
		"",
	}
	key := "TestKey"
	sha256Expectation := []string{
		"55162764899345021b3050ebd07138c123ecac31250f839b6aea60285e2f6136",
		"fb577cfa7e2cd09444b939e810c72ab67adc2c09cb5531a20adac71f6ef9e151",
	}
	for i := range data {
		result := HmacSha2Sum256([]byte(data[i]), []byte(key))
		strSum := hex.EncodeToString(result[:])
		assert.Equal(t, sha256Expectation[i], strSum)
	}
}

func TestHmacSha3Sum256(t *testing.T) {
	data := []string{
		"This is a test",
		"",
	}
	key := "TestKey"
	sha256Expectation := []string{
		"bb8db886269568dcfb4bc1b57f945a75b2616a9cc4aaef791b284939163ce50e",
		"0fec3e8ae639be34b20d5db20b1d3c9c1e12fbb78b83070c441e6035ea1a8975",
	}
	for i := range data {
		result := HmacSha3Sum256([]byte(data[i]), []byte(key))
		strSum := hex.EncodeToString(result[:])
		assert.Equal(t, sha256Expectation[i], strSum)
	}
}
