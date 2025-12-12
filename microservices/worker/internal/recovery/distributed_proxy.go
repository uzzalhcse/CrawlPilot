package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DistributedProxyManager handles proxy rotation across multiple workers
// Uses Redis for coordination to ensure:
// 1. Even distribution of proxy usage across workers
// 2. Domain-affinity: prefer proxies that work for specific domains
// 3. Lease-based allocation to prevent simultaneous use
// 4. Cooldown after failures to prevent hammering
type DistributedProxyManager struct {
	cache   *cache.Cache
	config  *ProxyRotationConfig
	localID string        // Unique ID for this worker instance
	dbPool  *pgxpool.Pool // Optional: database pool for syncing proxy state
}

// ProxyRotationConfig configures the distributed proxy manager
type ProxyRotationConfig struct {
	// Lease settings
	LeaseDuration    time.Duration // How long a proxy is "locked" after allocation (default: 30s)
	CooldownDuration time.Duration // How long to wait after failure (default: 60s)

	// Selection strategy
	Strategy           RotationStrategy
	MaxFailuresPerHour int           // Max failures before temporary disable (default: 5)
	HealthCheckTTL     time.Duration // How long health data is valid (default: 5m)

	// Domain affinity
	DomainAffinityWeight float64 // 0-1, how much to prefer domain-tested proxies (default: 0.7)
	MaxProxiesPerDomain  int     // Max proxies to track per domain (default: 20)
}

// RotationStrategy defines how proxies are selected
type RotationStrategy string

const (
	StrategyRoundRobin     RotationStrategy = "round_robin"     // Evenly distribute
	StrategyLeastUsed      RotationStrategy = "least_used"      // Prefer less used
	StrategyRandom         RotationStrategy = "random"          // Random selection
	StrategyDomainAffinity RotationStrategy = "domain_affinity" // Prefer proxies that work for domain
)

// DefaultProxyRotationConfig returns sensible defaults
func DefaultProxyRotationConfig() *ProxyRotationConfig {
	return &ProxyRotationConfig{
		LeaseDuration:        30 * time.Second,
		CooldownDuration:     60 * time.Second,
		Strategy:             StrategyDomainAffinity,
		MaxFailuresPerHour:   5,
		HealthCheckTTL:       5 * time.Minute,
		DomainAffinityWeight: 0.7,
		MaxProxiesPerDomain:  20,
	}
}

