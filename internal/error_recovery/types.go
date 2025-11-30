package error_recovery

import (
	"time"

	"github.com/google/uuid"
)

// ErrorPattern represents a detected pattern of errors
type ErrorPattern struct {
	Type             string  `json:"type"` // "rate_spike", "consecutive", "domain_specific"
	ErrorRate        float64 `json:"error_rate"`
	ConsecutiveCount int     `json:"consecutive_count"`
	DominantError    string  `json:"dominant_error"`
}

// ActivationDecision is the result of the analyzer's check
type ActivationDecision struct {
	ShouldActivate bool         `json:"should_activate"`
	Reason         string       `json:"reason"`
	ErrorPattern   ErrorPattern `json:"error_pattern"`
}

// Condition represents a condition for a rule to match
type Condition struct {
	Field    string      `json:"field"`    // "error_type", "status_code", "domain", "response_body"
	Operator string      `json:"operator"` // "equals", "contains", "regex", "gt", "lt"
	Value    interface{} `json:"value"`
}

// RuleContext defines the context in which a rule applies
type RuleContext struct {
	DomainPattern     string                 `json:"domain_pattern"` // "*.shopify.com", "example.com", "*"
	Variables         map[string]interface{} `json:"variables"`
	MaxRetries        int                    `json:"max_retries"`
	TimeoutMultiplier float64                `json:"timeout_multiplier"`
	LearnedFrom       string                 `json:"learned_from,omitempty"`
	LastUpdated       time.Time              `json:"last_updated"`
}

// Action represents an action to take when a rule matches
type Action struct {
	Type       string                 `json:"type"`       // "enable_stealth", "wait", "rotate_proxy"
	Parameters map[string]interface{} `json:"parameters"` // Supports {{variable}} substitution
	Condition  *Condition             `json:"condition,omitempty"`
}

// ContextAwareRule is a rule that maps conditions to actions
type ContextAwareRule struct {
	ID          string      `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Priority    int         `json:"priority" db:"priority"`
	Conditions  []Condition `json:"conditions" db:"conditions"`
	Context     RuleContext `json:"context" db:"context"`
	Actions     []Action    `json:"actions" db:"actions"`
	Confidence  float64     `json:"confidence" db:"confidence"`
	SuccessRate float64     `json:"success_rate" db:"success_rate"`
	UsageCount  int         `json:"usage_count" db:"usage_count"`
	CreatedBy   string      `json:"created_by" db:"created_by"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// Solution is the result of a rule or AI reasoning
type Solution struct {
	RuleName    string                 `json:"rule_name"`
	Actions     []Action               `json:"actions"`
	Confidence  float64                `json:"confidence"`
	Context     map[string]interface{} `json:"context"`
	Type        string                 `json:"type"` // "rule", "ai"
	Fingerprint string                 `json:"fingerprint"`
	RuleID      *uuid.UUID             `json:"rule_id,omitempty"`
}

// Config represents global configuration
type Config struct {
	Key   string      `json:"key" db:"config_key"`
	Value interface{} `json:"value" db:"config_value"`
}

func NewContextAwareRule(name string) *ContextAwareRule {
	return &ContextAwareRule{
		ID:         uuid.New().String(),
		Name:       name,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Conditions: []Condition{},
		Actions:    []Action{},
		Context: RuleContext{
			Variables: make(map[string]interface{}),
		},
	}
}
