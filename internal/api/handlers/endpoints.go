package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"webhookrelay/internal/endpoints"
)

type EndpointsHandler struct {
	service *endpoints.Service
}

func NewEndpointsHandler(service *endpoints.Service) *EndpointsHandler {
	return &EndpointsHandler{service: service}
}

type createEndpointRequest struct {
	URL string `json:"url"`
}

type endpointResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// Create handles POST /api/v1/endpoints.
func (h *EndpointsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ep, err := h.service.Create(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, endpoints.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("failed to create endpoint: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create endpoint")
		return
	}

	writeJSON(w, http.StatusCreated, endpointResponse{
		ID:     ep.ID,
		URL:    ep.URL,
		Secret: ep.Secret,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
