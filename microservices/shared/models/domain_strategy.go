package models

import (
	"encoding/json"
	"time"
)

// ProxyTier represents the escalation tier for proxy selection
type ProxyTier int

const (
	TierDirect      ProxyTier = 0 // No proxy, direct connection
	TierDatacenter  ProxyTier = 1 // Datacenter proxies - fast, cheap, more detectable
	TierResidential ProxyTier = 2 // Residential proxies - slower, stealthier
	TierMobile      ProxyTier = 3 // Mobile proxies - slowest, most trusted
)

// LearningStatus represents the learning state for a domain
type LearningStatus string

const (
	StatusNew         LearningStatus = "new"          // No data yet
	StatusLearning    LearningStatus = "learning"     // Gathering data
	StatusStable      LearningStatus = "stable"       // Confident in strategy
	StatusNeedsReview LearningStatus = "needs_review" // Issues detected
)

// SessionRequirement indicates if a domain requires session reuse for anti-bot bypass
type SessionRequirement int

const (
	SessionUnknown   SessionRequirement = 0 // Not yet determined
	SessionRequired  SessionRequirement = 1 // Must reuse sessions (anti-bot cookies detected)
	SessionOptional  SessionRequirement = 2 // Sessions help but not strictly required
	SessionNotNeeded SessionRequirement = 3 // Fresh sessions work fine
)

// AntiBotCookiePatterns contains known anti-bot cookie name patterns
// These cookies indicate the site uses anti-bot protection that grants sessions
var AntiBotCookiePatterns = []string{
	// Cloudflare
	"cf_clearance",
	"__cf_bm",
	"cf_chl_",
	// Datadome
	"datadome",
	// PerimeterX
	"_px",
	"_pxhd",
	"_pxvid",
	// Akamai
	"ak_bmsc",
	"bm_sv",
	"bm_sz",
	// Incapsula/Imperva
	"incap_ses_",
	"visid_incap_",
	// Kasada
	"x-kpsdk-",
	// hCaptcha session
	"hc_accessibility",
	// Shape Security
	"_abck",
}

// DomainStrategy stores learned anti-bot handling strategies per domain
type DomainStrategy struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`

	// Tier Settings (Crawlee pattern)
	RecommendedTier ProxyTier `json:"recommended_tier"` // 0=direct, 1=datacenter, 2=residential, 3=mobile
	TierConfidence  float64   `json:"tier_confidence"`  // 0.0-1.0

	// Session Settings (anti-bot cookie detection)
	SessionRequirement SessionRequirement `json:"session_requirement"`        // 0=unknown, 1=required, 2=optional, 3=not_needed
	DetectedCookies    []string           `json:"detected_cookies,omitempty"` // Which anti-bot cookies were detected

	// Rotation Settings
	OptimalRotationCount *int    `json:"optimal_rotation_count,omitempty"` // Rotate every N requests
	RotationConfidence   float64 `json:"rotation_confidence"`

	// Throttling
	AdaptiveDelayMs       int  `json:"adaptive_delay_ms"`                 // Delay between requests
	MaxConcurrentRequests *int `json:"max_concurrent_requests,omitempty"` // Detected limit

	// Success Metrics
	TotalRequests  int64   `json:"total_requests"`
	TotalSuccesses int64   `json:"total_successes"`
	TotalFailures  int64   `json:"total_failures"`
	SuccessRate    float64 `json:"success_rate"`

	// Learning Status
	LearningStatus LearningStatus `json:"learning_status"`
	SampleSize     int            `json:"sample_size"`
	LastBlockAt    *time.Time     `json:"last_block_at,omitempty"`
	LastSuccessAt  *time.Time     `json:"last_success_at,omitempty"`

	// Tier History
	TierAttempts map[int]*TierStats `json:"tier_attempts,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TierStats tracks success/failure for each tier
type TierStats struct {
	Attempts  int `json:"attempts"`
	Successes int `json:"successes"`
	Failures  int `json:"failures"`
}

// CalculateTierSuccessRate calculates success rate for a specific tier
func (d *DomainStrategy) CalculateTierSuccessRate(tier ProxyTier) float64 {
	if d.TierAttempts == nil {
		return 0
	}
	stats, ok := d.TierAttempts[int(tier)]
	if !ok || stats.Attempts == 0 {
		return 0
	}
	return float64(stats.Successes) / float64(stats.Attempts)
}

// ShouldEscalate determines if we should try a higher tier
func (d *DomainStrategy) ShouldEscalate(currentTier ProxyTier) bool {
	// If success rate for current tier is low, try higher
	successRate := d.CalculateTierSuccessRate(currentTier)
	return successRate < 0.5 && currentTier < TierMobile
}

// GetNextTier returns the next tier to try
func (d *DomainStrategy) GetNextTier(currentTier ProxyTier) ProxyTier {
	if currentTier >= TierMobile {
		return TierMobile
	}
	return currentTier + 1
}

// IsConfident returns true if we have enough data to be confident
func (d *DomainStrategy) IsConfident() bool {
	return d.SampleSize >= 50 && d.TierConfidence >= 0.7
}

// TierAttemptsJSON wraps tier attempts for DB storage
func (d *DomainStrategy) TierAttemptsJSON() ([]byte, error) {
	if d.TierAttempts == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(d.TierAttempts)
}

// ParseTierAttemptsJSON parses tier attempts from DB
func (d *DomainStrategy) ParseTierAttemptsJSON(data []byte) error {
	if len(data) == 0 {
		d.TierAttempts = make(map[int]*TierStats)
		return nil
	}
	return json.Unmarshal(data, &d.TierAttempts)
}
