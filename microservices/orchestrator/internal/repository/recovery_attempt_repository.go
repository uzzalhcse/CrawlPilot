package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
)

// RecoveryAttemptRepository handles recovery attempt CRUD operations
type RecoveryAttemptRepository struct {
	db *database.DB
}

// RecoveryAttempt represents a recovery attempt from the recovery_attempts table
type RecoveryAttempt struct {
	ID            string    `json:"id"`
	ExecutionID   string    `json:"execution_id"`
	TaskID        string    `json:"task_id"`
	WorkflowID    string    `json:"workflow_id"`
	URL           string    `json:"url"`
	Domain        string    `json:"domain"`
	ErrorPattern  string    `json:"error_pattern"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	StatusCode    int       `json:"status_code,omitempty"`
	Action        string    `json:"action,omitempty"`
	Source        string    `json:"source"` // rule, ai, default, none, pending
	RuleID        string    `json:"rule_id,omitempty"`
	AIReasoning   string    `json:"ai_reasoning,omitempty"`
	Status        string    `json:"status"` // pending, success, failed
	RetryDelayMs  int       `json:"retry_delay_ms,omitempty"`
	DurationMs    int       `json:"duration_ms,omitempty"`
	ProxyID       string    `json:"proxy_id,omitempty"`
	ProxyTier     int       `json:"proxy_tier,omitempty"`
	TierFrom      int       `json:"tier_from,omitempty"`
	TierTo        int       `json:"tier_to,omitempty"`
	Confidence    float64   `json:"confidence,omitempty"`
	TriggerReason string    `json:"trigger_reason,omitempty"`
	RetryCount    int       `json:"retry_count,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewRecoveryAttemptRepository creates a new recovery attempt repository
func NewRecoveryAttemptRepository(db *database.DB) *RecoveryAttemptRepository {
	return &RecoveryAttemptRepository{db: db}
}

// CreateAttempt creates a new recovery attempt record (called immediately on error detection)
func (r *RecoveryAttemptRepository) CreateAttempt(ctx context.Context, attempt *RecoveryAttempt) error {
	// Default to 'pending' if no status provided
	status := attempt.Status
	if status == "" {
		status = "pending"
	}
	query := `
		INSERT INTO recovery_attempts (
			execution_id, task_id, workflow_id, url, domain,
			error_pattern, error_message, status_code, status, source,
			confidence, trigger_reason
		) VALUES (
			$1, $2, NULLIF($3, '')::uuid, $4, $5,
			$6, $7, $8, $9, 'pending',
			$10, NULLIF($11, '')
		) RETURNING id, created_at
	`

	return r.db.Pool.QueryRow(ctx, query,
		attempt.ExecutionID, attempt.TaskID, attempt.WorkflowID, attempt.URL, attempt.Domain,
		attempt.ErrorPattern, attempt.ErrorMessage, attempt.StatusCode, status,
		attempt.Confidence, attempt.TriggerReason,
	).Scan(&attempt.ID, &attempt.CreatedAt)
}

// CreateAttemptsBatch creates multiple recovery attempts in a single transaction (high-throughput)
// Uses multi-value INSERT for better performance than individual inserts
func (r *RecoveryAttemptRepository) CreateAttemptsBatch(ctx context.Context, attempts []RecoveryAttempt) (int, error) {
	if len(attempts) == 0 {
		return 0, nil
	}

	// Build multi-value INSERT query
	// Format: INSERT INTO table (...) VALUES ($1,$2,...), ($n+1,$n+2,...), ...
	query := `
		INSERT INTO recovery_attempts (
			execution_id, task_id, workflow_id, url, domain,
			error_pattern, error_message, status_code, status, source,
			confidence, trigger_reason
		) VALUES `

	paramsPerRow := 11 // Added status as dynamic parameter
	args := make([]interface{}, 0, len(attempts)*paramsPerRow)
	valueStrings := make([]string, 0, len(attempts))

	for i, attempt := range attempts {
		base := i * paramsPerRow
		// Default to 'detected' if no status provided (for below-threshold errors)
		status := attempt.Status
		if status == "" {
			status = "detected"
		}
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d, $%d, NULLIF($%d, '')::uuid, $%d, $%d, $%d, $%d, $%d, $%d, 'pending', $%d, NULLIF($%d, ''))",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11))
		args = append(args,
			attempt.ExecutionID,
			attempt.TaskID,
			attempt.WorkflowID,
			attempt.URL,
			attempt.Domain,
			attempt.ErrorPattern,
			attempt.ErrorMessage,
			attempt.StatusCode,
			status,
			attempt.Confidence,
			attempt.TriggerReason,
		)
	}

	query += strings.Join(valueStrings, ", ")

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

// UpdateAttempt updates a recovery attempt with the outcome
func (r *RecoveryAttemptRepository) UpdateAttempt(ctx context.Context, id string, action, source, status, ruleID, aiReasoning string, retryDelayMs, durationMs int, proxyID string, proxyTier, tierFrom, tierTo, retryCount int) error {
	query := `
		UPDATE recovery_attempts SET
			action = $1,
			source = $2,
			status = $3,
			rule_id = NULLIF($4, ''),
			ai_reasoning = NULLIF($5, ''),
			retry_delay_ms = NULLIF($6, 0),
			duration_ms = NULLIF($7, 0),
			proxy_id = NULLIF($8, ''),
			proxy_tier = NULLIF($9, 0),
			tier_from = NULLIF($10, 0),
			tier_to = NULLIF($11, 0),
			retry_count = NULLIF($12, 0)
		WHERE id = $13
	`

	_, err := r.db.Pool.Exec(ctx, query, action, source, status, ruleID, aiReasoning, retryDelayMs, durationMs, proxyID, proxyTier, tierFrom, tierTo, retryCount, id)
	return err
}

// UpdateAttemptBatchItem represents a single update in a batch
type UpdateAttemptBatchItem struct {
	ID           string
	Action       string
	Source       string
	Status       string
	RuleID       string
	AIReasoning  string
	RetryDelayMs int
	DurationMs   int
	ProxyID      string
	ProxyTier    int
	TierFrom     int
	TierTo       int
	RetryCount   int
}

// UpdateAttemptsBatch updates multiple recovery attempts in a single query (high-throughput)
// Uses UPDATE FROM VALUES pattern for O(1) DB queries instead of O(N)
func (r *RecoveryAttemptRepository) UpdateAttemptsBatch(ctx context.Context, updates []UpdateAttemptBatchItem) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	// Build the VALUES clause with proper parameter indexing
	// Each row has 13 parameters
	paramsPerRow := 13
	valueStrings := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)*paramsPerRow)

	for i, u := range updates {
		base := i * paramsPerRow
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d::uuid, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11, base+12, base+13))
		args = append(args,
			u.ID,
			u.Action,
			u.Source,
			u.Status,
			u.RuleID,
			u.AIReasoning,
			u.RetryDelayMs,
			u.DurationMs,
			u.ProxyID,
			u.ProxyTier,
			u.TierFrom,
			u.TierTo,
			u.RetryCount,
		)
	}

	query := fmt.Sprintf(`
		UPDATE recovery_attempts AS t SET
			action = COALESCE(NULLIF(v.action, ''), t.action),
			source = COALESCE(NULLIF(v.source, ''), t.source),
			status = v.status,
			rule_id = COALESCE(NULLIF(v.rule_id, ''), t.rule_id),
			ai_reasoning = COALESCE(NULLIF(v.ai_reasoning, ''), t.ai_reasoning),
			retry_delay_ms = COALESCE(NULLIF(v.retry_delay_ms, 0), t.retry_delay_ms),
			duration_ms = COALESCE(NULLIF(v.duration_ms, 0), t.duration_ms),
			proxy_id = COALESCE(NULLIF(v.proxy_id, ''), t.proxy_id),
			proxy_tier = COALESCE(NULLIF(v.proxy_tier, 0), t.proxy_tier),
			tier_from = COALESCE(NULLIF(v.tier_from, 0), t.tier_from),
			tier_to = COALESCE(NULLIF(v.tier_to, 0), t.tier_to),
			retry_count = COALESCE(NULLIF(v.retry_count, 0), t.retry_count)
		FROM (VALUES %s) AS v(id, action, source, status, rule_id, ai_reasoning, retry_delay_ms, duration_ms, proxy_id, proxy_tier, tier_from, tier_to, retry_count)
		WHERE t.id = v.id
	`, strings.Join(valueStrings, ", "))

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

