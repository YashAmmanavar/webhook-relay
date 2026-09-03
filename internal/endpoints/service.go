package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"webhookrelay/internal/ssrf"
)

// ErrInvalidURL is returned when a caller supplies a URL that is not an
// absolute http:// or https:// URL, or one that resolves to a blocked
// address (see internal/ssrf).
var ErrInvalidURL = errors.New("url must be an absolute http or https URL")

// hostCheckTimeout bounds the DNS lookup done at registration time so a
// slow/unresponsive resolver can't hang endpoint creation.
const hostCheckTimeout = 3 * time.Second

// Service contains endpoint business logic, keeping HTTP handlers thin.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create validates the URL is well-formed and, as an early/coarse check,
// that its hostname doesn't currently resolve to a blocked address (doc
// section 21 — SSRF protection). This check is for good UX (reject an
// obviously bad URL immediately) — it is NOT the real security boundary,
// since DNS can change between now and actual delivery. The real
// enforcement is internal/delivery's dial hook, which validates the exact
// address being connected to at delivery time.
func (s *Service) Create(ctx context.Context, rawURL string) (*Endpoint, error) {
	rawURL = strings.TrimSpace(rawURL)
	u, err := validateURL(rawURL)
	if err != nil {
		return nil, err
	}

	checkCtx, cancel := context.WithTimeout(ctx, hostCheckTimeout)
	defer cancel()
	if err := ssrf.CheckHost(checkCtx, u.Hostname()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	return s.repo.Create(ctx, rawURL)
}

func (s *Service) Get(ctx context.Context, id string) (*Endpoint, error) {
	return s.repo.GetByID(ctx, id)
}

func validateURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, ErrInvalidURL
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" {
		return nil, ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrInvalidURL
	}
	return u, nil
}
