package endpoints

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Endpoint is a webhook destination that events can be sent to.
type Endpoint struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// generateID returns a random identifier of the form "<prefix><32 hex chars>",
// e.g. "ep_1a2b3c..." or "whsec_1a2b3c...".
func generateID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}
