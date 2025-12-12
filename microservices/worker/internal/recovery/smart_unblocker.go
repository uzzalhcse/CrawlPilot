package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// strategyCacheEntry holds a cached strategy with expiry
type strategyCacheEntry struct {
	strategy  *models.DomainStrategy
	expiresAt time.Time
}

// SmartUnblocker provides intelligent anti-bot handling using learned domain strategies
// Implements the hybrid approach: Tiered Proxy Escalation + Domain Memory + Non-Blocking Coordination
type SmartUnblocker struct {
	pool        *pgxpool.Pool
	cache       *cache.Cache
	tieredProxy *TieredProxyManager
	config      *SmartUnblockerConfig

	// In-memory cache for hot path (avoids Redis/DB for repeated domains)
	strategyCache sync.Map // domain -> *strategyCacheEntry
	cacheTTL      time.Duration
}

// SmartUnblockerConfig configures the smart unblocker
type SmartUnblockerConfig struct {
	// Learning
	MinSamplesForStable int           // Min requests before marking domain as stable (default: 100)
	ConfidenceThreshold float64       // Min confidence to trust learned strategy (default: 0.7)
	LearningTTL         time.Duration // How long learning data stays in Redis (default: 24h)

	// Persistence
	PersistThreshold int // Min requests before persisting to DB (default: 50)

	// Optimistic Mode
	StartAtTierZero bool // Start new domains at Tier 0 (default: true)

	// Throttling
	MaxAdaptiveDelayMs int // Max adaptive delay in ms (default: 5000)
	DelayIncrementMs   int // Delay increment on rate limit (default: 500)
	DelayDecrementMs   int // Delay decrement on success streak (default: 100)

	// Performance
	LocalCacheTTL time.Duration // How long to cache strategies in-memory (default: 30s)
}

// DefaultSmartUnblockerConfig returns sensible defaults
func DefaultSmartUnblockerConfig() *SmartUnblockerConfig {
	return &SmartUnblockerConfig{
		MinSamplesForStable: 100,
		ConfidenceThreshold: 0.7,
		LearningTTL:         24 * time.Hour,
		PersistThreshold:    10,
		StartAtTierZero:     true,
		MaxAdaptiveDelayMs:  5000,
		DelayIncrementMs:    500,
		DelayDecrementMs:    100,
		LocalCacheTTL:       30 * time.Second,
	}
}

// NewSmartUnblocker creates a new smart unblocker
func NewSmartUnblocker(pool *pgxpool.Pool, c *cache.Cache, tieredProxy *TieredProxyManager, config *SmartUnblockerConfig) *SmartUnblocker {
	if config == nil {
		config = DefaultSmartUnblockerConfig()
	}

	return &SmartUnblocker{
		pool:        pool,
		cache:       c,
		tieredProxy: tieredProxy,
		config:      config,
		cacheTTL:    config.LocalCacheTTL,
	}
}

// Redis keys for smart unblocker
const (
	keyDomainStrategy     = "unblocker:strategy:%s"       // Cached domain strategy
	keyExecutionStats     = "unblocker:exec:%s:domain:%s" // Stats per execution per domain
	keyConsecutiveSuccess = "unblocker:success:%s"        // Consecutive success count
	keyConsecutiveFailure = "unblocker:failure:%s"        // Consecutive failure count
	keyExecutionDomains   = "unblocker:exec:%s:domains"   // Set of domains per execution (for O(1) iteration)
)

