package delivery

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Attempt is a single logged delivery attempt for an event.
type Attempt struct {
	ID            int64     `json:"id"`
	EventID       string    `json:"event_id"`
	AttemptNumber int       `json:"attempt_number"`
	StatusCode    *int      `json:"status_code"`
	ResponseBody  *string   `json:"response_body"`
	Error         *string   `json:"error"`
	DurationMs    *int      `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

// AttemptRepository provides raw SQL access to the delivery_attempts table.
type AttemptRepository struct {
	pool *pgxpool.Pool
}

func NewAttemptRepository(pool *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{pool: pool}
}

// LogAttempt records the outcome of a single delivery attempt for an event.
func (r *AttemptRepository) LogAttempt(ctx context.Context, eventID string, attemptNumber int, result Result) error {
	var statusCode *int
	if result.StatusCode != 0 {
		statusCode = &result.StatusCode
	}

	var responseBody *string
	if result.ResponseBody != "" {
		responseBody = &result.ResponseBody
	}

	var errMsg *string
	if result.Err != nil {
		msg := result.Err.Error()
		errMsg = &msg
	}

	const query = `
		INSERT INTO delivery_attempts (event_id, attempt_number, status_code, response_body, error, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, eventID, attemptNumber, statusCode, responseBody, errMsg, result.Duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("failed to log delivery attempt for event %s: %w", eventID, err)
	}

	return nil
}

// ListByEvent returns every logged attempt for an event, oldest first.
func (r *AttemptRepository) ListByEvent(ctx context.Context, eventID string) ([]Attempt, error) {
	const query = `
		SELECT id, event_id, attempt_number, status_code, response_body, error, duration_ms, created_at
		FROM delivery_attempts
		WHERE event_id = $1
		ORDER BY attempt_number
	`

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list delivery attempts for event %s: %w", eventID, err)
	}
	defer rows.Close()

	attempts := []Attempt{}
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.ID, &a.EventID, &a.AttemptNumber, &a.StatusCode, &a.ResponseBody, &a.Error, &a.DurationMs, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan delivery attempt: %w", err)
		}
		attempts = append(attempts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed reading delivery attempts: %w", err)
	}

	return attempts, nil
}
