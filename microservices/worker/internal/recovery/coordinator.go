package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RecoveryCoordinator ensures only one worker handles recovery for a domain/pattern
// Uses Redis distributed locks to prevent redundant recovery work across workers.
//
// Problem: Without coordination, if 5 workers hit a rate limit:
//   - All 5 would independently trigger recovery
//   - All 5 might call AI for analysis
//   - All 5 would switch proxies/add delays
//
// Solution: First worker to detect the issue becomes "coordinator"
//   - Coordinator executes recovery and publishes result
//   - Other workers wait for result or use cached recovery plan
type RecoveryCoordinator struct {
	cache    *cache.Cache
	workerID string
	config   *CoordinatorConfig
}

// CoordinatorConfig holds configuration for the recovery coordinator
type CoordinatorConfig struct {
	LockTTL      time.Duration // How long to hold the recovery lock (default: 30s)
	ResultTTL    time.Duration // How long to cache recovery results (default: 5m)
	WaitTimeout  time.Duration // How long non-coordinators wait for result (default: 10s)
	PollInterval time.Duration // How often to check for result (default: 500ms)
}

// DefaultCoordinatorConfig returns sensible defaults
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		LockTTL:      30 * time.Second,
		ResultTTL:    5 * time.Minute,
		WaitTimeout:  10 * time.Second,
		PollInterval: 500 * time.Millisecond,
	}
}

// CoordinatorRole indicates whether this worker is coordinator or follower
type CoordinatorRole string

const (
	RoleCoordinator CoordinatorRole = "coordinator" // This worker handles recovery
	RoleFollower    CoordinatorRole = "follower"    // Another worker is handling it
	RoleNone        CoordinatorRole = "none"        // No coordination needed
)

// RecoveryResult is the cached result of a recovery operation
type RecoveryResult struct {
	Plan          *RecoveryPlan `json:"plan"`
	CoordinatorID string        `json:"coordinator_id"`
	Pattern       ErrorPattern  `json:"pattern"`
	Domain        string        `json:"domain"`
	CreatedAt     time.Time     `json:"created_at"`
	ExpiresAt     time.Time     `json:"expires_at"`
}

// Redis key patterns
const (
	keyRecoveryLock   = "recovery:lock:%s:%s"   // domain:pattern
	keyRecoveryResult = "recovery:result:%s:%s" // domain:pattern
)

// NewRecoveryCoordinator creates a new recovery coordinator
func NewRecoveryCoordinator(c *cache.Cache, workerID string, config *CoordinatorConfig) *RecoveryCoordinator {
	if config == nil {
		config = DefaultCoordinatorConfig()
	}

	return &RecoveryCoordinator{
		cache:    c,
		workerID: workerID,
		config:   config,
	}
}

// TryAcquireCoordination attempts to become the coordinator for a domain/pattern recovery
// Returns the role (coordinator/follower) and any existing result
func (c *RecoveryCoordinator) TryAcquireCoordination(ctx context.Context, domain string, pattern ErrorPattern) (CoordinatorRole, *RecoveryResult, error) {
	if c.cache == nil {
		return RoleNone, nil, nil // No Redis, no coordination
	}

	lockKey := fmt.Sprintf(keyRecoveryLock, domain, pattern)
	resultKey := fmt.Sprintf(keyRecoveryResult, domain, pattern)

	// First, check if there's already a cached result (another worker completed recovery)
	existingResult, err := c.getResult(ctx, resultKey)
	if err == nil && existingResult != nil && time.Now().Before(existingResult.ExpiresAt) {
		logger.Debug("Using cached recovery result",
			zap.String("domain", domain),
			zap.String("pattern", string(pattern)),
			zap.String("coordinator", existingResult.CoordinatorID),
		)
		return RoleFollower, existingResult, nil
	}

	// Try to acquire the lock
	success, err := c.cache.SetNX(ctx, lockKey, c.workerID, c.config.LockTTL)
	if err != nil {
		logger.Warn("Failed to acquire recovery lock", zap.Error(err))
		return RoleNone, nil, err
	}

	if success {
		// We are the coordinator!
		logger.Info("Acquired recovery coordinator role",
			zap.String("worker_id", c.workerID),
			zap.String("domain", domain),
			zap.String("pattern", string(pattern)),
		)
		return RoleCoordinator, nil, nil
	}

	// Another worker is the coordinator, we're a follower
	logger.Debug("Another worker is recovery coordinator, waiting for result",
		zap.String("domain", domain),
		zap.String("pattern", string(pattern)),
	)
	return RoleFollower, nil, nil
}

