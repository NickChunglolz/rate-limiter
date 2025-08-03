package domain

import (
	"time"
)

// RateLimitRule defines the rate limiting configuration
type RateLimitRule struct {
	ID           string        `json:"id"`
	Resource     string        `json:"resource"`
	Limit        int           `json:"limit"`
	Window       time.Duration `json:"window"`
	Algorithm    Algorithm     `json:"algorithm"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Algorithm represents different rate limiting algorithms
type Algorithm string

const (
	TokenBucket     Algorithm = "token_bucket"
	SlidingWindow   Algorithm = "sliding_window"
	FixedWindow     Algorithm = "fixed_window"
	LeakyBucket     Algorithm = "leaky_bucket"
)

// RateLimitState represents the current state of rate limiting for a client
type RateLimitState struct {
	ClientID       string    `json:"client_id"`
	Resource       string    `json:"resource"`
	RequestCount   int       `json:"request_count"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	RemainingQuota int       `json:"remaining_quota"`
	LastRequestAt  time.Time `json:"last_request_at"`
	IsBlocked      bool      `json:"is_blocked"`
	BlockedUntil   time.Time `json:"blocked_until"`
	Version        int       `json:"version"`
}

// RateLimitAggregate represents the domain aggregate
type RateLimitAggregate struct {
	ID       string         `json:"id"`
	State    RateLimitState `json:"state"`
	Rules    []RateLimitRule `json:"rules"`
	Events   []Event        `json:"events"`
	Version  int           `json:"version"`
}

// NewRateLimitAggregate creates a new rate limit aggregate
func NewRateLimitAggregate(clientID, resource string) *RateLimitAggregate {
	return &RateLimitAggregate{
		ID: clientID + ":" + resource,
		State: RateLimitState{
			ClientID:       clientID,
			Resource:       resource,
			RequestCount:   0,
			RemainingQuota: 0,
			IsBlocked:      false,
			Version:        0,
		},
		Events:  make([]Event, 0),
		Version: 0,
	}
}

// ApplyEvent applies an event to the aggregate
func (a *RateLimitAggregate) ApplyEvent(event Event) {
	switch e := event.(type) {
	case *RateLimitAppliedEvent:
		a.State.RequestCount = e.RequestCount
		a.State.WindowStart = e.WindowStart
		a.State.WindowEnd = e.WindowEnd
		a.State.RemainingQuota = e.RemainingQuota
		a.State.LastRequestAt = time.Now()
	case *RateLimitExceededEvent:
		a.State.IsBlocked = true
		a.State.BlockedUntil = e.BlockedUntil
		a.State.RequestCount = e.RequestCount
	case *RateLimitWindowResetEvent:
		a.State.RequestCount = 0
		a.State.WindowStart = e.WindowStart
		a.State.IsBlocked = false
		a.State.BlockedUntil = time.Time{}
	}
	a.Version++
	a.Events = append(a.Events, event)
}

// CanMakeRequest checks if a request can be made based on current state
func (a *RateLimitAggregate) CanMakeRequest(rule RateLimitRule) bool {
	now := time.Now()
	
	// Check if currently blocked
	if a.State.IsBlocked && now.Before(a.State.BlockedUntil) {
		return false
	}
	
	// Check if window has expired
	if now.After(a.State.WindowEnd) {
		return true // New window, allow request
	}
	
	// Check if within quota
	return a.State.RemainingQuota > 0
}
