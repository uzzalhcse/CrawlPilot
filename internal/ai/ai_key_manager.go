package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/uzzalhcse/crawlify/internal/storage"
	"go.uber.org/zap"
)

// AIKeyManager manages API key rotation
type AIKeyManager struct {
	keyRepo       *storage.AIKeyRepository
	logger        *zap.Logger
	currentKeyID  string
	currentAPIKey string
}

// NewAIKeyManager creates a new key manager
func NewAIKeyManager(keyRepo *storage.AIKeyRepository, logger *zap.Logger) *AIKeyManager {
	return &AIKeyManager{
		keyRepo: keyRepo,
		logger:  logger,
	}
}

// GetAPIKey returns the current API key for the specified provider, rotating if necessary
func (m *AIKeyManager) GetAPIKey(ctx context.Context, provider string) (string, error) {
	// If we don't have a current key for this provider, get one
	if m.currentAPIKey == "" {
		return m.RotateKey(ctx, provider)
	}
	return m.currentAPIKey, nil
}

// RotateKey gets the next available API key for the specified provider
func (m *AIKeyManager) RotateKey(ctx context.Context, provider string) (string, error) {
	key, err := m.keyRepo.GetNextAvailableKey(ctx, provider)
	if err != nil {
		m.logger.Error("Failed to get next API key", zap.String("provider", provider), zap.Error(err))
		return "", fmt.Errorf("no available %s API keys: %w", provider, err)
	}

	m.currentKeyID = key.ID
	m.currentAPIKey = key.APIKey

	m.logger.Info("Rotated to new API key",
		zap.String("provider", provider),
		zap.String("key_id", key.ID),
		zap.String("key_name", key.Name),
		zap.Int("total_requests", key.TotalRequests))

	return m.currentAPIKey, nil
}

// RecordSuccess records successful API call
func (m *AIKeyManager) RecordSuccess(ctx context.Context) error {
	if m.currentKeyID == "" {
		return nil
	}
	return m.keyRepo.RecordSuccess(ctx, m.currentKeyID)
}

// RecordFailure records failed API call and rotates if rate limited
func (m *AIKeyManager) RecordFailure(ctx context.Context, err error) error {
	if m.currentKeyID == "" {
		return nil
	}

	errMsg := err.Error()
	isRateLimit := isRateLimitError(errMsg)

	// Record the failure
	if recordErr := m.keyRepo.RecordFailure(ctx, m.currentKeyID, &errMsg, isRateLimit); recordErr != nil {
		m.logger.Error("Failed to record API key failure", zap.Error(recordErr))
	}

	// If rate limited, rotate to next key
	if isRateLimit {
		m.logger.Warn("API key rate limited, rotating to next key",
			zap.String("key_id", m.currentKeyID),
			zap.String("error", errMsg))

		// Clear current key to force rotation
		m.currentKeyID = ""
		m.currentAPIKey = ""
	}

	return nil
}

// GetCurrentKeyID returns the current key ID
func (m *AIKeyManager) GetCurrentKeyID() string {
	return m.currentKeyID
}

// isRateLimitError checks if error is a rate limit error
func isRateLimitError(errMsg string) bool {
	rateLimitIndicators := []string{
		"429",
		"quota exceeded",
		"rate limit",
		"too many requests",
		"Quota exceeded",
	}

	errLower := strings.ToLower(errMsg)
	for _, indicator := range rateLimitIndicators {
		if strings.Contains(errLower, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}
