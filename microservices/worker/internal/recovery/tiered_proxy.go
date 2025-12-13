package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// TieredProxyManager extends DistributedProxyManager with tier-based escalation
// Implements the Crawlee tiered proxy pattern for cost-effective anti-bot handling
type TieredProxyManager struct {
	*DistributedProxyManager
	config         *TieredProxyConfig
	knownProtected *KnownProtectedDomains // Skip Tier 0 for known protected domains
	dbPool         *pgxpool.Pool          // Database pool for tier persistence
}

// TieredProxyConfig configures the tiered proxy system
type TieredProxyConfig struct {
	// Tier thresholds
	EscalateAfterFailures int           // Failures before escalating to next tier (default: 2)
	TierCooldown          time.Duration // Cooldown before retrying lower tier (default: 10m)

	// Learning
	MinSamplesForConfidence int     // Min requests before considering tier stable (default: 20)
	SuccessRateThreshold    float64 // Success rate to consider tier working (default: 0.8)

	// De-escalation
	DeescalationMinSamples int // Min successful requests before de-escalating (default: 5)
}

// DefaultTieredProxyConfig returns sensible defaults
func DefaultTieredProxyConfig() *TieredProxyConfig {
	return &TieredProxyConfig{
		EscalateAfterFailures:   2,
		TierCooldown:            10 * time.Minute,
		MinSamplesForConfidence: 20,
		SuccessRateThreshold:    0.8,
		DeescalationMinSamples:  5,
	}
}

// NewTieredProxyManager creates a new tiered proxy manager
func NewTieredProxyManager(c *cache.Cache, baseConfig *ProxyRotationConfig, tieredConfig *TieredProxyConfig) *TieredProxyManager {
	if tieredConfig == nil {
		tieredConfig = DefaultTieredProxyConfig()
	}

	return &TieredProxyManager{
		DistributedProxyManager: NewDistributedProxyManager(c, baseConfig),
		config:                  tieredConfig,
		knownProtected:          NewKnownProtectedDomains(c),
	}
}

// Redis keys for tiered system
const (
	keyDomainTier   = "domain:tier:%s"             // Current tier for domain
	keyDomainStats  = "domain:stats:%s"            // Stats for domain (per execution)
	keyTierAttempts = "domain:tier_attempts:%s:%d" // Attempts per tier per domain
)

// GetProxyForTier gets a proxy for the specified tier
func (m *TieredProxyManager) GetProxyForTier(ctx context.Context, domain string, tier models.ProxyTier) (*Proxy, *ProxyLease, error) {
	if m.cache == nil {
		return nil, nil, fmt.Errorf("cache not initialized")
	}

	// Tier 0 = direct, no proxy
	if tier == models.TierDirect {
		return nil, nil, nil // No proxy needed
	}

	// Get proxies for the specified tier
	proxyID, err := m.selectProxyByTier(ctx, domain, int(tier))
	if err != nil {
		// If no proxy available at this tier, try lower tier
		if tier > models.TierDatacenter {
			logger.Debug("No proxy at tier, trying lower tier",
				zap.Int("requested_tier", int(tier)),
				zap.String("domain", domain),
			)
			return m.GetProxyForTier(ctx, domain, tier-1)
		}
		return nil, nil, err
	}

	if proxyID == "" {
		return nil, nil, fmt.Errorf("no available proxies for tier %d", tier)
	}

	// Acquire lease
	lease, err := m.acquireLease(ctx, proxyID, domain)
	if err != nil {
		// Proxy was taken, try again
		return m.GetProxyForTier(ctx, domain, tier)
	}

	// Get full proxy data
	proxy, err := m.getProxyData(ctx, proxyID)
	if err != nil {
		m.releaseLease(ctx, proxyID)
		return nil, nil, err
	}

	logger.Debug("Tiered proxy allocated",
		zap.String("proxy_id", proxyID),
		zap.Int("tier", int(tier)),
		zap.String("domain", domain),
	)

	return proxy, lease, nil
}

