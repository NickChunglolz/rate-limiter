package queries

import (
	"time"
)

// Query represents a query in the CQRS pattern
type Query interface {
	QueryID() string
	QueryType() string
	Timestamp() time.Time
}

// BaseQuery provides common query functionality
type BaseQuery struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Time time.Time `json:"timestamp"`
}

func (q BaseQuery) QueryID() string     { return q.ID }
func (q BaseQuery) QueryType() string   { return q.Type }
func (q BaseQuery) Timestamp() time.Time { return q.Time }

// GetRateLimitStatusQuery - Query for getting current rate limit status
type GetRateLimitStatusQuery struct {
	BaseQuery
	ClientID string `json:"client_id"`
	Resource string `json:"resource"`
}

// GetRateLimitHistoryQuery - Query for getting rate limit history
type GetRateLimitHistoryQuery struct {
	BaseQuery
	ClientID  string    `json:"client_id"`
	Resource  string    `json:"resource"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// GetActiveRulesQuery - Query for getting active rate limit rules
type GetActiveRulesQuery struct {
	BaseQuery
	Resource string `json:"resource,omitempty"`
}

// GetClientStatsQuery - Query for getting client statistics
type GetClientStatsQuery struct {
	BaseQuery
	ClientID  string    `json:"client_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// RateLimitStatus - Response for rate limit status queries
type RateLimitStatus struct {
	ClientID         string    `json:"client_id"`
	Resource         string    `json:"resource"`
	IsAllowed        bool      `json:"is_allowed"`
	RequestCount     int       `json:"request_count"`
	Limit            int       `json:"limit"`
	RemainingQuota   int       `json:"remaining_quota"`
	WindowStart      time.Time `json:"window_start"`
	WindowEnd        time.Time `json:"window_end"`
	ResetTime        time.Time `json:"reset_time"`
	IsBlocked        bool      `json:"is_blocked"`
	BlockedUntil     time.Time `json:"blocked_until,omitempty"`
	RetryAfter       int       `json:"retry_after,omitempty"`
}

// RateLimitHistory - Response for rate limit history queries
type RateLimitHistory struct {
	Events     []RateLimitEvent `json:"events"`
	TotalCount int              `json:"total_count"`
	HasMore    bool             `json:"has_more"`
}

// RateLimitEvent - Individual rate limit event in history
type RateLimitEvent struct {
	EventID     string            `json:"event_id"`
	EventType   string            `json:"event_type"`
	ClientID    string            `json:"client_id"`
	Resource    string            `json:"resource"`
	Timestamp   time.Time         `json:"timestamp"`
	RequestCount int              `json:"request_count,omitempty"`
	Limit       int              `json:"limit,omitempty"`
	IsBlocked   bool             `json:"is_blocked"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ClientStats - Response for client statistics queries
type ClientStats struct {
	ClientID          string                `json:"client_id"`
	TotalRequests     int                   `json:"total_requests"`
	BlockedRequests   int                   `json:"blocked_requests"`
	AllowedRequests   int                   `json:"allowed_requests"`
	ResourceStats     []ResourceStats       `json:"resource_stats"`
	TimeSeriesData    []TimeSeriesDataPoint `json:"time_series_data"`
}

// ResourceStats - Statistics for a specific resource
type ResourceStats struct {
	Resource        string  `json:"resource"`
	TotalRequests   int     `json:"total_requests"`
	BlockedRequests int     `json:"blocked_requests"`
	AllowedRequests int     `json:"allowed_requests"`
	BlockedRate     float64 `json:"blocked_rate"`
}

// TimeSeriesDataPoint - Time series data point for statistics
type TimeSeriesDataPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalRequests   int       `json:"total_requests"`
	BlockedRequests int       `json:"blocked_requests"`
	AllowedRequests int       `json:"allowed_requests"`
}
