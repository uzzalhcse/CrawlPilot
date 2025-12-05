package recovery

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RuleEngine manages and matches recovery rules
type RuleEngine struct {
	pool        *pgxpool.Pool
	rules       []*RecoveryRule
	mu          sync.RWMutex
	lastRefresh time.Time
	cacheTTL    time.Duration
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(pool *pgxpool.Pool) (*RuleEngine, error) {
	re := &RuleEngine{
		pool:     pool,
		rules:    make([]*RecoveryRule, 0),
		cacheTTL: 5 * time.Minute,
	}

	if err := re.Refresh(context.Background()); err != nil {
		return nil, err
	}

	return re, nil
}

// Refresh reloads rules from the database
func (re *RuleEngine) Refresh(ctx context.Context) error {
	query := `
		SELECT id, name, description, priority, enabled, pattern,
		       conditions, action, action_params, max_retries, retry_delay,
		       is_learned, learned_from, success_count, failure_count,
		       created_at, updated_at
		FROM recovery_rules
		WHERE enabled = true
		ORDER BY priority ASC
	`

	rows, err := re.pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	rules := make([]*RecoveryRule, 0)

	for rows.Next() {
		r := &RecoveryRule{}
		var conditionsJSON, paramsJSON []byte
		var learnedFrom *string // Handle NULL

		err := rows.Scan(
			&r.ID, &r.Name, &r.Description, &r.Priority, &r.Enabled,
			&r.Pattern, &conditionsJSON, &r.Action, &paramsJSON,
			&r.MaxRetries, &r.RetryDelay, &r.IsLearned, &learnedFrom,
			&r.SuccessCount, &r.FailureCount, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			logger.Warn("Failed to scan rule", zap.Error(err))
			continue
		}

		// Handle nullable learned_from
		if learnedFrom != nil {
			r.LearnedFrom = *learnedFrom
		}

		// Parse JSON fields
		json.Unmarshal(conditionsJSON, &r.Conditions)
		json.Unmarshal(paramsJSON, &r.ActionParams)

		rules = append(rules, r)
	}

	// Sort by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	re.mu.Lock()
	re.rules = rules
	re.lastRefresh = time.Now()
	re.mu.Unlock()

	logger.Info("Loaded recovery rules",
		zap.Int("count", len(rules)),
		zap.Int("learned", countLearned(rules)),
	)

	return nil
}

// Match finds the best matching rule for an error
func (re *RuleEngine) Match(ctx context.Context, err *DetectedError) *RecoveryRule {
	// Check if cache needs refresh
	if time.Since(re.lastRefresh) > re.cacheTTL {
		go re.Refresh(ctx) // Async refresh
	}

	re.mu.RLock()
	defer re.mu.RUnlock()

	for _, rule := range re.rules {
		if re.matchRule(rule, err) {
			logger.Debug("Rule matched",
				zap.String("rule_id", rule.ID),
				zap.String("rule_name", rule.Name),
				zap.String("error_pattern", string(err.Pattern)),
			)
			return rule
		}
	}

	return nil
}

// matchRule checks if a rule matches an error
func (re *RuleEngine) matchRule(rule *RecoveryRule, err *DetectedError) bool {
	// Pattern must match
	if rule.Pattern != err.Pattern && rule.Pattern != "" {
		return false
	}

	// All conditions must match
	for _, cond := range rule.Conditions {
		if !cond.Match(err) {
			return false
		}
	}

	return true
}

// RecordRuleOutcome records the success/failure of a rule execution
func (re *RuleEngine) RecordRuleOutcome(ctx context.Context, ruleID string, success bool) error {
	var field string
	if success {
		field = "success_count"
	} else {
		field = "failure_count"
	}

	query := "UPDATE recovery_rules SET " + field + " = " + field + " + 1, updated_at = NOW() WHERE id = $1"
	_, err := re.pool.Exec(ctx, query, ruleID)
	return err
}

// CreateRule creates a new rule (from frontend)
func (re *RuleEngine) CreateRule(ctx context.Context, rule *RecoveryRule) error {
	if rule.ID == "" {
		rule.ID = generateID()
	}

	conditionsJSON, _ := json.Marshal(rule.Conditions)
	paramsJSON, _ := json.Marshal(rule.ActionParams)

	query := `
		INSERT INTO recovery_rules (
			id, name, description, priority, enabled, pattern,
			conditions, action, action_params, max_retries, retry_delay,
			is_learned, success_count, failure_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			false, 0, 0, NOW(), NOW()
		)
	`

	_, err := re.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.Priority, rule.Enabled,
		string(rule.Pattern), conditionsJSON, string(rule.Action), paramsJSON,
		rule.MaxRetries, rule.RetryDelay,
	)
	if err != nil {
		return err
	}

	// Refresh cache
	return re.Refresh(ctx)
}

// UpdateRule updates an existing rule
func (re *RuleEngine) UpdateRule(ctx context.Context, rule *RecoveryRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	paramsJSON, _ := json.Marshal(rule.ActionParams)

	query := `
		UPDATE recovery_rules SET
			name = $2, description = $3, priority = $4, enabled = $5,
			pattern = $6, conditions = $7, action = $8, action_params = $9,
			max_retries = $10, retry_delay = $11, updated_at = NOW()
		WHERE id = $1
	`

	_, err := re.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.Priority, rule.Enabled,
		string(rule.Pattern), conditionsJSON, string(rule.Action), paramsJSON,
		rule.MaxRetries, rule.RetryDelay,
	)
	if err != nil {
		return err
	}

	return re.Refresh(ctx)
}

// DeleteRule deletes a rule
func (re *RuleEngine) DeleteRule(ctx context.Context, ruleID string) error {
	query := `DELETE FROM recovery_rules WHERE id = $1`
	_, err := re.pool.Exec(ctx, query, ruleID)
	if err != nil {
		return err
	}
	return re.Refresh(ctx)
}

// GetAllRules returns all rules for the frontend
func (re *RuleEngine) GetAllRules(ctx context.Context) ([]*RecoveryRule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	// Return a copy
	result := make([]*RecoveryRule, len(re.rules))
	copy(result, re.rules)
	return result, nil
}

// ToRecoveryPlan converts a matched rule to a recovery plan
func (re *RuleEngine) ToRecoveryPlan(rule *RecoveryRule) *RecoveryPlan {
	return &RecoveryPlan{
		Action:      rule.Action,
		Params:      rule.ActionParams,
		Reason:      "Matched rule: " + rule.Name,
		ShouldRetry: rule.Action != ActionSendToDLQ && rule.Action != ActionSkipDomain,
		RetryDelay:  time.Duration(rule.RetryDelay) * time.Second,
		Source:      sourceFromRule(rule),
		RuleID:      rule.ID,
	}
}

func sourceFromRule(rule *RecoveryRule) string {
	if rule.IsLearned {
		return "learned"
	}
	return "rule"
}

func countLearned(rules []*RecoveryRule) int {
	count := 0
	for _, r := range rules {
		if r.IsLearned {
			count++
		}
	}
	return count
}
