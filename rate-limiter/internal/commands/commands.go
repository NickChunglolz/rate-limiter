package commands

import (
	"time"
)

// Command represents a command in the CQRS pattern
type Command interface {
	CommandID() string
	CommandType() string
	Timestamp() time.Time
}

// BaseCommand provides common command functionality
type BaseCommand struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Time time.Time `json:"timestamp"`
}

func (c BaseCommand) CommandID() string     { return c.ID }
func (c BaseCommand) CommandType() string   { return c.Type }
func (c BaseCommand) Timestamp() time.Time  { return c.Time }

// CheckRateLimitCommand - Query command for checking rate limits
type CheckRateLimitCommand struct {
	BaseCommand
	ClientID   string            `json:"client_id"`
	Resource   string            `json:"resource"`
	IPAddress  string            `json:"ip_address"`
	UserAgent  string            `json:"user_agent"`
	Metadata   map[string]string `json:"metadata"`
}

// ApplyRateLimitCommand - Command for applying/updating rate limits
type ApplyRateLimitCommand struct {
	BaseCommand
	ClientID     string    `json:"client_id"`
	Resource     string    `json:"resource"`
	RequestedAt  time.Time `json:"requested_at"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
}

// CreateRuleCommand - Command for creating rate limit rules
type CreateRuleCommand struct {
	BaseCommand
	Resource  string        `json:"resource"`
	Limit     int           `json:"limit"`
	Window    time.Duration `json:"window"`
	Algorithm string        `json:"algorithm"`
}

// UpdateRuleCommand - Command for updating rate limit rules
type UpdateRuleCommand struct {
	BaseCommand
	RuleID    string        `json:"rule_id"`
	Resource  string        `json:"resource"`
	Limit     int           `json:"limit"`
	Window    time.Duration `json:"window"`
	Algorithm string        `json:"algorithm"`
}

// ResetRateLimitCommand - Command for resetting rate limits
type ResetRateLimitCommand struct {
	BaseCommand
	ClientID string `json:"client_id"`
	Resource string `json:"resource"`
}
