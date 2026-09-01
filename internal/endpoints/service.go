package endpoints

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// ErrInvalidURL is returned when a caller supplies a URL that is not an
// absolute http:// or https:// URL.
var ErrInvalidURL = errors.New("url must be an absolute http or https URL")

// Service contains endpoint business logic, keeping HTTP handlers thin.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, rawURL string) (*Endpoint, error) {
	rawURL = strings.TrimSpace(rawURL)
	if err := validateURL(rawURL); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, rawURL)
}

func (s *Service) Get(ctx context.Context, id string) (*Endpoint, error) {
	return s.repo.GetByID(ctx, id)
}

func validateURL(raw string) error {
	if raw == "" {
		return ErrInvalidURL
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}
	return nil
}
