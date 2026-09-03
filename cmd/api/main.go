package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"webhookrelay/internal/api"
	"webhookrelay/internal/api/handlers"
	"webhookrelay/internal/api/middleware"
	"webhookrelay/internal/database"
	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
)

// devAPIKey is used only when WEBHOOK_RELAY_API_KEY isn't set. Fine for
// localhost testing — never use this for anything reachable beyond that.
const devAPIKey = "sk_dev_local_only_insecure_key"

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

	apiKey := os.Getenv("WEBHOOK_RELAY_API_KEY")
	if apiKey == "" {
		apiKey = devAPIKey
		log.Println("WARNING: WEBHOOK_RELAY_API_KEY not set, using an insecure default dev key — do not expose this server beyond localhost")
	}

	router := api.NewRouter(endpointsHandler, eventsHandler)
	handler := middleware.CORS(middleware.RateLimit(middleware.Auth(apiKey, middleware.BodyLimit(router))))

	const addr = ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
