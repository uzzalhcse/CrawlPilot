package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/internal/error_recovery"
)

// RecoveryHistoryRecord represents a single error recovery event
type RecoveryHistoryRecord struct {
	ID          uuid.UUID `json:"id"`
	ExecutionID uuid.UUID `json:"execution_id"`
	WorkflowID  uuid.UUID `json:"workflow_id"`

	// Error Details
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
	StatusCode   *int   `json:"status_code,omitempty"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	NodeID       string `json:"node_id"`
	PhaseID      string `json:"phase_id"`

	// Pattern Analysis
	PatternDetected  bool     `json:"pattern_detected"`
	PatternType      string   `json:"pattern_type,omitempty"`
	ActivationReason string   `json:"activation_reason,omitempty"`
	ErrorRate        *float64 `json:"error_rate,omitempty"`

	// Recovery Solution
	RuleID       *uuid.UUID `json:"rule_id,omitempty"`
	RuleName     string     `json:"rule_name,omitempty"`
	SolutionType string     `json:"solution_type"` // 'rule', 'ai', 'none'
	Confidence   *float64   `json:"confidence,omitempty"`

	// Actions Applied
	ActionsApplied []error_recovery.Action `json:"actions_applied,omitempty"`

	// Outcome
	RecoveryAttempted  bool `json:"recovery_attempted"`
	RecoverySuccessful bool `json:"recovery_successful"`
	RetryCount         int  `json:"retry_count"`
	TimeToRecoveryMs   int  `json:"time_to_recovery_ms"`

	// Context
	RequestContext map[string]interface{} `json:"request_context,omitempty"`

	// Timestamps
	DetectedAt  time.Time  `json:"detected_at"`
	RecoveredAt *time.Time `json:"recovered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// RecoveryStats represents aggregated recovery statistics
type RecoveryStats struct {
	TotalAttempts        int                       `json:"total_attempts"`
	SuccessfulRecoveries int                       `json:"successful_recoveries"`
	SuccessRate          float64                   `json:"success_rate"`
	AvgRecoveryTimeMs    float64                   `json:"avg_recovery_time_ms"`
	ByErrorType          map[string]ErrorTypeStats `json:"by_error_type"`
	ByRule               map[string]RuleStats      `json:"by_rule"`
	ByDomain             map[string]DomainStats    `json:"by_domain"`
	Timeline             []TimelinePoint           `json:"timeline"`
}

type ErrorTypeStats struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

type RuleStats struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	AvgTime     float64 `json:"avg_time_ms"`
}

type DomainStats struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
}

type TimelinePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Attempts  int       `json:"attempts"`
	Successes int       `json:"successes"`
}

// StatsFilter defines filtering options for statistics
type StatsFilter struct {
	StartTime  *time.Time
	EndTime    *time.Time
	WorkflowID *uuid.UUID
	Domain     *string
	ErrorType  *string
}

// ErrorRecoveryHistoryRepository handles recovery history persistence
type ErrorRecoveryHistoryRepository struct {
	db *PostgresDB
}

// NewErrorRecoveryHistoryRepository creates a new repository
func NewErrorRecoveryHistoryRepository(db *PostgresDB) *ErrorRecoveryHistoryRepository {
	return &ErrorRecoveryHistoryRepository{db: db}
}

