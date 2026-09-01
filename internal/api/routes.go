package api

import (
	"net/http"

	"webhookrelay/internal/api/handlers"
)

// NewRouter registers all HTTP routes and returns the top-level handler.
func NewRouter(endpointsHandler *handlers.EndpointsHandler, eventsHandler *handlers.EventsHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/endpoints", endpointsHandler.Create)
	mux.HandleFunc("POST /api/v1/events", eventsHandler.Create)
	mux.HandleFunc("GET /api/v1/events", eventsHandler.List)
	mux.HandleFunc("GET /api/v1/events/{id}", eventsHandler.Get)
	mux.HandleFunc("POST /api/v1/events/{id}/retry", eventsHandler.Retry)
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
