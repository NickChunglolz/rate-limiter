package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/NickChunglolz/rate-limiter/internal/api"
	"github.com/NickChunglolz/rate-limiter/internal/handlers"
	"github.com/NickChunglolz/rate-limiter/internal/infrastructure"
)

func main() {
	// Initialize infrastructure components
	eventStore := infrastructure.NewInMemoryEventStore()
	ruleRepository := infrastructure.NewInMemoryRuleRepository()
	readModel := infrastructure.NewInMemoryReadModel()
	eventBus := infrastructure.NewEventBus()
	
	// Initialize CQRS handlers
	commandHandler := handlers.NewRateLimitCommandHandler(eventStore, ruleRepository)
	queryHandler := handlers.NewRateLimitQueryHandler(readModel, ruleRepository)
	
	// Initialize service and HTTP handler
	service := api.NewRateLimiterService(commandHandler, queryHandler)
	httpHandler := api.NewHTTPHandler(service)
	
	// Setup event projection to read model
	go setupEventProjection(eventBus, readModel)
	
	// Create some default rules for demonstration
	setupDefaultRules(service)
	
	// Setup HTTP routes
	mux := httpHandler.SetupRoutes()
	
	// Add middleware for logging and CORS
	handler := loggingMiddleware(corsMiddleware(mux))
	
	// Start server
	addr := ":8080"
	fmt.Printf("Rate Limiter server starting on %s\n", addr)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /api/v1/ratelimit/check")
	fmt.Println("  GET  /api/v1/ratelimit/status")
	fmt.Println("  GET  /api/v1/ratelimit/history")
	fmt.Println("  GET  /api/v1/ratelimit/stats")
	fmt.Println("  POST /api/v1/ratelimit/rules")
	fmt.Println("  POST /api/v1/ratelimit/reset")
	
	log.Fatal(http.ListenAndServe(addr, handler))
}

// setupEventProjection sets up event projection from command side to query side
func setupEventProjection(eventBus *infrastructure.EventBus, readModel *infrastructure.InMemoryReadModel) {
	// Subscribe to all events
	events := eventBus.Subscribe("*")
	
	for event := range events {
		ctx := context.Background()
		if err := readModel.UpdateFromEvent(ctx, event); err != nil {
			log.Printf("Error updating read model from event: %v", err)
		}
	}
}

// setupDefaultRules creates some default rate limiting rules
func setupDefaultRules(service *api.RateLimiterService) {
	ctx := context.Background()
	
	// API rate limit: 100 requests per minute
	err := service.CreateRule(ctx, "api", 100, time.Minute, "sliding_window")
	if err != nil {
		log.Printf("Error creating API rule: %v", err)
	}
	
	// Login rate limit: 5 attempts per 15 minutes
	err = service.CreateRule(ctx, "login", 5, 15*time.Minute, "fixed_window")
	if err != nil {
		log.Printf("Error creating login rule: %v", err)
	}
	
	// Upload rate limit: 10 uploads per hour
	err = service.CreateRule(ctx, "upload", 10, time.Hour, "sliding_window")
	if err != nil {
		log.Printf("Error creating upload rule: %v", err)
	}
	
	fmt.Println("Default rate limiting rules created:")
	fmt.Println("  - api: 100 requests/minute (sliding window)")
	fmt.Println("  - login: 5 attempts/15 minutes (fixed window)")
	fmt.Println("  - upload: 10 uploads/hour (sliding window)")
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
