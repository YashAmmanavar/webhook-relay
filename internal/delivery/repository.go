package delivery

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
