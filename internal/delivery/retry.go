package delivery

import (
	"math/rand"
	"time"
)

// MaxAttempts is the maximum number of delivery attempts made before an
// event is considered permanently failed (doc section 12).
const MaxAttempts = 6

// baseDelay is the starting backoff delay; each subsequent retry doubles it:
// 1m, 2m, 4m, 8m, 16m for attempts 1-5 (doc section 12).
const baseDelay = 1 * time.Minute

// retryableStatusCodes are response codes that indicate a transient problem
// on the receiving end, worth retrying (doc section 11).
var retryableStatusCodes = map[int]bool{
	408: true,
	429: true,
	500: true,
	502: true,
	503: true,
	504: true,
}

// nonRetryableStatusCodes indicate a permanent problem with the request
// itself — retrying won't help (doc section 11).
var nonRetryableStatusCodes = map[int]bool{
	400: true,
	401: true,
	403: true,
	404: true,
	422: true,
}

// IsRetryable reports whether a delivery outcome is worth retrying. A
// network error or timeout (no response at all) is always retryable. For an
// HTTP response, the doc's explicit lists take precedence; any other status
// defaults to retryable only if it's a 5xx (an unlisted 4xx is assumed to be
// a permanent client-side problem).
func IsRetryable(statusCode int, err error) bool {
	if err != nil {
		return true
	}
	if retryableStatusCodes[statusCode] {
		return true
	}
	if nonRetryableStatusCodes[statusCode] {
		return false
	}
	return statusCode >= 500
}

// BackoffDelay returns how long to wait before making another delivery
// attempt, given how many attempts have been made so far (>= 1), plus up to
// 20% random jitter so many failed events don't retry in lockstep.
func BackoffDelay(attemptCount int) time.Duration {
	delay := baseDelay * time.Duration(int64(1)<<uint(attemptCount-1))
	jitter := time.Duration(rand.Int63n(int64(delay)/5 + 1))
	return delay + jitter
}