// ProxyLease represents a leased proxy for exclusive use
type ProxyLease struct {
	ProxyID   string    `json:"proxy_id"`
	WorkerID  string    `json:"worker_id"`
	Domain    string    `json:"domain"`
	LeasedAt  time.Time `json:"leased_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ProxyHealth stores health metrics in Redis
type ProxyHealth struct {
	ProxyID        string    `json:"proxy_id"`
	TotalRequests  int64     `json:"total_requests"`
	SuccessCount   int64     `json:"success_count"`
	FailureCount   int64     `json:"failure_count"`
	LastSuccess    time.Time `json:"last_success"`
	LastFailure    time.Time `json:"last_failure"`
	HourlyFailures int       `json:"hourly_failures"`
	IsDisabled     bool      `json:"is_disabled"`
	DisabledUntil  time.Time `json:"disabled_until"`
}

// NewDistributedProxyManager creates a new distributed proxy manager
func NewDistributedProxyManager(c *cache.Cache, config *ProxyRotationConfig) *DistributedProxyManager {
	if config == nil {
		config = DefaultProxyRotationConfig()
	}

	// Generate unique worker ID
	localID := fmt.Sprintf("worker-%d-%d", time.Now().UnixNano(), rand.Intn(10000))

	return &DistributedProxyManager{
		cache:   c,
		config:  config,
		localID: localID,
	}
}

// SetDBPool sets the database pool for syncing proxy state to the database
// This should be called after creation to enable database updates
func (m *DistributedProxyManager) SetDBPool(pool *pgxpool.Pool) {
	m.dbPool = pool
}

// Redis key helpers
const (
	keyProxyPool     = "proxy:pool"           // Sorted set: proxy IDs scored by usage count
	keyProxyDomain   = "proxy:domain:%s"      // Sorted set: proxies that work for domain
	keyProxyLease    = "proxy:lease:%s"       // String: lease info for proxy
	keyProxyCooldown = "proxy:cooldown:%s:%s" // String: cooldown for proxy+domain
	keyProxyHealth   = "proxy:health:%s"      // Hash: health metrics for proxy
	keyProxyRotation = "proxy:rotation:%s"    // Counter: round-robin index per domain
	keyProxyData     = "proxy:data:%s"        // String: full proxy configuration
)

// GetProxy acquires a proxy lease for the given domain
// Returns the best available proxy based on strategy
func (m *DistributedProxyManager) GetProxy(ctx context.Context, domain string) (*Proxy, *ProxyLease, error) {
	if m.cache == nil {
		return nil, nil, fmt.Errorf("cache not initialized")
	}

	// Try to get a proxy based on strategy
	var proxyID string
	var err error

	switch m.config.Strategy {
	case StrategyDomainAffinity:
		proxyID, err = m.selectDomainAffinity(ctx, domain)
	case StrategyLeastUsed:
		proxyID, err = m.selectLeastUsed(ctx, domain)
	case StrategyRandom:
		proxyID, err = m.selectRandom(ctx, domain)
	default: // StrategyRoundRobin
		proxyID, err = m.selectRoundRobin(ctx, domain)
	}

	if err != nil {
		return nil, nil, err
	}
	if proxyID == "" {
		return nil, nil, fmt.Errorf("no available proxies for domain %s", domain)
	}

	// Try to acquire lease
	lease, err := m.acquireLease(ctx, proxyID, domain)
	if err != nil {
		// Proxy was taken, try again with next best
		return m.GetProxy(ctx, domain)
	}

	// Get full proxy data
	proxy, err := m.getProxyData(ctx, proxyID)
	if err != nil {
		m.releaseLease(ctx, proxyID)
		return nil, nil, err
	}

	logger.Debug("Proxy allocated",
		zap.String("proxy_id", proxyID),
		zap.String("domain", domain),
		zap.String("worker", m.localID),
	)

	return proxy, lease, nil
}

// selectDomainAffinity selects a proxy that has worked for this domain before
func (m *DistributedProxyManager) selectDomainAffinity(ctx context.Context, domain string) (string, error) {
	domainKey := fmt.Sprintf(keyProxyDomain, domain)

	// First, try domain-specific proxies (sorted by success rate)
	// Use weighted random: 70% chance to pick from domain-tested, 30% from general pool
	if rand.Float64() < m.config.DomainAffinityWeight {
		proxies, err := m.getAvailableFromSet(ctx, domainKey, domain, 5)
		if err == nil && len(proxies) > 0 {
			// Pick random from top 5 for this domain
			return proxies[rand.Intn(len(proxies))], nil
		}
	}

	// Fall back to least-used from general pool
	return m.selectLeastUsed(ctx, domain)
}

// selectLeastUsed selects the proxy with fewest total uses
func (m *DistributedProxyManager) selectLeastUsed(ctx context.Context, domain string) (string, error) {
	// Get bottom 10 proxies by usage count
	proxies, err := m.getAvailableFromSet(ctx, keyProxyPool, domain, 10)
	if err != nil {
		return "", err
	}
	if len(proxies) == 0 {
		return "", fmt.Errorf("no proxies available")
	}

	// Pick random from bottom 10 to add some distribution
	return proxies[rand.Intn(len(proxies))], nil
}

// selectRoundRobin selects the next proxy in rotation for this domain
func (m *DistributedProxyManager) selectRoundRobin(ctx context.Context, domain string) (string, error) {
	rotationKey := fmt.Sprintf(keyProxyRotation, domain)

	// Atomically increment rotation counter
	counter, err := m.cache.Increment(ctx, rotationKey)
	if err != nil {
		counter = 0
	}

	// Get all available proxies
	proxies, err := m.getAvailableFromSet(ctx, keyProxyPool, domain, 100)
	if err != nil || len(proxies) == 0 {
		return "", fmt.Errorf("no proxies available")
	}

	// Round-robin selection
	index := int(counter) % len(proxies)
	return proxies[index], nil
}

// selectRandom selects a random available proxy
func (m *DistributedProxyManager) selectRandom(ctx context.Context, domain string) (string, error) {
	proxies, err := m.getAvailableFromSet(ctx, keyProxyPool, domain, 50)
	if err != nil || len(proxies) == 0 {
		return "", fmt.Errorf("no proxies available")
	}

	return proxies[rand.Intn(len(proxies))], nil
}

// getAvailableFromSet gets available proxies from a sorted set
// Filters out: leased, cooldown, disabled
func (m *DistributedProxyManager) getAvailableFromSet(ctx context.Context, setKey, domain string, limit int) ([]string, error) {
	// Get proxies sorted by score (usage count)
	members, err := m.cache.ZRangeByScore(ctx, setKey, 0, int(time.Now().Unix()), limit*2)
	if err != nil {
		return nil, err
	}

	// Debug: log pool size
	if len(members) == 0 {
		poolCount, _ := m.cache.ZCard(ctx, setKey)
		logger.Debug("Proxy pool query returned zero members",
			zap.String("set_key", setKey),
			zap.Int64("pool_total_count", poolCount),
			zap.Int("limit_requested", limit*2),
		)
	}

	available := make([]string, 0, limit)
	for _, proxyID := range members {
		if len(available) >= limit {
			break
		}

		// Check if leased
		leaseKey := fmt.Sprintf(keyProxyLease, proxyID)
		exists, _ := m.cache.Exists(ctx, leaseKey)
		if exists {
			continue
		}

		// Check if in cooldown for this domain
		// DISABLED: Cooldown wastes billable time on Cloud Run/serverless
		// When all proxies are in cooldown, we'd rather try again than wait
		// cooldownKey := fmt.Sprintf(keyProxyCooldown, proxyID, domain)
		// exists, _ = m.cache.Exists(ctx, cooldownKey)
		// if exists {
		// 	continue
		// }

		// Check if disabled
		health, _ := m.getHealth(ctx, proxyID)
		if health != nil && health.IsDisabled && time.Now().Before(health.DisabledUntil) {
			continue
		}

		available = append(available, proxyID)
	}

	// Debug: log if proxies existed but all were filtered out
	if len(available) == 0 && len(members) > 0 {
		logger.Debug("All proxies filtered by availability checks",
			zap.String("set_key", setKey),
			zap.Int("pool_members_found", len(members)),
			zap.String("domain", domain),
			zap.String("reason", "all proxies leased, in cooldown, or disabled"),
		)
	}

	return available, nil
}

// acquireLease tries to acquire an exclusive lease on a proxy
func (m *DistributedProxyManager) acquireLease(ctx context.Context, proxyID, domain string) (*ProxyLease, error) {
	leaseKey := fmt.Sprintf(keyProxyLease, proxyID)

	lease := &ProxyLease{
		ProxyID:   proxyID,
		WorkerID:  m.localID,
		Domain:    domain,
		LeasedAt:  time.Now(),
		ExpiresAt: time.Now().Add(m.config.LeaseDuration),
	}

	leaseData, _ := json.Marshal(lease)

	// SetNX returns true only if key didn't exist (we got the lock)
	success, err := m.cache.SetNX(ctx, leaseKey, string(leaseData), m.config.LeaseDuration)
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, fmt.Errorf("proxy %s already leased", proxyID)
	}

	return lease, nil
}

// releaseLease releases a proxy lease
func (m *DistributedProxyManager) releaseLease(ctx context.Context, proxyID string) error {
	leaseKey := fmt.Sprintf(keyProxyLease, proxyID)
	return m.cache.Delete(ctx, leaseKey)
}

// RecordSuccess records a successful request with a proxy
// OPTIMIZED: Uses pipeline to batch 6 Redis calls into 1 round-trip
func (m *DistributedProxyManager) RecordSuccess(ctx context.Context, proxyID, domain string) error {
	// Release lease (separate call - needs SetNX check)
	m.releaseLease(ctx, proxyID)

	// PIPELINE: Batch all updates into single round-trip
	healthKey := fmt.Sprintf(keyProxyHealth, proxyID)
	domainKey := fmt.Sprintf(keyProxyDomain, domain)

	pipe := m.cache.Pipeline()
	pipe.HIncrBy(ctx, healthKey, "total_requests", 1)
	pipe.HIncrBy(ctx, healthKey, "success_count", 1)
	pipe.HSet(ctx, healthKey, "last_success", time.Now().Format(time.RFC3339))
	pipe.ZIncrBy(ctx, domainKey, 1, proxyID)
	pipe.Expire(ctx, domainKey, 24*time.Hour)
	pipe.ZIncrBy(ctx, keyProxyPool, 1, proxyID)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Warn("Pipeline error in RecordSuccess", zap.Error(err))
	}

	logger.Debug("Proxy success recorded",
		zap.String("proxy_id", proxyID),
		zap.String("domain", domain),
	)

	return nil
}

// RecordFailure records a failed request with a proxy
// OPTIMIZED: Uses pipeline where possible to reduce round-trips
func (m *DistributedProxyManager) RecordFailure(ctx context.Context, proxyID, domain string, pattern ErrorPattern) error {
	// Release lease (separate call)
	m.releaseLease(ctx, proxyID)

	healthKey := fmt.Sprintf(keyProxyHealth, proxyID)
	domainKey := fmt.Sprintf(keyProxyDomain, domain)
	cooldownKey := fmt.Sprintf(keyProxyCooldown, proxyID, domain)
	cooldownDuration := m.getCooldownDuration(pattern)

	// PIPELINE: Batch health metrics updates
	pipe := m.cache.Pipeline()
	pipe.HIncrBy(ctx, healthKey, "total_requests", 1)
	pipe.HIncrBy(ctx, healthKey, "failure_count", 1)
	pipe.HSet(ctx, healthKey, "last_failure", time.Now().Format(time.RFC3339))
	pipe.ZIncrBy(ctx, domainKey, -0.5, proxyID) // Decrease domain affinity
	pipe.Set(ctx, cooldownKey, "1", cooldownDuration)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Warn("Pipeline error in RecordFailure", zap.Error(err))
	}

	// Immediately disable proxies with auth failures (bad credentials)
	if pattern == PatternProxyAuthFailed {
		m.disableProxy(ctx, proxyID, 24*time.Hour)
		logger.Warn("Proxy disabled due to authentication failure (bad credentials)",
			zap.String("proxy_id", proxyID),
			zap.String("domain", domain),
		)
		return nil
	}

	// Increment hourly failure count (needs return value, can't pipeline)
	hourlyKey := fmt.Sprintf("proxy:hourly_fail:%s:%d", proxyID, time.Now().Hour())
	count, _ := m.cache.Increment(ctx, hourlyKey)
	m.cache.Expire(ctx, hourlyKey, 2*time.Hour)

	// Check if should disable
	if int(count) >= m.config.MaxFailuresPerHour {
		m.disableProxy(ctx, proxyID, 30*time.Minute)
	}

	logger.Debug("Proxy failure recorded",
		zap.String("proxy_id", proxyID),
		zap.String("domain", domain),
		zap.String("pattern", string(pattern)),
		zap.Duration("cooldown", cooldownDuration),
	)

	return nil
}

// getCooldownDuration returns cooldown duration based on error pattern
func (m *DistributedProxyManager) getCooldownDuration(pattern ErrorPattern) time.Duration {
	base := m.config.CooldownDuration

	switch pattern {
	case PatternBlocked:
		return base * 5 // 5 minutes for blocked
	case PatternRateLimited:
		return base * 2 // 2 minutes for rate limit
	case PatternCaptcha:
		return base * 10 // 10 minutes for captcha
	default:
		return base
	}
}

// disableProxy temporarily disables a proxy in Redis and optionally syncs to database
// OPTIMIZED: Uses pipeline for Redis, async for DB write (non-blocking)
func (m *DistributedProxyManager) disableProxy(ctx context.Context, proxyID string, duration time.Duration) {
	healthKey := fmt.Sprintf(keyProxyHealth, proxyID)

	// Race condition protection: use SetNX to ensure only one worker disables the proxy
	lockKey := fmt.Sprintf("proxy:disable_lock:%s", proxyID)
	acquired, _ := m.cache.SetNX(ctx, lockKey, m.localID, duration)
	if !acquired {
		// Another worker is already handling this, just update local Redis state
		pipe := m.cache.Pipeline()
		pipe.HSet(ctx, healthKey, "is_disabled", "true")
		pipe.HSet(ctx, healthKey, "disabled_until", time.Now().Add(duration).Format(time.RFC3339))
		if _, err := pipe.Exec(ctx); err != nil {
			logger.Warn("Pipeline error in disableProxy (secondary)", zap.Error(err))
		}
		return
	}

	// PIPELINE: Update Redis health state
	pipe := m.cache.Pipeline()
	pipe.HSet(ctx, healthKey, "is_disabled", "true")
	pipe.HSet(ctx, healthKey, "disabled_until", time.Now().Add(duration).Format(time.RFC3339))
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Warn("Pipeline error in disableProxy", zap.Error(err))
	}

	// ASYNC: Sync to database without blocking the hot path
	if m.dbPool != nil {
		go func(proxyID string, duration time.Duration) {
			dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			query := `UPDATE proxies SET is_healthy = false, updated_at = NOW() WHERE id = $1`
			_, dbErr := m.dbPool.Exec(dbCtx, query, proxyID)
			if dbErr != nil {
				logger.Warn("Failed to sync proxy disable to database",
					zap.String("proxy_id", proxyID),
					zap.Error(dbErr),
				)
			} else {
				logger.Info("Proxy disabled and synced to database",
					zap.String("proxy_id", proxyID),
					zap.Duration("duration", duration),
				)
			}
		}(proxyID, duration)
	} else {
		logger.Warn("Proxy disabled (Redis only - no DB pool)",
			zap.String("proxy_id", proxyID),
			zap.Duration("duration", duration),
		)
	}
}

// getHealth gets health metrics for a proxy
func (m *DistributedProxyManager) getHealth(ctx context.Context, proxyID string) (*ProxyHealth, error) {
	healthKey := fmt.Sprintf(keyProxyHealth, proxyID)
	data, err := m.cache.HGetAll(ctx, healthKey)
	if err != nil {
		return nil, err
	}

	health := &ProxyHealth{ProxyID: proxyID}
	if v, ok := data["total_requests"]; ok {
		health.TotalRequests, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data["success_count"]; ok {
		health.SuccessCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data["failure_count"]; ok {
		health.FailureCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data["is_disabled"]; ok {
		health.IsDisabled = v == "true"
	}
	if v, ok := data["disabled_until"]; ok {
		health.DisabledUntil, _ = time.Parse(time.RFC3339, v)
	}

	return health, nil
}

// getProxyData gets full proxy configuration from Redis
func (m *DistributedProxyManager) getProxyData(ctx context.Context, proxyID string) (*Proxy, error) {
	dataKey := fmt.Sprintf(keyProxyData, proxyID)
	data, err := m.cache.Get(ctx, dataKey)
	if err != nil || data == "" {
		return nil, fmt.Errorf("proxy %s not found", proxyID)
	}

	var proxy Proxy
	if err := json.Unmarshal([]byte(data), &proxy); err != nil {
		return nil, err
	}

	return &proxy, nil
}

// SeedProxies seeds proxies into Redis from database or JSON
func (m *DistributedProxyManager) SeedProxies(ctx context.Context, proxies []Proxy) error {
	for _, proxy := range proxies {
		// Store proxy data
		dataKey := fmt.Sprintf(keyProxyData, proxy.ID)
		data, _ := json.Marshal(proxy)
		if err := m.cache.Set(ctx, dataKey, string(data), 0); err != nil {
			return err
		}

		// Add to pool with initial score 0 (least used)
		if err := m.cache.ZAdd(ctx, keyProxyPool, 0, proxy.ID); err != nil {
			return err
		}
	}

	logger.Info("Proxies seeded to Redis",
		zap.Int("count", len(proxies)),
	)

	return nil
}

// GetStats returns statistics for all proxies
func (m *DistributedProxyManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Get all proxies from pool
	proxyIDs, err := m.cache.ZRangeByScore(ctx, keyProxyPool, 0, int(time.Now().Unix()+86400), 1000)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_proxies":   len(proxyIDs),
		"available":       0,
		"leased":          0,
		"disabled":        0,
		"total_requests":  int64(0),
		"total_successes": int64(0),
		"total_failures":  int64(0),
	}

	for _, proxyID := range proxyIDs {
		// Check if leased
		leaseKey := fmt.Sprintf(keyProxyLease, proxyID)
		leased, _ := m.cache.Exists(ctx, leaseKey)
		if leased {
			stats["leased"] = stats["leased"].(int) + 1
		}

		// Get health
		health, _ := m.getHealth(ctx, proxyID)
		if health != nil {
			if health.IsDisabled {
				stats["disabled"] = stats["disabled"].(int) + 1
			} else if !leased {
				stats["available"] = stats["available"].(int) + 1
			}
			stats["total_requests"] = stats["total_requests"].(int64) + health.TotalRequests
			stats["total_successes"] = stats["total_successes"].(int64) + health.SuccessCount
			stats["total_failures"] = stats["total_failures"].(int64) + health.FailureCount
		}
	}

	return stats, nil
}

// GetProxyHealthList returns health info for all proxies
func (m *DistributedProxyManager) GetProxyHealthList(ctx context.Context) ([]ProxyHealth, error) {
	proxyIDs, err := m.cache.ZRangeByScore(ctx, keyProxyPool, 0, int(time.Now().Unix()+86400), 1000)
	if err != nil {
		return nil, err
	}

	healthList := make([]ProxyHealth, 0, len(proxyIDs))
	for _, proxyID := range proxyIDs {
		health, err := m.getHealth(ctx, proxyID)
		if err == nil && health != nil {
			healthList = append(healthList, *health)
		}
	}

	return healthList, nil
}

// CleanupExpiredLeases removes expired leases (safety cleanup)
func (m *DistributedProxyManager) CleanupExpiredLeases(ctx context.Context) int {
	// Leases auto-expire via TTL, but this can be called for explicit cleanup
	// In production, this would scan for any orphaned leases
	return 0
}
