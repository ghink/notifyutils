package sign

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"
)

// RFC 4231 test case 2, so a regression in the primitive is caught against an
// external source of truth rather than against this package's own output.
const (
	rfc4231Key     = "Jefe"
	rfc4231Message = "what do ya want for nothing?"
	rfc4231Hex     = "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"
	rfc4231Base64  = "W9zBRr9gdU5qBCQmCJV1x1oAPwidJzmDnexYuWTsOEM="
)

func TestHMACSHA256(t *testing.T) {
	got := hex.EncodeToString(hmacSHA256([]byte(rfc4231Key), []byte(rfc4231Message)))

	if got != rfc4231Hex {
		t.Errorf("HMACSHA256() = %s, want %s", got, rfc4231Hex)
	}
}

func TestHMACSHA256Hex(t *testing.T) {
	got := HMACSHA256Hex([]byte(rfc4231Key), []byte(rfc4231Message))

	if got != rfc4231Hex {
		t.Errorf("HMACSHA256Hex() = %s, want %s", got, rfc4231Hex)
	}
}

func TestHMACSHA256Base64(t *testing.T) {
	got := HMACSHA256Base64([]byte(rfc4231Key), []byte(rfc4231Message))

	if got != rfc4231Base64 {
		t.Errorf("HMACSHA256Base64() = %s, want %s", got, rfc4231Base64)
	}
	// Cross-check against the hex form through an independent encoding path.
	digest, err := hex.DecodeString(rfc4231Hex)
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString(digest); got != want {
		t.Errorf("HMACSHA256Base64() = %s, want %s", got, want)
	}
}

func TestHMACSHA256EmptyKey(t *testing.T) {
	// A signature over an empty key is still well defined; it must not panic.
	if got := hmacSHA256(nil, []byte("x")); len(got) != 32 {
		t.Errorf("digest length = %d, want 32", len(got))
	}
}

func TestTimestamps(t *testing.T) {
	now := time.Now()

	if diff := UnixSeconds() - now.Unix(); diff < -1 || diff > 1 {
		t.Errorf("UnixSeconds() = %d, want within a second of %d", UnixSeconds(), now.Unix())
	}
	if diff := UnixMilli() - now.UnixMilli(); diff < -1000 || diff > 1000 {
		t.Errorf("UnixMilli() = %d, want within a second of %d", UnixMilli(), now.UnixMilli())
	}
}
