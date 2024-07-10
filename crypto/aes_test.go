package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESSuccess(t *testing.T) {
	encString := "Things to be encrypted"
	data := []byte(encString)
	key := []byte("1234567890abcdef")
	buf, err := AESGCMEncrypt(data, key)
	assert.Nil(t, err)

	d, err := AESGCMDecrypt(buf, key)
	assert.Nil(t, err)
	assert.Equal(t, string(d), encString)
}

func TestAESFail(t *testing.T) {
	encString := "Things to be encrypted"
	data := []byte(encString)
	key := []byte("1234")
	_, err := AESGCMEncrypt(data, key)
	assert.NotNil(t, err)
}