// Create saves a new recovery history record
func (r *ErrorRecoveryHistoryRepository) Create(ctx context.Context, record *RecoveryHistoryRecord) error {
	// Marshal actions and context to JSON
	actionsJSON, err := json.Marshal(record.ActionsApplied)
	if err != nil {
		return err
	}

	contextJSON, err := json.Marshal(record.RequestContext)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO error_recovery_history (
			execution_id, workflow_id, error_type, error_message, status_code,
			url, domain, node_id, phase_id, pattern_detected, pattern_type,
			activation_reason, error_rate, rule_id, rule_name, solution_type,
			confidence, actions_applied, recovery_attempted, recovery_successful,
			retry_count, time_to_recovery_ms, request_context, detected_at, recovered_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		) RETURNING id, created_at`

	err = r.db.Pool.QueryRow(ctx, query,
		record.ExecutionID, record.WorkflowID, record.ErrorType, record.ErrorMessage,
		record.StatusCode, record.URL, record.Domain, record.NodeID, record.PhaseID,
		record.PatternDetected, record.PatternType, record.ActivationReason, record.ErrorRate,
		record.RuleID, record.RuleName, record.SolutionType, record.Confidence,
		actionsJSON, record.RecoveryAttempted, record.RecoverySuccessful,
		record.RetryCount, record.TimeToRecoveryMs, contextJSON, record.DetectedAt,
		record.RecoveredAt,
	).Scan(&record.ID, &record.CreatedAt)

	return err
}

// GetByExecutionID retrieves all recovery events for an execution
func (r *ErrorRecoveryHistoryRepository) GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]RecoveryHistoryRecord, error) {
	query := `
		SELECT id, execution_id, workflow_id, error_type, error_message, status_code,
		       url, domain, node_id, phase_id, pattern_detected, pattern_type,
		       activation_reason, error_rate, rule_id, rule_name, solution_type,
		       confidence, actions_applied, recovery_attempted, recovery_successful,
		       retry_count, time_to_recovery_ms, request_context, detected_at,
		       recovered_at, created_at
		FROM error_recovery_history
		WHERE execution_id = $1
		ORDER BY detected_at ASC`

	rows, err := r.db.Pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

// GetByWorkflowID retrieves recent recovery events for a workflow
func (r *ErrorRecoveryHistoryRepository) GetByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit int) ([]RecoveryHistoryRecord, error) {
	query := `
		SELECT id, execution_id, workflow_id, error_type, error_message, status_code,
		       url, domain, node_id, phase_id, pattern_detected, pattern_type,
		       activation_reason, error_rate, rule_id, rule_name, solution_type,
		       confidence, actions_applied, recovery_attempted, recovery_successful,
		       retry_count, time_to_recovery_ms, request_context, detected_at,
		       recovered_at, created_at
		FROM error_recovery_history
		WHERE workflow_id = $1
		ORDER BY detected_at DESC
		LIMIT $2`

	rows, err := r.db.Pool.Query(ctx, query, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

// GetRecent retrieves the most recent recovery events
func (r *ErrorRecoveryHistoryRepository) GetRecent(ctx context.Context, limit int) ([]RecoveryHistoryRecord, error) {
	query := `
		SELECT id, execution_id, workflow_id, error_type, error_message, status_code,
		       url, domain, node_id, phase_id, pattern_detected, pattern_type,
		       activation_reason, error_rate, rule_id, rule_name, solution_type,
		       confidence, actions_applied, recovery_attempted, recovery_successful,
		       retry_count, time_to_recovery_ms, request_context, detected_at,
		       recovered_at, created_at
		FROM error_recovery_history
		ORDER BY detected_at DESC
		LIMIT $1`

	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

// GetStats retrieves aggregated statistics
func (r *ErrorRecoveryHistoryRepository) GetStats(ctx context.Context, filter StatsFilter) (*RecoveryStats, error) {
	stats := &RecoveryStats{
		ByErrorType: make(map[string]ErrorTypeStats),
		ByRule:      make(map[string]RuleStats),
		ByDomain:    make(map[string]DomainStats),
		Timeline:    []TimelinePoint{},
	}

	// Build WHERE clause
	where := "WHERE recovery_attempted = true"
	args := []interface{}{}
	argPos := 1

	if filter.StartTime != nil {
		where += ` AND detected_at >= $` + string(rune('0'+argPos))
		args = append(args, *filter.StartTime)
		argPos++
	}
	if filter.EndTime != nil {
		where += ` AND detected_at <= $` + string(rune('0'+argPos))
		args = append(args, *filter.EndTime)
		argPos++
	}
	if filter.WorkflowID != nil {
		where += ` AND workflow_id = $` + string(rune('0'+argPos))
		args = append(args, *filter.WorkflowID)
		argPos++
	}

	// Get overall stats
	overallQuery := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN recovery_successful THEN 1 ELSE 0 END) as successes,
			AVG(time_to_recovery_ms) as avg_time
		FROM error_recovery_history
		` + where

	err := r.db.Pool.QueryRow(ctx, overallQuery, args...).Scan(
		&stats.TotalAttempts,
		&stats.SuccessfulRecoveries,
		&stats.AvgRecoveryTimeMs,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	if stats.TotalAttempts > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRecoveries) / float64(stats.TotalAttempts)
	}

	// Get by error type
	errorTypeQuery := `
		SELECT 
			error_type,
			COUNT(*) as count,
			SUM(CASE WHEN recovery_successful THEN 1 ELSE 0 END)::FLOAT / COUNT(*) as success_rate
		FROM error_recovery_history
		` + where + `
		GROUP BY error_type`

	rows, err := r.db.Pool.Query(ctx, errorTypeQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var errorType string
		var count int
		var successRate float64
		if err := rows.Scan(&errorType, &count, &successRate); err != nil {
			continue
		}
		stats.ByErrorType[errorType] = ErrorTypeStats{Count: count, SuccessRate: successRate}
	}

	// Get by rule
	ruleQuery := `
		SELECT 
			rule_name,
			COUNT(*) as count,
			SUM(CASE WHEN recovery_successful THEN 1 ELSE 0 END)::FLOAT / COUNT(*) as success_rate,
			AVG(time_to_recovery_ms) as avg_time
		FROM error_recovery_history
		` + where + ` AND rule_name IS NOT NULL
		GROUP BY rule_name`

	rows2, err := r.db.Pool.Query(ctx, ruleQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var ruleName string
		var count int
		var successRate, avgTime float64
		if err := rows2.Scan(&ruleName, &count, &successRate, &avgTime); err != nil {
			continue
		}
		stats.ByRule[ruleName] = RuleStats{Count: count, SuccessRate: successRate, AvgTime: avgTime}
	}

	return stats, nil
}

// scanRecords scans multiple recovery records from rows
func (r *ErrorRecoveryHistoryRepository) scanRecords(rows pgx.Rows) ([]RecoveryHistoryRecord, error) {
	var records []RecoveryHistoryRecord

	for rows.Next() {
		var record RecoveryHistoryRecord
		var actionsJSON, contextJSON []byte

		err := rows.Scan(
			&record.ID, &record.ExecutionID, &record.WorkflowID, &record.ErrorType,
			&record.ErrorMessage, &record.StatusCode, &record.URL, &record.Domain,
			&record.NodeID, &record.PhaseID, &record.PatternDetected, &record.PatternType,
			&record.ActivationReason, &record.ErrorRate, &record.RuleID, &record.RuleName,
			&record.SolutionType, &record.Confidence, &actionsJSON, &record.RecoveryAttempted,
			&record.RecoverySuccessful, &record.RetryCount, &record.TimeToRecoveryMs,
			&contextJSON, &record.DetectedAt, &record.RecoveredAt, &record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if len(actionsJSON) > 0 {
			json.Unmarshal(actionsJSON, &record.ActionsApplied)
		}
		if len(contextJSON) > 0 {
			json.Unmarshal(contextJSON, &record.RequestContext)
		}

		records = append(records, record)
	}

	return records, rows.Err()
}
