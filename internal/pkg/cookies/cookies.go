package cookies

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

var signKeyValueError = func(err error) error {
	return fmt.Errorf("error signing key value: %w", err)
}

func SignKeyValue(key string, value string, secretKey []byte) (string, error) {
	if key == "" {
		return "", signKeyValueError(errors.New("empty key"))
	}

	if value == "" {
		return "", signKeyValueError(errors.New("empty value"))
	}

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(key))
	mac.Write([]byte(value))
	signature := mac.Sum(nil)

	var result bytes.Buffer
	result.Write(signature)
	result.Write([]byte(value))

	return base64.StdEncoding.EncodeToString(result.Bytes()), nil
}

var verifySignedKeyValueError = func(err error) error {
	return fmt.Errorf("error verifying signed key value: %w", err)
}

func VerifySignedKeyValue(key string, signedValue string, secretKey []byte) (string, error) {
	if key == "" {
		return "", verifySignedKeyValueError(errors.New("empty key"))
	}

	if signedValue == "" {
		return "", verifySignedKeyValueError(errors.New("empty signedValue"))
	}

	signedValueBytes, err := base64.StdEncoding.DecodeString(signedValue)
	if err != nil {
		return "", verifySignedKeyValueError(err)
	}

	if len(signedValueBytes) < sha256.Size {
		return "", verifySignedKeyValueError(errors.New("signed value is too short"))
	}

	signature := signedValueBytes[:sha256.Size]
	value := signedValueBytes[sha256.Size:]

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(key))
	mac.Write(value)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return "", verifySignedKeyValueError(errors.New("invalid signature"))
	}

	return string(value), nil
}
