package apikey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries        int           // maximum number of retries across all keys
	InitialBackoff    time.Duration // initial backoff duration
	MaxBackoff        time.Duration // maximum backoff duration
	BackoffMultiplier float64       // backoff multiplier
	RateLimitWindow   time.Duration // how long to mark a key as rate limited
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        10, // try up to 10 times (cycling through keys)
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RateLimitWindow:   5 * time.Minute, // assume rate limit lasts 5 minutes
	}
}

// RetryResult contains the result of a retry operation
type RetryResult struct {
	Success      bool
	Response     interface{}
	Error        error
	KeyUsed      *APIKey
	AttemptsUsed int
	TotalTime    time.Duration
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check for common rate limit error patterns
	rateLimitIndicators := []string{
		"429",
		"too many requests",
		"rate limit",
		"ratelimit",
		"rate_limit",
		"quota exceeded",
		"resource exhausted",
		"limit exceeded",
		"throttled",
	}

	for _, indicator := range rateLimitIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}

// RetryWithRotation executes a function with automatic key rotation on failures
func (m *Manager) RetryWithRotation(
	ctx context.Context,
	provider string,
	config *RetryConfig,
	fn func(ctx context.Context, apiKey string) (interface{}, error),
) (*RetryResult, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	startTime := time.Now()
	var excludedKeys []primitive.ObjectID
	var lastError error
	var lastKey *APIKey

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		// Get next available key (excluding previously failed ones this round)
		key, err := m.GetNextAvailableKey(ctx, provider, excludedKeys)
		if err != nil {
			if errors.Is(err, ErrNoKeysAvailable) {
				// No more keys available, wait and try again after clearing exclusions
				if attempt < config.MaxRetries-1 {
					backoff := calculateBackoff(attempt, config)
					fmt.Printf("⏳ All keys exhausted, waiting %v before retry (attempt %d/%d)...\n",
						backoff, attempt+1, config.MaxRetries)
					time.Sleep(backoff)
					excludedKeys = nil // clear exclusions and try again
					continue
				}
			}
			return nil, fmt.Errorf("failed to get API key: %w", err)
		}

		lastKey = key
		requestStart := time.Now()

		// Execute the function with this key
		result, err := fn(ctx, key.Key)
		responseTime := time.Since(requestStart).Milliseconds()

		if err == nil {
			// Success! Record usage and return
			if err := m.UseKey(ctx, key.ID, true, "", responseTime); err != nil {
				fmt.Printf("Warning: failed to record key usage: %v\n", err)
			}

			return &RetryResult{
				Success:      true,
				Response:     result,
				KeyUsed:      key,
				AttemptsUsed: attempt + 1,
				TotalTime:    time.Since(startTime),
			}, nil
		}

		// Error occurred
		lastError = err

		// Check if it's a rate limit error
		if IsRateLimitError(err) {
			fmt.Printf("⚠️  Rate limit hit on key %s (attempt %d/%d): %v\n",
				maskKey(key.Key), attempt+1, config.MaxRetries, err)

			// Mark this key as rate limited
			if err := m.MarkRateLimited(ctx, key.ID, config.RateLimitWindow); err != nil {
				fmt.Printf("Warning: failed to mark key as rate limited: %v\n", err)
			}

			// Record failed usage
			if err := m.UseKey(ctx, key.ID, false, err.Error(), responseTime); err != nil {
				fmt.Printf("Warning: failed to record key usage: %v\n", err)
			}

			// Add to excluded keys for this round
			excludedKeys = append(excludedKeys, key.ID)

			// Wait with exponential backoff before trying next key
			if attempt < config.MaxRetries-1 {
				backoff := calculateBackoff(attempt, config)
				fmt.Printf("⏳ Rotating to next API key after %v...\n", backoff)
				time.Sleep(backoff)
			}

			continue
		}

		// Other error (not rate limit)
		fmt.Printf("❌ Request failed with key %s (attempt %d/%d): %v\n",
			maskKey(key.Key), attempt+1, config.MaxRetries, err)

		// Record failed usage
		if err := m.UseKey(ctx, key.ID, false, err.Error(), responseTime); err != nil {
			fmt.Printf("Warning: failed to record key usage: %v\n", err)
		}

		// For non-rate-limit errors, we might want to retry with same key first
		// But for now, exclude it and try another key
		excludedKeys = append(excludedKeys, key.ID)

		// Wait before retry
		if attempt < config.MaxRetries-1 {
			backoff := calculateBackoff(attempt, config)
			time.Sleep(backoff)
		}
	}

	// All retries exhausted
	return &RetryResult{
		Success:      false,
		Error:        lastError,
		KeyUsed:      lastKey,
		AttemptsUsed: config.MaxRetries,
		TotalTime:    time.Since(startTime),
	}, fmt.Errorf("all retries exhausted after %d attempts: %w", config.MaxRetries, lastError)
}

// calculateBackoff calculates exponential backoff duration
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	backoff := float64(config.InitialBackoff) * pow(config.BackoffMultiplier, float64(attempt))

	if backoff > float64(config.MaxBackoff) {
		return config.MaxBackoff
	}

	return time.Duration(backoff)
}

// pow is a simple integer power function
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// maskKey masks an API key for logging
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// SimpleRetry is a simpler retry function that just cycles through all available keys once
func (m *Manager) SimpleRetry(
	ctx context.Context,
	provider string,
	fn func(ctx context.Context, apiKey string) (interface{}, error),
) (interface{}, error) {
	// Get all keys for this provider
	keys, err := m.GetAllKeys(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no API keys available for provider: %s", provider)
	}

	var lastError error

	// Try each key once
	for i, key := range keys {
		if !key.IsAvailable() {
			continue
		}

		requestStart := time.Now()
		result, err := fn(ctx, key.Key)
		responseTime := time.Since(requestStart).Milliseconds()

		if err == nil {
			// Success
			if err := m.UseKey(ctx, key.ID, true, "", responseTime); err != nil {
				fmt.Printf("Warning: failed to record key usage: %v\n", err)
			}
			return result, nil
		}

		lastError = err

		// Record failure
		if err := m.UseKey(ctx, key.ID, false, err.Error(), responseTime); err != nil {
			fmt.Printf("Warning: failed to record key usage: %v\n", err)
		}

		// If rate limited, mark it
		if IsRateLimitError(err) {
			fmt.Printf("⚠️  Rate limit hit on key %d/%d: %v\n", i+1, len(keys), err)
			if err := m.MarkRateLimited(ctx, key.ID, 5*time.Minute); err != nil {
				fmt.Printf("Warning: failed to mark key as rate limited: %v\n", err)
			}
		} else {
			fmt.Printf("❌ Request failed with key %d/%d: %v\n", i+1, len(keys), err)
		}

		// Short delay before trying next key
		if i < len(keys)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return nil, fmt.Errorf("all keys failed, last error: %w", lastError)
}
