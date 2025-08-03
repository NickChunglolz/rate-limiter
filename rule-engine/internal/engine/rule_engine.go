package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/NickChunglolz/rule-engine/internal/domain"
)

// RuleEngine provides rule evaluation capabilities
type RuleEngine struct {
	ruleRepository RuleRepository
	eventPublisher EventPublisher
}

// RuleRepository defines the interface for rule storage
type RuleRepository interface {
	GetActiveRules(ctx context.Context) ([]domain.Rule, error)
	GetRulesByType(ctx context.Context, ruleType domain.RuleType) ([]domain.Rule, error)
	GetRulesByTags(ctx context.Context, tags []string) ([]domain.Rule, error)
	SaveRule(ctx context.Context, rule domain.Rule) error
	UpdateRule(ctx context.Context, rule domain.Rule) error
	DeleteRule(ctx context.Context, ruleID string) error
	GetRuleByID(ctx context.Context, ruleID string) (*domain.Rule, error)
}

// EventPublisher defines the interface for publishing rule evaluation events
type EventPublisher interface {
	PublishRuleEvaluated(ctx context.Context, result domain.RuleEvaluationResult) error
	PublishRuleMatched(ctx context.Context, result domain.RuleEvaluationResult) error
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(ruleRepository RuleRepository, eventPublisher EventPublisher) *RuleEngine {
	return &RuleEngine{
		ruleRepository: ruleRepository,
		eventPublisher: eventPublisher,
	}
}

// EvaluateRules evaluates all active rules against the given context
func (e *RuleEngine) EvaluateRules(ctx context.Context, evalCtx domain.RuleEvaluationContext) ([]domain.RuleEvaluationResult, error) {
	// Get all active rules
	rules, err := e.ruleRepository.GetActiveRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rules: %w", err)
	}
	
	// Sort rules by priority (higher priority first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	
	var results []domain.RuleEvaluationResult
	
	// Evaluate each rule
	for _, rule := range rules {
		result := rule.EvaluateRule(evalCtx)
		results = append(results, result)
		
		// Publish evaluation event
		if err := e.eventPublisher.PublishRuleEvaluated(ctx, result); err != nil {
			// Log error but continue evaluation
			fmt.Printf("Error publishing rule evaluated event: %v\n", err)
		}
		
		// If rule matched, publish matched event
		if result.Matched {
			if err := e.eventPublisher.PublishRuleMatched(ctx, result); err != nil {
				// Log error but continue evaluation
				fmt.Printf("Error publishing rule matched event: %v\n", err)
			}
		}
	}
	
	return results, nil
}

// EvaluateRulesByType evaluates rules of a specific type
func (e *RuleEngine) EvaluateRulesByType(ctx context.Context, ruleType domain.RuleType, evalCtx domain.RuleEvaluationContext) ([]domain.RuleEvaluationResult, error) {
	rules, err := e.ruleRepository.GetRulesByType(ctx, ruleType)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules by type: %w", err)
	}
	
	// Sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	
	var results []domain.RuleEvaluationResult
	
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		
		result := rule.EvaluateRule(evalCtx)
		results = append(results, result)
		
		// Publish events
		if err := e.eventPublisher.PublishRuleEvaluated(ctx, result); err != nil {
			fmt.Printf("Error publishing rule evaluated event: %v\n", err)
		}
		
		if result.Matched {
			if err := e.eventPublisher.PublishRuleMatched(ctx, result); err != nil {
				fmt.Printf("Error publishing rule matched event: %v\n", err)
			}
		}
	}
	
	return results, nil
}

// GetMatchedActions returns all actions from matched rules
func (e *RuleEngine) GetMatchedActions(results []domain.RuleEvaluationResult) []domain.RuleAction {
	var actions []domain.RuleAction
	
	for _, result := range results {
		if result.Matched {
			actions = append(actions, result.Actions...)
		}
	}
	
	return actions
}

// HasBlockingAction checks if any of the results contain a blocking action
func (e *RuleEngine) HasBlockingAction(results []domain.RuleEvaluationResult) bool {
	for _, result := range results {
		if result.Matched {
			for _, action := range result.Actions {
				if action.Type == "deny" || action.Type == "block" {
					return true
				}
			}
		}
	}
	return false
}

// GetRateLimitActions returns all rate limiting actions from matched rules
func (e *RuleEngine) GetRateLimitActions(results []domain.RuleEvaluationResult) []domain.RuleAction {
	var rateLimitActions []domain.RuleAction
	
	for _, result := range results {
		if result.Matched {
			for _, action := range result.Actions {
				if action.Type == "rate_limit" {
					rateLimitActions = append(rateLimitActions, action)
				}
			}
		}
	}
	
	return rateLimitActions
}

// CreateRule creates a new rule
func (e *RuleEngine) CreateRule(ctx context.Context, rule domain.Rule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}
	
	return e.ruleRepository.SaveRule(ctx, rule)
}

// UpdateRule updates an existing rule
func (e *RuleEngine) UpdateRule(ctx context.Context, rule domain.Rule) error {
	rule.UpdatedAt = time.Now()
	return e.ruleRepository.UpdateRule(ctx, rule)
}

// DeleteRule deletes a rule
func (e *RuleEngine) DeleteRule(ctx context.Context, ruleID string) error {
	return e.ruleRepository.DeleteRule(ctx, ruleID)
}

// GetRule retrieves a rule by ID
func (e *RuleEngine) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	return e.ruleRepository.GetRuleByID(ctx, ruleID)
}

// ValidateRule validates a rule's structure and conditions
func (e *RuleEngine) ValidateRule(rule domain.Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	
	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}
	
	// Validate conditions
	for i, condition := range rule.Conditions {
		if condition.Field == "" {
			return fmt.Errorf("condition %d: field is required", i)
		}
		
		if condition.Operator == "" {
			return fmt.Errorf("condition %d: operator is required", i)
		}
		
		// Validate operator
		validOperators := []string{
			"equals", "not_equals", "contains", "starts_with", "ends_with",
			"in", "not_in", "greater_than", "less_than", "greater_equal", "less_equal",
		}
		
		validOp := false
		for _, op := range validOperators {
			if condition.Operator == op {
				validOp = true
				break
			}
		}
		
		if !validOp {
			return fmt.Errorf("condition %d: invalid operator '%s'", i, condition.Operator)
		}
	}
	
	// Validate actions
	for i, action := range rule.Actions {
		if action.Type == "" {
			return fmt.Errorf("action %d: type is required", i)
		}
		
		// Validate action type
		validActions := []string{
			"allow", "deny", "block", "rate_limit", "throttle", "log", "alert",
		}
		
		validAction := false
		for _, act := range validActions {
			if action.Type == act {
				validAction = true
				break
			}
		}
		
		if !validAction {
			return fmt.Errorf("action %d: invalid action type '%s'", i, action.Type)
		}
	}
	
	return nil
}
