package recovery

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// KnownProtectedDomains maintains a list of domains that never work with Tier 0 (direct).
// This prevents wasting the first request on domains that always require proxies.
type KnownProtectedDomains struct {
	cache *cache.Cache

	// In-memory cache for hot path
	staticDomains map[string]models.ProxyTier // Hardcoded/configured domains
	learnedCache  sync.Map                    // domain -> learned minimum tier
}

// Redis key for learned protected domains
const keyLearnedProtectedDomain = "protected:learned:%s"

// NewKnownProtectedDomains creates an empty tracker (for testing or when DB not available)
func NewKnownProtectedDomains(c *cache.Cache) *KnownProtectedDomains {
	return &KnownProtectedDomains{
		cache:         c,
		staticDomains: make(map[string]models.ProxyTier),
	}
}

// NewKnownProtectedDomainsFromDB creates tracker by loading from domain_strategies table
// This is the ONLY source of truth - no hardcoded defaults
func NewKnownProtectedDomainsFromDB(ctx context.Context, c *cache.Cache, pool *pgxpool.Pool) *KnownProtectedDomains {
	kpd := &KnownProtectedDomains{
		cache:         c,
		staticDomains: make(map[string]models.ProxyTier),
	}

	// Load learned strategies from domain_strategies table
	if pool != nil {
		query := `SELECT domain, recommended_tier FROM domain_strategies WHERE learning_status IN ('stable', 'learning')`
		rows, err := pool.Query(ctx, query)
		if err != nil {
			logger.Warn("Failed to load domain strategies from DB", zap.Error(err))
		} else {
			defer rows.Close()
			loadedCount := 0
			for rows.Next() {
				var domain string
				var tier int
				if err := rows.Scan(&domain, &tier); err == nil && tier > 0 {
					kpd.staticDomains[domain] = models.ProxyTier(tier)
					loadedCount++
				}
			}
			if loadedCount > 0 {
				logger.Info("Loaded domain strategies from DB", zap.Int("count", loadedCount))
			}
		}
	}

	return kpd
}

// GetMinimumTier returns the minimum tier required for a domain.
// Returns TierDirect (0) if the domain is not known to be protected.
func (kpd *KnownProtectedDomains) GetMinimumTier(ctx context.Context, domain string) models.ProxyTier {
	// Normalize domain (remove www. prefix, lowercase)
	domain = normalizeDomainName(domain)

	// Check static list first (fastest)
	if tier, ok := kpd.staticDomains[domain]; ok {
		return tier
	}

	// Check parent domain (e.g., "www.amazon.com" -> "amazon.com")
	if parentTier, ok := kpd.checkParentDomain(domain); ok {
		return parentTier
	}

	// Check in-memory learned cache
	if tier, ok := kpd.learnedCache.Load(domain); ok {
		return tier.(models.ProxyTier)
	}

	// Check Redis for learned tier
	if kpd.cache != nil {
		key := kpd.learnedKey(domain)
		if tierStr, err := kpd.cache.Get(ctx, key); err == nil && tierStr != "" {
			tier := parseIntSimple(tierStr)
			if tier > 0 {
				modelTier := models.ProxyTier(tier)
				// Cache in memory
				kpd.learnedCache.Store(domain, modelTier)
				return modelTier
			}
		}
	}

	// Not protected - use Tier 0 (direct)
	return models.TierDirect
}

// LearnMinimumTier learns that a domain requires at least the specified tier
// This is called when Tier 0 fails for a domain
func (kpd *KnownProtectedDomains) LearnMinimumTier(ctx context.Context, domain string, tier models.ProxyTier) {
	domain = normalizeDomainName(domain)

	// Update in-memory cache
	kpd.learnedCache.Store(domain, tier)

	// Persist to Redis (24 hour TTL)
	if kpd.cache != nil {
		key := kpd.learnedKey(domain)
		kpd.cache.Set(ctx, key, formatInt(int(tier)), 24*time.Hour)
	}
}

// IsProtected checks if a domain is known to be protected
func (kpd *KnownProtectedDomains) IsProtected(ctx context.Context, domain string) bool {
	return kpd.GetMinimumTier(ctx, domain) > models.TierDirect
}

// checkParentDomain checks if any parent domain is in the static list
func (kpd *KnownProtectedDomains) checkParentDomain(domain string) (models.ProxyTier, bool) {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return 0, false
	}

	// Try progressively shorter domain suffixes
	for i := 1; i < len(parts)-1; i++ {
		parentDomain := strings.Join(parts[i:], ".")
		if tier, ok := kpd.staticDomains[parentDomain]; ok {
			return tier, true
		}
	}

	return 0, false
}

// AddStaticDomain adds a domain to the static protected list (runtime)
func (kpd *KnownProtectedDomains) AddStaticDomain(domain string, tier models.ProxyTier) {
	domain = normalizeDomainName(domain)
	kpd.staticDomains[domain] = tier
}

// GetAllStaticDomains returns all statically configured protected domains
func (kpd *KnownProtectedDomains) GetAllStaticDomains() map[string]models.ProxyTier {
	result := make(map[string]models.ProxyTier)
	for domain, tier := range kpd.staticDomains {
		result[domain] = tier
	}
	return result
}

// ClearLearnedCache clears the in-memory learned cache
func (kpd *KnownProtectedDomains) ClearLearnedCache() {
	kpd.learnedCache = sync.Map{}
}

func (kpd *KnownProtectedDomains) learnedKey(domain string) string {
	return "protected:learned:" + domain
}

// Helper functions
func normalizeDomainName(domain string) string {
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(domain, "www.")
	if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
		domain = domain[:colonIdx]
	}
	return domain
}

func parseIntSimple(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
