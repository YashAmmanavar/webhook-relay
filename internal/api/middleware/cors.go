package middleware

import "net/http"

// CORS allows the dashboard (served from a different origin/port) to call
// this API from the browser. allowedOrigin is normally "*" for local dev,
// but should be set to the dashboard's real deployed origin in production
// (via the ALLOWED_ORIGIN env var — see cmd/api/main.go) now that requests
// carry an API key worth not exposing to arbitrary origins.
func CORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
