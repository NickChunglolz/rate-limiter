package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandler provides HTTP endpoints for the rate limiter
type HTTPHandler struct {
	service *RateLimiterService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service *RateLimiterService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// CheckRateLimitHandler handles rate limit check requests
func (h *HTTPHandler) CheckRateLimitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ClientID  string `json:"client_id"`
		Resource  string `json:"resource"`
		IPAddress string `json:"ip_address,omitempty"`
		UserAgent string `json:"user_agent,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.ClientID == "" || req.Resource == "" {
		http.Error(w, "client_id and resource are required", http.StatusBadRequest)
		return
	}
	
	// Use IP from request if not provided
	if req.IPAddress == "" {
		req.IPAddress = r.RemoteAddr
	}
	
	// Use User-Agent from request if not provided
	if req.UserAgent == "" {
		req.UserAgent = r.UserAgent()
	}
	
	status, err := h.service.CheckRateLimit(r.Context(), req.ClientID, req.Resource, req.IPAddress, req.UserAgent)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Set appropriate status code
	statusCode := http.StatusOK
	if !status.IsAllowed {
		statusCode = http.StatusTooManyRequests
		
		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(status.RemainingQuota))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.ResetTime.Unix(), 10))
		
		if status.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(status.RetryAfter))
		}
	} else {
		// Set rate limit headers for successful requests too
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(status.RemainingQuota))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.ResetTime.Unix(), 10))
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// GetStatusHandler handles rate limit status requests
func (h *HTTPHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientID := r.URL.Query().Get("client_id")
	resource := r.URL.Query().Get("resource")
	
	if clientID == "" || resource == "" {
		http.Error(w, "client_id and resource are required", http.StatusBadRequest)
		return
	}
	
	status, err := h.service.GetRateLimitStatus(r.Context(), clientID, resource)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetHistoryHandler handles rate limit history requests
func (h *HTTPHandler) GetHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientID := r.URL.Query().Get("client_id")
	resource := r.URL.Query().Get("resource")
	
	if clientID == "" || resource == "" {
		http.Error(w, "client_id and resource are required", http.StatusBadRequest)
		return
	}
	
	// Parse optional parameters
	var startTime, endTime time.Time
	var err error
	
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "Invalid start_time format", http.StatusBadRequest)
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}
	
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "Invalid end_time format", http.StatusBadRequest)
			return
		}
	} else {
		endTime = time.Now()
	}
	
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}
	
	history, err := h.service.GetRateLimitHistory(r.Context(), clientID, resource, startTime, endTime, limit, offset)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetStatsHandler handles client statistics requests
func (h *HTTPHandler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}
	
	// Parse optional time range
	var startTime, endTime time.Time
	var err error
	
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "Invalid start_time format", http.StatusBadRequest)
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}
	
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "Invalid end_time format", http.StatusBadRequest)
			return
		}
	} else {
		endTime = time.Now()
	}
	
	stats, err := h.service.GetClientStats(r.Context(), clientID, startTime, endTime)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// CreateRuleHandler handles rule creation requests
func (h *HTTPHandler) CreateRuleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Resource  string `json:"resource"`
		Limit     int    `json:"limit"`
		Window    string `json:"window"`    // e.g., "1h", "5m", "30s"
		Algorithm string `json:"algorithm"` // e.g., "sliding_window", "fixed_window"
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Resource == "" || req.Limit <= 0 || req.Window == "" {
		http.Error(w, "resource, limit, and window are required", http.StatusBadRequest)
		return
	}
	
	window, err := time.ParseDuration(req.Window)
	if err != nil {
		http.Error(w, "Invalid window format", http.StatusBadRequest)
		return
	}
	
	if req.Algorithm == "" {
		req.Algorithm = "sliding_window" // default
	}
	
	err = h.service.CreateRule(r.Context(), req.Resource, req.Limit, window, req.Algorithm)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// ResetHandler handles rate limit reset requests
func (h *HTTPHandler) ResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ClientID string `json:"client_id"`
		Resource string `json:"resource"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.ClientID == "" || req.Resource == "" {
		http.Error(w, "client_id and resource are required", http.StatusBadRequest)
		return
	}
	
	err := h.service.ResetRateLimit(r.Context(), req.ClientID, req.Resource)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

// SetupRoutes sets up HTTP routes
func (h *HTTPHandler) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/api/v1/ratelimit/check", h.CheckRateLimitHandler)
	mux.HandleFunc("/api/v1/ratelimit/status", h.GetStatusHandler)
	mux.HandleFunc("/api/v1/ratelimit/history", h.GetHistoryHandler)
	mux.HandleFunc("/api/v1/ratelimit/stats", h.GetStatsHandler)
	mux.HandleFunc("/api/v1/ratelimit/rules", h.CreateRuleHandler)
	mux.HandleFunc("/api/v1/ratelimit/reset", h.ResetHandler)
	
	return mux
}
