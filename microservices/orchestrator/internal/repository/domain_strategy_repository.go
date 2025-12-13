package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
)

// DomainStrategyRepository handles domain_strategies CRUD operations
type DomainStrategyRepository struct {
	db *database.DB
}

// normalizeDomain normalizes a domain for consistent DB lookups
func normalizeDomain(domain string) string {
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(domain, "www.")
	return domain
}

// DomainStrategy represents a learned strategy from the domain_strategies table
type DomainStrategy struct {
	ID                   string     `json:"id"`
	Domain               string     `json:"domain"`
	RecommendedTier      int        `json:"recommended_tier"`
	TierConfidence       float64    `json:"tier_confidence"`
	SessionRequirement   int        `json:"session_requirement"`
	DetectedCookies      []string   `json:"detected_cookies,omitempty"`
	OptimalRotationCount *int       `json:"optimal_rotation_count,omitempty"`
	RotationConfidence   float64    `json:"rotation_confidence"`
	AdaptiveDelayMs      int        `json:"adaptive_delay_ms"`
	MaxConcurrentReqs    *int       `json:"max_concurrent_requests,omitempty"`
	TotalRequests        int64      `json:"total_requests"`
	TotalSuccesses       int64      `json:"total_successes"`
	TotalFailures        int64      `json:"total_failures"`
	SuccessRate          float64    `json:"success_rate"`
	LearningStatus       string     `json:"learning_status"` // new, learning, stable, needs_review
	SampleSize           int        `json:"sample_size"`
	LastBlockAt          *time.Time `json:"last_block_at,omitempty"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty"`
	TierAttempts         string     `json:"tier_attempts,omitempty"` // JSON string
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// NewDomainStrategyRepository creates a new domain strategy repository
func NewDomainStrategyRepository(db *database.DB) *DomainStrategyRepository {
	return &DomainStrategyRepository{db: db}
}

// GetAll returns all domain strategies with optional filtering
func (r *DomainStrategyRepository) GetAll(ctx context.Context, status string, limit, offset int) ([]DomainStrategy, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM domain_strategies WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if status != "" {
		countQuery += fmt.Sprintf(` AND learning_status = $%d`, argNum)
		args = append(args, status)
		argNum++
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Main query
	query := `SELECT 
		id, domain, recommended_tier, tier_confidence,
		session_requirement, detected_cookies,
		optimal_rotation_count, rotation_confidence,
		adaptive_delay_ms, max_concurrent_requests,
		total_requests, total_successes, total_failures, success_rate,
		learning_status, sample_size,
		last_block_at, last_success_at,
		COALESCE(tier_attempts::text, '{}'),
		created_at, updated_at
	FROM domain_strategies WHERE 1=1`

	args = []interface{}{}
	argNum = 1

	if status != "" {
		query += fmt.Sprintf(` AND learning_status = $%d`, argNum)
		args = append(args, status)
		argNum++
	}

	query += fmt.Sprintf(` ORDER BY total_requests DESC, updated_at DESC LIMIT $%d OFFSET $%d`, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	strategies := make([]DomainStrategy, 0)
	for rows.Next() {
		var s DomainStrategy
		if err := rows.Scan(
			&s.ID, &s.Domain, &s.RecommendedTier, &s.TierConfidence,
			&s.SessionRequirement, &s.DetectedCookies,
			&s.OptimalRotationCount, &s.RotationConfidence,
			&s.AdaptiveDelayMs, &s.MaxConcurrentReqs,
			&s.TotalRequests, &s.TotalSuccesses, &s.TotalFailures, &s.SuccessRate,
			&s.LearningStatus, &s.SampleSize,
			&s.LastBlockAt, &s.LastSuccessAt,
			&s.TierAttempts,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			continue
		}
		strategies = append(strategies, s)
	}

	return strategies, total, nil
}

// GetByDomain returns a single domain strategy
func (r *DomainStrategyRepository) GetByDomain(ctx context.Context, domain string) (*DomainStrategy, error) {
	domain = normalizeDomain(domain)
	query := `SELECT 
		id, domain, recommended_tier, tier_confidence,
		session_requirement, detected_cookies,
		optimal_rotation_count, rotation_confidence,
		adaptive_delay_ms, max_concurrent_requests,
		total_requests, total_successes, total_failures, success_rate,
		learning_status, sample_size,
		last_block_at, last_success_at,
		COALESCE(tier_attempts::text, '{}'),
		created_at, updated_at
	FROM domain_strategies WHERE domain = $1`

	var s DomainStrategy
	err := r.db.Pool.QueryRow(ctx, query, domain).Scan(
		&s.ID, &s.Domain, &s.RecommendedTier, &s.TierConfidence,
		&s.SessionRequirement, &s.DetectedCookies,
		&s.OptimalRotationCount, &s.RotationConfidence,
		&s.AdaptiveDelayMs, &s.MaxConcurrentReqs,
		&s.TotalRequests, &s.TotalSuccesses, &s.TotalFailures, &s.SuccessRate,
		&s.LearningStatus, &s.SampleSize,
		&s.LastBlockAt, &s.LastSuccessAt,
		&s.TierAttempts,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// Delete removes a domain strategy (clears learning)
func (r *DomainStrategyRepository) Delete(ctx context.Context, domain string) error {
	domain = normalizeDomain(domain)
	query := `DELETE FROM domain_strategies WHERE domain = $1`
	result, err := r.db.Pool.Exec(ctx, query, domain)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("domain not found: %s", domain)
	}
	return nil
}

// DeleteAll removes all domain strategies (clear all learning)
func (r *DomainStrategyRepository) DeleteAll(ctx context.Context) (int64, error) {
	query := `DELETE FROM domain_strategies`
	result, err := r.db.Pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetStats returns aggregate statistics for domain strategies
func (r *DomainStrategyRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
		COUNT(*) as total,
		COUNT(*) FILTER (WHERE learning_status = 'new') as new_count,
		COUNT(*) FILTER (WHERE learning_status = 'learning') as learning,
		COUNT(*) FILTER (WHERE learning_status = 'stable') as stable,
		COUNT(*) FILTER (WHERE learning_status = 'needs_review') as needs_review,
		AVG(success_rate) as avg_success_rate,
		SUM(total_requests) as total_requests_all
	FROM domain_strategies`

	var total, newCount, learning, stable, needsReview int
	var avgSuccessRate, totalRequests *float64

	err := r.db.Pool.QueryRow(ctx, query).Scan(
		&total, &newCount, &learning, &stable, &needsReview,
		&avgSuccessRate, &totalRequests,
	)
	if err != nil {
		return nil, err
	}

	avgRate := 0.0
	if avgSuccessRate != nil {
		avgRate = *avgSuccessRate * 100
	}

	return map[string]interface{}{
		"total":            total,
		"new":              newCount,
		"learning":         learning,
		"stable":           stable,
		"needs_review":     needsReview,
		"avg_success_rate": avgRate,
	}, nil
}
