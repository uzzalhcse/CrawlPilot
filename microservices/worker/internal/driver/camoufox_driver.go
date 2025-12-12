package driver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	captcha "github.com/uzzalhcse/camoufox-captcha-go"
	camoufox "github.com/uzzalhcse/camoufox-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/services"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
	"go.uber.org/zap"
)

// CamoufoxDriver implements the Driver interface using camoufox-go anti-detect browser
type CamoufoxDriver struct {
	camoufox *camoufox.Camoufox
	config   *config.BrowserConfig
	profile  *models.BrowserProfile
}

// CamoufoxOptions for configuring the driver
type CamoufoxOptions struct {
	GeoIP            string // "auto" or IP address for geolocation
	VirtualHeadless  bool   // Use Xvfb virtual display
	ForceScopeAccess bool   // Enable shadow DOM access for CAPTCHA
	BlockImages      bool   // Block image loading
	BlockWebGL       bool   // Block WebGL
	BlockWebRTC      bool   // Block WebRTC
}

// NewCamoufoxDriver creates a new CamoufoxDriver with default settings
func NewCamoufoxDriver(cfg *config.BrowserConfig) (*CamoufoxDriver, error) {
	return newCamoufoxDriverInternal(cfg, nil, nil)
}

// NewCamoufoxDriverWithProfile creates a CamoufoxDriver using browser profile settings
func NewCamoufoxDriverWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*CamoufoxDriver, error) {
	return newCamoufoxDriverInternal(cfg, profile, nil)
}

// NewCamoufoxDriverWithFingerprint creates a CamoufoxDriver with a locked fingerprint.
// This is used for domain-locked CAPTCHA session sharing - all browsers for the same domain
// use the same fingerprint to ensure cookies remain valid.
func NewCamoufoxDriverWithFingerprint(cfg *config.BrowserConfig, profile *models.BrowserProfile, fingerprint *camoufox.Fingerprint) (*CamoufoxDriver, error) {
	return newCamoufoxDriverInternal(cfg, profile, fingerprint)
}

// newCamoufoxDriverInternal creates the driver with optional profile and fingerprint
func newCamoufoxDriverInternal(cfg *config.BrowserConfig, profile *models.BrowserProfile, fingerprint *camoufox.Fingerprint) (*CamoufoxDriver, error) {
	// Use shared browser factory for consistent configuration with orchestrator (includes proxy validation)
	opts, err := services.BuildCamoufoxOptions(profile, cfg.Headless)
	if err != nil {
		return nil, fmt.Errorf("failed to build camoufox options: %w", err)
	}

	// CRITICAL: These options MUST be true for CAPTCHA solving to work
	// ForceScopeAccess enables shadowRootUnl for closed Shadow DOM access
	// DisableCOOP enables cross-origin iframe access
	opts.ForceScopeAccess = true
	opts.DisableCOOP = true

	// Use pre-defined fingerprint if provided (for domain-locked sessions)
	if fingerprint != nil {
		opts.Fingerprint = fingerprint
		logger.Debug("Using pre-defined fingerprint for domain-locked session")
	}

	cam, err := camoufox.NewBrowser(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create camoufox browser: %w", err)
	}

	logger.Info("Camoufox driver initialized",
		zap.Bool("headless", cfg.Headless),
		zap.Bool("force_scope_access", opts.ForceScopeAccess),
		zap.Bool("disable_coop", opts.DisableCOOP),
		zap.Bool("virtual_headless", opts.VirtualHeadless),
		zap.String("geo_ip", opts.GeoIP),
		zap.Bool("fingerprint_locked", fingerprint != nil),
	)

	return &CamoufoxDriver{
		camoufox: cam,
		config:   cfg,
		profile:  profile,
	}, nil
}

// Note: buildCamoufoxOptions has been moved to shared/services/browser_factory.go
// as services.BuildCamoufoxOptions for unified configuration across orchestrator and worker

