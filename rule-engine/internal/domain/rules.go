package domain

import (
	"time"
)

// RuleType defines different types of rules
type RuleType string

const (
	RateLimitRule    RuleType = "rate_limit"
	ThrottleRule     RuleType = "throttle"
	BlacklistRule    RuleType = "blacklist"
	WhitelistRule    RuleType = "whitelist"
	GeofenceRule     RuleType = "geofence"
	TimeBasedRule    RuleType = "time_based"
)

// RuleCondition defines conditions for rule evaluation
type RuleCondition struct {
	Field    string      `json:"field"`    // e.g., "client_id", "ip_address", "user_agent"
	Operator string      `json:"operator"` // e.g., "equals", "contains", "regex", "in"
	Value    interface{} `json:"value"`    // The value to compare against
}

// RuleAction defines actions to take when a rule matches
type RuleAction struct {
	Type       string                 `json:"type"`       // e.g., "allow", "deny", "rate_limit", "throttle"
	Parameters map[string]interface{} `json:"parameters"` // Action-specific parameters
}

// Rule represents a business rule in the system
type Rule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        RuleType        `json:"type"`
	Description string          `json:"description"`
	Priority    int             `json:"priority"`    // Higher number = higher priority
	Enabled     bool            `json:"enabled"`
	Conditions  []RuleCondition `json:"conditions"`  // All conditions must match (AND logic)
	Actions     []RuleAction    `json:"actions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CreatedBy   string          `json:"created_by"`
	Tags        []string        `json:"tags"`
}

// RuleEvaluationContext contains data for rule evaluation
type RuleEvaluationContext struct {
	ClientID    string            `json:"client_id"`
	Resource    string            `json:"resource"`
	IPAddress   string            `json:"ip_address"`
	UserAgent   string            `json:"user_agent"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata"`
	RequestData map[string]interface{} `json:"request_data"`
}

// RuleEvaluationResult contains the result of rule evaluation
type RuleEvaluationResult struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Matched     bool                   `json:"matched"`
	Actions     []RuleAction           `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
}

// RuleSet represents a collection of rules
type RuleSet struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rules       []Rule    `json:"rules"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EvaluateRule evaluates a rule against the given context
func (r *Rule) EvaluateRule(ctx RuleEvaluationContext) RuleEvaluationResult {
	result := RuleEvaluationResult{
		RuleID:      r.ID,
		RuleName:    r.Name,
		Matched:     false,
		Actions:     make([]RuleAction, 0),
		Metadata:    make(map[string]interface{}),
		EvaluatedAt: time.Now(),
	}
	
	if !r.Enabled {
		return result
	}
	
	// Evaluate all conditions (AND logic)
	matched := true
	for _, condition := range r.Conditions {
		if !r.evaluateCondition(condition, ctx) {
			matched = false
			break
		}
	}
	
	result.Matched = matched
	if matched {
		result.Actions = r.Actions
	}
	
	return result
}

// evaluateCondition evaluates a single condition
func (r *Rule) evaluateCondition(condition RuleCondition, ctx RuleEvaluationContext) bool {
	var fieldValue interface{}
	
	// Get field value from context
	switch condition.Field {
	case "client_id":
		fieldValue = ctx.ClientID
	case "resource":
		fieldValue = ctx.Resource
	case "ip_address":
		fieldValue = ctx.IPAddress
	case "user_agent":
		fieldValue = ctx.UserAgent
	case "timestamp":
		fieldValue = ctx.Timestamp
	default:
		// Check metadata
		if val, exists := ctx.Metadata[condition.Field]; exists {
			fieldValue = val
		} else if val, exists := ctx.RequestData[condition.Field]; exists {
			fieldValue = val
		} else {
			return false // Field not found
		}
	}
	
	// Evaluate based on operator
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return containsString(str, substr)
			}
		}
		return false
	case "starts_with":
		if str, ok := fieldValue.(string); ok {
			if prefix, ok := condition.Value.(string); ok {
				return len(str) >= len(prefix) && str[:len(prefix)] == prefix
			}
		}
		return false
	case "ends_with":
		if str, ok := fieldValue.(string); ok {
			if suffix, ok := condition.Value.(string); ok {
				return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
			}
		}
		return false
	case "in":
		if values, ok := condition.Value.([]interface{}); ok {
			for _, val := range values {
				if fieldValue == val {
					return true
				}
			}
		}
		return false
	case "not_in":
		if values, ok := condition.Value.([]interface{}); ok {
			for _, val := range values {
				if fieldValue == val {
					return false
				}
			}
			return true
		}
		return false
	case "greater_than":
		return compareNumbers(fieldValue, condition.Value) > 0
	case "less_than":
		return compareNumbers(fieldValue, condition.Value) < 0
	case "greater_equal":
		return compareNumbers(fieldValue, condition.Value) >= 0
	case "less_equal":
		return compareNumbers(fieldValue, condition.Value) <= 0
	default:
		return false
	}
}

// Helper function to check if string contains substring
func containsString(str, substr string) bool {
	return len(str) >= len(substr) && str != substr && findSubstring(str, substr) != -1
}

// Helper function to find substring
func findSubstring(str, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(str) {
		return -1
	}
	
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function to compare numbers
func compareNumbers(a, b interface{}) int {
	var aVal, bVal float64
	var ok bool
	
	// Convert a to float64
	switch v := a.(type) {
	case int:
		aVal = float64(v)
	case int64:
		aVal = float64(v)
	case float64:
		aVal = v
	case float32:
		aVal = float64(v)
	default:
		return 0 // Cannot compare
	}
	
	// Convert b to float64
	switch v := b.(type) {
	case int:
		bVal = float64(v)
	case int64:
		bVal = float64(v)
	case float64:
		bVal = v
	case float32:
		bVal = float64(v)
	default:
		return 0 // Cannot compare
	}
	
	if !ok {
		return 0
	}
	
	if aVal > bVal {
		return 1
	} else if aVal < bVal {
		return -1
	}
	return 0
}
