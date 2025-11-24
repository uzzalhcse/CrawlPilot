package storage

import (
	"context"
	"encoding/json"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// FixSuggestionRepository handles database operations for fix suggestions
type FixSuggestionRepository struct {
	db *PostgresDB
}

// NewFixSuggestionRepository creates a new fix suggestion repository
func NewFixSuggestionRepository(db *PostgresDB) *FixSuggestionRepository {
	return &FixSuggestionRepository{db: db}
}

// Create creates a new fix suggestion
func (r *FixSuggestionRepository) Create(ctx context.Context, suggestion *models.FixSuggestion) error {
	alternativesJSON, _ := json.Marshal(suggestion.AlternativeSelectors)
	configJSON, _ := json.Marshal(suggestion.SuggestedNodeConfig)
	verificationJSON, _ := json.Marshal(suggestion.VerificationResult)

	query := `
		INSERT INTO fix_suggestions
		(id, snapshot_id, workflow_id, node_id, suggested_selector, alternative_selectors,
		 suggested_node_config, fix_explanation, confidence_score, status, ai_model, ai_response_raw, verification_result)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		suggestion.ID,
		suggestion.SnapshotID,
		suggestion.WorkflowID,
		suggestion.NodeID,
		suggestion.SuggestedSelector,
		alternativesJSON,
		configJSON,
		suggestion.FixExplanation,
		suggestion.ConfidenceScore,
		suggestion.Status,
		suggestion.AIModel,
		suggestion.AIResponseRaw,
		verificationJSON,
	)

	if err != nil {
		logger.Error("Failed to create fix suggestion", zap.Error(err))
		return err
	}

	return nil
}

// GetByID retrieves a fix suggestion by ID
func (r *FixSuggestionRepository) GetByID(ctx context.Context, id string) (*models.FixSuggestion, error) {
	query := `
		SELECT id, snapshot_id, workflow_id, node_id, suggested_selector, alternative_selectors,
		       suggested_node_config, fix_explanation, confidence_score, status, reviewed_by,
		       reviewed_at, applied_at, reverted_at, ai_model, ai_prompt_tokens, ai_response_tokens,
		       ai_response_raw, verification_result, created_at, updated_at
		FROM fix_suggestions
		WHERE id = $1
	`

	suggestion := &models.FixSuggestion{}
	var alternativesJSON, configJSON, verificationJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&suggestion.ID,
		&suggestion.SnapshotID,
		&suggestion.WorkflowID,
		&suggestion.NodeID,
		&suggestion.SuggestedSelector,
		&alternativesJSON,
		&configJSON,
		&suggestion.FixExplanation,
		&suggestion.ConfidenceScore,
		&suggestion.Status,
		&suggestion.ReviewedBy,
		&suggestion.ReviewedAt,
		&suggestion.AppliedAt,
		&suggestion.RevertedAt,
		&suggestion.AIModel,
		&suggestion.AIPromptTokens,
		&suggestion.AIResponseTokens,
		&suggestion.AIResponseRaw,
		&verificationJSON,
		&suggestion.CreatedAt,
		&suggestion.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal(alternativesJSON, &suggestion.AlternativeSelectors)
	json.Unmarshal(configJSON, &suggestion.SuggestedNodeConfig)
	json.Unmarshal(verificationJSON, &suggestion.VerificationResult)

	return suggestion, nil
}

// GetBySnapshotID retrieves all fix suggestions for a snapshot
func (r *FixSuggestionRepository) GetBySnapshotID(ctx context.Context, snapshotID string) ([]*models.FixSuggestion, error) {
	query := `
		SELECT id, snapshot_id, workflow_id, node_id, suggested_selector, alternative_selectors,
		       suggested_node_config, fix_explanation, confidence_score, status, reviewed_by,
		       reviewed_at, applied_at, reverted_at, ai_model, ai_prompt_tokens, ai_response_tokens,
		       verification_result, created_at, updated_at
		FROM fix_suggestions
		WHERE snapshot_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*models.FixSuggestion
	for rows.Next() {
		suggestion := &models.FixSuggestion{}
		var alternativesJSON, configJSON, verificationJSON []byte

		err := rows.Scan(
			&suggestion.ID,
			&suggestion.SnapshotID,
			&suggestion.WorkflowID,
			&suggestion.NodeID,
			&suggestion.SuggestedSelector,
			&alternativesJSON,
			&configJSON,
			&suggestion.FixExplanation,
			&suggestion.ConfidenceScore,
			&suggestion.Status,
			&suggestion.ReviewedBy,
			&suggestion.ReviewedAt,
			&suggestion.AppliedAt,
			&suggestion.RevertedAt,
			&suggestion.AIModel,
			&suggestion.AIPromptTokens,
			&suggestion.AIResponseTokens,
			&verificationJSON,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		json.Unmarshal(alternativesJSON, &suggestion.AlternativeSelectors)
		json.Unmarshal(configJSON, &suggestion.SuggestedNodeConfig)
		json.Unmarshal(verificationJSON, &suggestion.VerificationResult)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// UpdateStatus updates the status of a fix suggestion
func (r *FixSuggestionRepository) UpdateStatus(ctx context.Context, id string, status string, reviewedBy string) error {
	query := `
		UPDATE fix_suggestions
		SET status = $1, reviewed_by = $2, reviewed_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	_, err := r.db.Pool.Exec(ctx, query, status, reviewedBy, id)
	if err != nil {
		logger.Error("Failed to update fix suggestion status", zap.Error(err))
		return err
	}

	return nil
}

// MarkAsApplied marks a suggestion as applied
func (r *FixSuggestionRepository) MarkAsApplied(ctx context.Context, id string) error {
	query := `
		UPDATE fix_suggestions
		SET status = 'applied', applied_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		logger.Error("Failed to mark suggestion as applied", zap.Error(err))
		return err
	}

	return nil
}

// MarkAsReverted marks a suggestion as reverted
func (r *FixSuggestionRepository) MarkAsReverted(ctx context.Context, id string) error {
	query := `
		UPDATE fix_suggestions
		SET status = 'reverted', reverted_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		logger.Error("Failed to mark suggestion as reverted", zap.Error(err))
		return err
	}

	return nil
}
