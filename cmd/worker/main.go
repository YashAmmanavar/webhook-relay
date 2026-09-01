package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"webhookrelay/internal/database"
	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
)

// readBlock is how long a single Read call waits for new messages before
// returning empty-handed, so the shutdown signal gets checked regularly.
const readBlock = 5 * time.Second

// retryScanInterval is how often the worker checks for RETRYING events whose
// backoff has elapsed and re-queues them.
const retryScanInterval = 15 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	pool, err := database.NewPool(connectCtx, database.DefaultConfig())
	cancel()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	redisCtx, redisCancel := context.WithTimeout(ctx, 5*time.Second)
	redisClient, err := queue.NewClient(redisCtx, queue.DefaultConfig())
	redisCancel()
	if err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}
	defer redisClient.Close()
	eventQueue := queue.New(redisClient)

	if err := eventQueue.EnsureGroup(ctx); err != nil {
		log.Fatalf("could not set up consumer group: %v", err)
	}

	endpointsService := endpoints.NewService(endpoints.NewRepository(pool))
	attemptRepo := delivery.NewAttemptRepository(pool)
	eventsService := events.NewService(events.NewRepository(pool), endpointsService, delivery.NewClient(), attemptRepo, eventQueue)

	consumerName := consumerName()
	log.Printf("worker %q started, waiting for events", consumerName)

	go runRetryScanner(ctx, eventsService)

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		default:
		}

		messages, err := eventQueue.Read(ctx, consumerName, readBlock)
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
