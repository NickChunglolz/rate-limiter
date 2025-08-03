package domain

import (
	"time"
)

// Event represents a domain event in the system
type Event interface {
	EventID() string
	EventType() string
	Timestamp() time.Time
	AggregateID() string
}

// BaseEvent provides common event functionality
type BaseEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Time        time.Time `json:"timestamp"`
	AggrID      string    `json:"aggregate_id"`
	Version     int       `json:"version"`
}

func (e BaseEvent) EventID() string     { return e.ID }
func (e BaseEvent) EventType() string   { return e.Type }
func (e BaseEvent) Timestamp() time.Time { return e.Time }
func (e BaseEvent) AggregateID() string { return e.AggrID }

// RateLimitRequestedEvent - Command side event
type RateLimitRequestedEvent struct {
	BaseEvent
	ClientID    string    `json:"client_id"`
	Resource    string    `json:"resource"`
	RequestedAt time.Time `json:"requested_at"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
}

// RateLimitAppliedEvent - Write side event
type RateLimitAppliedEvent struct {
	BaseEvent
	ClientID       string    `json:"client_id"`
	Resource       string    `json:"resource"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	RequestCount   int       `json:"request_count"`
	Limit          int       `json:"limit"`
	RemainingQuota int       `json:"remaining_quota"`
}

// RateLimitExceededEvent - Command side event
type RateLimitExceededEvent struct {
	BaseEvent
	ClientID     string    `json:"client_id"`
	Resource     string    `json:"resource"`
	RequestCount int       `json:"request_count"`
	Limit        int       `json:"limit"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
	BlockedUntil time.Time `json:"blocked_until"`
}

// RateLimitWindowResetEvent - Query side optimization event
type RateLimitWindowResetEvent struct {
	BaseEvent
	ClientID    string    `json:"client_id"`
	Resource    string    `json:"resource"`
	WindowStart time.Time `json:"window_start"`
}
