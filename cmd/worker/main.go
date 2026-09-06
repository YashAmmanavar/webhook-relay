package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhookrelay/internal/database"
	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
	"webhookrelay/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	pool, err := database.NewPool(connectCtx, database.ConnStringFromEnv())
	cancel()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	redisCtx, redisCancel := context.WithTimeout(ctx, 5*time.Second)
	redisClient, err := queue.NewClient(redisCtx, queue.Config{Addr: queue.AddrFromEnv()})
	redisCancel()
	if err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}
	defer redisClient.Close()
	eventQueue := queue.New(redisClient)

	endpointsService := endpoints.NewService(endpoints.NewRepository(pool))
	attemptRepo := delivery.NewAttemptRepository(pool)
	eventsService := events.NewService(events.NewRepository(pool), endpointsService, delivery.NewClient(), attemptRepo, eventQueue)

	worker.Run(ctx, eventsService, eventQueue)
}
