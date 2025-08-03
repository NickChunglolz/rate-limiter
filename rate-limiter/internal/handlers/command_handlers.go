package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/NickChunglolz/rate-limiter/internal/commands"
	"github.com/NickChunglolz/rate-limiter/internal/domain"
)

// CommandHandler handles commands in the CQRS pattern
type CommandHandler interface {
	Handle(ctx context.Context, cmd commands.Command) error
}

// EventStore defines the interface for event storage
type EventStore interface {
	SaveEvents(ctx context.Context, aggregateID string, events []domain.Event, expectedVersion int) error
	GetEvents(ctx context.Context, aggregateID string) ([]domain.Event, error)
}

// RuleRepository defines the interface for rule storage
type RuleRepository interface {
	Save(ctx context.Context, rule domain.RateLimitRule) error
	GetByResource(ctx context.Context, resource string) ([]domain.RateLimitRule, error)
	GetByID(ctx context.Context, id string) (*domain.RateLimitRule, error)
	Update(ctx context.Context, rule domain.RateLimitRule) error
	Delete(ctx context.Context, id string) error
}

// RateLimitCommandHandler handles rate limiting commands
type RateLimitCommandHandler struct {
	eventStore     EventStore
	ruleRepository RuleRepository
}

// NewRateLimitCommandHandler creates a new command handler
func NewRateLimitCommandHandler(eventStore EventStore, ruleRepository RuleRepository) *RateLimitCommandHandler {
	return &RateLimitCommandHandler{
		eventStore:     eventStore,
		ruleRepository: ruleRepository,
	}
}

// Handle processes different types of commands
func (h *RateLimitCommandHandler) Handle(ctx context.Context, cmd commands.Command) error {
	switch c := cmd.(type) {
	case *commands.ApplyRateLimitCommand:
		return h.handleApplyRateLimit(ctx, c)
	case *commands.CreateRuleCommand:
		return h.handleCreateRule(ctx, c)
	case *commands.UpdateRuleCommand:
		return h.handleUpdateRule(ctx, c)
	case *commands.ResetRateLimitCommand:
		return h.handleResetRateLimit(ctx, c)
	default:
		return fmt.Errorf("unknown command type: %T", cmd)
	}
}

// handleApplyRateLimit processes rate limit application
func (h *RateLimitCommandHandler) handleApplyRateLimit(ctx context.Context, cmd *commands.ApplyRateLimitCommand) error {
	aggregateID := cmd.ClientID + ":" + cmd.Resource
	
	// Get existing events for the aggregate
	events, err := h.eventStore.GetEvents(ctx, aggregateID)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}
	
	// Reconstruct aggregate from events
	aggregate := domain.NewRateLimitAggregate(cmd.ClientID, cmd.Resource)
	for _, event := range events {
		aggregate.ApplyEvent(event)
	}
	
	// Get applicable rules
	rules, err := h.ruleRepository.GetByResource(ctx, cmd.Resource)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}
	
	if len(rules) == 0 {
		return fmt.Errorf("no rules found for resource: %s", cmd.Resource)
	}
	
	// Apply the most restrictive rule
	rule := rules[0] // For simplicity, using first rule
	
	var newEvents []domain.Event
	
	if aggregate.CanMakeRequest(rule) {
		// Allow the request and update state
		event := &domain.RateLimitAppliedEvent{
			BaseEvent: domain.BaseEvent{
				ID:      fmt.Sprintf("applied-%d", time.Now().UnixNano()),
				Type:    "RateLimitApplied",
				Time:    time.Now(),
				AggrID:  aggregateID,
				Version: aggregate.Version + 1,
			},
			ClientID:       cmd.ClientID,
			Resource:       cmd.Resource,
			WindowStart:    time.Now().Truncate(rule.Window),
			WindowEnd:      time.Now().Truncate(rule.Window).Add(rule.Window),
			RequestCount:   aggregate.State.RequestCount + 1,
			Limit:          rule.Limit,
			RemainingQuota: rule.Limit - (aggregate.State.RequestCount + 1),
		}
		newEvents = append(newEvents, event)
	} else {
		// Block the request
		event := &domain.RateLimitExceededEvent{
			BaseEvent: domain.BaseEvent{
				ID:      fmt.Sprintf("exceeded-%d", time.Now().UnixNano()),
				Type:    "RateLimitExceeded",
				Time:    time.Now(),
				AggrID:  aggregateID,
				Version: aggregate.Version + 1,
			},
			ClientID:     cmd.ClientID,
			Resource:     cmd.Resource,
			RequestCount: aggregate.State.RequestCount + 1,
			Limit:        rule.Limit,
			WindowStart:  aggregate.State.WindowStart,
			WindowEnd:    aggregate.State.WindowEnd,
			BlockedUntil: aggregate.State.WindowEnd,
		}
		newEvents = append(newEvents, event)
	}
	
	// Save events
	return h.eventStore.SaveEvents(ctx, aggregateID, newEvents, aggregate.Version)
}

// handleCreateRule creates a new rate limit rule
func (h *RateLimitCommandHandler) handleCreateRule(ctx context.Context, cmd *commands.CreateRuleCommand) error {
	rule := domain.RateLimitRule{
		ID:        fmt.Sprintf("rule-%d", time.Now().UnixNano()),
		Resource:  cmd.Resource,
		Limit:     cmd.Limit,
		Window:    cmd.Window,
		Algorithm: domain.Algorithm(cmd.Algorithm),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return h.ruleRepository.Save(ctx, rule)
}

// handleUpdateRule updates an existing rate limit rule
func (h *RateLimitCommandHandler) handleUpdateRule(ctx context.Context, cmd *commands.UpdateRuleCommand) error {
	rule, err := h.ruleRepository.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}
	
	rule.Resource = cmd.Resource
	rule.Limit = cmd.Limit
	rule.Window = cmd.Window
	rule.Algorithm = domain.Algorithm(cmd.Algorithm)
	rule.UpdatedAt = time.Now()
	
	return h.ruleRepository.Update(ctx, *rule)
}

// handleResetRateLimit resets rate limit for a client/resource
func (h *RateLimitCommandHandler) handleResetRateLimit(ctx context.Context, cmd *commands.ResetRateLimitCommand) error {
	aggregateID := cmd.ClientID + ":" + cmd.Resource
	
	event := &domain.RateLimitWindowResetEvent{
		BaseEvent: domain.BaseEvent{
			ID:      fmt.Sprintf("reset-%d", time.Now().UnixNano()),
			Type:    "RateLimitWindowReset",
			Time:    time.Now(),
			AggrID:  aggregateID,
			Version: 1,
		},
		ClientID:    cmd.ClientID,
		Resource:    cmd.Resource,
		WindowStart: time.Now(),
	}
	
	return h.eventStore.SaveEvents(ctx, aggregateID, []domain.Event{event}, 0)
}
