package delivery

import (
	"errors"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{"network error", 0, errors.New("connection refused"), true},
		{"408 request timeout", 408, nil, true},
		{"429 too many requests", 429, nil, true},
		{"500 internal server error", 500, nil, true},
		{"502 bad gateway", 502, nil, true},
		{"503 service unavailable", 503, nil, true},
		{"504 gateway timeout", 504, nil, true},
		{"400 bad request", 400, nil, false},
		{"401 unauthorized", 401, nil, false},
		{"403 forbidden", 403, nil, false},
		{"404 not found", 404, nil, false},
		{"422 unprocessable entity", 422, nil, false},
		{"unlisted 5xx defaults retryable", 599, nil, true},
		{"unlisted 4xx defaults non-retryable", 418, nil, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsRetryable(c.statusCode, c.err); got != c.want {
				t.Errorf("IsRetryable(%d, %v) = %v, want %v", c.statusCode, c.err, got, c.want)
			}
		})
	}
}

func TestBackoffDelay(t *testing.T) {
	// Expected base delays before jitter: 1m, 2m, 4m, 8m, 16m for attempts 1-5.
	wantBase := []time.Duration{
		1 * time.Minute,
		2 * time.Minute,
		4 * time.Minute,
		8 * time.Minute,
		16 * time.Minute,
	}

	for i, base := range wantBase {
		attempt := i + 1
		got := BackoffDelay(attempt)
		maxWithJitter := base + base/5 + 1
		if got < base || got > maxWithJitter {
			t.Errorf("BackoffDelay(%d) = %v, want between %v and %v", attempt, got, base, maxWithJitter)
		}
	}
}