// LoadStrategiesForExecution loads known strategies from DB at execution start
// Returns map of domain -> strategy for quick lookup
func (u *SmartUnblocker) LoadStrategiesForExecution(ctx context.Context, domains []string) (map[string]*models.DomainStrategy, error) {
	strategies := make(map[string]*models.DomainStrategy)

	if u.pool == nil || len(domains) == 0 {
		return strategies, nil
	}

	// Batch load from DB
	query := `
		SELECT id, domain, recommended_tier, tier_confidence, 
		       session_requirement, detected_cookies,
		       optimal_rotation_count, rotation_confidence,
		       adaptive_delay_ms, max_concurrent_requests,
		       total_requests, total_successes, total_failures,
		       learning_status, sample_size, tier_attempts,
		       created_at, updated_at
		FROM domain_strategies
		WHERE domain = ANY($1)
	`

	rows, err := u.pool.Query(ctx, query, domains)
	if err != nil {
		logger.Error("Failed to load domain strategies", zap.Error(err))
		return strategies, nil // Return empty, will use optimistic mode
	}
	defer rows.Close()

	for rows.Next() {
		var s models.DomainStrategy
		var optRotation, maxConcurrent *int
		var tierAttemptsJSON []byte

		err := rows.Scan(
			&s.ID, &s.Domain, &s.RecommendedTier, &s.TierConfidence,
			&s.SessionRequirement, &s.DetectedCookies,
			&optRotation, &s.RotationConfidence,
			&s.AdaptiveDelayMs, &maxConcurrent,
			&s.TotalRequests, &s.TotalSuccesses, &s.TotalFailures,
			&s.LearningStatus, &s.SampleSize, &tierAttemptsJSON,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			logger.Warn("Failed to scan domain strategy", zap.Error(err))
			continue
		}

		s.OptimalRotationCount = optRotation
		s.MaxConcurrentRequests = maxConcurrent
		s.ParseTierAttemptsJSON(tierAttemptsJSON)

		strategies[s.Domain] = &s

		// Also cache in Redis for fast access
		u.cacheStrategy(ctx, &s)
	}

	logger.Info("Loaded domain strategies",
		zap.Int("count", len(strategies)),
		zap.Int("requested", len(domains)),
	)

	return strategies, nil
}

// GetStrategy returns the strategy for a domain
// OPTIMIZED: Uses 3-tier cache: in-memory → Redis → DB
func (u *SmartUnblocker) GetStrategy(ctx context.Context, domain string) *models.DomainStrategy {
	// Tier 1: Check in-memory cache (fastest, no network)
	if cached, ok := u.strategyCache.Load(domain); ok {
		entry := cached.(*strategyCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.strategy
		}
		// Expired, delete from cache
		u.strategyCache.Delete(domain)
	}

	// Tier 2: Try Redis cache
	cacheKey := fmt.Sprintf(keyDomainStrategy, domain)
	cached, err := u.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var strategy models.DomainStrategy
		if err := json.Unmarshal([]byte(cached), &strategy); err == nil {
			// Store in in-memory cache for future fast access
			u.strategyCache.Store(domain, &strategyCacheEntry{
				strategy:  &strategy,
				expiresAt: time.Now().Add(u.cacheTTL),
			})
			return &strategy
		}
	}

	// Tier 3: Try DB
	if u.pool != nil {
		strategy, err := u.loadStrategyFromDB(ctx, domain)
		if err == nil && strategy != nil {
			u.cacheStrategy(ctx, strategy)
			// Also store in in-memory cache
			u.strategyCache.Store(domain, &strategyCacheEntry{
				strategy:  strategy,
				expiresAt: time.Now().Add(u.cacheTTL),
			})
			return strategy
		}
	}

	// No strategy found - optimistic: start at Tier 0
	if u.config.StartAtTierZero {
		optimistic := &models.DomainStrategy{
			Domain:          domain,
			RecommendedTier: models.TierDirect,
			LearningStatus:  models.StatusNew,
		}
		// Cache optimistic strategy briefly
		u.strategyCache.Store(domain, &strategyCacheEntry{
			strategy:  optimistic,
			expiresAt: time.Now().Add(10 * time.Second),
		})
		return optimistic
	}

	return nil
}

