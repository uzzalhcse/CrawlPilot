// Package recovery provides intelligent error recovery for web scraping tasks.
// It uses a layered approach: rule-based recovery first, with AI fallback.
// Successful AI recoveries are learned and converted to rules for future use.
package recovery

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// ErrorPattern represents a categorized error type
type ErrorPattern string

const (
	PatternBlocked       ErrorPattern = "blocked"
	PatternRateLimited   ErrorPattern = "rate_limited"
	PatternCaptcha       ErrorPattern = "captcha"
	PatternTimeout       ErrorPattern = "timeout"
	PatternConnectionErr ErrorPattern = "connection_error"
	PatternLayoutChanged ErrorPattern = "layout_changed"
	PatternAuthRequired  ErrorPattern = "auth_required"
	PatternNotFound      ErrorPattern = "not_found"
	PatternServerError   ErrorPattern = "server_error"
	PatternUnknown       ErrorPattern = "unknown"
)

// ActionType represents a recovery action to take
type ActionType string

const (
	ActionSwitchProxy  ActionType = "switch_proxy"
	ActionAddDelay     ActionType = "add_delay"
	ActionSkipDomain   ActionType = "skip_domain"
	ActionRetryBrowser ActionType = "retry_with_browser"
	ActionSendToDLQ    ActionType = "send_to_dlq"
	ActionRotateUA     ActionType = "rotate_user_agent"
	ActionClearCookies ActionType = "clear_cookies"
	ActionRetry        ActionType = "retry"
)

