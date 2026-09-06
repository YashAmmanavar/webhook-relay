// Command server runs the HTTP API and the delivery worker in a single
// process — a deployment-specific packaging choice for hosts whose free
// tier only supports one always-on web service, not a separate
// continuously-running background worker (e.g. Render). cmd/api and
// cmd/worker remain as independent binaries for local dev and for any
// deployment that can run them as separate, independently-scalable
// services.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhookrelay/internal/api"
	"webhookrelay/internal/api/handlers"
	"webhookrelay/internal/api/middleware"
	"webhookrelay/internal/database"
	"webhookrelay/internal/delivery"
	"webhookrelay/internal/endpoints"
	"webhookrelay/internal/events"
	"webhookrelay/internal/queue"
	"webhookrelay/internal/worker"
)

// devAPIKey is used only when WEBHOOK_RELAY_API_KEY isn't set. Fine for
// localhost testing — never use this for anything reachable beyond that.
const devAPIKey = "sk_dev_local_only_insecure_key"

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

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	router := api.NewRouter(endpointsHandler, eventsHandler)
	handler := middleware.CORS(allowedOrigin, middleware.RateLimit(middleware.Auth(apiKey, middleware.BodyLimit(router))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// The worker loop runs in the background; the HTTP server in the
	// foreground is what a PaaS's health check / keep-alive ping sees, and
	// keeping this one process alive keeps the worker goroutine alive too.
	go worker.Run(ctx, eventsService, eventQueue)

	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("combined server (api+worker) listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
