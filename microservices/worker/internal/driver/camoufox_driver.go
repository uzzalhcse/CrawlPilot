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
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
	"go.uber.org/zap"
)

// CamoufoxDriver implements the Driver interface using camoufox-go anti-detect browser
type CamoufoxDriver struct {
	camoufox    *camoufox.Camoufox
	config      *config.BrowserConfig
	profile     *models.BrowserProfile
	autoCaptcha bool // Enable auto CAPTCHA solving middleware
}

// CamoufoxOptions for configuring the driver
type CamoufoxOptions struct {
	GeoIP            string // "auto" or IP address for geolocation
	VirtualHeadless  bool   // Use Xvfb virtual display
	ForceScopeAccess bool   // Enable shadow DOM access for CAPTCHA
	AutoCaptcha      bool   // Enable auto CAPTCHA solving
	BlockImages      bool   // Block image loading
	BlockWebGL       bool   // Block WebGL
	BlockWebRTC      bool   // Block WebRTC
}

// NewCamoufoxDriver creates a new CamoufoxDriver with default settings
func NewCamoufoxDriver(cfg *config.BrowserConfig) (*CamoufoxDriver, error) {
	return newCamoufoxDriverInternal(cfg, nil)
}

// NewCamoufoxDriverWithProfile creates a CamoufoxDriver using browser profile settings
func NewCamoufoxDriverWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*CamoufoxDriver, error) {
	return newCamoufoxDriverInternal(cfg, profile)
}

// newCamoufoxDriverInternal creates the driver with optional profile
func newCamoufoxDriverInternal(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*CamoufoxDriver, error) {
	opts := buildCamoufoxOptions(cfg, profile)

	cam, err := camoufox.NewBrowser(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create camoufox browser: %w", err)
	}

	// Determine auto captcha setting (default: true)
	autoCaptcha := true
	if profile != nil && !profile.AutoCaptchaSolve {
		autoCaptcha = false
	}

	logger.Info("Camoufox driver initialized",
		zap.Bool("headless", cfg.Headless),
		zap.Bool("auto_captcha", autoCaptcha),
		zap.Bool("virtual_headless", opts.VirtualHeadless),
		zap.String("geo_ip", opts.GeoIP),
	)

	return &CamoufoxDriver{
		camoufox:    cam,
		config:      cfg,
		profile:     profile,
		autoCaptcha: autoCaptcha,
	}, nil
}

// buildCamoufoxOptions maps config and profile to camoufox.Options
func buildCamoufoxOptions(cfg *config.BrowserConfig, profile *models.BrowserProfile) camoufox.Options {
	opts := camoufox.Options{
		OS:                   "linux", // Default to linux for server environments
		Headless:             cfg.Headless,
		ForceScopeAccess:     true, // Required for CAPTCHA solving
		DisableCOOP:          true, // Required for cross-origin iframe access
		Humanize:             1.0,  // Default: enable human-like mouse movement (1 second max)
		IncludeDefaultAddons: true, // Default: include uBlock Origin
	}

	if profile == nil {
		return opts
	}

	// Target OS for fingerprint
	if profile.TargetOS != "" {
		opts.OS = profile.TargetOS
	}

	// Map profile fields to camoufox options
	if profile.GeoIP != "" {
		opts.GeoIP = profile.GeoIP
	}

	if profile.VirtualHeadless {
		opts.VirtualHeadless = true
		opts.Headless = false // Virtual headless runs as headed in Xvfb
	}

	if !profile.ForceScopeAccess {
		opts.ForceScopeAccess = false
	}

	if profile.BlockImages {
		opts.BlockImages = true
	}

	if profile.BlockWebGL {
		opts.BlockWebGL = true
	}

	if profile.DisableWebRTC {
		opts.BlockWebRTC = true
	}

	// Humanize setting (0 = disabled, > 0 = max duration in seconds)
	if profile.Humanize > 0 {
		opts.Humanize = profile.Humanize
	} else if profile.Humanize == 0 {
		// Explicit 0 means disabled - but we default to 1.0 unless explicitly set to 0
		// Since Go zero value is 0, we check if it's been explicitly set via DB
		// For now, keep the default enabled
	}

	// Include default addons (uBlock Origin) - default true
	opts.IncludeDefaultAddons = profile.IncludeDefaultAddons

	// Enable browser caching
	if profile.EnableCache {
		opts.EnableCache = true
	}

	// User data directory for persistent sessions
	if profile.UserDataDir != "" {
		opts.UserDataDir = profile.UserDataDir
	}

	// Screen dimensions
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		opts.Screen = &camoufox.Screen{
			MinWidth:  profile.ScreenWidth - 100,
			MaxWidth:  profile.ScreenWidth + 100,
			MinHeight: profile.ScreenHeight - 100,
			MaxHeight: profile.ScreenHeight + 100,
		}
	}

	// Timezone
	if profile.Timezone != "" {
		opts.Timezone = profile.Timezone
	}

	// Locale
	if profile.Locale != "" {
		opts.Locale = []string{profile.Locale}
	} else if len(profile.Languages) > 0 {
		opts.Locale = profile.Languages
	}

	// Proxy configuration
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		opts.Proxy = &camoufox.ProxyConfig{
			Server:   profile.ProxyServer,
			Username: profile.ProxyUsername,
			Password: profile.ProxyPassword,
		}
	}

	return opts
}

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
		page:        page,
		camoufox:    d.camoufox,
		autoCaptcha: d.autoCaptcha,
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

// CamoufoxPage implements the Page interface with auto CAPTCHA middleware
type CamoufoxPage struct {
	page        playwright.Page
	camoufox    *camoufox.Camoufox
	autoCaptcha bool
	closed      bool
	mu          sync.Mutex
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

// Goto navigates to the URL with auto CAPTCHA detection and solving middleware
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
	if err != nil {
		return err
	}

	// Auto CAPTCHA solving middleware
	if p.autoCaptcha {
		if err := p.trySolveCaptcha(); err != nil {
			logger.Warn("Auto CAPTCHA solving attempt failed",
				zap.String("url", url),
				zap.Error(err),
			)
			// Don't return error - CAPTCHA solving is best-effort
		}
	}

	return nil
}

// trySolveCaptcha detects and attempts to solve CAPTCHA challenges
func (p *CamoufoxPage) trySolveCaptcha() error {
	// Wait a bit for page to stabilize
	time.Sleep(500 * time.Millisecond)

	// Detect Cloudflare challenge
	if p.detectCloudflareChallenge() {
		logger.Info("Cloudflare challenge detected, attempting to solve")

		success, err := captcha.SolveCaptchaPage(p.page, captcha.SolveOptions{
			CaptchaType:   captcha.CaptchaCloudflare,
			ChallengeType: captcha.ChallengeInterstitial,
			Debug:         false,
		})

		if err != nil {
			return fmt.Errorf("captcha solve error: %w", err)
		}

		if success {
			logger.Info("Cloudflare challenge solved successfully")
		} else {
			logger.Warn("Cloudflare challenge not solved")
		}
	}

	return nil
}

// detectCloudflareChallenge checks if the page has a Cloudflare challenge
func (p *CamoufoxPage) detectCloudflareChallenge() bool {
	// Check for common Cloudflare indicators
	content, err := p.page.Content()
	if err != nil {
		return false
	}

	// Check for Cloudflare challenge markers
	indicators := []string{
		"cf-turnstile",
		"cf_chl_opt",
		"challenge-running",
		"Just a moment",
		"Checking your browser",
		"cf-challenge",
	}

	for _, indicator := range indicators {
		if containsString(content, indicator) {
			return true
		}
	}

	return false
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
