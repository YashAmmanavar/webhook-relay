package middleware

import "net/http"

// maxRequestBodyBytes caps how much of a request body the API will read, so
// a client can't exhaust memory by sending an oversized payload (doc section
// 21: "Maximum payload: 1 MB").
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// BodyLimit wraps every request body with http.MaxBytesReader. A handler
// that reads past the limit gets a *http.MaxBytesError, which the affected
// handlers check for to return 413 instead of a generic 400.
func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}