func (d *CamoufoxDriver) NewPage(ctx context.Context) (Page, error) {
	// Check for proxy override in context
	var proxyConfig *browser.ProxyConfig
	if proxyCfg, ok := ctx.Value(ProxyKey).(*browser.ProxyConfig); ok && proxyCfg != nil {
		proxyConfig = proxyCfg
	}

	// Get page from camoufox
	page, err := d.camoufox.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create camoufox page: %w", err)
	}

	// If proxy override specified, we need to handle it separately
	// Note: camoufox applies proxy at browser level, context-level proxy requires new browser
	if proxyConfig != nil {
		logger.Warn("Context-level proxy override not fully supported in camoufox, use profile-level proxy",
			zap.String("proxy", proxyConfig.Server),
		)
	}

	return &CamoufoxPage{
		page:     page,
		camoufox: d.camoufox,
	}, nil
}

func (d *CamoufoxDriver) Close() error {
	if d.camoufox != nil {
		return d.camoufox.Close()
	}
	return nil
}

func (d *CamoufoxDriver) Name() string {
	return "camoufox"
}

// SetSessionChecker is a no-op for CamoufoxDriver.
// Camoufox doesn't use pooled contexts - sessions are managed at browser level,
// so anti-bot cookies are naturally preserved within the same browser instance.
func (d *CamoufoxDriver) SetSessionChecker(checker browser.SessionChecker) {
	// No-op: Camoufox maintains session state within the browser instance
	logger.Debug("CamoufoxDriver: SetSessionChecker called (no-op, sessions managed at browser level)")
}

// CamoufoxPage implements the Page interface
type CamoufoxPage struct {
	page     playwright.Page
	camoufox *camoufox.Camoufox
	closed   bool
	mu       sync.Mutex
}

func (p *CamoufoxPage) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	return p.page.Close()
}

func (p *CamoufoxPage) DriverName() string {
	return "camoufox"
}

// Goto navigates to the URL
func (p *CamoufoxPage) Goto(url string, options ...PageOption) error {
	opts := &PageOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageGotoOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}
	if opts.WaitUntil != "" {
		val := playwright.WaitUntilState(opts.WaitUntil)
		pwOpts.WaitUntil = &val
	}

	_, err := p.page.Goto(url, pwOpts)
	return err
}

// SolveCaptcha attempts to detect and solve CAPTCHA challenges on the current page.
// Implements the CaptchaSolver interface.
func (p *CamoufoxPage) SolveCaptcha(opts CaptchaSolveOptions) (bool, error) {
	// Quick detection FIRST - avoid waiting if no CAPTCHA present
	challengeType := p.detectCloudflareChallenge()
	if challengeType == "" {
		return false, nil // No CAPTCHA detected - no delay incurred
	}

	logger.Info("Cloudflare challenge detected, attempting to solve",
		zap.String("detected_type", challengeType),
	)

	// Wait a bit for page to stabilize only AFTER detection
	time.Sleep(500 * time.Millisecond)

	// Convert to captcha package options
	captchaOpts := captcha.SolveOptions{
		CaptchaType:             captcha.CaptchaCloudflare,
		ExpectedContentSelector: opts.ExpectedContentSelector,
		Debug:                   opts.Debug,
	}

	// Determine challenge type - use auto-detected if not specified
	if opts.ChallengeType != "" {
		// User specified a type
		switch opts.ChallengeType {
		case "turnstile":
			captchaOpts.ChallengeType = captcha.ChallengeTurnstile
		default:
			captchaOpts.ChallengeType = captcha.ChallengeInterstitial
		}
	} else {
		// Auto-detected type
		switch challengeType {
		case "turnstile":
			captchaOpts.ChallengeType = captcha.ChallengeTurnstile
		default:
			captchaOpts.ChallengeType = captcha.ChallengeInterstitial
		}
	}

	// Set solve attempts if specified
	if opts.SolveAttempts > 0 {
		captchaOpts.SolveAttempts = opts.SolveAttempts
	}

	logger.Debug("Attempting CAPTCHA solve",
		zap.String("challenge_type", string(captchaOpts.ChallengeType)),
	)

	success, err := captcha.SolveCaptchaPage(p.page, captchaOpts)
	if err != nil {
		return false, fmt.Errorf("captcha solve error: %w", err)
	}

	// If first attempt failed and we auto-detected, try the other type
	if !success && opts.ChallengeType == "" {
		var alternateType captcha.ChallengeType
		if captchaOpts.ChallengeType == captcha.ChallengeInterstitial {
			alternateType = captcha.ChallengeTurnstile
		} else {
			alternateType = captcha.ChallengeInterstitial
		}

		logger.Debug("First attempt failed, trying alternate challenge type",
			zap.String("alternate_type", string(alternateType)),
		)

		captchaOpts.ChallengeType = alternateType
		success, err = captcha.SolveCaptchaPage(p.page, captchaOpts)
		if err != nil {
			return false, fmt.Errorf("captcha solve error (alternate): %w", err)
		}
	}

	if success {
		logger.Info("Cloudflare challenge solved successfully")
	} else {
		logger.Warn("Cloudflare challenge not solved")
	}

	return success, nil
}

