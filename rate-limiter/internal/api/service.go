package api

import (
	"context"
	"fmt"
	"time"

	"github.com/NickChunglolz/rate-limiter/internal/commands"
	"github.com/NickChunglolz/rate-limiter/internal/handlers"
	"github.com/NickChunglolz/rate-limiter/internal/queries"
)

// RateLimiterService provides the main API for the rate limiter
type RateLimiterService struct {
	commandHandler handlers.CommandHandler
	queryHandler   handlers.QueryHandler
}

// NewRateLimiterService creates a new rate limiter service
func NewRateLimiterService(commandHandler handlers.CommandHandler, queryHandler handlers.QueryHandler) *RateLimiterService {
	return &RateLimiterService{
		commandHandler: commandHandler,
		queryHandler:   queryHandler,
	}
}

// CheckRateLimit checks if a request is allowed and applies the rate limit
func (s *RateLimiterService) CheckRateLimit(ctx context.Context, clientID, resource, ipAddress, userAgent string) (*queries.RateLimitStatus, error) {
	// First, check current status
	statusQuery := &queries.GetRateLimitStatusQuery{
		BaseQuery: queries.BaseQuery{
			ID:   fmt.Sprintf("status-%d", time.Now().UnixNano()),
			Type: "GetRateLimitStatus",
			Time: time.Now(),
		},
		ClientID: clientID,
		Resource: resource,
	}
	
	result, err := s.queryHandler.Handle(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit status: %w", err)
	}
	
	currentStatus := result.(*queries.RateLimitStatus)
	
	// If already blocked, return current status
	if currentStatus.IsBlocked && time.Now().Before(currentStatus.BlockedUntil) {
		return currentStatus, nil
	}
	
	// Apply rate limit (this will update the state)
	applyCmd := &commands.ApplyRateLimitCommand{
		BaseCommand: commands.BaseCommand{
			ID:   fmt.Sprintf("apply-%d", time.Now().UnixNano()),
			Type: "ApplyRateLimit",
			Time: time.Now(),
		},
		ClientID:    clientID,
		Resource:    resource,
		RequestedAt: time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}
	
	err = s.commandHandler.Handle(ctx, applyCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to apply rate limit: %w", err)
	}
	
	// Get updated status
	result, err = s.queryHandler.Handle(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated rate limit status: %w", err)
	}
	
	return result.(*queries.RateLimitStatus), nil
}

// GetRateLimitStatus gets the current rate limit status for a client/resource
func (s *RateLimiterService) GetRateLimitStatus(ctx context.Context, clientID, resource string) (*queries.RateLimitStatus, error) {
	query := &queries.GetRateLimitStatusQuery{
		BaseQuery: queries.BaseQuery{
			ID:   fmt.Sprintf("status-%d", time.Now().UnixNano()),
			Type: "GetRateLimitStatus",
			Time: time.Now(),
		},
		ClientID: clientID,
		Resource: resource,
	}
	
	result, err := s.queryHandler.Handle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit status: %w", err)
	}
	
	return result.(*queries.RateLimitStatus), nil
}

// GetRateLimitHistory gets the rate limit history for a client/resource
func (s *RateLimiterService) GetRateLimitHistory(ctx context.Context, clientID, resource string, startTime, endTime time.Time, limit, offset int) (*queries.RateLimitHistory, error) {
	query := &queries.GetRateLimitHistoryQuery{
		BaseQuery: queries.BaseQuery{
			ID:   fmt.Sprintf("history-%d", time.Now().UnixNano()),
			Type: "GetRateLimitHistory",
			Time: time.Now(),
		},
		ClientID:  clientID,
		Resource:  resource,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
	}
	
	result, err := s.queryHandler.Handle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit history: %w", err)
	}
	
	return result.(*queries.RateLimitHistory), nil
}

// GetClientStats gets statistics for a client
func (s *RateLimiterService) GetClientStats(ctx context.Context, clientID string, startTime, endTime time.Time) (*queries.ClientStats, error) {
	query := &queries.GetClientStatsQuery{
		BaseQuery: queries.BaseQuery{
			ID:   fmt.Sprintf("stats-%d", time.Now().UnixNano()),
			Type: "GetClientStats",
			Time: time.Now(),
		},
		ClientID:  clientID,
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	result, err := s.queryHandler.Handle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get client stats: %w", err)
	}
	
	return result.(*queries.ClientStats), nil
}

// CreateRule creates a new rate limit rule
func (s *RateLimiterService) CreateRule(ctx context.Context, resource string, limit int, window time.Duration, algorithm string) error {
	cmd := &commands.CreateRuleCommand{
		BaseCommand: commands.BaseCommand{
			ID:   fmt.Sprintf("create-rule-%d", time.Now().UnixNano()),
			Type: "CreateRule",
			Time: time.Now(),
		},
		Resource:  resource,
		Limit:     limit,
		Window:    window,
		Algorithm: algorithm,
	}
	
	return s.commandHandler.Handle(ctx, cmd)
}

// UpdateRule updates an existing rate limit rule
func (s *RateLimiterService) UpdateRule(ctx context.Context, ruleID, resource string, limit int, window time.Duration, algorithm string) error {
	cmd := &commands.UpdateRuleCommand{
		BaseCommand: commands.BaseCommand{
			ID:   fmt.Sprintf("update-rule-%d", time.Now().UnixNano()),
			Type: "UpdateRule",
			Time: time.Now(),
		},
		RuleID:    ruleID,
		Resource:  resource,
		Limit:     limit,
		Window:    window,
		Algorithm: algorithm,
	}
	
	return s.commandHandler.Handle(ctx, cmd)
}

// ResetRateLimit resets the rate limit for a client/resource
func (s *RateLimiterService) ResetRateLimit(ctx context.Context, clientID, resource string) error {
	cmd := &commands.ResetRateLimitCommand{
		BaseCommand: commands.BaseCommand{
			ID:   fmt.Sprintf("reset-%d", time.Now().UnixNano()),
			Type: "ResetRateLimit",
			Time: time.Now(),
		},
		ClientID: clientID,
		Resource: resource,
	}
	
	return s.commandHandler.Handle(ctx, cmd)
}