// loadStrategyFromDB loads a single strategy from DB
func (u *SmartUnblocker) loadStrategyFromDB(ctx context.Context, domain string) (*models.DomainStrategy, error) {
	query := `
		SELECT id, domain, recommended_tier, tier_confidence, 
		       session_requirement, detected_cookies,
		       optimal_rotation_count, rotation_confidence,
		       adaptive_delay_ms, max_concurrent_requests,
		       total_requests, total_successes, total_failures,
		       learning_status, sample_size, tier_attempts,
		       created_at, updated_at
		FROM domain_strategies
		WHERE domain = $1
	`

	var s models.DomainStrategy
	var optRotation, maxConcurrent *int
	var tierAttemptsJSON []byte

	err := u.pool.QueryRow(ctx, query, domain).Scan(
		&s.ID, &s.Domain, &s.RecommendedTier, &s.TierConfidence,
		&s.SessionRequirement, &s.DetectedCookies,
		&optRotation, &s.RotationConfidence,
		&s.AdaptiveDelayMs, &maxConcurrent,
		&s.TotalRequests, &s.TotalSuccesses, &s.TotalFailures,
		&s.LearningStatus, &s.SampleSize, &tierAttemptsJSON,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	s.OptimalRotationCount = optRotation
	s.MaxConcurrentRequests = maxConcurrent
	s.ParseTierAttemptsJSON(tierAttemptsJSON)

	return &s, nil
}

// cacheStrategy caches a strategy in Redis (full JSON, not just TierAttempts)
func (u *SmartUnblocker) cacheStrategy(ctx context.Context, s *models.DomainStrategy) {
	cacheKey := fmt.Sprintf(keyDomainStrategy, s.Domain)
	data, err := json.Marshal(s)
	if err != nil {
		logger.Warn("Failed to marshal strategy for cache", zap.Error(err))
		return
	}
	u.cache.Set(ctx, cacheKey, string(data), u.config.LearningTTL)
}

// RecordResult records a request result for learning
// OPTIMIZED: Uses Redis pipeline to batch 7+ operations into 1 round-trip
func (u *SmartUnblocker) RecordResult(ctx context.Context, executionID, domain string, tier models.ProxyTier, success bool, statusCode int, responseTime time.Duration) {
	if u.cache == nil {
		return
	}

	// Pre-compute keys
	statsKey := fmt.Sprintf(keyExecutionStats, executionID, domain)
	domainsKey := fmt.Sprintf(keyExecutionDomains, executionID)
	successKey := fmt.Sprintf(keyConsecutiveSuccess, domain)
	failureKey := fmt.Sprintf(keyConsecutiveFailure, domain)

	// Use pipeline to batch all Redis operations
	pipe := u.cache.Pipeline()

	// Track domain in execution set (for efficient persistence iteration)
	pipe.SAdd(ctx, domainsKey, domain)
	pipe.Expire(ctx, domainsKey, 2*time.Hour)

	if success {
		pipe.HIncrBy(ctx, statsKey, "successes", 1)
		pipe.HIncrBy(ctx, statsKey, fmt.Sprintf("tier_%d_success", tier), 1)
		pipe.Incr(ctx, successKey)
		pipe.Expire(ctx, successKey, 10*time.Minute)
		pipe.Del(ctx, failureKey)
	} else {
		pipe.HIncrBy(ctx, statsKey, "failures", 1)
		pipe.HIncrBy(ctx, statsKey, fmt.Sprintf("tier_%d_failure", tier), 1)
		pipe.HSet(ctx, statsKey, "last_status_code", fmt.Sprintf("%d", statusCode))
		pipe.Incr(ctx, failureKey)
		pipe.Expire(ctx, failureKey, 10*time.Minute)
		pipe.Del(ctx, successKey)
	}

	pipe.HIncrBy(ctx, statsKey, "total_requests", 1)
	pipe.Expire(ctx, statsKey, 2*time.Hour)

	// Execute all commands in single round-trip
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Warn("Failed to record result in pipeline", zap.Error(err))
	}

	// Also record in tiered proxy manager (separate for now - could also be pipelined)
	if u.tieredProxy != nil {
		u.tieredProxy.RecordTierResult(ctx, domain, tier, success)
	}
}

// GetOptimalTier returns the best tier to use for a domain
func (u *SmartUnblocker) GetOptimalTier(ctx context.Context, domain string) models.ProxyTier {
	// Check cached/learned strategy first
	strategy := u.GetStrategy(ctx, domain)
	if strategy != nil && strategy.LearningStatus == models.StatusStable {
		return strategy.RecommendedTier
	}

	// Check tiered proxy manager for real-time data
	if u.tieredProxy != nil {
		return u.tieredProxy.GetOptimalTierForDomain(ctx, domain)
	}

	// Optimistic default
	return models.TierDirect
}

// GetAdaptiveDelay returns the recommended delay for a domain
func (u *SmartUnblocker) GetAdaptiveDelay(ctx context.Context, domain string) time.Duration {
	strategy := u.GetStrategy(ctx, domain)
	if strategy != nil && strategy.AdaptiveDelayMs > 0 {
		return time.Duration(strategy.AdaptiveDelayMs) * time.Millisecond
	}
	return 0
}

// ShouldEscalate checks if we should try a higher tier
func (u *SmartUnblocker) ShouldEscalate(ctx context.Context, domain string, currentTier models.ProxyTier) (bool, models.ProxyTier) {
	if u.tieredProxy == nil {
		return false, currentTier
	}
	return u.tieredProxy.ShouldEscalateTier(ctx, domain, currentTier)
}