// selectProxyByTier selects a proxy from the specified tier
// FIXED: No longer falls back to in-memory filtering which caused race conditions
// All proxies MUST be in tier-specific pools (seeded on startup)
func (m *TieredProxyManager) selectProxyByTier(ctx context.Context, domain string, tier int) (string, error) {
	// Key for tier-specific proxies
	tierPoolKey := fmt.Sprintf("proxy:pool:tier:%d", tier)

	// Get available proxies from tier pool directly
	proxies, err := m.getAvailableFromSet(ctx, tierPoolKey, domain, 10)
	if err != nil {
		return "", fmt.Errorf("failed to get proxies for tier %d: %w", tier, err)
	}
	if len(proxies) == 0 {
		// No fallback to main pool - tier pools should be complete
		// Log and return error so caller can escalate to next tier
		logger.Debug("No available proxies in tier pool",
			zap.Int("tier", tier),
			zap.String("domain", domain),
		)
		return "", fmt.Errorf("no proxies available for tier %d", tier)
	}

	// Random selection from available
	return proxies[rand.Intn(len(proxies))], nil
}

// GetOptimalTierForDomain returns the recommended tier for a domain
// IMPROVED: Checks known protected domains to skip Tier 0 waste
func (m *TieredProxyManager) GetOptimalTierForDomain(ctx context.Context, domain string) models.ProxyTier {
	// Normalize domain for consistent lookups
	domain = normalizeDomainName(domain)

	// First: Check if domain has a DB-defined strategy (authoritative source)
	if m.knownProtected != nil {
		minTier, fromDB := m.knownProtected.GetMinimumTierWithSource(ctx, domain)
		if fromDB {
			// DB is the source of truth - use this tier, even if it's 0 (direct)
			logger.Debug("Domain tier from DB (authoritative)",
				zap.String("domain", domain),
				zap.Int("tier", int(minTier)),
			)
			return minTier
		}
		// Not in DB, but knownProtected might have learned it from Redis
		if minTier > models.TierDirect {
			logger.Debug("Domain requires minimum tier (learned)",
				zap.String("domain", domain),
				zap.Int("min_tier", int(minTier)),
			)
			return minTier
		}
	}

	// Second: Check cached/learned tier from previous requests (only for domains NOT in DB)
	tierKey := fmt.Sprintf(keyDomainTier, domain)
	tierStr, err := m.cache.Get(ctx, tierKey)
	if err == nil && tierStr != "" {
		var tier int
		fmt.Sscanf(tierStr, "%d", &tier)
		return models.ProxyTier(tier)
	}

	// Default: start at Tier 0 (direct) - optimistic for unknown domains
	return models.TierDirect
}

// SetDomainTier sets the recommended tier for a domain (Redis only - use SetDomainTierWithDB for persistence)
func (m *TieredProxyManager) SetDomainTier(ctx context.Context, domain string, tier models.ProxyTier) error {
	domain = normalizeDomainName(domain)
	tierKey := fmt.Sprintf(keyDomainTier, domain)
	return m.cache.Set(ctx, tierKey, fmt.Sprintf("%d", tier), 24*time.Hour)
}

// SetDBPool sets the database pool for tier persistence
func (m *TieredProxyManager) SetDBPool(pool *pgxpool.Pool) {
	m.dbPool = pool
}