// GetAttempts returns recovery attempts with filtering
func (r *RecoveryAttemptRepository) GetAttempts(ctx context.Context, executionID, status, pattern string, limit, offset int) ([]RecoveryAttempt, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM recovery_attempts WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if executionID != "" {
		countQuery += fmt.Sprintf(` AND execution_id = $%d`, argNum)
		args = append(args, executionID)
		argNum++
	}
	if status != "" {
		countQuery += fmt.Sprintf(` AND status = $%d`, argNum)
		args = append(args, status)
		argNum++
	}
	if pattern != "" {
		countQuery += fmt.Sprintf(` AND error_pattern = $%d`, argNum)
		args = append(args, pattern)
		argNum++
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Main query
	query := `SELECT id, execution_id, task_id, workflow_id, url, domain,
	          error_pattern, error_message, COALESCE(status_code, 0),
	          COALESCE(action, ''), source, COALESCE(rule_id, ''),
	          COALESCE(ai_reasoning, ''), status,
	          COALESCE(retry_delay_ms, 0), COALESCE(duration_ms, 0),
	          COALESCE(proxy_id, ''), COALESCE(proxy_tier, 0),
	          COALESCE(tier_from, 0), COALESCE(tier_to, 0),
	          COALESCE(confidence, 0), COALESCE(trigger_reason, ''),
	          COALESCE(retry_count, 0),
	          created_at, updated_at
	          FROM recovery_attempts WHERE 1=1`

	args = []interface{}{}
	argNum = 1

	if executionID != "" {
		query += fmt.Sprintf(` AND execution_id = $%d`, argNum)
		args = append(args, executionID)
		argNum++
	}
	if status != "" {
		query += fmt.Sprintf(` AND status = $%d`, argNum)
		args = append(args, status)
		argNum++
	}
	if pattern != "" {
		query += fmt.Sprintf(` AND error_pattern = $%d`, argNum)
		args = append(args, pattern)
		argNum++
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	attempts := make([]RecoveryAttempt, 0)
	for rows.Next() {
		var a RecoveryAttempt
		if err := rows.Scan(
			&a.ID, &a.ExecutionID, &a.TaskID, &a.WorkflowID, &a.URL, &a.Domain,
			&a.ErrorPattern, &a.ErrorMessage, &a.StatusCode,
			&a.Action, &a.Source, &a.RuleID,
			&a.AIReasoning, &a.Status,
			&a.RetryDelayMs, &a.DurationMs,
			&a.ProxyID, &a.ProxyTier,
			&a.TierFrom, &a.TierTo,
			&a.Confidence, &a.TriggerReason,
			&a.RetryCount,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			continue
		}
		attempts = append(attempts, a)
	}

	return attempts, total, nil
}

// GetStats returns aggregate statistics for recovery attempts
func (r *RecoveryAttemptRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE status = 'pending') as pending,
	          COUNT(*) FILTER (WHERE status = 'success') as success,
	          COUNT(*) FILTER (WHERE status = 'failed') as failed,
	          COUNT(*) FILTER (WHERE source = 'rule') as from_rules,
	          COUNT(*) FILTER (WHERE source = 'ai') as from_ai,
	          COUNT(*) FILTER (WHERE source = 'default') as from_default,
	          COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 hour') as last_hour,
	          COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as last_24h
	          FROM recovery_attempts`

	row := r.db.Pool.QueryRow(ctx, query)

	var total, pending, success, failed, fromRules, fromAI, fromDefault, lastHour, last24h int
	if err := row.Scan(&total, &pending, &success, &failed, &fromRules, &fromAI, &fromDefault, &lastHour, &last24h); err != nil {
		return nil, err
	}

	successRate := 0.0
	if total-pending > 0 {
		successRate = float64(success) / float64(total-pending) * 100
	}

	return map[string]interface{}{
		"total":        total,
		"pending":      pending,
		"success":      success,
		"failed":       failed,
		"success_rate": successRate,
		"from_rules":   fromRules,
		"from_ai":      fromAI,
		"from_default": fromDefault,
		"last_hour":    lastHour,
		"last_24h":     last24h,
	}, nil
}

// GetPatternStats returns recovery attempts grouped by error pattern
func (r *RecoveryAttemptRepository) GetPatternStats(ctx context.Context) ([]map[string]interface{}, error) {
	query := `SELECT error_pattern, 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE status = 'success') as success,
	          COUNT(*) FILTER (WHERE status = 'failed') as failed
	          FROM recovery_attempts
	          GROUP BY error_pattern
	          ORDER BY total DESC
	          LIMIT 10`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]map[string]interface{}, 0)
	for rows.Next() {
		var pattern string
		var total, success, failed int
		if err := rows.Scan(&pattern, &total, &success, &failed); err != nil {
			continue
		}
		successRate := 0.0
		if total > 0 {
			successRate = float64(success) / float64(total) * 100
		}
		stats = append(stats, map[string]interface{}{
			"pattern":      pattern,
			"total":        total,
			"success":      success,
			"failed":       failed,
			"success_rate": successRate,
		})
	}
	return stats, nil
}
