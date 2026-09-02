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

// List handles GET /api/v1/events, with an optional ?status= filter.
func (h *EventsHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	evs, err := h.service.List(r.Context(), status)
	if err != nil {
		if errors.Is(err, events.ErrInvalidStatus) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("failed to list events: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	writeJSON(w, http.StatusOK, evs)
}

// Get handles GET /api/v1/events/{id}, returning the event plus its full
// delivery attempt history.
func (h *EventsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	detail, err := h.service.GetWithAttempts(r.Context(), id)
	if err != nil {
		if errors.Is(err, events.ErrNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		log.Printf("failed to get event %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to get event")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// Retry handles POST /api/v1/events/{id}/retry.
func (h *EventsHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ev, err := h.service.Retry(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, events.ErrNotFound):
			writeError(w, http.StatusNotFound, "event not found")
		case errors.Is(err, events.ErrEventNotFailed):
			writeError(w, http.StatusConflict, err.Error())
		default:
			log.Printf("failed to retry event %s: %v", id, err)
			writeError(w, http.StatusInternalServerError, "failed to retry event")
		}
		return
	}

	writeJSON(w, http.StatusOK, createEventResponse{
		EventID: ev.ID,
		Status:  ev.Status,
	})
}
