package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// rateLimitRPS and rateLimitBurst define a per-client token bucket: the
// sustained rate a client can make requests at, plus how far above that they
// can briefly burst. Not tunable via config for now — the doc doesn't call
// for a specific policy, just "prevent abuse" (section 21), and these are
// generous enough not to get in the way of normal use.
const (
	rateLimitRPS   = 5
	rateLimitBurst = 20
)

// RateLimit throttles requests per client IP using a token bucket, to guard
// against a single client hammering the API. The health check is exempt, so
// monitoring polls can't trip it.
//
// The per-IP limiter map is never evicted, so it grows for the life of the
// process (one entry per distinct IP ever seen). Fine for this single-
// operator MVP; a long-running multi-tenant deployment would want to expire
// idle entries.
func RateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		l, ok := limiters[key]
		if !ok {
			l = rate.NewLimiter(rate.Limit(rateLimitRPS), rateLimitBurst)
			limiters[key] = l
		}
		return l
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		if !getLimiter(clientIP(r)).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded, slow down"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
