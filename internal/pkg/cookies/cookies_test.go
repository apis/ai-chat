package cookies

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignKeyValuePositive(t *testing.T) {
	key := "testKey"
	value := "testValue"
	secretKey := []byte("testSecretKey")

	signedValue, err := SignKeyValue(key, value, secretKey)
	assert.NotEmpty(t, signedValue)
	assert.NoError(t, err)
}

func TestSignKeyValueNegativeEmptyKey(t *testing.T) {
	key := ""
	value := "testValue"
	secretKey := []byte("testSecretKey")

	signedValue, err := SignKeyValue(key, value, secretKey)
	assert.Empty(t, signedValue)
	assert.EqualError(t, err, "error signing key value: empty key")
}

func TestSignKeyValueNegativeEmptyValue(t *testing.T) {
	key := "testKey"
	value := ""
	secretKey := []byte("testSecretKey")

	signedValue, err := SignKeyValue(key, value, secretKey)
	assert.Empty(t, signedValue)
	assert.EqualError(t, err, "error signing key value: empty value")
}

func TestVerifySignedKeyValuePositive(t *testing.T) {
	key := "testKey"
	value := "testValue"
	secretKey := []byte("testSecretKey")

	signedValue, err := SignKeyValue(key, value, secretKey)
	assert.NotEmpty(t, signedValue)
	assert.NoError(t, err)

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Equal(t, value, verifiedValue)
	assert.NoError(t, err)
}

func TestVerifySignedKeyValueNegativeEmptyKey(t *testing.T) {
	key := ""
	signedValue := "testValue"
	secretKey := []byte("testSecretKey")

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Empty(t, verifiedValue)
	assert.EqualError(t, err, "error verifying signed key value: empty key")
}

func TestVerifySignedKeyValueNegativeEmptySignedValue(t *testing.T) {
	key := "testKey"
	signedValue := ""
	secretKey := []byte("testSecretKey")

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Empty(t, verifiedValue)
	assert.EqualError(t, err, "error verifying signed key value: empty signedValue")
}

func TestVerifySignedKeyValueNegativeSignedValueEncoding(t *testing.T) {
	key := "testKey"
	signedValue := "It is not BASE64 encoded value"
	secretKey := []byte("testSecretKey")

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Empty(t, verifiedValue)
	assert.EqualError(t, err, "error verifying signed key value: illegal base64 data at input byte 2")
}

func TestVerifySignedKeyValueNegativeSignedValueTooShort(t *testing.T) {
	key := "testKey"
	signedValue := base64.StdEncoding.EncodeToString([]byte("test"))
	secretKey := []byte("testSecretKey")

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Empty(t, verifiedValue)
	assert.EqualError(t, err, "error verifying signed key value: signed value is too short")
}

func TestVerifySignedKeyValueNegativeInvalidSignature(t *testing.T) {
	key := "testKey"
	signedValue := base64.StdEncoding.EncodeToString([]byte("test test test test test test test test test test test"))
	secretKey := []byte("testSecretKey")

	verifiedValue, err := VerifySignedKeyValue(key, signedValue, secretKey)
	assert.Empty(t, verifiedValue)
	assert.EqualError(t, err, "error verifying signed key value: invalid signature")
}
