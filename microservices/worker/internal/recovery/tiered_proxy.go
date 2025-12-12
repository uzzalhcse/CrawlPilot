package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// TieredProxyManager extends DistributedProxyManager with tier-based escalation
// Implements the Crawlee tiered proxy pattern for cost-effective anti-bot handling
type TieredProxyManager struct {
	*DistributedProxyManager
	config *TieredProxyConfig
}

// TieredProxyConfig configures the tiered proxy system
type TieredProxyConfig struct {
	// Tier thresholds
	EscalateAfterFailures int           // Failures before escalating to next tier (default: 2)
	TierCooldown          time.Duration // Cooldown before retrying lower tier (default: 10m)

	// Learning
	MinSamplesForConfidence int     // Min requests before considering tier stable (default: 20)
	SuccessRateThreshold    float64 // Success rate to consider tier working (default: 0.8)
}

// DefaultTieredProxyConfig returns sensible defaults
func DefaultTieredProxyConfig() *TieredProxyConfig {
	return &TieredProxyConfig{
		EscalateAfterFailures:   2,
		TierCooldown:            10 * time.Minute,
		MinSamplesForConfidence: 20,
		SuccessRateThreshold:    0.8,
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
func (m *TieredProxyManager) selectProxyByTier(ctx context.Context, domain string, tier int) (string, error) {
	// Key for tier-specific proxies
	tierPoolKey := fmt.Sprintf("proxy:pool:tier:%d", tier)

	// Get available proxies from tier
	proxies, err := m.getAvailableFromSet(ctx, tierPoolKey, domain, 10)
	if err != nil || len(proxies) == 0 {
		// Try main pool filtered by tier
		return m.selectFromMainPoolByTier(ctx, domain, tier)
	}

	// Random selection from available
	return proxies[rand.Intn(len(proxies))], nil
}

// selectFromMainPoolByTier selects from main pool filtering by tier
func (m *TieredProxyManager) selectFromMainPoolByTier(ctx context.Context, domain string, tier int) (string, error) {
	// Get all proxies and filter by tier
	allProxies, err := m.getAvailableFromSet(ctx, keyProxyPool, domain, 100)
	if err != nil {
		return "", err
	}

	// Filter by tier
	var tierProxies []string
	for _, proxyID := range allProxies {
		proxy, err := m.getProxyData(ctx, proxyID)
		if err != nil {
			continue
		}
		if proxy.Tier == tier {
			tierProxies = append(tierProxies, proxyID)
		}
	}

	if len(tierProxies) == 0 {
		return "", fmt.Errorf("no proxies available for tier %d", tier)
	}

	return tierProxies[rand.Intn(len(tierProxies))], nil
}

// GetOptimalTierForDomain returns the recommended tier for a domain
func (m *TieredProxyManager) GetOptimalTierForDomain(ctx context.Context, domain string) models.ProxyTier {
	tierKey := fmt.Sprintf(keyDomainTier, domain)

	// Check cached tier
	tierStr, err := m.cache.Get(ctx, tierKey)
	if err == nil && tierStr != "" {
		var tier int
		fmt.Sscanf(tierStr, "%d", &tier)
		return models.ProxyTier(tier)
	}

	// Default: start at Tier 0 (direct) - optimistic approach
	return models.TierDirect
}

// SetDomainTier sets the recommended tier for a domain
func (m *TieredProxyManager) SetDomainTier(ctx context.Context, domain string, tier models.ProxyTier) error {
	tierKey := fmt.Sprintf(keyDomainTier, domain)
	return m.cache.Set(ctx, tierKey, fmt.Sprintf("%d", tier), 24*time.Hour)
}

// RecordTierResult records success/failure for tier learning
func (m *TieredProxyManager) RecordTierResult(ctx context.Context, domain string, tier models.ProxyTier, success bool) error {
	attemptKey := fmt.Sprintf(keyTierAttempts, domain, tier)

	if success {
		m.cache.HIncrBy(ctx, attemptKey, "successes", 1)
	} else {
		m.cache.HIncrBy(ctx, attemptKey, "failures", 1)
	}
	m.cache.HIncrBy(ctx, attemptKey, "attempts", 1)
	m.cache.Expire(ctx, attemptKey, 24*time.Hour)

	return nil
}

// ShouldEscalateTier determines if we should try a higher tier
func (m *TieredProxyManager) ShouldEscalateTier(ctx context.Context, domain string, currentTier models.ProxyTier) (bool, models.ProxyTier) {
	if currentTier >= models.TierMobile {
		return false, currentTier // Already at highest tier
	}

	attemptKey := fmt.Sprintf(keyTierAttempts, domain, currentTier)
	data, err := m.cache.HGetAll(ctx, attemptKey)
	if err != nil {
		return false, currentTier
	}

	var failures, attempts int
	fmt.Sscanf(data["failures"], "%d", &failures)
	fmt.Sscanf(data["attempts"], "%d", &attempts)

	// Escalate after N consecutive failures
	if failures >= m.config.EscalateAfterFailures {
		nextTier := currentTier + 1
		logger.Info("Escalating proxy tier",
			zap.String("domain", domain),
			zap.Int("from_tier", int(currentTier)),
			zap.Int("to_tier", int(nextTier)),
			zap.Int("failures", failures),
		)
		return true, nextTier
	}

	return false, currentTier
}

// GetTierStats returns statistics for all tiers for a domain
func (m *TieredProxyManager) GetTierStats(ctx context.Context, domain string) map[models.ProxyTier]*models.TierStats {
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
func (m *TieredProxyManager) SeedProxiesWithTiers(ctx context.Context, proxies []Proxy) error {
	// Clear existing pools to remove stale proxies from previous runs
	m.cache.Delete(ctx, keyProxyPool)
	// Also clear tier-specific pools
	for tier := 0; tier <= 3; tier++ {
		tierPoolKey := fmt.Sprintf("proxy:pool:tier:%d", tier)
		m.cache.Delete(ctx, tierPoolKey)
	}

	for _, proxy := range proxies {
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
	}

	logger.Info("Proxies seeded with tiers",
		zap.Int("count", len(proxies)),
	)

	return nil
}
