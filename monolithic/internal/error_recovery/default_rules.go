package error_recovery

import (
	"time"

	"github.com/google/uuid"
)

// GetDefaultRules returns a set of predefined error recovery rules
func GetDefaultRules() []ContextAwareRule {
	now := time.Now()

	return []ContextAwareRule{
		{
			ID:          uuid.New().String(),
			Name:        "cloudflare_adaptive_stealth",
			Description: "Adaptive stealth mode for Cloudflare protection",
			Priority:    10,
			Conditions: []Condition{
				{Field: "status_code", Operator: "equals", Value: 403},
				{Field: "response_body", Operator: "contains", Value: "cloudflare"},
			},
			Context: RuleContext{
				DomainPattern: "*",
				Variables: map[string]interface{}{
					"wait_time":     15,
					"stealth_level": "aggressive",
				},
				MaxRetries:  3,
				LastUpdated: now,
			},
			Actions: []Action{
				{Type: "enable_stealth", Parameters: map[string]interface{}{"level": "aggressive"}},
				{Type: "wait", Parameters: map[string]interface{}{"duration": 15}},
			},
			Confidence:  0.90,
			SuccessRate: 0.90,
			CreatedBy:   "predefined",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			Name:        "shopify_rate_limit_adaptive",
			Description: "Handles Shopify rate limiting with adaptive backoff",
			Priority:    8,
			Conditions: []Condition{
				{Field: "status_code", Operator: "equals", Value: 429},
				{Field: "domain", Operator: "regex", Value: `.*\.myshopify\.com`},
			},
			Context: RuleContext{
				DomainPattern: "*.myshopify.com",
				Variables: map[string]interface{}{
					"retry_delay": 60,
				},
				MaxRetries:  3,
				LastUpdated: now,
			},
			Actions: []Action{
				{Type: "pause_execution", Parameters: map[string]interface{}{}},
				{Type: "wait", Parameters: map[string]interface{}{"duration": 60}},
				{Type: "reduce_workers", Parameters: map[string]interface{}{"count": 2}},
				{Type: "add_delay", Parameters: map[string]interface{}{"duration": 2000}},
				{Type: "resume_execution", Parameters: map[string]interface{}{}},
			},
			Confidence:  0.95,
			SuccessRate: 0.95,
			CreatedBy:   "predefined",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			Name:        "generic_rate_limit_429",
			Description: "Generic rate limit handler for any domain (429 errors)",
			Priority:    7,
			Conditions: []Condition{
				{Field: "status_code", Operator: "equals", Value: 429},
			},
			Context: RuleContext{
				DomainPattern: "*",
				Variables: map[string]interface{}{
					"backoff_time": 30,
				},
				MaxRetries:  3,
				LastUpdated: now,
			},
			Actions: []Action{
				{Type: "pause_execution", Parameters: map[string]interface{}{}},
				{Type: "wait", Parameters: map[string]interface{}{"duration": 30}},
				{Type: "reduce_workers", Parameters: map[string]interface{}{"count": 1}},
				{Type: "add_delay", Parameters: map[string]interface{}{"duration": 1000}},
				{Type: "resume_execution", Parameters: map[string]interface{}{}},
			},
			Confidence:  0.85,
			SuccessRate: 0.85,
			CreatedBy:   "predefined",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New().String(),
			Name:        "forbidden_stealth_escalation",
			Description: "Escalates stealth mode for generic 403 errors",
			Priority:    5,
			Conditions: []Condition{
				{Field: "status_code", Operator: "equals", Value: 403},
			},
			Context: RuleContext{
				DomainPattern: "*",
				Variables: map[string]interface{}{
					"stealth_level": "moderate",
				},
				MaxRetries:  2,
				LastUpdated: now,
			},
			Actions: []Action{
				{Type: "enable_stealth", Parameters: map[string]interface{}{"level": "moderate"}},
				{Type: "wait", Parameters: map[string]interface{}{"duration": 5}},
			},
			Confidence:  0.75,
			SuccessRate: 0.75,
			CreatedBy:   "predefined",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}
