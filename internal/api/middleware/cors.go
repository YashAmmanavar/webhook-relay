package middleware

import "net/http"

// CORS allows the dashboard (served from a different origin/port during
// development) to call this API from the browser. Wide open for now since
// there's no auth yet — tighten this to a specific configured origin once
// Phase 9 (security) adds API authentication.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
