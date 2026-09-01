package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when an event with the given ID does not exist.
var ErrNotFound = errors.New("event not found")

// Repository provides raw SQL access to the events table.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create generates a new event ID and persists the event with status
// PENDING (the table default), returning the stored row.
func (r *Repository) Create(ctx context.Context, endpointID, eventType string, payload json.RawMessage) (*Event, error) {
	id, err := generateID("evt_")
	if err != nil {
		return nil, err
	}

	const query = `
		INSERT INTO events (id, endpoint_id, event_type, payload)
		VALUES ($1, $2, $3, $4)
		RETURNING id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
	`

	var ev Event
	err = r.pool.QueryRow(ctx, query, id, endpointID, eventType, payload).Scan(
		&ev.ID, &ev.EndpointID, &ev.EventType, &ev.Payload, &ev.Status, &ev.AttemptCount, &ev.NextRetryAt, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return &ev, nil
}

// GetByID fetches a single event by its ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Event, error) {
	const query = `
		SELECT id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var ev Event
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ev.ID, &ev.EndpointID, &ev.EventType, &ev.Payload, &ev.Status, &ev.AttemptCount, &ev.NextRetryAt, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event %q: %w", id, err)
	}

	return &ev, nil
}

// listLimit caps how many events a single List call can return, so the
// dashboard's list view can't accidentally pull an unbounded result set.
const listLimit = 200

// List returns the most recent events, optionally filtered by status.
func (r *Repository) List(ctx context.Context, status string) ([]*Event, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		const query = `
			SELECT id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
			FROM events
			ORDER BY created_at DESC
			LIMIT $1
		`
		rows, err = r.pool.Query(ctx, query, listLimit)
	} else {
		const query = `
			SELECT id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
			FROM events
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, status, listLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	events := []*Event{}
	for rows.Next() {
		var ev Event
		if err := rows.Scan(
			&ev.ID, &ev.EndpointID, &ev.EventType, &ev.Payload, &ev.Status, &ev.AttemptCount, &ev.NextRetryAt, &ev.CreatedAt, &ev.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed reading events: %w", err)
	}

	return events, nil
}

// UpdateStatus sets an event's status, attempt count, and next retry time
// (nil clears it — used for terminal states and once a due retry has been
// re-queued), returning the updated row.
func (r *Repository) UpdateStatus(ctx context.Context, id, status string, attemptCount int, nextRetryAt *time.Time) (*Event, error) {
	const query = `
		UPDATE events
		SET status = $2, attempt_count = $3, next_retry_at = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
	`

	var ev Event
	err := r.pool.QueryRow(ctx, query, id, status, attemptCount, nextRetryAt).Scan(
		&ev.ID, &ev.EndpointID, &ev.EventType, &ev.Payload, &ev.Status, &ev.AttemptCount, &ev.NextRetryAt, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update event %q: %w", id, err)
	}

	return &ev, nil
}

// FindDueRetries returns events in RETRYING status whose next_retry_at has
// passed, so the worker's retry scanner can re-queue them for delivery.
func (r *Repository) FindDueRetries(ctx context.Context) ([]*Event, error) {
	const query = `
		SELECT id, endpoint_id, event_type, payload, status, attempt_count, next_retry_at, created_at, updated_at
		FROM events
		WHERE status = 'RETRYING' AND next_retry_at IS NOT NULL AND next_retry_at <= now()
		ORDER BY next_retry_at
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query due retries: %w", err)
	}
	defer rows.Close()

	var due []*Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(
			&ev.ID, &ev.EndpointID, &ev.EventType, &ev.Payload, &ev.Status, &ev.AttemptCount, &ev.NextRetryAt, &ev.CreatedAt, &ev.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan due retry: %w", err)
		}
		due = append(due, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed reading due retries: %w", err)
	}

	return due, nil
}