// SupportsCaptchaSolving returns true as Camoufox has full CAPTCHA solving support
// via shadowRootUnl for accessing closed Shadow DOM elements.
func (p *CamoufoxPage) SupportsCaptchaSolving() bool {
	return true
}

// GetFingerprint returns the browser's fingerprint data for session caching.
// Returns (userAgent, platform, language, languages, hardwareConcurrency, screenWidth, screenHeight)
// This implements the FingerprintProvider interface.
func (p *CamoufoxPage) GetFingerprint() (string, string, string, []string, int, int, int) {
	fp := p.camoufox.Fingerprint()
	if fp == nil {
		return "", "", "", nil, 0, 0, 0
	}

	return fp.Navigator.UserAgent,
		fp.Navigator.Platform,
		fp.Navigator.Language,
		fp.Navigator.Languages,
		fp.Navigator.HardwareConcurrency,
		fp.Screen.Width,
		fp.Screen.Height
}

// detectCloudflareChallenge checks if the page has a Cloudflare challenge
// Returns the detected challenge type: "turnstile", "interstitial", or "" if none
// Detection logic based on playwright-captcha Python repo
func (p *CamoufoxPage) detectCloudflareChallenge() string {
	// Use CSS selector-based detection for reliability (from playwright-captcha)

	// Check for INTERSTITIAL first (full-page challenge) - this is the more common case
	// Key selector: script[src*="/cdn-cgi/challenge-platform/"]
	interstitialSelectors := []string{
		`script[src*="/cdn-cgi/challenge-platform/"]`,
	}
	for _, selector := range interstitialSelectors {
		el, err := p.page.QuerySelector(selector)
		if err == nil && el != nil {
			return "interstitial"
		}
	}

	// Check for TURNSTILE (embedded widget captcha)
	// Key selectors from playwright-captcha
	turnstileSelectors := []string{
		`input[name="cf-turnstile-response"]`,
		`script[src*="challenges.cloudflare.com/turnstile"]`,
	}
	for _, selector := range turnstileSelectors {
		el, err := p.page.QuerySelector(selector)
		if err == nil && el != nil {
			return "turnstile"
		}
	}

	// Fallback: Check page content for additional indicators
	content, err := p.page.Content()
	if err != nil {
		return ""
	}

	// Additional interstitial indicators (text-based fallback)
	interstitialIndicators := []string{
		"cf_chl_opt",
		"challenge-running",
		"Just a moment",
		"Checking your browser",
	}
	for _, indicator := range interstitialIndicators {
		if containsString(content, indicator) {
			return "interstitial"
		}
	}

	// Additional turnstile indicators (text-based fallback)
	if containsString(content, "cf-turnstile") {
		return "turnstile"
	}

	return ""
}

