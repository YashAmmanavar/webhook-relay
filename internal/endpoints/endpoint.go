package endpoints

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Endpoint is a webhook destination that events can be sent to.
//
// Secret is deliberately excluded from JSON (`json:"-"`) even though the
// struct itself has no other reason to avoid marshaling: it's a credential,
// and the only place it should ever reach an API response is the one-time
// POST /api/v1/endpoints creation response, which builds its own explicit
// DTO rather than marshaling this struct directly (see
// handlers.endpointResponse). Keeping Secret unexported from JSON here means
// a future handler that carelessly returns a raw *Endpoint (e.g. a GET
// /endpoints list) can't leak every secret by accident — it would have to
// deliberately reference .Secret the way the creation handler already does.
type Endpoint struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
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
