package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func SignData(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func CheckSignature(mac1, mac2 string) bool {
	return hmac.Equal([]byte(mac1), []byte(mac2))
}