// WaitForResult waits for the coordinator to publish recovery result
// Returns the result or nil if timeout
func (c *RecoveryCoordinator) WaitForResult(ctx context.Context, domain string, pattern ErrorPattern) *RecoveryResult {
	if c.cache == nil {
		return nil
	}

	resultKey := fmt.Sprintf(keyRecoveryResult, domain, pattern)
	deadline := time.Now().Add(c.config.WaitTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		result, err := c.getResult(ctx, resultKey)
		if err == nil && result != nil {
			logger.Debug("Received recovery result from coordinator",
				zap.String("domain", domain),
				zap.String("pattern", string(pattern)),
				zap.String("action", string(result.Plan.Action)),
			)
			return result
		}

		time.Sleep(c.config.PollInterval)
	}

	logger.Warn("Timeout waiting for recovery result",
		zap.String("domain", domain),
		zap.String("pattern", string(pattern)),
	)
	return nil
}

// PublishResult publishes the recovery result for other workers to use
func (c *RecoveryCoordinator) PublishResult(ctx context.Context, domain string, pattern ErrorPattern, plan *RecoveryPlan) error {
	if c.cache == nil {
		return nil
	}

	resultKey := fmt.Sprintf(keyRecoveryResult, domain, pattern)
	lockKey := fmt.Sprintf(keyRecoveryLock, domain, pattern)

	result := &RecoveryResult{
		Plan:          plan,
		CoordinatorID: c.workerID,
		Pattern:       pattern,
		Domain:        domain,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(c.config.ResultTTL),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// Publish result
	if err := c.cache.Set(ctx, resultKey, string(data), c.config.ResultTTL); err != nil {
		return err
	}

	// Release lock (allow new coordination for future issues)
	c.cache.Delete(ctx, lockKey)

	logger.Info("Published recovery result",
		zap.String("domain", domain),
		zap.String("pattern", string(pattern)),
		zap.String("action", string(plan.Action)),
		zap.Duration("ttl", c.config.ResultTTL),
	)

	return nil
}

// ReleaseLock releases the coordination lock (call if recovery fails)
func (c *RecoveryCoordinator) ReleaseLock(ctx context.Context, domain string, pattern ErrorPattern) error {
	if c.cache == nil {
		return nil
	}

	lockKey := fmt.Sprintf(keyRecoveryLock, domain, pattern)
	return c.cache.Delete(ctx, lockKey)
}

// getResult retrieves cached recovery result
func (c *RecoveryCoordinator) getResult(ctx context.Context, key string) (*RecoveryResult, error) {
	data, err := c.cache.Get(ctx, key)
	if err != nil || data == "" {
		return nil, err
	}

	var result RecoveryResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetStats returns statistics about coordination
func (c *RecoveryCoordinator) GetStats(ctx context.Context) map[string]interface{} {
	if c.cache == nil {
		return nil
	}

	stats := map[string]interface{}{
		"worker_id":  c.workerID,
		"lock_ttl":   c.config.LockTTL.String(),
		"result_ttl": c.config.ResultTTL.String(),
	}

	// Count active locks
	locks, _ := c.cache.Keys(ctx, "recovery:lock:*")
	stats["active_locks"] = len(locks)

	// Count cached results
	results, _ := c.cache.Keys(ctx, "recovery:result:*")
	stats["cached_results"] = len(results)

	return stats
}

// InvalidateResult removes a cached result (e.g., if conditions changed)
func (c *RecoveryCoordinator) InvalidateResult(ctx context.Context, domain string, pattern ErrorPattern) error {
	if c.cache == nil {
		return nil
	}

	resultKey := fmt.Sprintf(keyRecoveryResult, domain, pattern)
	return c.cache.Delete(ctx, resultKey)
}
