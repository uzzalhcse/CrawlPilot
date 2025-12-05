package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// LearningSystem captures successful AI recoveries and promotes them to rules
type LearningSystem struct {
	pool               *pgxpool.Pool
	promotionThreshold int // Number of successes before promoting to rule
}

// NewLearningSystem creates a new learning system
func NewLearningSystem(pool *pgxpool.Pool) *LearningSystem {
	return &LearningSystem{
		pool:               pool,
		promotionThreshold: 3, // Promote to rule after 3 successes
	}
}

// RecordAIAction records an AI-suggested action for learning
func (l *LearningSystem) RecordAIAction(ctx context.Context, action *LearnedAction) error {
	// Generate error signature (hash of pattern + domain + action)
	action.ErrorSignature = l.generateSignature(action)

	query := `
		INSERT INTO learned_actions (
			id, execution_id, task_id, error_pattern, error_signature,
			domain, action, action_params, ai_reasoning, success,
			promoted_to_rule, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			false, NOW()
		)
	`

	paramsJSON, _ := json.Marshal(action.ActionParams)

	_, err := l.pool.Exec(ctx, query,
		action.ID, action.ExecutionID, action.TaskID,
		string(action.ErrorPattern), action.ErrorSignature,
		action.Domain, string(action.Action), paramsJSON,
		action.AIReasoning, action.Success,
	)
	if err != nil {
		return fmt.Errorf("failed to record learned action: %w", err)
	}

	// Check if we should promote to a rule
	if action.Success {
		if err := l.checkAndPromote(ctx, action); err != nil {
			logger.Warn("Failed to check promotion", zap.Error(err))
		}
	}

	return nil
}

// UpdateOutcome updates the success/failure of a learned action
func (l *LearningSystem) UpdateOutcome(ctx context.Context, actionID string, success bool) error {
	query := `UPDATE learned_actions SET success = $1 WHERE id = $2`
	_, err := l.pool.Exec(ctx, query, success, actionID)
	if err != nil {
		return fmt.Errorf("failed to update outcome: %w", err)
	}

	// Check promotion if successful
	if success {
		var action LearnedAction
		getQuery := `SELECT error_pattern, error_signature, domain, action, action_params, ai_reasoning
		             FROM learned_actions WHERE id = $1`
		var paramsJSON []byte
		err := l.pool.QueryRow(ctx, getQuery, actionID).Scan(
			&action.ErrorPattern, &action.ErrorSignature, &action.Domain,
			&action.Action, &paramsJSON, &action.AIReasoning,
		)
		if err == nil {
			json.Unmarshal(paramsJSON, &action.ActionParams)
			l.checkAndPromote(ctx, &action)
		}
	}

	return nil
}

// generateSignature creates a unique signature for an error-action pair
func (l *LearningSystem) generateSignature(action *LearnedAction) string {
	data := fmt.Sprintf("%s:%s:%s", action.ErrorPattern, action.Domain, action.Action)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // First 16 bytes
}

// checkAndPromote checks if an action should be promoted to a rule
func (l *LearningSystem) checkAndPromote(ctx context.Context, action *LearnedAction) error {
	// Count successful actions with same signature
	query := `
		SELECT COUNT(*) FROM learned_actions 
		WHERE error_signature = $1 AND success = true AND promoted_to_rule = false
	`
	var count int
	err := l.pool.QueryRow(ctx, query, action.ErrorSignature).Scan(&count)
	if err != nil {
		return err
	}

	if count >= l.promotionThreshold {
		return l.promoteToRule(ctx, action)
	}

	return nil
}

// promoteToRule creates a new rule from learned actions
func (l *LearningSystem) promoteToRule(ctx context.Context, action *LearnedAction) error {
	// Create the rule
	ruleID := generateID()

	// Build conditions based on the action
	conditions := []RuleCondition{
		{Field: "domain", Operator: "contains", Value: extractBaseDomain(action.Domain)},
	}
	conditionsJSON, _ := json.Marshal(conditions)
	paramsJSON, _ := json.Marshal(action.ActionParams)

	query := `
		INSERT INTO recovery_rules (
			id, name, description, priority, enabled, pattern,
			conditions, action, action_params, max_retries, retry_delay,
			is_learned, learned_from, success_count, failure_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, 100, true, $4,
			$5, $6, $7, 3, 5,
			true, $8, $9, 0,
			NOW(), NOW()
		)
	`

	ruleName := fmt.Sprintf("Learned: %s for %s", action.Action, action.Domain)
	ruleDesc := fmt.Sprintf("Auto-learned from AI. Reasoning: %s", truncateContent(action.AIReasoning, 200))

	_, err := l.pool.Exec(ctx, query,
		ruleID, ruleName, ruleDesc, string(action.ErrorPattern),
		conditionsJSON, string(action.Action), paramsJSON,
		action.ErrorSignature, l.promotionThreshold,
	)
	if err != nil {
		return fmt.Errorf("failed to create rule: %w", err)
	}

	// Mark all matching learned actions as promoted
	updateQuery := `
		UPDATE learned_actions 
		SET promoted_to_rule = true, rule_id = $1 
		WHERE error_signature = $2 AND success = true
	`
	_, err = l.pool.Exec(ctx, updateQuery, ruleID, action.ErrorSignature)
	if err != nil {
		logger.Warn("Failed to mark actions as promoted", zap.Error(err))
	}

	logger.Info("Promoted AI action to rule",
		zap.String("rule_id", ruleID),
		zap.String("pattern", string(action.ErrorPattern)),
		zap.String("domain", action.Domain),
		zap.String("action", string(action.Action)),
	)

	return nil
}

// GetLearnedStats returns statistics about learned actions
func (l *LearningSystem) GetLearnedStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total learned actions
	var totalCount, successCount, promotedCount int
	err := l.pool.QueryRow(ctx, `SELECT COUNT(*) FROM learned_actions`).Scan(&totalCount)
	if err == nil {
		stats["total_actions"] = totalCount
	}

	err = l.pool.QueryRow(ctx, `SELECT COUNT(*) FROM learned_actions WHERE success = true`).Scan(&successCount)
	if err == nil {
		stats["successful_actions"] = successCount
	}

	err = l.pool.QueryRow(ctx, `SELECT COUNT(*) FROM learned_actions WHERE promoted_to_rule = true`).Scan(&promotedCount)
	if err == nil {
		stats["promoted_to_rules"] = promotedCount
	}

	// Auto-learned rules
	var autoRulesCount int
	err = l.pool.QueryRow(ctx, `SELECT COUNT(*) FROM recovery_rules WHERE is_learned = true`).Scan(&autoRulesCount)
	if err == nil {
		stats["auto_learned_rules"] = autoRulesCount
	}

	return stats, nil
}

// CleanupOldActions removes old learned actions that weren't successful
func (l *LearningSystem) CleanupOldActions(ctx context.Context, olderThan time.Duration) (int, error) {
	query := `
		DELETE FROM learned_actions 
		WHERE success = false AND created_at < $1 AND promoted_to_rule = false
	`
	cutoff := time.Now().Add(-olderThan)
	result, err := l.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// extractBaseDomain extracts the base domain from a host
func extractBaseDomain(host string) string {
	// Simple extraction - remove common subdomains
	// In production, use a proper public suffix list
	parts := []string{}
	for _, p := range splitHost(host) {
		if p != "www" && p != "m" && p != "mobile" {
			parts = append(parts, p)
		}
	}
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return host
}

func splitHost(host string) []string {
	result := []string{}
	current := ""
	for _, c := range host {
		if c == '.' || c == ':' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			if c == ':' {
				break // Stop at port
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
