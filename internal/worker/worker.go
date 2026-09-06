// Package worker contains the delivery worker's consume loop: pulling event
// IDs off the queue and delivering them, plus the retry scanner that
// re-queues events whose backoff has elapsed. Shared by cmd/worker (a
// standalone worker process) and cmd/server (worker + API combined into one
// process, for deployment targets that only support a single always-on
// service).
package worker

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
)

// readBlock is how long a single Read call waits for new messages before
// returning empty-handed, so a shutdown signal gets checked regularly.
const readBlock = 5 * time.Second

// retryScanInterval is how often the scanner checks for RETRYING events
// whose backoff has elapsed and re-queues them.
const retryScanInterval = 15 * time.Second

// Run ensures the consumer group exists, then consumes events from the
// queue and delivers them — running the retry scanner alongside it — until
// ctx is cancelled. Blocks until shutdown.
func Run(ctx context.Context, eventsService *events.Service, eventQueue *queue.Queue) {
	if err := eventQueue.EnsureGroup(ctx); err != nil {
		log.Fatalf("could not set up consumer group: %v", err)
	}

	name := consumerName()
	log.Printf("worker %q started, waiting for events", name)

	go runRetryScanner(ctx, eventsService)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down")
			return
		default:
		}

		messages, err := eventQueue.Read(ctx, name, readBlock)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				continue
			}
			log.Printf("error reading from queue: %v", err)
			continue
		}

		for _, msg := range messages {
			if err := eventsService.Deliver(ctx, msg.EventID); err != nil {
				log.Printf("delivery failed for event %s: %v", msg.EventID, err)
			}
			if err := eventQueue.Ack(ctx, msg.ID); err != nil {
				log.Printf("failed to ack message %s for event %s: %v", msg.ID, msg.EventID, err)
			}
		}
	}
}

// consumerName identifies this worker process to Redis, distinguishing it
// from other worker processes/replicas reading the same consumer group.
func consumerName() string {
	host, err := os.Hostname()
	if err != nil {
		host = "worker"
	}
	return host + "-" + strconv.Itoa(os.Getpid())
}

// runRetryScanner periodically re-queues RETRYING events whose backoff delay
// has elapsed, until ctx is cancelled.
func runRetryScanner(ctx context.Context, eventsService *events.Service) {
	ticker := time.NewTicker(retryScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := eventsService.EnqueueDueRetries(ctx)
			if err != nil {
				log.Printf("retry scan failed: %v", err)
				continue
			}
			if n > 0 {
				log.Printf("re-queued %d due retries", n)
			}
		}
	}
}
