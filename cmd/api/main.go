package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"webhookrelay/internal/api"
	"webhookrelay/internal/api/handlers"
	"webhookrelay/internal/database"
	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
)

func main() {
	connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := database.NewPool(connectCtx, database.DefaultConfig())
	cancel()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer pool.Close()

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	redisClient, err := queue.NewClient(redisCtx, queue.DefaultConfig())
	redisCancel()
	if err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}
	defer redisClient.Close()
	eventQueue := queue.New(redisClient)

	endpointsRepo := endpoints.NewRepository(pool)
	endpointsService := endpoints.NewService(endpointsRepo)
	endpointsHandler := handlers.NewEndpointsHandler(endpointsService)

	deliveryClient := delivery.NewClient()
	attemptRepo := delivery.NewAttemptRepository(pool)

	eventsRepo := events.NewRepository(pool)
	eventsService := events.NewService(eventsRepo, endpointsService, deliveryClient, attemptRepo, eventQueue)
	eventsHandler := handlers.NewEventsHandler(eventsService)

	router := api.NewRouter(endpointsHandler, eventsHandler)

	const addr = ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