// containsString is a simple substring check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (p *CamoufoxPage) Content() (string, error) {
	return p.page.Content()
}

func (p *CamoufoxPage) Title() (string, error) {
	return p.page.Title()
}

func (p *CamoufoxPage) URL() (string, error) {
	return p.page.URL(), nil
}

func (p *CamoufoxPage) Click(selector string, options ...ElementOption) error {
	opts := &ElementOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageClickOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}

	return p.page.Click(selector, pwOpts)
}

func (p *CamoufoxPage) Type(selector, text string, options ...ElementOption) error {
	opts := &ElementOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageTypeOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}
	if opts.Delay > 0 {
		pwOpts.Delay = playwright.Float(float64(opts.Delay.Milliseconds()))
	}

	return p.page.Type(selector, text, pwOpts)
}

func (p *CamoufoxPage) Fill(selector, text string, options ...ElementOption) error {
	opts := &ElementOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageFillOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}

	return p.page.Fill(selector, text, pwOpts)
}

func (p *CamoufoxPage) Hover(selector string, options ...ElementOption) error {
	opts := &ElementOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageHoverOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}

	return p.page.Hover(selector, pwOpts)
}

func (p *CamoufoxPage) WaitForSelector(selector string, options ...WaitOption) error {
	opts := &WaitOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageWaitForSelectorOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}
	if opts.State != "" {
		var state *playwright.WaitForSelectorState
		switch opts.State {
		case "attached":
			state = playwright.WaitForSelectorStateAttached
		case "detached":
			state = playwright.WaitForSelectorStateDetached
		case "visible":
			state = playwright.WaitForSelectorStateVisible
		case "hidden":
			state = playwright.WaitForSelectorStateHidden
		}
		if state != nil {
			pwOpts.State = state
		}
	}

	_, err := p.page.WaitForSelector(selector, pwOpts)
	return err
}

func (p *CamoufoxPage) WaitForURL(url string, options ...WaitOption) error {
	opts := &WaitOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageWaitForURLOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}
	if opts.WaitUntil != "" {
		val := playwright.WaitUntilState(opts.WaitUntil)
		pwOpts.WaitUntil = &val
	}

	return p.page.WaitForURL(url, pwOpts)
}

func (p *CamoufoxPage) WaitForState(state string, options ...WaitOption) error {
	opts := &WaitOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageWaitForLoadStateOptions{}
	if opts.Timeout > 0 {
		pwOpts.Timeout = playwright.Float(float64(opts.Timeout.Milliseconds()))
	}
	if state != "" {
		val := playwright.LoadState(state)
		pwOpts.State = &val
	}

	return p.page.WaitForLoadState(pwOpts)
}

func (p *CamoufoxPage) WaitForFunction(expression string, args ...interface{}) error {
	_, err := p.page.WaitForFunction(expression, args, playwright.PageWaitForFunctionOptions{})
	return err
}

func (p *CamoufoxPage) Evaluate(expression string, args ...interface{}) (interface{}, error) {
	return p.page.Evaluate(expression, args...)
}

func (p *CamoufoxPage) AddInitScript(script string) error {
	return p.page.AddInitScript(playwright.Script{Content: playwright.String(script)})
}

func (p *CamoufoxPage) QuerySelector(selector string) (Element, error) {
	el, err := p.page.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	if el == nil {
		return nil, nil
	}
	return &CamoufoxElement{el: el}, nil
}

func (p *CamoufoxPage) QuerySelectorAll(selector string) ([]Element, error) {
	els, err := p.page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	elements := make([]Element, len(els))
	for i, el := range els {
		elements[i] = &CamoufoxElement{el: el}
	}
	return elements, nil
}

