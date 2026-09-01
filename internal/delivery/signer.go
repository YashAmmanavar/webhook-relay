package delivery

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Sign computes the webhook signature for a payload (doc section 13):
// HMAC-SHA256(secret, "<unix timestamp>.<raw payload bytes>"), returned as
// "v1=<hex-encoded mac>" so the scheme can change later without breaking
// receivers that already parse the "v1=" prefix.
func Sign(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