// PersistLearnedStrategies persists learned strategies to DB at end of execution
// OPTIMIZED: Uses SMEMBERS instead of pattern SCAN for O(1) domain iteration
func (u *SmartUnblocker) PersistLearnedStrategies(ctx context.Context, executionID string) error {
	if u.pool == nil || u.cache == nil {
		return nil
	}

	// Get domains from set (O(1) vs O(n) pattern scan)
	domainsKey := fmt.Sprintf(keyExecutionDomains, executionID)
	domains, err := u.cache.SMembers(ctx, domainsKey)
	if err != nil {
		return err
	}

	var statsKeysToDelete []string
	for _, domain := range domains {
		// Get stats key for this domain
		statsKey := fmt.Sprintf(keyExecutionStats, executionID, domain)
		statsKeysToDelete = append(statsKeysToDelete, statsKey)

		// Get stats from Redis
		data, err := u.cache.HGetAll(ctx, statsKey)
		if err != nil {
			continue
		}

		// Parse stats
		var totalRequests, successes, failures int64
		fmt.Sscanf(data["total_requests"], "%d", &totalRequests)
		fmt.Sscanf(data["successes"], "%d", &successes)
		fmt.Sscanf(data["failures"], "%d", &failures)

		// Parse session requirement (from RecordSessionCookies)
		sessionRequired := models.SessionUnknown
		if data["session_required"] == "1" {
			sessionRequired = models.SessionRequired
		}
		var detectedCookies []string
		if cookieStr := data["detected_cookies"]; cookieStr != "" {
			// Split by comma
			for _, c := range splitCookies(cookieStr) {
				if c != "" {
					detectedCookies = append(detectedCookies, c)
				}
			}
		}

		// Only persist if enough data
		if totalRequests < int64(u.config.PersistThreshold) {
			continue
		}

		// Get tier stats from tiered proxy manager
		var tierAttempts map[int]*models.TierStats
		if u.tieredProxy != nil {
			ts := u.tieredProxy.GetTierStats(ctx, domain)
			tierAttempts = make(map[int]*models.TierStats)
			for tier, stats := range ts {
				tierAttempts[int(tier)] = stats
			}
		}

		// Calculate recommended tier
		recommendedTier := u.calculateBestTier(tierAttempts)
		tierConfidence := u.calculateTierConfidence(tierAttempts, recommendedTier)
		learningStatus := models.StatusLearning
		if totalRequests >= int64(u.config.MinSamplesForStable) && tierConfidence >= u.config.ConfidenceThreshold {
			learningStatus = models.StatusStable
		}

		// Upsert to DB
		ds := &models.DomainStrategy{TierAttempts: tierAttempts}
		tierAttemptsJSON, _ := ds.TierAttemptsJSON()

		query := `
			INSERT INTO domain_strategies 
				(domain, recommended_tier, tier_confidence, 
				 session_requirement, detected_cookies,
				 total_requests, total_successes, total_failures,
				 sample_size, learning_status, tier_attempts)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (domain) DO UPDATE SET
				recommended_tier = CASE 
					WHEN EXCLUDED.tier_confidence > domain_strategies.tier_confidence 
					THEN EXCLUDED.recommended_tier 
					ELSE domain_strategies.recommended_tier 
				END,
				tier_confidence = GREATEST(EXCLUDED.tier_confidence, domain_strategies.tier_confidence),
				session_requirement = CASE
					WHEN EXCLUDED.session_requirement > domain_strategies.session_requirement
					THEN EXCLUDED.session_requirement
					ELSE domain_strategies.session_requirement
				END,
				detected_cookies = COALESCE(EXCLUDED.detected_cookies, domain_strategies.detected_cookies),
				total_requests = domain_strategies.total_requests + EXCLUDED.total_requests,
				total_successes = domain_strategies.total_successes + EXCLUDED.total_successes,
				total_failures = domain_strategies.total_failures + EXCLUDED.total_failures,
				sample_size = domain_strategies.sample_size + EXCLUDED.sample_size,
				learning_status = CASE 
					WHEN domain_strategies.sample_size + EXCLUDED.sample_size >= $12 THEN 'stable'
					ELSE 'learning' 
				END,
				tier_attempts = EXCLUDED.tier_attempts,
				updated_at = NOW()
		`

		_, err = u.pool.Exec(ctx, query,
			domain, recommendedTier, tierConfidence,
			sessionRequired, detectedCookies,
			totalRequests, successes, failures,
			totalRequests, learningStatus, tierAttemptsJSON,
			u.config.MinSamplesForStable,
		)
		if err != nil {
			logger.Error("Failed to persist domain strategy",
				zap.String("domain", domain),
				zap.Error(err),
			)
		} else {
			logger.Info("Persisted domain strategy",
				zap.String("domain", domain),
				zap.Int("recommended_tier", int(recommendedTier)),
				zap.String("status", string(learningStatus)),
			)
		}
	}

	// Cleanup execution stats keys from Redis
	for _, key := range statsKeysToDelete {
		u.cache.Delete(ctx, key)
	}
	// Cleanup domains set
	u.cache.Delete(ctx, domainsKey)

	return nil
}

