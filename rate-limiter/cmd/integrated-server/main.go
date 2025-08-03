package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	rateLimiterAPI "github.com/NickChunglolz/rate-limiter/internal/api"
	rateLimiterHandlers "github.com/NickChunglolz/rate-limiter/internal/handlers"
	rateLimiterInfra "github.com/NickChunglolz/rate-limiter/internal/infrastructure"
	"github.com/NickChunglolz/rate-limiter/internal/integration"
	ruleEngine "github.com/NickChunglolz/rule-engine/engine"
	ruleDomain "github.com/NickChunglolz/rule-engine/domain"
	ruleInfra "github.com/NickChunglolz/rule-engine/infrastructure"
)

func main() {
	// Initialize Rate Limiter components
	eventStore := rateLimiterInfra.NewInMemoryEventStore()
	rateLimitRuleRepository := rateLimiterInfra.NewInMemoryRuleRepository()
	readModel := rateLimiterInfra.NewInMemoryReadModel()
	eventBus := rateLimiterInfra.NewEventBus()

	commandHandler := rateLimiterHandlers.NewRateLimitCommandHandler(eventStore, rateLimitRuleRepository)
	queryHandler := rateLimiterHandlers.NewRateLimitQueryHandler(readModel, rateLimitRuleRepository)
	rateLimiterService := rateLimiterAPI.NewRateLimiterService(commandHandler, queryHandler)

	// Initialize Rule Engine components
	ruleRepository := ruleInfra.NewInMemoryRuleRepository()
	eventPublisher := ruleInfra.NewSimpleEventPublisher()
	ruleEngineService := ruleEngine.NewRuleEngine(ruleRepository, eventPublisher)

	// Initialize Integrated Service
	integratedService := integration.NewIntegratedRateLimiterService(rateLimiterService, ruleEngineService)

	// Setup event projection
	go setupEventProjection(eventBus, readModel)

	// Setup default rules and rate limits
	setupDefaultConfiguration(rateLimiterService, ruleEngineService)

	// Setup HTTP server with integrated endpoints
	mux := setupIntegratedRoutes(integratedService)
	handler := loggingMiddleware(corsMiddleware(mux))

	// Start server
	addr := ":8081"
	fmt.Printf("Integrated Rate Limiter with Rule Engine server starting on %s\n", addr)
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /health         - Health check")
	fmt.Println("  POST /api/v1/check   - Integrated request check")
	fmt.Println("  POST /api/v1/security/block-ips - Block IP addresses")
	fmt.Println("  POST /api/v1/security/rate-limit-resources - Rate limit resources")

	log.Fatal(http.ListenAndServe(addr, handler))
}

func setupEventProjection(eventBus *rateLimiterInfra.EventBus, readModel *rateLimiterInfra.InMemoryReadModel) {
	events := eventBus.Subscribe("*")
	for event := range events {
		ctx := context.Background()
		if err := readModel.UpdateFromEvent(ctx, event); err != nil {
			log.Printf("Error updating read model from event: %v", err)
		}
	}
}