// SetDomainTierWithDB sets the tier in Redis AND persists to domain_strategies table
// This ensures learned tiers survive beyond the 24h Redis TTL
func (m *TieredProxyManager) SetDomainTierWithDB(ctx context.Context, domain string, tier models.ProxyTier, confidence float64) error {
	domain = normalizeDomainName(domain)

	logger.Debug("SetDomainTierWithDB called",
		zap.String("domain", domain),
		zap.Int("tier", int(tier)),
		zap.Float64("confidence", confidence),
		zap.Bool("dbPool_nil", m.dbPool == nil),
	)

	// 1. Set in Redis cache (fast path)
	tierKey := fmt.Sprintf(keyDomainTier, domain)
	if err := m.cache.Set(ctx, tierKey, fmt.Sprintf("%d", tier), 24*time.Hour); err != nil {
		logger.Warn("Failed to set tier in Redis", zap.String("domain", domain), zap.Error(err))
		// Continue to try DB persistence
	}

	// 2. Update in-memory cache (knownProtected)
	if m.knownProtected != nil {
		m.knownProtected.AddStaticDomain(domain, tier)
	}

	// 3. Persist to database (durable storage)
	if m.dbPool == nil {
		logger.Warn("SetDomainTierWithDB: dbPool is nil, cannot persist to database",
			zap.String("domain", domain),
			zap.Int("tier", int(tier)),
		)
		return nil
	}

	query := `
		INSERT INTO domain_strategies (domain, recommended_tier, tier_confidence, learning_status, sample_size)
		VALUES ($1, $2, $3, 'stable', 1)
		ON CONFLICT (domain) DO UPDATE SET
			recommended_tier = CASE 
				-- Always escalate: Use new tier if it's higher (we learned from failure)
				WHEN EXCLUDED.recommended_tier > domain_strategies.recommended_tier THEN EXCLUDED.recommended_tier
				-- De-escalate: Use new tier if it's lower AND has good confidence
				WHEN EXCLUDED.recommended_tier < domain_strategies.recommended_tier 
					AND EXCLUDED.tier_confidence >= 0.5 THEN EXCLUDED.recommended_tier
				ELSE domain_strategies.recommended_tier
			END,
			tier_confidence = GREATEST(EXCLUDED.tier_confidence, domain_strategies.tier_confidence),
			learning_status = 'stable',
			sample_size = domain_strategies.sample_size + 1,
			updated_at = NOW()
	`
	if _, err := m.dbPool.Exec(ctx, query, domain, tier, confidence); err != nil {
		logger.Warn("Failed to persist tier to DB",
			zap.String("domain", domain),
			zap.Int("tier", int(tier)),
			zap.Error(err),
		)
		// Don't fail - Redis write may have succeeded
	} else {
		logger.Info("Persisted domain tier to DB",
			zap.String("domain", domain),
			zap.Int("tier", int(tier)),
			zap.Float64("confidence", confidence),
		)
	}

	return nil
}

// RecordTierResult records success/failure for tier learning
// Also triggers tier de-escalation if lower tier succeeds (with min sample check)
func (m *TieredProxyManager) RecordTierResult(ctx context.Context, domain string, tier models.ProxyTier, success bool) error {
	domain = normalizeDomainName(domain)
	attemptKey := fmt.Sprintf(keyTierAttempts, domain, tier)

	if success {
		m.cache.HIncrBy(ctx, attemptKey, "successes", 1)

		// BIDIRECTIONAL LEARNING: Only de-escalate after minimum successful samples
		if m.knownProtected != nil && m.config != nil {
			// Get current success count for this tier
			data, _ := m.cache.HGetAll(ctx, attemptKey)
			successes := 0
			if val, ok := data["successes"]; ok {
				fmt.Sscanf(val, "%d", &successes)
			}

			// Only de-escalate if we have enough successful samples
			minSamples := m.config.DeescalationMinSamples
			if minSamples <= 0 {
				minSamples = 5 // Default fallback
			}
			if successes >= minSamples {
				m.knownProtected.UpdateTierIfLower(ctx, domain, tier)
			}
		}
	} else {
		m.cache.HIncrBy(ctx, attemptKey, "failures", 1)
		logger.Debug("Recorded tier failure",
			zap.String("domain", domain),
			zap.Int("tier", int(tier)),
			zap.String("key", attemptKey),
		)
	}
	m.cache.HIncrBy(ctx, attemptKey, "attempts", 1)
	m.cache.Expire(ctx, attemptKey, 24*time.Hour)

	return nil
}

