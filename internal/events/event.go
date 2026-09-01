package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Status values match the CHECK constraint on the events table.
const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusRetrying   = "RETRYING"
	StatusDelivered  = "DELIVERED"
	StatusFailed     = "FAILED"
)

// Event is a single webhook delivery attempt record, queued for delivery to
// an endpoint.
type Event struct {
	ID           string          `json:"id"`
	EndpointID   string          `json:"endpoint_id"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	AttemptCount int             `json:"attempt_count"`
	NextRetryAt  *time.Time      `json:"next_retry_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// generateID returns a random identifier of the form "<prefix><32 hex chars>",
// e.g. "evt_1a2b3c...".
func generateID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}
