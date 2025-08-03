package integration

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rateLimiterAPI "github.com/NickChunglolz/rate-limiter/internal/api"
	rateLimiterQueries "github.com/NickChunglolz/rate-limiter/internal/queries"
	ruleEngine "github.com/NickChunglolz/rule-engine/engine"
	ruleDomain "github.com/NickChunglolz/rule-engine/domain"
)

// IntegratedRateLimiterService combines rate limiting with rule engine
type IntegratedRateLimiterService struct {
	rateLimiterService *rateLimiterAPI.RateLimiterService
	ruleEngine         *ruleEngine.RuleEngine
}

// NewIntegratedRateLimiterService creates a new integrated service
func NewIntegratedRateLimiterService(
	rateLimiterService *rateLimiterAPI.RateLimiterService,
	ruleEngine *ruleEngine.RuleEngine,
) *IntegratedRateLimiterService {
	return &IntegratedRateLimiterService{
		rateLimiterService: rateLimiterService,
		ruleEngine:         ruleEngine,
	}
}

// CheckRequestWithRules checks a request against both rules and rate limits
func (s *IntegratedRateLimiterService) CheckRequestWithRules(
	ctx context.Context,
	clientID, resource, ipAddress, userAgent string,
	metadata map[string]string,
	requestData map[string]interface{},
) (*RequestCheckResult, error) {
	
	// Create rule evaluation context
	evalCtx := ruleDomain.RuleEvaluationContext{
		ClientID:    clientID,
		Resource:    resource,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Timestamp:   time.Now(),
		Metadata:    metadata,
		RequestData: requestData,
	}
	
	// Evaluate rules first
	ruleResults, err := s.ruleEngine.EvaluateRules(ctx, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate rules: %w", err)
	}
	
	// Check for blocking actions
	if s.ruleEngine.HasBlockingAction(ruleResults) {
		return &RequestCheckResult{
			Allowed:           false,
			Reason:            "blocked by rule",
			RuleResults:       ruleResults,
			RateLimitStatus:   nil,
			BlockingRuleID:    s.getFirstBlockingRuleID(ruleResults),
		}, nil
	}
	
	// Check for rate limiting actions
	rateLimitActions := s.ruleEngine.GetRateLimitActions(ruleResults)
	if len(rateLimitActions) > 0 {
		// Apply dynamic rate limiting based on rule actions
		err := s.applyDynamicRateLimiting(ctx, rateLimitActions, resource)
		if err != nil {
			return nil, fmt.Errorf("failed to apply dynamic rate limiting: %w", err)
		}
	}
	
	// Check rate limits
	rateLimitStatus, err := s.rateLimiterService.CheckRateLimit(ctx, clientID, resource, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	
	result := &RequestCheckResult{
		Allowed:         rateLimitStatus.IsAllowed,
		Reason:          s.determineReason(rateLimitStatus, ruleResults),
		RuleResults:     ruleResults,
		RateLimitStatus: rateLimitStatus,
	}
	
	if !rateLimitStatus.IsAllowed {
		result.Reason = "rate limited"
	}
	
	return result, nil
}

// RequestCheckResult contains the result of an integrated request check
type RequestCheckResult struct {
	Allowed           bool                              `json:"allowed"`
	Reason            string                            `json:"reason"`
	RuleResults       []ruleDomain.RuleEvaluationResult `json:"rule_results"`
	RateLimitStatus   *rateLimiterQueries.RateLimitStatus `json:"rate_limit_status"`
	BlockingRuleID    string                            `json:"blocking_rule_id,omitempty"`
	AppliedActions    []ruleDomain.RuleAction           `json:"applied_actions"`
}

// applyDynamicRateLimiting applies rate limiting rules dynamically
func (s *IntegratedRateLimiterService) applyDynamicRateLimiting(
	ctx context.Context,
	actions []ruleDomain.RuleAction,
	resource string,
) error {
	for _, action := range actions {
		if action.Type == "rate_limit" {
			// Extract rate limiting parameters from action
			limit, limitOK := action.Parameters["limit"]
			window, windowOK := action.Parameters["window"]
			algorithm, algorithmOK := action.Parameters["algorithm"]
			
			if !limitOK || !windowOK {
				continue // Skip invalid action
			}
			
			// Convert parameters
			var limitInt int
			var windowDuration time.Duration
			var algorithmStr string
			
			switch v := limit.(type) {
			case int:
				limitInt = v
			case float64:
				limitInt = int(v)
			case string:
				if parsed, err := strconv.Atoi(v); err == nil {
					limitInt = parsed
				}
			}
			
			switch v := window.(type) {
			case string:
				if parsed, err := time.ParseDuration(v); err == nil {
					windowDuration = parsed
				}
			case int:
				windowDuration = time.Duration(v) * time.Second
			case float64:
				windowDuration = time.Duration(v) * time.Second
			}
			
			if algorithmOK {
				if alg, ok := algorithm.(string); ok {
					algorithmStr = alg
				}
			} else {
				algorithmStr = "sliding_window" // default
			}
			
			if limitInt > 0 && windowDuration > 0 {
				// Create or update the rate limiting rule
				err := s.rateLimiterService.CreateRule(ctx, resource, limitInt, windowDuration, algorithmStr)
				if err != nil {
					return fmt.Errorf("failed to create dynamic rate limit rule: %w", err)
				}
			}
		}
	}
	
	return nil
}

// getFirstBlockingRuleID returns the ID of the first blocking rule
func (s *IntegratedRateLimiterService) getFirstBlockingRuleID(results []ruleDomain.RuleEvaluationResult) string {
	for _, result := range results {
		if result.Matched {
			for _, action := range result.Actions {
				if action.Type == "deny" || action.Type == "block" {
					return result.RuleID
				}
			}
		}
	}
	return ""
}

// determineReason determines the reason for allowing/blocking a request
func (s *IntegratedRateLimiterService) determineReason(
	rateLimitStatus *rateLimiterQueries.RateLimitStatus,
	ruleResults []ruleDomain.RuleEvaluationResult,
) string {
	if !rateLimitStatus.IsAllowed {
		return "rate limited"
	}
	
	// Check if any rules matched
	for _, result := range ruleResults {
		if result.Matched {
			for _, action := range result.Actions {
				switch action.Type {
				case "allow":
					return "allowed by rule"
				case "throttle":
					return "throttled by rule"
				}
			}
		}
	}
	
	return "allowed"
}

// CreateSecurityRule creates a security-focused rule
func (s *IntegratedRateLimiterService) CreateSecurityRule(
	ctx context.Context,
	name, description string,
	conditions []ruleDomain.RuleCondition,
	actions []ruleDomain.RuleAction,
	priority int,
) error {
	rule := ruleDomain.Rule{
		ID:          fmt.Sprintf("security-rule-%d", time.Now().UnixNano()),
		Name:        name,
		Type:        ruleDomain.RateLimitRule,
		Description: description,
		Priority:    priority,
		Enabled:     true,
		Conditions:  conditions,
		Actions:     actions,
		Tags:        []string{"security", "auto-generated"},
	}
	
	return s.ruleEngine.CreateRule(ctx, rule)
}

// CreateIPBasedRule creates an IP-based blocking or rate limiting rule
func (s *IntegratedRateLimiterService) CreateIPBasedRule(
	ctx context.Context,
	ipAddresses []string,
	action string, // "block" or "rate_limit"
	parameters map[string]interface{},
) error {
	// Convert IP addresses to interface{} slice
	var ipValues []interface{}
	for _, ip := range ipAddresses {
		ipValues = append(ipValues, ip)
	}
	
	conditions := []ruleDomain.RuleCondition{
		{
			Field:    "ip_address",
			Operator: "in",
			Value:    ipValues,
		},
	}
	
	var actions []ruleDomain.RuleAction
	if action == "block" {
		actions = append(actions, ruleDomain.RuleAction{
			Type:       "deny",
			Parameters: parameters,
		})
	} else if action == "rate_limit" {
		actions = append(actions, ruleDomain.RuleAction{
			Type:       "rate_limit",
			Parameters: parameters,
		})
	}
	
	rule := ruleDomain.Rule{
		ID:          fmt.Sprintf("ip-rule-%d", time.Now().UnixNano()),
		Name:        fmt.Sprintf("IP-based %s rule", action),
		Type:        ruleDomain.BlacklistRule,
		Description: fmt.Sprintf("Auto-generated IP-based %s rule", action),
		Priority:    100,
		Enabled:     true,
		Conditions:  conditions,
		Actions:     actions,
		Tags:        []string{"ip-based", "auto-generated"},
	}
	
	return s.ruleEngine.CreateRule(ctx, rule)
}

// CreateResourceBasedRule creates a resource-specific rule
func (s *IntegratedRateLimiterService) CreateResourceBasedRule(
	ctx context.Context,
	resources []string,
	limit int,
	window time.Duration,
	algorithm string,
) error {
	// Convert resources to interface{} slice
	var resourceValues []interface{}
	for _, resource := range resources {
		resourceValues = append(resourceValues, resource)
	}
	
	conditions := []ruleDomain.RuleCondition{
		{
			Field:    "resource",
			Operator: "in",
			Value:    resourceValues,
		},
	}
	
	actions := []ruleDomain.RuleAction{
		{
			Type: "rate_limit",
			Parameters: map[string]interface{}{
				"limit":     limit,
				"window":    window.String(),
				"algorithm": algorithm,
			},
		},
	}
	
	rule := ruleDomain.Rule{
		ID:          fmt.Sprintf("resource-rule-%d", time.Now().UnixNano()),
		Name:        "Resource-based rate limiting rule",
		Type:        ruleDomain.RateLimitRule,
		Description: "Auto-generated resource-specific rate limiting rule",
		Priority:    50,
		Enabled:     true,
		Conditions:  conditions,
		Actions:     actions,
		Tags:        []string{"resource-based", "auto-generated"},
	}
	
	return s.ruleEngine.CreateRule(ctx, rule)
}
