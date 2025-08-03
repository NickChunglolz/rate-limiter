package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NickChunglolz/rate-limiter/internal/domain"
	"github.com/NickChunglolz/rate-limiter/internal/queries"
)

// InMemoryReadModel implements ReadModel interface for testing/development
type InMemoryReadModel struct {
	statuses map[string]*queries.RateLimitStatus
	history  map[string][]queries.RateLimitEvent
	stats    map[string]*queries.ClientStats
	mutex    sync.RWMutex
}

// NewInMemoryReadModel creates a new in-memory read model
func NewInMemoryReadModel() *InMemoryReadModel {
	return &InMemoryReadModel{
		statuses: make(map[string]*queries.RateLimitStatus),
		history:  make(map[string][]queries.RateLimitEvent),
		stats:    make(map[string]*queries.ClientStats),
	}
}

// GetRateLimitStatus retrieves current rate limit status
func (r *InMemoryReadModel) GetRateLimitStatus(ctx context.Context, clientID, resource string) (*queries.RateLimitStatus, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	key := clientID + ":" + resource
	status, exists := r.statuses[key]
	if !exists {
		// Return default status
		return &queries.RateLimitStatus{
			ClientID:       clientID,
			Resource:       resource,
			IsAllowed:      true,
			RequestCount:   0,
			Limit:          0,
			RemainingQuota: 0,
			WindowStart:    time.Now(),
			WindowEnd:      time.Now().Add(time.Hour),
			ResetTime:      time.Now().Add(time.Hour),
			IsBlocked:      false,
		}, nil
	}
	
	// Deep copy to avoid race conditions
	result := *status
	return &result, nil
}

