package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// StreamName is the Redis Stream that event delivery jobs are queued on.
const StreamName = "webhookrelay:events"

// GroupName is the consumer group all workers read from, so that a message
// is only delivered to one worker at a time and can be reclaimed if that
// worker crashes before acknowledging it.
const GroupName = "webhookrelay-workers"

// pendingIdleThreshold is how long a message may sit claimed-but-unacked by
// a consumer before another consumer is allowed to reclaim it (e.g. after a
// worker crash).
const pendingIdleThreshold = 30 * time.Second

// reclaimBatchSize and readBatchSize cap how many messages are pulled per
// Read call.
const (
	reclaimBatchSize = 10
	readBatchSize    = 10
)

// Queue provides producer and consumer access to the events delivery stream.
type Queue struct {
	client *redis.Client
}

func New(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// Enqueue adds an event ID to the stream for a worker to pick up.
func (q *Queue) Enqueue(ctx context.Context, eventID string) error {
	err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{"event_id": eventID},
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue event %s: %w", eventID, err)
	}
	return nil
}

// EnsureGroup creates the consumer group (and the stream itself, if it
// doesn't exist yet) starting from the beginning of the stream. Safe to call
// every time a worker starts — an already-existing group is not an error.
func (q *Queue) EnsureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, StreamName, GroupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// Message is a single queued event delivery job.
type Message struct {
	ID      string // Redis stream entry ID, needed to Ack.
	EventID string
}

// Read waits up to block for new messages for the given consumer name. It
// first reclaims any messages that were claimed by a previous consumer but
// never acknowledged for longer than pendingIdleThreshold (e.g. because that
// worker crashed), so at-least-once delivery holds across worker restarts.
func (q *Queue) Read(ctx context.Context, consumer string, block time.Duration) ([]Message, error) {
	claimed, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   StreamName,
		Group:    GroupName,
		Consumer: consumer,
		MinIdle:  pendingIdleThreshold,
		Start:    "0",
		Count:    reclaimBatchSize,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to reclaim pending messages: %w", err)
	}
	if len(claimed) > 0 {
		return toMessages(claimed), nil
	}

	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    GroupName,
		Consumer: consumer,
		Streams:  []string{StreamName, ">"},
		Count:    readBatchSize,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}
	if len(streams) == 0 {
		return nil, nil
	}

	return toMessages(streams[0].Messages), nil
}

// Ack acknowledges that a message has been fully processed, removing it from
// the consumer group's pending entries list.
func (q *Queue) Ack(ctx context.Context, id string) error {
	if err := q.client.XAck(ctx, StreamName, GroupName, id).Err(); err != nil {
		return fmt.Errorf("failed to ack message %s: %w", id, err)
	}
	return nil
}

func toMessages(entries []redis.XMessage) []Message {
	msgs := make([]Message, 0, len(entries))
	for _, e := range entries {
		eventID, _ := e.Values["event_id"].(string)
		msgs = append(msgs, Message{ID: e.ID, EventID: eventID})
	}
	return msgs
}
