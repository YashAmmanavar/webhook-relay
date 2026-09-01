package delivery

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSign(t *testing.T) {
	secret := "whsec_test123"
	var timestamp int64 = 1780000000
	payload := []byte(`{"payment_id":"pay_123","amount":4999}`)

	got := Sign(secret, timestamp, payload)

	if !strings.HasPrefix(got, "v1=") {
		t.Fatalf("Sign() = %q, want v1= prefix", got)
	}

	// Recompute independently to confirm Sign matches the documented scheme:
	// HMAC-SHA256(secret, timestamp + "." + raw_body).
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("1780000000."))
	mac.Write(payload)
	want := "v1=" + hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Errorf("Sign() = %q, want %q", got, want)
	}
}

func TestSign_DifferentInputsProduceDifferentSignatures(t *testing.T) {
	base := Sign("secret", 1000, []byte("payload"))

	if s := Sign("other-secret", 1000, []byte("payload")); s == base {
		t.Error("different secret produced the same signature")
	}
	if s := Sign("secret", 2000, []byte("payload")); s == base {
		t.Error("different timestamp produced the same signature")
	}
	if s := Sign("secret", 1000, []byte("other-payload")); s == base {
		t.Error("different payload produced the same signature")
	}
}
