package delivery

import (
	"errors"
	"testing"

	"webhookrelay/internal/ssrf"
)

func TestClient_Deliver_BlocksLoopback(t *testing.T) {
	client := NewClient()
	result := client.Deliver(t.Context(), Request{
		URL:     "http://127.0.0.1:1/webhook",
		EventID: "evt_test",
		Secret:  "secret",
		Payload: []byte(`{}`),
	})

	if result.Delivered {
		t.Fatal("expected delivery to a loopback address to fail, but it succeeded")
	}
	if !errors.Is(result.Err, ssrf.ErrBlockedAddress) {
		t.Errorf("expected result.Err to wrap ssrf.ErrBlockedAddress, got: %v", result.Err)
	}
}

func TestClient_Deliver_BlocksLinkLocal(t *testing.T) {
	// 169.254.169.254 is the cloud metadata address every major provider
	// uses — the single most important address to keep blocked.
	client := NewClient()
	result := client.Deliver(t.Context(), Request{
		URL:     "http://169.254.169.254/latest/meta-data/",
		EventID: "evt_test",
		Secret:  "secret",
		Payload: []byte(`{}`),
	})

	if result.Delivered {
		t.Fatal("expected delivery to the cloud metadata address to fail, but it succeeded")
	}
	if !errors.Is(result.Err, ssrf.ErrBlockedAddress) {
		t.Errorf("expected result.Err to wrap ssrf.ErrBlockedAddress, got: %v", result.Err)
	}
}

func TestClient_Deliver_AllowsPublicAddress(t *testing.T) {
	// Hits a real external service — only fails if OUR code blocked it, not
	// for unrelated network issues, so it stays meaningful without being
	// fragile to transient connectivity problems.
	client := NewClient()
	result := client.Deliver(t.Context(), Request{
		URL:     "https://httpstat.us/200",
		EventID: "evt_test",
		Secret:  "secret",
		Payload: []byte(`{}`),
	})

	if result.Err != nil && errors.Is(result.Err, ssrf.ErrBlockedAddress) {
		t.Errorf("public address was incorrectly blocked: %v", result.Err)
	}
}