// calculateBestTier finds the tier with highest success rate
func (u *SmartUnblocker) calculateBestTier(tierAttempts map[int]*models.TierStats) models.ProxyTier {
	bestTier := models.TierDirect
	bestSuccessRate := 0.0

	for tier, stats := range tierAttempts {
		if stats.Attempts == 0 {
			continue
		}
		successRate := float64(stats.Successes) / float64(stats.Attempts)
		if successRate > bestSuccessRate {
			bestSuccessRate = successRate
			bestTier = models.ProxyTier(tier)
		}
	}

	return bestTier
}

// calculateTierConfidence calculates confidence in the recommended tier
func (u *SmartUnblocker) calculateTierConfidence(tierAttempts map[int]*models.TierStats, tier models.ProxyTier) float64 {
	stats, ok := tierAttempts[int(tier)]
	if !ok || stats.Attempts == 0 {
		return 0
	}

	// Confidence is success rate * sample size factor
	successRate := float64(stats.Successes) / float64(stats.Attempts)
	sampleFactor := float64(stats.Attempts) / float64(u.config.MinSamplesForStable)
	if sampleFactor > 1 {
		sampleFactor = 1
	}

	return successRate * sampleFactor
}

// extractDomainFromKey extracts domain from Redis key
func extractDomainFromKey(key string) string {
	// Key format: unblocker:exec:{exec_id}:domain:{domain}
	// We need to extract the domain part
	parts := splitKey(key)
	if len(parts) >= 5 {
		return parts[len(parts)-1]
	}
	return ""
}

// splitKey splits a Redis key by ':'
func splitKey(key string) []string {
	var parts []string
	current := ""
	for _, c := range key {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// =====================================================
// SESSION DETECTION (Anti-Bot Cookie Based)
// =====================================================

// DetectAntiBotCookies checks if any cookie names match known anti-bot patterns
// Returns list of detected anti-bot cookies
func DetectAntiBotCookies(cookieNames []string) []string {
	var detected []string
	for _, name := range cookieNames {
		for _, pattern := range models.AntiBotCookiePatterns {
			// Check if cookie name starts with or equals the pattern
			if len(name) >= len(pattern) && name[:len(pattern)] == pattern {
				detected = append(detected, name)
				break
			}
		}
	}
	return detected
}

// RecordSessionCookies records detected anti-bot cookies for a domain
// This updates the session requirement in Redis for later persistence
func (u *SmartUnblocker) RecordSessionCookies(ctx context.Context, executionID, domain string, cookieNames []string) {
	if u.cache == nil {
		return
	}

	// Check for anti-bot cookies
	detected := DetectAntiBotCookies(cookieNames)
	if len(detected) == 0 {
		return
	}

	// Store in Redis hash for this domain
	statsKey := fmt.Sprintf(keyExecutionStats, executionID, domain)

	pipe := u.cache.Pipeline()
	pipe.HSet(ctx, statsKey, "session_required", "1")
	pipe.HSet(ctx, statsKey, "detected_cookies", joinCookies(detected))
	pipe.Exec(ctx)

	// Also update in-memory cache if exists
	if cached, ok := u.strategyCache.Load(domain); ok {
		entry := cached.(*strategyCacheEntry)
		entry.strategy.SessionRequirement = models.SessionRequired
		entry.strategy.DetectedCookies = detected
	}

	logger.Info("Anti-bot cookies detected",
		zap.String("domain", domain),
		zap.Strings("cookies", detected),
	)
}

// GetSessionRequirement returns whether a domain needs session reuse
func (u *SmartUnblocker) GetSessionRequirement(ctx context.Context, domain string) models.SessionRequirement {
	strategy := u.GetStrategy(ctx, domain)
	if strategy != nil {
		return strategy.SessionRequirement
	}
	return models.SessionUnknown
}

// NeedsSessionReuse returns true if the domain requires session reuse for anti-bot bypass
func (u *SmartUnblocker) NeedsSessionReuse(ctx context.Context, domain string) bool {
	req := u.GetSessionRequirement(ctx, domain)
	return req == models.SessionRequired
}

// joinCookies joins cookie names for storage
func joinCookies(cookies []string) string {
	result := ""
	for i, c := range cookies {
		if i > 0 {
			result += ","
		}
		result += c
	}
	return result
}

// splitCookies splits comma-separated cookie string
func splitCookies(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
