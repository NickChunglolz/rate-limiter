package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/NickChunglolz/rate-limiter/internal/domain"
)

// InMemoryEventStore implements EventStore interface for testing/development
type InMemoryEventStore struct {
	events map[string][]domain.Event
	mutex  sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]domain.Event),
	}
}

// SaveEvents saves events for an aggregate
func (s *InMemoryEventStore) SaveEvents(ctx context.Context, aggregateID string, events []domain.Event, expectedVersion int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	existingEvents := s.events[aggregateID]
	if len(existingEvents) != expectedVersion {
		return fmt.Errorf("concurrency conflict: expected version %d, got %d", expectedVersion, len(existingEvents))
	}
	
	s.events[aggregateID] = append(existingEvents, events...)
	return nil
}

// GetEvents retrieves all events for an aggregate
func (s *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string) ([]domain.Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	events := s.events[aggregateID]
	if events == nil {
		return make([]domain.Event, 0), nil
	}
	
	// Deep copy to avoid race conditions
	result := make([]domain.Event, len(events))
	copy(result, events)
	return result, nil
}

// InMemoryRuleRepository implements RuleRepository interface for testing/development
type InMemoryRuleRepository struct {
	rules map[string]domain.RateLimitRule
	mutex sync.RWMutex
}

// NewInMemoryRuleRepository creates a new in-memory rule repository
func NewInMemoryRuleRepository() *InMemoryRuleRepository {
	return &InMemoryRuleRepository{
		rules: make(map[string]domain.RateLimitRule),
	}
}

// Save saves a rate limit rule
func (r *InMemoryRuleRepository) Save(ctx context.Context, rule domain.RateLimitRule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.rules[rule.ID] = rule
	return nil
}

// GetByResource retrieves rules by resource
func (r *InMemoryRuleRepository) GetByResource(ctx context.Context, resource string) ([]domain.RateLimitRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var result []domain.RateLimitRule
	for _, rule := range r.rules {
		if rule.Resource == resource {
			result = append(result, rule)
		}
	}
	
	return result, nil
}

// GetByID retrieves a rule by ID
func (r *InMemoryRuleRepository) GetByID(ctx context.Context, id string) (*domain.RateLimitRule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	rule, exists := r.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	
	return &rule, nil
}

// Update updates an existing rule
func (r *InMemoryRuleRepository) Update(ctx context.Context, rule domain.RateLimitRule) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.rules[rule.ID]; !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}
	
	r.rules[rule.ID] = rule
	return nil
}

// Delete deletes a rule
func (r *InMemoryRuleRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.rules[id]; !exists {
		return fmt.Errorf("rule not found: %s", id)
	}
	
	delete(r.rules, id)
	return nil
}

// RedisEventStore implements EventStore interface using Redis
type RedisEventStore struct {
	// Redis client would be here
	// For now, just embed the in-memory implementation
	*InMemoryEventStore
}

// NewRedisEventStore creates a new Redis-based event store
func NewRedisEventStore() *RedisEventStore {
	return &RedisEventStore{
		InMemoryEventStore: NewInMemoryEventStore(),
	}
}

// PostgreSQLRuleRepository implements RuleRepository interface using PostgreSQL
type PostgreSQLRuleRepository struct {
	// Database connection would be here
	// For now, just embed the in-memory implementation
	*InMemoryRuleRepository
}

// NewPostgreSQLRuleRepository creates a new PostgreSQL-based rule repository
func NewPostgreSQLRuleRepository() *PostgreSQLRuleRepository {
	return &PostgreSQLRuleRepository{
		InMemoryRuleRepository: NewInMemoryRuleRepository(),
	}
}

// EventBus handles event publishing and subscription
type EventBus struct {
	subscribers map[string][]chan domain.Event
	mutex       sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan domain.Event),
	}
}

// Subscribe subscribes to events of a specific type
func (b *EventBus) Subscribe(eventType string) <-chan domain.Event {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	ch := make(chan domain.Event, 100) // Buffered channel
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

// Publish publishes an event
func (b *EventBus) Publish(event domain.Event) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	eventType := event.EventType()
	if channels, exists := b.subscribers[eventType]; exists {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}
	
	// Also publish to "all" subscribers
	if channels, exists := b.subscribers["*"]; exists {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}
}
