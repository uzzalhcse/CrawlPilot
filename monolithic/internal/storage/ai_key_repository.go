package storage

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// AIAPIKey represents an AI API key with usage tracking
type AIAPIKey struct {
	ID                 string     `db:"id"`
	APIKey             string     `db:"api_key"`
	Name               string     `db:"name"`
	Provider           string     `db:"provider"` // gemini or openrouter
	TotalRequests      int        `db:"total_requests"`
	SuccessfulRequests int        `db:"successful_requests"`
	FailedRequests     int        `db:"failed_requests"`
	LastUsedAt         *time.Time `db:"last_used_at"`
	LastErrorAt        *time.Time `db:"last_error_at"`
	LastErrorMessage   *string    `db:"last_error_message"`
	CooldownUntil      *time.Time `db:"cooldown_until"`
	IsActive           bool       `db:"is_active"`
	IsRateLimited      bool       `db:"is_rate_limited"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// AIKeyRepository manages Gemini API keys
type AIKeyRepository struct {
	db     *PostgresDB
	logger *zap.Logger
}

// NewAIKeyRepository creates a new repository
func NewAIKeyRepository(db *PostgresDB, logger *zap.Logger) *AIKeyRepository {
	return &AIKeyRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new API key
func (r *AIKeyRepository) Create(ctx context.Context, name, apiKey string) error {
	query := `
		INSERT INTO ai_api_keys (name, api_key)
		VALUES ($1, $2)
	`
	_, err := r.db.Pool.Exec(ctx, query, name, apiKey)
	return err
}

// GetNextAvailableKey returns the next available API key for the specified provider
func (r *AIKeyRepository) GetNextAvailableKey(ctx context.Context, provider string) (*AIAPIKey, error) {
	query := `
		SELECT id, api_key, name, total_requests, successful_requests, failed_requests,
		       last_used_at, last_error_at, last_error_message, cooldown_until,
		       is_active, is_rate_limited, created_at, updated_at, provider
		FROM ai_api_keys
		WHERE is_active = true
		  AND is_rate_limited = false
		  AND provider = $1
		  AND (cooldown_until IS NULL OR cooldown_until < NOW())
		ORDER BY total_requests ASC, last_used_at ASC NULLS FIRST
		LIMIT 1
	`

	key := &AIAPIKey{}

	err := r.db.Pool.QueryRow(ctx, query, provider).Scan(
		&key.ID, &key.APIKey, &key.Name, &key.TotalRequests, &key.SuccessfulRequests,
		&key.FailedRequests, &key.LastUsedAt, &key.LastErrorAt, &key.LastErrorMessage,
		&key.CooldownUntil, &key.IsActive, &key.IsRateLimited, &key.CreatedAt, &key.UpdatedAt,
		&key.Provider,
	)

	if err != nil {
		return nil, fmt.Errorf("no available API keys for provider %s: %w", provider, err)
	}

	return key, nil
}

// RecordSuccess updates key after successful use
func (r *AIKeyRepository) RecordSuccess(ctx context.Context, keyID string) error {
	query := `
		UPDATE ai_api_keys
		SET total_requests = total_requests + 1,
		    successful_requests = successful_requests + 1,
		    last_used_at = NOW(),
		    is_rate_limited = false,
		    cooldown_until = NULL
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, keyID)
	return err
}

// RecordFailure records a failed request and applies cooldown if rate limited
func (r *AIKeyRepository) RecordFailure(ctx context.Context, keyID string, errorMsg *string, isRateLimit bool) error {
	var query string

	if isRateLimit {
		// Apply 1 minute cooldown for rate limit errors
		query = `
			UPDATE ai_api_keys
			SET total_requests = total_requests + 1,
			    failed_requests = failed_requests + 1,
			    last_error_at = NOW(),
			    last_error_message = $2,
			    is_rate_limited = true,
			    cooldown_until = NOW() + INTERVAL '1 minute'
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE ai_api_keys
			SET total_requests = total_requests + 1,
			    failed_requests = failed_requests + 1,
			    last_error_at = NOW(),
			    last_error_message = $2
			WHERE id = $1
		`
	}

	_, err := r.db.Pool.Exec(ctx, query, keyID, errorMsg)
	return err
}

// GetAllKeys returns all API keys for management
func (r *AIKeyRepository) GetAllKeys(ctx context.Context) ([]*AIAPIKey, error) {
	query := `
		SELECT id, api_key, name, total_requests, successful_requests, failed_requests,
		       last_used_at, last_error_at, last_error_message, cooldown_until,
		       is_active, is_rate_limited, created_at, updated_at
		FROM ai_api_keys
		ORDER BY total_requests ASC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*AIAPIKey
	for rows.Next() {
		key := &AIAPIKey{}
		err := rows.Scan(
			&key.ID, &key.APIKey, &key.Name, &key.TotalRequests, &key.SuccessfulRequests,
			&key.FailedRequests, &key.LastUsedAt, &key.LastErrorAt, &key.LastErrorMessage,
			&key.CooldownUntil, &key.IsActive, &key.IsRateLimited, &key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// SetActive enables/disables an API key
func (r *AIKeyRepository) SetActive(ctx context.Context, keyID string, active bool) error {
	query := `UPDATE ai_api_keys SET is_active = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, active, keyID)
	return err
}

// Delete removes an API key
func (r *AIKeyRepository) Delete(ctx context.Context, keyID string) error {
	query := `DELETE FROM ai_api_keys WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, keyID)
	return err
}

// ResetCooldowns clears all cooldowns (for manual reset)
func (r *AIKeyRepository) ResetCooldowns(ctx context.Context) error {
	query := `
		UPDATE ai_api_keys
		SET is_rate_limited = false, cooldown_until = NULL
		WHERE is_rate_limited = true OR cooldown_until IS NOT NULL
	`
	_, err := r.db.Pool.Exec(ctx, query)
	return err
}
