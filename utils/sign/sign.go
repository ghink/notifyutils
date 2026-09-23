package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// HMACSHA256 is the shared primitive behind the request signatures of the robot
// channels. What differs per channel is what is used as key and what as message, so
// only the primitive is shared; each driver documents its own composition.
func hmacSHA256(key, message []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(message)
	return m.Sum(nil)
}

func HMACSHA256Base64(key, message []byte) string {
	return base64.StdEncoding.EncodeToString(hmacSHA256(key, message))
}

func HMACSHA256Hex(key, message []byte) string {
	return hex.EncodeToString(hmacSHA256(key, message))
}

func UnixSeconds() int64 {
	return time.Now().Unix()
}

func UnixMilli() int64 {
	return time.Now().UnixMilli()
}