func (p *CamoufoxPage) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	opts := &ScreenshotOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.PageScreenshotOptions{}
	if opts.FullPage {
		pwOpts.FullPage = playwright.Bool(true)
	}
	if opts.Type != "" {
		val := playwright.ScreenshotType(opts.Type)
		pwOpts.Type = &val
	}
	if opts.Quality > 0 {
		pwOpts.Quality = playwright.Int(opts.Quality)
	}

	return p.page.Screenshot(pwOpts)
}

func (p *CamoufoxPage) GetCookies() ([]*http.Cookie, error) {
	ctx := p.page.Context()
	pwCookies, err := ctx.Cookies()
	if err != nil {
		return nil, err
	}

	cookies := make([]*http.Cookie, len(pwCookies))
	for i, c := range pwCookies {
		cookies[i] = &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  time.Unix(int64(c.Expires), 0),
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: http.SameSiteDefaultMode,
		}
		if c.SameSite != nil {
			if c.SameSite == playwright.SameSiteAttributeStrict {
				cookies[i].SameSite = http.SameSiteStrictMode
			} else if c.SameSite == playwright.SameSiteAttributeLax {
				cookies[i].SameSite = http.SameSiteLaxMode
			} else if c.SameSite == playwright.SameSiteAttributeNone {
				cookies[i].SameSite = http.SameSiteNoneMode
			}
		}
	}
	return cookies, nil
}

func (p *CamoufoxPage) SetCookies(cookies []*http.Cookie) error {
	ctx := p.page.Context()
	pwCookies := make([]playwright.OptionalCookie, len(cookies))
	for i, c := range cookies {
		sameSite := playwright.SameSiteAttributeNone
		if c.SameSite == http.SameSiteStrictMode {
			sameSite = playwright.SameSiteAttributeStrict
		} else if c.SameSite == http.SameSiteLaxMode {
			sameSite = playwright.SameSiteAttributeLax
		}

		pwCookies[i] = playwright.OptionalCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   playwright.String(c.Domain),
			Path:     playwright.String(c.Path),
			Expires:  playwright.Float(float64(c.Expires.Unix())),
			Secure:   playwright.Bool(c.Secure),
			HttpOnly: playwright.Bool(c.HttpOnly),
			SameSite: sameSite,
		}
	}
	return ctx.AddCookies(pwCookies)
}

// CamoufoxElement implements the Element interface
type CamoufoxElement struct {
	el playwright.ElementHandle
}

func (e *CamoufoxElement) Text() (string, error) {
	return e.el.TextContent()
}

func (e *CamoufoxElement) Attribute(name string) (string, error) {
	return e.el.GetAttribute(name)
}

func (e *CamoufoxElement) InnerHTML() (string, error) {
	return e.el.InnerHTML()
}

func (e *CamoufoxElement) Click() error {
	return e.el.Click()
}

func (e *CamoufoxElement) Type(text string) error {
	return e.el.Type(text)
}

func (e *CamoufoxElement) Fill(text string) error {
	return e.el.Fill(text)
}

func (e *CamoufoxElement) Hover() error {
	return e.el.Hover()
}

func (e *CamoufoxElement) QuerySelector(selector string) (Element, error) {
	el, err := e.el.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	if el == nil {
		return nil, nil
	}
	return &CamoufoxElement{el: el}, nil
}

func (e *CamoufoxElement) QuerySelectorAll(selector string) ([]Element, error) {
	els, err := e.el.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	elements := make([]Element, len(els))
	for i, el := range els {
		elements[i] = &CamoufoxElement{el: el}
	}
	return elements, nil
}

func (e *CamoufoxElement) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	opts := &ScreenshotOptions{}
	for _, opt := range options {
		opt(opts)
	}

	pwOpts := playwright.ElementHandleScreenshotOptions{}
	if opts.Type != "" {
		val := playwright.ScreenshotType(opts.Type)
		pwOpts.Type = &val
	}
	if opts.Quality > 0 {
		pwOpts.Quality = playwright.Int(opts.Quality)
	}

	return e.el.Screenshot(pwOpts)
}