// ShouldEscalateTier determines if we should try a higher tier
func (m *TieredProxyManager) ShouldEscalateTier(ctx context.Context, domain string, currentTier models.ProxyTier) (bool, models.ProxyTier) {
	domain = normalizeDomainName(domain)
	if currentTier >= models.TierMobile {
		return false, currentTier // Already at highest tier
	}

	attemptKey := fmt.Sprintf(keyTierAttempts, domain, currentTier)
	data, err := m.cache.HGetAll(ctx, attemptKey)
	if err != nil {
		logger.Debug("Failed to get tier attempts from cache",
			zap.Error(err),
			zap.String("domain", domain),
			zap.Int("tier", int(currentTier)),
		)
		return false, currentTier
	}

	var failures, attempts int
	fmt.Sscanf(data["failures"], "%d", &failures)
	fmt.Sscanf(data["attempts"], "%d", &attempts)

	logger.Debug("Checking tier escalation",
		zap.String("domain", domain),
		zap.Int("current_tier", int(currentTier)),
		zap.Int("failures", failures),
		zap.Int("attempts", attempts),
		zap.Int("threshold", m.config.EscalateAfterFailures),
	)

	// Escalate after N consecutive failures
	if failures >= m.config.EscalateAfterFailures {
		nextTier := currentTier + 1
		logger.Info("Escalating proxy tier",
			zap.String("domain", domain),
			zap.Int("from_tier", int(currentTier)),
			zap.Int("to_tier", int(nextTier)),
			zap.Int("failures", failures),
		)
		// Learn that this domain requires at least the next tier
		if m.knownProtected != nil {
			m.knownProtected.LearnMinimumTier(ctx, domain, nextTier)
		}
		return true, nextTier
	}

	return false, currentTier
}

// GetTierStats returns statistics for all tiers for a domain
func (m *TieredProxyManager) GetTierStats(ctx context.Context, domain string) map[models.ProxyTier]*models.TierStats {
	domain = normalizeDomainName(domain)
	stats := make(map[models.ProxyTier]*models.TierStats)

	for tier := models.TierDirect; tier <= models.TierMobile; tier++ {
		attemptKey := fmt.Sprintf(keyTierAttempts, domain, tier)
		data, err := m.cache.HGetAll(ctx, attemptKey)
		if err != nil {
			continue
		}

		var attempts, successes, failures int
		fmt.Sscanf(data["attempts"], "%d", &attempts)
		fmt.Sscanf(data["successes"], "%d", &successes)
		fmt.Sscanf(data["failures"], "%d", &failures)

		if attempts > 0 {
			stats[tier] = &models.TierStats{
				Attempts:  attempts,
				Successes: successes,
				Failures:  failures,
			}
		}
	}

	return stats
}

// SeedProxiesWithTiers seeds proxies with tier information
// IMPROVED: Skips unhealthy/invalid proxies to prevent seeding dead proxies
func (m *TieredProxyManager) SeedProxiesWithTiers(ctx context.Context, proxies []Proxy) error {
	// Clear existing pools to remove stale proxies from previous runs
	m.cache.Delete(ctx, keyProxyPool)
	// Also clear tier-specific pools
	for tier := 0; tier <= 3; tier++ {
		tierPoolKey := fmt.Sprintf("proxy:pool:tier:%d", tier)
		m.cache.Delete(ctx, tierPoolKey)
	}

	seededCount := 0
	skippedUnhealthy := 0

	for _, proxy := range proxies {
		// FIXED: Skip unhealthy or invalid proxies
		if !proxy.IsHealthy || !proxy.Valid {
			skippedUnhealthy++
			continue
		}

		// Store proxy data
		dataKey := fmt.Sprintf(keyProxyData, proxy.ID)
		data, _ := json.Marshal(proxy)
		if err := m.cache.Set(ctx, dataKey, string(data), 0); err != nil {
			return err
		}

		// Add to main pool
		if err := m.cache.ZAdd(ctx, keyProxyPool, 0, proxy.ID); err != nil {
			return err
		}

		// Add to tier-specific pool
		tierPoolKey := fmt.Sprintf("proxy:pool:tier:%d", proxy.Tier)
		if err := m.cache.ZAdd(ctx, tierPoolKey, 0, proxy.ID); err != nil {
			return err
		}

		seededCount++
	}

	if skippedUnhealthy > 0 {
		logger.Warn("Skipped unhealthy/invalid proxies during seeding",
			zap.Int("skipped", skippedUnhealthy),
		)
	}

	logger.Info("Proxies seeded with tiers",
		zap.Int("seeded", seededCount),
		zap.Int("skipped_unhealthy", skippedUnhealthy),
		zap.Int("total_input", len(proxies)),
	)

	return nil
}
