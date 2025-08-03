package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/NickChunglolz/rate-limiter/internal/queries"
)

// QueryHandler handles queries in the CQRS pattern
type QueryHandler interface {
	Handle(ctx context.Context, query queries.Query) (interface{}, error)
}

// ReadModel defines the interface for read model storage
type ReadModel interface {
	GetRateLimitStatus(ctx context.Context, clientID, resource string) (*queries.RateLimitStatus, error)
	GetRateLimitHistory(ctx context.Context, clientID, resource string, startTime, endTime time.Time, limit, offset int) (*queries.RateLimitHistory, error)
	GetClientStats(ctx context.Context, clientID string, startTime, endTime time.Time) (*queries.ClientStats, error)
	UpdateFromEvent(ctx context.Context, event interface{}) error
}

// RateLimitQueryHandler handles rate limiting queries
type RateLimitQueryHandler struct {
	readModel      ReadModel
	ruleRepository RuleRepository
}

// NewRateLimitQueryHandler creates a new query handler
func NewRateLimitQueryHandler(readModel ReadModel, ruleRepository RuleRepository) *RateLimitQueryHandler {
	return &RateLimitQueryHandler{
		readModel:      readModel,
		ruleRepository: ruleRepository,
	}
}

// Handle processes different types of queries
func (h *RateLimitQueryHandler) Handle(ctx context.Context, query queries.Query) (interface{}, error) {
	switch q := query.(type) {
	case *queries.GetRateLimitStatusQuery:
		return h.handleGetRateLimitStatus(ctx, q)
	case *queries.GetRateLimitHistoryQuery:
		return h.handleGetRateLimitHistory(ctx, q)
	case *queries.GetActiveRulesQuery:
		return h.handleGetActiveRules(ctx, q)
	case *queries.GetClientStatsQuery:
		return h.handleGetClientStats(ctx, q)
	default:
		return nil, fmt.Errorf("unknown query type: %T", query)
	}
}

// handleGetRateLimitStatus retrieves current rate limit status
func (h *RateLimitQueryHandler) handleGetRateLimitStatus(ctx context.Context, query *queries.GetRateLimitStatusQuery) (*queries.RateLimitStatus, error) {
	status, err := h.readModel.GetRateLimitStatus(ctx, query.ClientID, query.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit status: %w", err)
	}
	
	return status, nil
}

// handleGetRateLimitHistory retrieves rate limit history
func (h *RateLimitQueryHandler) handleGetRateLimitHistory(ctx context.Context, query *queries.GetRateLimitHistoryQuery) (*queries.RateLimitHistory, error) {
	history, err := h.readModel.GetRateLimitHistory(ctx, query.ClientID, query.Resource, query.StartTime, query.EndTime, query.Limit, query.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit history: %w", err)
	}
	
	return history, nil
}

// handleGetActiveRules retrieves active rate limit rules
func (h *RateLimitQueryHandler) handleGetActiveRules(ctx context.Context, query *queries.GetActiveRulesQuery) ([]interface{}, error) {
	var rules []interface{}
	
	if query.Resource != "" {
		resourceRules, err := h.ruleRepository.GetByResource(ctx, query.Resource)
		if err != nil {
			return nil, fmt.Errorf("failed to get rules for resource: %w", err)
		}
		for _, rule := range resourceRules {
			rules = append(rules, rule)
		}
	} else {
		// Return all rules - this would need a method to get all rules
		// For now, return empty slice
		rules = make([]interface{}, 0)
	}
	
	return rules, nil
}

// handleGetClientStats retrieves client statistics
func (h *RateLimitQueryHandler) handleGetClientStats(ctx context.Context, query *queries.GetClientStatsQuery) (*queries.ClientStats, error) {
	stats, err := h.readModel.GetClientStats(ctx, query.ClientID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get client stats: %w", err)
	}
	
	return stats, nil
}
