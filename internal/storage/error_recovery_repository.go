package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/internal/error_recovery"
)

type ErrorRecoveryRepository struct {
	db *PostgresDB
}

func NewErrorRecoveryRepository(db *PostgresDB) *ErrorRecoveryRepository {
	return &ErrorRecoveryRepository{db: db}
}

// GetConfig retrieves a configuration value by key
func (r *ErrorRecoveryRepository) GetConfig(ctx context.Context, key string) (interface{}, error) {
	var value interface{}
	err := r.db.Pool.QueryRow(ctx, "SELECT config_value FROM error_recovery_configs WHERE config_key = $1", key).Scan(&value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return value, nil
}

// UpdateConfig updates or inserts a configuration value
func (r *ErrorRecoveryRepository) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO error_recovery_configs (config_key, config_value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (config_key) DO UPDATE
		SET config_value = EXCLUDED.config_value, updated_at = NOW()
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

// CreateRule creates a new context-aware rule
func (r *ErrorRecoveryRepository) CreateRule(ctx context.Context, rule *error_recovery.ContextAwareRule) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO context_aware_rules (
			id, name, description, priority, conditions, context, actions,
			confidence, success_rate, usage_count, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		rule.ID, rule.Name, rule.Description, rule.Priority, rule.Conditions, rule.Context, rule.Actions,
		rule.Confidence, rule.SuccessRate, rule.UsageCount, rule.CreatedBy, rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create rule: %w", err)
	}
	return nil
}

// GetRule retrieves a rule by ID
func (r *ErrorRecoveryRepository) GetRule(ctx context.Context, id string) (*error_recovery.ContextAwareRule, error) {
	var rule error_recovery.ContextAwareRule
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, priority, conditions, context, actions,
		       confidence, success_rate, usage_count, created_by, created_at, updated_at
		FROM context_aware_rules WHERE id = $1
	`, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Priority, &rule.Conditions, &rule.Context, &rule.Actions,
		&rule.Confidence, &rule.SuccessRate, &rule.UsageCount, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	return &rule, nil
}

// UpdateRule updates an existing rule
func (r *ErrorRecoveryRepository) UpdateRule(ctx context.Context, rule *error_recovery.ContextAwareRule) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE context_aware_rules SET
			name = $2, description = $3, priority = $4, conditions = $5, context = $6, actions = $7,
			confidence = $8, success_rate = $9, usage_count = $10, updated_at = NOW()
		WHERE id = $1
	`,
		rule.ID, rule.Name, rule.Description, rule.Priority, rule.Conditions, rule.Context, rule.Actions,
		rule.Confidence, rule.SuccessRate, rule.UsageCount,
	)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}
	return nil
}

// DeleteRule deletes a rule by ID
func (r *ErrorRecoveryRepository) DeleteRule(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM context_aware_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	return nil
}

// ListRules lists all rules, optionally filtered (e.g., by priority)
func (r *ErrorRecoveryRepository) ListRules(ctx context.Context) ([]error_recovery.ContextAwareRule, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, description, priority, conditions, context, actions,
		       confidence, success_rate, usage_count, created_by, created_at, updated_at
		FROM context_aware_rules
		ORDER BY priority DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []error_recovery.ContextAwareRule
	for rows.Next() {
		var rule error_recovery.ContextAwareRule
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.Priority, &rule.Conditions, &rule.Context, &rule.Actions,
			&rule.Confidence, &rule.SuccessRate, &rule.UsageCount, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}
