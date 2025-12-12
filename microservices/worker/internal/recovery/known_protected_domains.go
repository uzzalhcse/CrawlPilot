package recovery

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/cache"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// KnownProtectedDomains maintains a list of domains that never work with Tier 0 (direct).
// This prevents wasting the first request on domains that always require proxies.
type KnownProtectedDomains struct {
	cache *cache.Cache

	// In-memory cache for hot path
	staticDomains map[string]models.ProxyTier // Hardcoded/configured domains
	learnedCache  sync.Map                    // domain -> learned minimum tier
}

// defaultProtectedDomains is the fallback list if database is not configured
// These are major sites with strong anti-bot protection
var defaultProtectedDomains = map[string]models.ProxyTier{
	// E-commerce (aggressive bot protection)
	"amazon.com":    models.TierResidential,
	"amazon.co.uk":  models.TierResidential,
	"amazon.de":     models.TierResidential,
	"amazon.co.jp":  models.TierResidential,
	"ebay.com":      models.TierDatacenter,
	"walmart.com":   models.TierResidential,
	"target.com":    models.TierResidential,
	"bestbuy.com":   models.TierResidential,
	"homedepot.com": models.TierDatacenter,
	"lowes.com":     models.TierDatacenter,
	"costco.com":    models.TierResidential,
	"wayfair.com":   models.TierResidential,
	"etsy.com":      models.TierDatacenter,

	// Social media
	"linkedin.com":  models.TierResidential,
	"facebook.com":  models.TierResidential,
	"instagram.com": models.TierResidential,
	"twitter.com":   models.TierResidential,
	"x.com":         models.TierResidential,
	"tiktok.com":    models.TierMobile,

	// Tech companies with strong protection
	"google.com":    models.TierResidential,
	"microsoft.com": models.TierDatacenter,
	"apple.com":     models.TierResidential,

	// Travel (notorious for anti-bot)
	"booking.com":     models.TierResidential,
	"expedia.com":     models.TierResidential,
	"airbnb.com":      models.TierResidential,
	"tripadvisor.com": models.TierResidential,
	"hotels.com":      models.TierResidential,
	"kayak.com":       models.TierResidential,

	// Real estate (strong protection)
	"zillow.com":  models.TierResidential,
	"redfin.com":  models.TierResidential,
	"realtor.com": models.TierResidential,
	"trulia.com":  models.TierResidential,

	// Job sites
	"indeed.com":    models.TierResidential,
	"glassdoor.com": models.TierResidential,
	"monster.com":   models.TierDatacenter,

	// Financial
	"bloomberg.com": models.TierDatacenter,
	"reuters.com":   models.TierDatacenter,

	// Cloudflare-protected by default
	"cloudflare.com": models.TierResidential,
}

// Redis key for learned protected domains
const keyLearnedProtectedDomain = "protected:learned:%s"

// NewKnownProtectedDomains creates a new known protected domains tracker
func NewKnownProtectedDomains(c *cache.Cache) *KnownProtectedDomains {
	return &KnownProtectedDomains{
		cache:         c,
		staticDomains: defaultProtectedDomains,
	}
}

// NewKnownProtectedDomainsWithConfig creates tracker with domains from ConfigManager
// If database has protected_domains configured, uses those; otherwise uses defaults
func NewKnownProtectedDomainsWithConfig(c *cache.Cache, configuredDomains map[string]int) *KnownProtectedDomains {
	kpd := &KnownProtectedDomains{
		cache:         c,
		staticDomains: make(map[string]models.ProxyTier),
	}

	if configuredDomains != nil && len(configuredDomains) > 0 {
		// Use configured domains from database
		for domain, tier := range configuredDomains {
			kpd.staticDomains[domain] = models.ProxyTier(tier)
		}
	} else {
		// Fall back to hardcoded defaults
		kpd.staticDomains = defaultProtectedDomains
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