func setupDefaultConfiguration(rateLimiterService *rateLimiterAPI.RateLimiterService, ruleEngineService *ruleEngine.RuleEngine) {
	ctx := context.Background()

	// Create default rate limiting rules
	rateLimiterService.CreateRule(ctx, "api", 100, time.Minute, "sliding_window")
	rateLimiterService.CreateRule(ctx, "login", 5, 15*time.Minute, "fixed_window")
	rateLimiterService.CreateRule(ctx, "upload", 10, time.Hour, "sliding_window")

	// Create default security rules

	// 1. Block suspicious user agents
	suspiciousUserAgentRule := ruleDomain.Rule{
		ID:          "block-suspicious-agents",
		Name:        "Block Suspicious User Agents",
		Type:        ruleDomain.BlacklistRule,
		Description: "Block requests from suspicious user agents",
		Priority:    200,
		Enabled:     true,
		Conditions: []ruleDomain.RuleCondition{
			{
				Field:    "user_agent",
				Operator: "contains",
				Value:    "bot",
			},
		},
		Actions: []ruleDomain.RuleAction{
			{
				Type: "deny",
				Parameters: map[string]interface{}{
					"reason": "suspicious user agent",
				},
			},
		},
		Tags: []string{"security", "user-agent"},
	}
	ruleEngineService.CreateRule(ctx, suspiciousUserAgentRule)

	// 2. Aggressive rate limiting for login attempts
	aggressiveLoginRule := ruleDomain.Rule{
		ID:          "aggressive-login-rate-limit",
		Name:        "Aggressive Login Rate Limiting",
		Type:        ruleDomain.RateLimitRule,
		Description: "Apply stricter rate limiting for login endpoints",
		Priority:    150,
		Enabled:     true,
		Conditions: []ruleDomain.RuleCondition{
			{
				Field:    "resource",
				Operator: "equals",
				Value:    "login",
			},
		},
		Actions: []ruleDomain.RuleAction{
			{
				Type: "rate_limit",
				Parameters: map[string]interface{}{
					"limit":     3,
					"window":    "5m",
					"algorithm": "fixed_window",
				},
			},
		},
		Tags: []string{"security", "login"},
	}
	ruleEngineService.CreateRule(ctx, aggressiveLoginRule)

	// 3. Whitelist internal IPs
	whitelistRule := ruleDomain.Rule{
		ID:          "whitelist-internal-ips",
		Name:        "Whitelist Internal IPs",
		Type:        ruleDomain.WhitelistRule,
		Description: "Allow all requests from internal IP ranges",
		Priority:    300,
		Enabled:     true,
		Conditions: []ruleDomain.RuleCondition{
			{
				Field:    "ip_address",
				Operator: "starts_with",
				Value:    "192.168.",
			},
		},
		Actions: []ruleDomain.RuleAction{
			{
				Type: "allow",
				Parameters: map[string]interface{}{
					"reason": "internal IP",
				},
			},
		},
		Tags: []string{"security", "whitelist"},
	}
	ruleEngineService.CreateRule(ctx, whitelistRule)

	fmt.Println("Default configuration created:")
	fmt.Println("Rate Limiting Rules:")
	fmt.Println("  - api: 100 requests/minute")
	fmt.Println("  - login: 5 attempts/15 minutes")
	fmt.Println("  - upload: 10 uploads/hour")
	fmt.Println("Security Rules:")
	fmt.Println("  - Block suspicious user agents")
	fmt.Println("  - Aggressive login rate limiting (3/5min)")
	fmt.Println("  - Whitelist internal IPs (192.168.x.x)")
}

func setupIntegratedRoutes(service *integration.IntegratedRateLimiterService) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "integrated-rate-limiter",
		})
	})

	// Integrated request check endpoint
	mux.HandleFunc("/api/v1/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ClientID    string                 `json:"client_id"`
			Resource    string                 `json:"resource"`
			IPAddress   string                 `json:"ip_address,omitempty"`
			UserAgent   string                 `json:"user_agent,omitempty"`
			Metadata    map[string]string      `json:"metadata,omitempty"`
			RequestData map[string]interface{} `json:"request_data,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.ClientID == "" || req.Resource == "" {
			http.Error(w, "client_id and resource are required", http.StatusBadRequest)
			return
		}

		// Use request IP and User-Agent if not provided
		if req.IPAddress == "" {
			req.IPAddress = r.RemoteAddr
		}
		if req.UserAgent == "" {
			req.UserAgent = r.UserAgent()
		}
		if req.Metadata == nil {
			req.Metadata = make(map[string]string)
		}
		if req.RequestData == nil {
			req.RequestData = make(map[string]interface{})
		}

		result, err := service.CheckRequestWithRules(
			r.Context(),
			req.ClientID,
			req.Resource,
			req.IPAddress,
			req.UserAgent,
			req.Metadata,
			req.RequestData,
		)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		statusCode := http.StatusOK
		if !result.Allowed {
			statusCode = http.StatusTooManyRequests
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	})

	// Block IPs endpoint
	mux.HandleFunc("/api/v1/security/block-ips", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			IPAddresses []string `json:"ip_addresses"`
			Reason      string   `json:"reason,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.IPAddresses) == 0 {
			http.Error(w, "ip_addresses is required", http.StatusBadRequest)
			return
		}

		parameters := map[string]interface{}{
			"reason": req.Reason,
		}
		if req.Reason == "" {
			parameters["reason"] = "blocked by admin"
		}

		err := service.CreateIPBasedRule(r.Context(), req.IPAddresses, "block", parameters)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})

	// Rate limit resources endpoint
	mux.HandleFunc("/api/v1/security/rate-limit-resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Resources []string `json:"resources"`
			Limit     int      `json:"limit"`
			Window    string   `json:"window"`
			Algorithm string   `json:"algorithm,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.Resources) == 0 || req.Limit <= 0 || req.Window == "" {
			http.Error(w, "resources, limit, and window are required", http.StatusBadRequest)
			return
		}

		window, err := time.ParseDuration(req.Window)
		if err != nil {
			http.Error(w, "Invalid window format", http.StatusBadRequest)
			return
		}

		if req.Algorithm == "" {
			req.Algorithm = "sliding_window"
		}

		err = service.CreateResourceBasedRule(r.Context(), req.Resources, req.Limit, window, req.Algorithm)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})

	return mux
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
	})
}

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

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