// DetectedError contains information about a detected error
type DetectedError struct {
	Pattern     ErrorPattern      `json:"pattern"`
	Confidence  float64           `json:"confidence"`
	RawError    string            `json:"raw_error"`
	Domain      string            `json:"domain"`
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	PageContent string            `json:"page_content,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	DetectedAt  time.Time         `json:"detected_at"`
}

// RecoveryPlan represents the planned recovery action
type RecoveryPlan struct {
	Action      ActionType             `json:"action"`
	Params      map[string]interface{} `json:"params"`
	Reason      string                 `json:"reason"`
	ShouldRetry bool                   `json:"should_retry"`
	RetryDelay  time.Duration          `json:"retry_delay"`
	Source      string                 `json:"source"` // "rule", "ai", "learned"
	RuleID      string                 `json:"rule_id,omitempty"`
}

// RecoveryRule represents a configured recovery rule (from DB/frontend)
type RecoveryRule struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Description  string                 `json:"description" db:"description"`
	Priority     int                    `json:"priority" db:"priority"` // Lower = higher priority
	Enabled      bool                   `json:"enabled" db:"enabled"`
	Pattern      ErrorPattern           `json:"pattern" db:"pattern"`
	Conditions   []RuleCondition        `json:"conditions" db:"conditions"`
	Action       ActionType             `json:"action" db:"action"`
	ActionParams map[string]interface{} `json:"action_params" db:"action_params"`
	MaxRetries   int                    `json:"max_retries" db:"max_retries"`
	RetryDelay   int                    `json:"retry_delay" db:"retry_delay"` // seconds
	IsLearned    bool                   `json:"is_learned" db:"is_learned"`   // From AI learning
	LearnedFrom  string                 `json:"learned_from,omitempty" db:"learned_from"`
	SuccessCount int                    `json:"success_count" db:"success_count"`
	FailureCount int                    `json:"failure_count" db:"failure_count"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// RuleCondition represents a condition for rule matching
type RuleCondition struct {
	Field    string `json:"field"`    // "domain", "status_code", "error_contains", "url_pattern"
	Operator string `json:"operator"` // "equals", "contains", "regex", "gt", "lt"
	Value    string `json:"value"`
}

// Match checks if a condition matches the error
func (c *RuleCondition) Match(err *DetectedError) bool {
	switch c.Field {
	case "domain":
		return c.matchString(err.Domain)
	case "url_pattern":
		return c.matchString(err.URL)
	case "error_contains":
		return c.matchString(err.RawError)
	case "status_code":
		return c.matchInt(err.StatusCode)
	case "page_content":
		return c.matchString(err.PageContent)
	default:
		return false
	}
}

func (c *RuleCondition) matchString(value string) bool {
	switch c.Operator {
	case "equals":
		return strings.EqualFold(value, c.Value)
	case "contains":
		return strings.Contains(strings.ToLower(value), strings.ToLower(c.Value))
	case "regex":
		matched, _ := regexp.MatchString(c.Value, value)
		return matched
	default:
		return false
	}
}

func (c *RuleCondition) matchInt(value int) bool {
	// Parse target value
	var target int
	_, _ = strings.NewReader(c.Value).Read([]byte{byte(target)})

	switch c.Operator {
	case "equals":
		return value == target
	case "gt":
		return value > target
	case "lt":
		return value < target
	default:
		return false
	}
}

// LearnedAction represents an AI action that was successful
// Used for learning and auto-promotion to rules
type LearnedAction struct {
	ID             string                 `json:"id" db:"id"`
	ExecutionID    string                 `json:"execution_id" db:"execution_id"`
	TaskID         string                 `json:"task_id" db:"task_id"`
	ErrorPattern   ErrorPattern           `json:"error_pattern" db:"error_pattern"`
	ErrorSignature string                 `json:"error_signature" db:"error_signature"` // Hash of error characteristics
	Domain         string                 `json:"domain" db:"domain"`
	Action         ActionType             `json:"action" db:"action"`
	ActionParams   map[string]interface{} `json:"action_params" db:"action_params"`
	AIReasoning    string                 `json:"ai_reasoning" db:"ai_reasoning"`
	Success        bool                   `json:"success" db:"success"`
	PromotedToRule bool                   `json:"promoted_to_rule" db:"promoted_to_rule"`
	RuleID         *string                `json:"rule_id,omitempty" db:"rule_id"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// Proxy represents a proxy server configuration
type Proxy struct {
	ID             string    `json:"id" db:"id"`
	ProxyID        string    `json:"proxy_id" db:"proxy_id"`
	Server         string    `json:"server" db:"server"`
	Username       string    `json:"username" db:"username"`
	Password       string    `json:"password" db:"password"`
	ProxyAddress   string    `json:"proxy_address" db:"proxy_address"`
	Port           int       `json:"port" db:"port"`
	Valid          bool      `json:"valid" db:"valid"`
	LastVerified   time.Time `json:"last_verification" db:"last_verified"`
	CountryCode    string    `json:"country_code" db:"country_code"`
	CityName       string    `json:"city_name" db:"city_name"`
	ASNName        string    `json:"asn_name" db:"asn_name"`
	ASNNumber      int       `json:"asn_number" db:"asn_number"`
	ConfidenceHigh bool      `json:"high_country_confidence" db:"confidence_high"`
	ProxyType      string    `json:"proxy_type" db:"proxy_type"` // static, rotating
	FailureCount   int       `json:"failure_count" db:"failure_count"`
	SuccessCount   int       `json:"success_count" db:"success_count"`
	LastUsed       time.Time `json:"last_used" db:"last_used"`
	IsHealthy      bool      `json:"is_healthy" db:"is_healthy"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ProxyURL returns the full proxy URL with authentication
func (p *Proxy) ProxyURL() string {
	if p.Username != "" && p.Password != "" {
		return "http://" + p.Username + ":" + p.Password + "@" + p.Server
	}
	return "http://" + p.Server
}

// DomainStatus tracks health status of a domain
type DomainStatus struct {
	Domain           string        `json:"domain"`
	FailureCount     int           `json:"failure_count"`
	SuccessCount     int           `json:"success_count"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	LastFailure      time.Time     `json:"last_failure"`
	LastSuccess      time.Time     `json:"last_success"`
	IsBlocked        bool          `json:"is_blocked"`
	BlockedUntil     time.Time     `json:"blocked_until"`
	RecommendedWait  time.Duration `json:"recommended_wait"`
	LastPattern      ErrorPattern  `json:"last_pattern"`
	WorkingProxies   []string      `json:"working_proxies"` // Proxy IDs that work for this domain
}

// RecoveryAttempt tracks a recovery attempt for logging/learning
type RecoveryAttempt struct {
	ID            string         `json:"id"`
	TaskID        string         `json:"task_id"`
	ExecutionID   string         `json:"execution_id"`
	DetectedError *DetectedError `json:"detected_error"`
	Plan          *RecoveryPlan  `json:"plan"`
	Success       bool           `json:"success"`
	Duration      time.Duration  `json:"duration"`
	Timestamp     time.Time      `json:"timestamp"`
}

// Manager is the main interface for the recovery system
type Manager interface {
	// TryRecover attempts to recover from an error
	// Returns a recovery plan if recovery is possible, nil otherwise
	TryRecover(ctx context.Context, taskID, executionID, url string, err error, pageContent string) (*RecoveryPlan, error)

	// RecordOutcome records whether a recovery attempt succeeded
	RecordOutcome(ctx context.Context, attempt *RecoveryAttempt) error

	// GetDomainStatus returns the health status of a domain
	GetDomainStatus(ctx context.Context, domain string) (*DomainStatus, error)

	// RefreshRules reloads rules from the database
	RefreshRules(ctx context.Context) error

	// Close cleans up resources
	Close() error
}
