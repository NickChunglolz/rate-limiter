package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/NickChunglolz/rule-engine/domain"
)

// InMemoryRuleRepository implements RuleRepository interface for testing/development
type InMemoryRuleRepository struct {
	rules map[string]domain.Rule
	mutex sync.RWMutex
}

// NewInMemoryRuleRepository creates a new in-memory rule repository
func NewInMemoryRuleRepository() *InMemoryRuleRepository {
	return &InMemoryRuleRepository{
		rules: make(map[string]domain.Rule),
	}
}

// GetActiveRules retrieves all active rules
func (r *InMemoryRuleRepository) GetActiveRules(ctx context.Context) ([]domain.Rule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var activeRules []domain.Rule
	for _, rule := range r.rules {
		if rule.Enabled {
			activeRules = append(activeRules, rule)
		}
	}
	
	return activeRules, nil
}

// GetRulesByType retrieves rules by type
func (r *InMemoryRuleRepository) GetRulesByType(ctx context.Context, ruleType domain.RuleType) ([]domain.Rule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var rules []domain.Rule
	for _, rule := range r.rules {
		if rule.Type == ruleType && rule.Enabled {
			rules = append(rules, rule)
		}
	}
	
	return rules, nil
}

// GetRulesByTags retrieves rules by tags
func (r *InMemoryRuleRepository) GetRulesByTags(ctx context.Context, tags []string) ([]domain.Rule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var rules []domain.Rule
	for _, rule := range r.rules {
		if rule.Enabled && r.hasAnyTag(rule.Tags, tags) {
			rules = append(rules, rule)
		}
	}
	
	return rules, nil
}

// SaveRule saves a rule
func (r *InMemoryRuleRepository) SaveRule(ctx context.Context, rule domain.Rule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.rules[rule.ID] = rule
	return nil
}

// UpdateRule updates an existing rule
func (r *InMemoryRuleRepository) UpdateRule(ctx context.Context, rule domain.Rule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.rules[rule.ID]; !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}
	
	r.rules[rule.ID] = rule
	return nil
}

// DeleteRule deletes a rule
func (r *InMemoryRuleRepository) DeleteRule(ctx context.Context, ruleID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}
	
	delete(r.rules, ruleID)
	return nil
}

// GetRuleByID retrieves a rule by ID
func (r *InMemoryRuleRepository) GetRuleByID(ctx context.Context, ruleID string) (*domain.Rule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	rule, exists := r.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}
	
	return &rule, nil
}

// hasAnyTag checks if rule has any of the specified tags
func (r *InMemoryRuleRepository) hasAnyTag(ruleTags, searchTags []string) bool {
	for _, ruleTag := range ruleTags {
		for _, searchTag := range searchTags {
			if ruleTag == searchTag {
				return true
			}
		}
	}
	return false
}

// SimpleEventPublisher implements EventPublisher interface for basic event publishing
type SimpleEventPublisher struct {
	subscribers []chan domain.RuleEvaluationResult
	mutex       sync.RWMutex
}

// NewSimpleEventPublisher creates a new simple event publisher
func NewSimpleEventPublisher() *SimpleEventPublisher {
	return &SimpleEventPublisher{
		subscribers: make([]chan domain.RuleEvaluationResult, 0),
	}
}

// PublishRuleEvaluated publishes a rule evaluated event
func (p *SimpleEventPublisher) PublishRuleEvaluated(ctx context.Context, result domain.RuleEvaluationResult) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	// Send to all subscribers
	for _, ch := range p.subscribers {
		select {
		case ch <- result:
		default:
			// Channel is full, skip this subscriber
		}
	}
	
	return nil
}

// PublishRuleMatched publishes a rule matched event
func (p *SimpleEventPublisher) PublishRuleMatched(ctx context.Context, result domain.RuleEvaluationResult) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	// Send to all subscribers
	for _, ch := range p.subscribers {
		select {
		case ch <- result:
		default:
			// Channel is full, skip this subscriber
		}
	}
	
	return nil
}

// Subscribe adds a subscriber for rule evaluation events
func (p *SimpleEventPublisher) Subscribe() <-chan domain.RuleEvaluationResult {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	ch := make(chan domain.RuleEvaluationResult, 100)
	p.subscribers = append(p.subscribers, ch)
	return ch
}