// GetRateLimitHistory retrieves rate limit history
func (r *InMemoryReadModel) GetRateLimitHistory(ctx context.Context, clientID, resource string, startTime, endTime time.Time, limit, offset int) (*queries.RateLimitHistory, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	key := clientID + ":" + resource
	allEvents := r.history[key]
	
	// Filter by time range
	var filteredEvents []queries.RateLimitEvent
	for _, event := range allEvents {
		if event.Timestamp.After(startTime) && event.Timestamp.Before(endTime) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	
	// Apply pagination
	totalCount := len(filteredEvents)
	start := offset
	end := offset + limit
	
	if start >= totalCount {
		return &queries.RateLimitHistory{
			Events:     make([]queries.RateLimitEvent, 0),
			TotalCount: totalCount,
			HasMore:    false,
		}, nil
	}
	
	if end > totalCount {
		end = totalCount
	}
	
	pagedEvents := filteredEvents[start:end]
	hasMore := end < totalCount
	
	return &queries.RateLimitHistory{
		Events:     pagedEvents,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// GetClientStats retrieves client statistics
func (r *InMemoryReadModel) GetClientStats(ctx context.Context, clientID string, startTime, endTime time.Time) (*queries.ClientStats, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	stats, exists := r.stats[clientID]
	if !exists {
		// Return default stats
		return &queries.ClientStats{
			ClientID:        clientID,
			TotalRequests:   0,
			BlockedRequests: 0,
			AllowedRequests: 0,
			ResourceStats:   make([]queries.ResourceStats, 0),
			TimeSeriesData:  make([]queries.TimeSeriesDataPoint, 0),
		}, nil
	}
	
	// Deep copy to avoid race conditions
	result := *stats
	return &result, nil
}

// UpdateFromEvent updates the read model from domain events
func (r *InMemoryReadModel) UpdateFromEvent(ctx context.Context, event interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	switch e := event.(type) {
	case *domain.RateLimitAppliedEvent:
		return r.updateFromRateLimitApplied(e)
	case *domain.RateLimitExceededEvent:
		return r.updateFromRateLimitExceeded(e)
	case *domain.RateLimitWindowResetEvent:
		return r.updateFromWindowReset(e)
	default:
		return fmt.Errorf("unknown event type: %T", event)
	}
}

// updateFromRateLimitApplied updates read model from RateLimitAppliedEvent
func (r *InMemoryReadModel) updateFromRateLimitApplied(event *domain.RateLimitAppliedEvent) error {
	key := event.ClientID + ":" + event.Resource
	
	// Update status
	status := &queries.RateLimitStatus{
		ClientID:       event.ClientID,
		Resource:       event.Resource,
		IsAllowed:      true,
		RequestCount:   event.RequestCount,
		Limit:          event.Limit,
		RemainingQuota: event.RemainingQuota,
		WindowStart:    event.WindowStart,
		WindowEnd:      event.WindowEnd,
		ResetTime:      event.WindowEnd,
		IsBlocked:      false,
	}
	r.statuses[key] = status
	
	// Add to history
	historyEvent := queries.RateLimitEvent{
		EventID:      event.EventID(),
		EventType:    event.EventType(),
		ClientID:     event.ClientID,
		Resource:     event.Resource,
		Timestamp:    event.Timestamp(),
		RequestCount: event.RequestCount,
		Limit:        event.Limit,
		IsBlocked:    false,
	}
	r.history[key] = append(r.history[key], historyEvent)
	
	// Update client stats
	r.updateClientStats(event.ClientID, event.Resource, true)
	
	return nil
}

// updateFromRateLimitExceeded updates read model from RateLimitExceededEvent
func (r *InMemoryReadModel) updateFromRateLimitExceeded(event *domain.RateLimitExceededEvent) error {
	key := event.ClientID + ":" + event.Resource
	
	// Calculate retry after in seconds
	retryAfter := int(time.Until(event.BlockedUntil).Seconds())
	if retryAfter < 0 {
		retryAfter = 0
	}
	
	// Update status
	status := &queries.RateLimitStatus{
		ClientID:       event.ClientID,
		Resource:       event.Resource,
		IsAllowed:      false,
		RequestCount:   event.RequestCount,
		Limit:          event.Limit,
		RemainingQuota: 0,
		WindowStart:    event.WindowStart,
		WindowEnd:      event.WindowEnd,
		ResetTime:      event.WindowEnd,
		IsBlocked:      true,
		BlockedUntil:   event.BlockedUntil,
		RetryAfter:     retryAfter,
	}
	r.statuses[key] = status
	
	// Add to history
	historyEvent := queries.RateLimitEvent{
		EventID:      event.EventID(),
		EventType:    event.EventType(),
		ClientID:     event.ClientID,
		Resource:     event.Resource,
		Timestamp:    event.Timestamp(),
		RequestCount: event.RequestCount,
		Limit:        event.Limit,
		IsBlocked:    true,
	}
	r.history[key] = append(r.history[key], historyEvent)
	
	// Update client stats
	r.updateClientStats(event.ClientID, event.Resource, false)
	
	return nil
}

// updateFromWindowReset updates read model from RateLimitWindowResetEvent
func (r *InMemoryReadModel) updateFromWindowReset(event *domain.RateLimitWindowResetEvent) error {
	key := event.ClientID + ":" + event.Resource
	
	// Reset status
	if status, exists := r.statuses[key]; exists {
		status.RequestCount = 0
		status.RemainingQuota = status.Limit
		status.WindowStart = event.WindowStart
		status.IsBlocked = false
		status.BlockedUntil = time.Time{}
		status.RetryAfter = 0
	}
	
	// Add to history
	historyEvent := queries.RateLimitEvent{
		EventID:   event.EventID(),
		EventType: event.EventType(),
		ClientID:  event.ClientID,
		Resource:  event.Resource,
		Timestamp: event.Timestamp(),
		IsBlocked: false,
	}
	r.history[key] = append(r.history[key], historyEvent)
	
	return nil
}

// updateClientStats updates client statistics
func (r *InMemoryReadModel) updateClientStats(clientID, resource string, allowed bool) {
	stats, exists := r.stats[clientID]
	if !exists {
		stats = &queries.ClientStats{
			ClientID:        clientID,
			TotalRequests:   0,
			BlockedRequests: 0,
			AllowedRequests: 0,
			ResourceStats:   make([]queries.ResourceStats, 0),
			TimeSeriesData:  make([]queries.TimeSeriesDataPoint, 0),
		}
		r.stats[clientID] = stats
	}
	
	// Update total stats
	stats.TotalRequests++
	if allowed {
		stats.AllowedRequests++
	} else {
		stats.BlockedRequests++
	}
	
	// Update resource-specific stats
	var resourceStats *queries.ResourceStats
	for i := range stats.ResourceStats {
		if stats.ResourceStats[i].Resource == resource {
			resourceStats = &stats.ResourceStats[i]
			break
		}
	}
	
	if resourceStats == nil {
		stats.ResourceStats = append(stats.ResourceStats, queries.ResourceStats{
			Resource:        resource,
			TotalRequests:   0,
			BlockedRequests: 0,
			AllowedRequests: 0,
		})
		resourceStats = &stats.ResourceStats[len(stats.ResourceStats)-1]
	}
	
	resourceStats.TotalRequests++
	if allowed {
		resourceStats.AllowedRequests++
	} else {
		resourceStats.BlockedRequests++
	}
	
	// Calculate blocked rate
	if resourceStats.TotalRequests > 0 {
		resourceStats.BlockedRate = float64(resourceStats.BlockedRequests) / float64(resourceStats.TotalRequests)
	}
	
	// Update time series data (simplified - could be more sophisticated)
	now := time.Now().Truncate(time.Minute) // Group by minute
	var dataPoint *queries.TimeSeriesDataPoint
	for i := range stats.TimeSeriesData {
		if stats.TimeSeriesData[i].Timestamp.Equal(now) {
			dataPoint = &stats.TimeSeriesData[i]
			break
		}
	}
	
	if dataPoint == nil {
		stats.TimeSeriesData = append(stats.TimeSeriesData, queries.TimeSeriesDataPoint{
			Timestamp:       now,
			TotalRequests:   0,
			BlockedRequests: 0,
			AllowedRequests: 0,
		})
		dataPoint = &stats.TimeSeriesData[len(stats.TimeSeriesData)-1]
	}
	
	dataPoint.TotalRequests++
	if allowed {
		dataPoint.AllowedRequests++
	} else {
		dataPoint.BlockedRequests++
	}
}
