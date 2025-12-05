package executor

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retries (default: 3)
	InitialDelay   time.Duration // Initial delay before first retry (default: 1s)
	MaxDelay       time.Duration // Maximum delay between retries (default: 30s)
	BackoffFactor  float64       // Multiplier for delay after each retry (default: 2.0)
	JitterFraction float64       // Random jitter as fraction of delay (default: 0.1)
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		BackoffFactor:  2.0,
		JitterFraction: 0.1,
	}
}

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// WithRetry executes a function with exponential backoff retry
func WithRetry(fn func() error, config RetryConfig) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()

		if err == nil {
			if attempt > 0 {
				logger.Info("Retry succeeded",
					zap.Int("attempt", attempt),
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			logger.Debug("Error is not retryable, stopping",
				zap.Error(err),
			)
			return err
		}

		// Don't sleep after the last attempt
		if attempt >= config.MaxRetries {
			break
		}

		delay := calculateBackoff(attempt, config)

		logger.Warn("Retrying after error",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", config.MaxRetries),
			zap.Duration("delay", delay),
			zap.Error(err),
		)

		time.Sleep(delay)
	}

	logger.Error("All retries exhausted",
		zap.Int("attempts", config.MaxRetries+1),
		zap.Error(lastErr),
	)

	return lastErr
}

// calculateBackoff calculates the delay for the given attempt
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: delay = initialDelay * (backoffFactor ^ attempt)
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))

	// Apply max delay cap
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Apply jitter to prevent thundering herd
	if config.JitterFraction > 0 {
		jitter := delay * config.JitterFraction * (rand.Float64()*2 - 1) // -jitter to +jitter
		delay = delay + jitter
	}

	return time.Duration(delay)
}

// isRetryable determines if an error should trigger a retry
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Explicit retryable error
	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return true
	}

	// Network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	errStr := strings.ToLower(err.Error())

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	// Connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	// Browser automation errors
	if strings.Contains(errStr, "navigation failed") ||
		strings.Contains(errStr, "page crashed") ||
		strings.Contains(errStr, "target closed") ||
		strings.Contains(errStr, "browser disconnected") {
		return true
	}

	// HTTP errors that warrant retry (rate limit, server errors)
	if strings.Contains(errStr, "status 429") ||
		strings.Contains(errStr, "status 500") ||
		strings.Contains(errStr, "status 502") ||
		strings.Contains(errStr, "status 503") ||
		strings.Contains(errStr, "status 504") {
		return true
	}

	return false
}

// MarkRetryable wraps an error to mark it as retryable
func MarkRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}
