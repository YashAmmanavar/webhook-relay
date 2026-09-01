package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"webhookrelay/internal/events"
)

type EventsHandler struct {
	service *events.Service
}

func NewEventsHandler(service *events.Service) *EventsHandler {
	return &EventsHandler{service: service}
}

type createEventRequest struct {
	EndpointID string          `json:"endpoint_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
}

type createEventResponse struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

// Create handles POST /api/v1/events.
func (h *EventsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ev, err := h.service.Create(r.Context(), req.EndpointID, req.EventType, req.Payload)
	if err != nil {
		switch {
		case errors.Is(err, events.ErrInvalidEventType), errors.Is(err, events.ErrInvalidPayload):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, events.ErrEndpointNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			log.Printf("failed to create event: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create event")
		}
		return
	}

	writeJSON(w, http.StatusCreated, createEventResponse{
		EventID: ev.ID,
		Status:  ev.Status,
	})
}
