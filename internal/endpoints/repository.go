package endpoints

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when an endpoint with the given ID does not exist.
var ErrNotFound = errors.New("endpoint not found")

// Repository provides raw SQL access to the endpoints table.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create generates a new endpoint ID and secret, persists the endpoint, and
// returns the stored row.
func (r *Repository) Create(ctx context.Context, url string) (*Endpoint, error) {
	id, err := generateID("ep_")
	if err != nil {
		return nil, err
	}
	secret, err := generateID("whsec_")
	if err != nil {
		return nil, err
	}

	const query = `
		INSERT INTO endpoints (id, url, secret)
		VALUES ($1, $2, $3)
		RETURNING id, url, secret, created_at, updated_at
	`

	var ep Endpoint
	err = r.pool.QueryRow(ctx, query, id, url, secret).Scan(
		&ep.ID, &ep.URL, &ep.Secret, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert endpoint: %w", err)
	}

	return &ep, nil
}

// GetByID fetches a single endpoint by its ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Endpoint, error) {
	const query = `
		SELECT id, url, secret, created_at, updated_at
		FROM endpoints
		WHERE id = $1
	`

	var ep Endpoint
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ep.ID, &ep.URL, &ep.Secret, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint %q: %w", id, err)
	}

	return &ep, nil
}
