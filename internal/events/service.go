package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/queue"
)

var (
	ErrInvalidEventType = errors.New("event_type is required")
	ErrInvalidPayload   = errors.New("payload is required")
	ErrEndpointNotFound = errors.New("endpoint not found")
)

// Service contains event business logic, keeping HTTP handlers thin.
type Service struct {
	repo      *Repository
	endpoints *endpoints.Service
	delivery  *delivery.Client
	attempts  *delivery.AttemptRepository
	queue     *queue.Queue
}

func NewService(repo *Repository, endpointsService *endpoints.Service, deliveryClient *delivery.Client, attemptRepo *delivery.AttemptRepository, q *queue.Queue) *Service {
	return &Service{repo: repo, endpoints: endpointsService, delivery: deliveryClient, attempts: attemptRepo, queue: q}
}

// Create validates the request, persists a new event as PENDING, and queues
// it for delivery. Delivery itself happens out-of-band, via a worker calling
// Deliver — see Phase 4.
func (s *Service) Create(ctx context.Context, endpointID, eventType string, payload json.RawMessage) (*Event, error) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return nil, ErrInvalidEventType
	}
	if len(payload) == 0 {
		return nil, ErrInvalidPayload
	}

	if _, err := s.endpoints.Get(ctx, endpointID); err != nil {
		if errors.Is(err, endpoints.ErrNotFound) {
			return nil, ErrEndpointNotFound
		}
		return nil, fmt.Errorf("failed to verify endpoint: %w", err)
	}

	ev, err := s.repo.Create(ctx, endpointID, eventType, payload)
	if err != nil {
		return nil, err
	}

	if err := s.queue.Enqueue(ctx, ev.ID); err != nil {
		return nil, fmt.Errorf("event %s created but failed to queue for delivery: %w", ev.ID, err)
	}

	return ev, nil
}

// Deliver loads an event by ID, attempts delivery to its endpoint, and
// updates its status based on the outcome: DELIVERED on success; RETRYING
// (with a backoff next_retry_at) if the failure is retryable and attempts
// remain; otherwise FAILED. Called by the worker process after it dequeues
// an event ID — both for a first attempt and for a re-queued retry.
func (s *Service) Deliver(ctx context.Context, eventID string) error {
	ev, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to load event %s: %w", eventID, err)
	}

	ep, err := s.endpoints.Get(ctx, ev.EndpointID)
	if err != nil {
		return fmt.Errorf("failed to load endpoint %s for event %s: %w", ev.EndpointID, eventID, err)
	}

	result := s.delivery.Deliver(ctx, delivery.Request{
		URL:     ep.URL,
		EventID: ev.ID,
		Secret:  ep.Secret,
		Payload: ev.Payload,
	})
	attemptCount := ev.AttemptCount + 1

	var errs []error
	if err := s.attempts.LogAttempt(ctx, ev.ID, attemptCount, result); err != nil {
		errs = append(errs, err)
	}

	var status string
	var nextRetryAt *time.Time
	switch {
	case result.Delivered:
		status = StatusDelivered
	case delivery.IsRetryable(result.StatusCode, result.Err) && attemptCount < delivery.MaxAttempts:
		status = StatusRetrying
		t := time.Now().Add(delivery.BackoffDelay(attemptCount))
		nextRetryAt = &t
	default:
		status = StatusFailed
	}

	if _, err := s.repo.UpdateStatus(ctx, ev.ID, status, attemptCount, nextRetryAt); err != nil {
		errs = append(errs, fmt.Errorf("failed to record delivery result: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("delivery attempted for event %s: %w", eventID, errors.Join(errs...))
	}

	return nil
}

// EnqueueDueRetries finds RETRYING events whose backoff has elapsed and
// re-queues them for delivery, clearing next_retry_at so the scanner doesn't
// pick the same event up again before Deliver processes it. Returns how many
// were re-queued. Intended to be called on a timer by the worker process.
func (s *Service) EnqueueDueRetries(ctx context.Context) (int, error) {
	due, err := s.repo.FindDueRetries(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find due retries: %w", err)
	}

	for _, ev := range due {
		if err := s.queue.Enqueue(ctx, ev.ID); err != nil {
			return 0, fmt.Errorf("failed to re-queue event %s: %w", ev.ID, err)
		}
		if _, err := s.repo.UpdateStatus(ctx, ev.ID, StatusRetrying, ev.AttemptCount, nil); err != nil {
			return 0, fmt.Errorf("failed to clear next_retry_at for event %s: %w", ev.ID, err)
		}
	}

	return len(due), nil
}
