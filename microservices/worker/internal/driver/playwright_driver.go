package driver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
)

// ProxyKey is now defined in options.go

// PlaywrightDriver implements the Driver interface using playwright-go
type PlaywrightDriver struct {
	pool    *browser.Pool
	profile *models.BrowserProfile // Optional profile for fingerprint settings
}

// NewPlaywrightDriver creates a new PlaywrightDriver
func NewPlaywrightDriver(cfg *config.BrowserConfig) (*PlaywrightDriver, error) {
	pool, err := browser.NewPool(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}

	return &PlaywrightDriver{
		pool: pool,
	}, nil
}

// NewPlaywrightDriverWithProfile creates a PlaywrightDriver using browser profile settings
// This enables profile-specific fingerprints, browser type selection, and custom configurations
func NewPlaywrightDriverWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*PlaywrightDriver, error) {
	// Create pool with profile settings
	pool, err := browser.NewPoolWithProfile(cfg, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool with profile: %w", err)
	}

	return &PlaywrightDriver{
		pool:    pool,
		profile: profile,
	}, nil
}

func (d *PlaywrightDriver) NewPage(ctx context.Context) (Page, error) {
	var browserCtx playwright.BrowserContext
	var err error
	var isProxy bool

	// Check for proxy in context
	if proxyCfg, ok := ctx.Value(ProxyKey).(*browser.ProxyConfig); ok && proxyCfg != nil {
		browserCtx, err = d.pool.CreateContextWithProxy(proxyCfg)
		isProxy = true
	} else {
		browserCtx, err = d.pool.Acquire(ctx)
	}

	if err != nil {
		return nil, err
	}

	page, err := browserCtx.NewPage()
	if err != nil {
		browserCtx.Close()
		return nil, err
	}

	return &PlaywrightPage{
		page:       page,
		browserCtx: browserCtx,
		pool:       d.pool,
		isProxy:    isProxy,
	}, nil
}

func (d *PlaywrightDriver) Close() error {
	return d.pool.Close()
}

func (d *PlaywrightDriver) Name() string {
	return "playwright"
}

// PlaywrightPage implements the Page interface
type PlaywrightPage struct {
	page       playwright.Page
	browserCtx playwright.BrowserContext
	pool       *browser.Pool
	isProxy    bool
	closed     bool
	mu         sync.Mutex
}

// NewPlaywrightPage creates a new PlaywrightPage from an existing playwright.Page
// Deprecated: Use NewPage via Driver interface
func NewPlaywrightPage(page playwright.Page) *PlaywrightPage {
	return &PlaywrightPage{page: page}
}

func (p *PlaywrightPage) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	// Close page first
	if err := p.page.Close(); err != nil {
		// Log error but continue to release context
		// logger.Warn("Failed to close playwright page", zap.Error(err))
	}

	// Release or close context
	if p.isProxy {
		return p.browserCtx.Close()
	}

	if p.pool != nil && p.browserCtx != nil {
		p.pool.Release(p.browserCtx)
	}
	return nil
}

func (p *PlaywrightPage) DriverName() string {
	return "playwright"
}

func (p *PlaywrightPage) Goto(url string, options ...PageOption) error {
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

func (p *PlaywrightPage) Content() (string, error) {
	return p.page.Content()
}

func (p *PlaywrightPage) Title() (string, error) {
	return p.page.Title()
}

func (p *PlaywrightPage) URL() (string, error) {
	return p.page.URL(), nil
}

func (p *PlaywrightPage) Click(selector string, options ...ElementOption) error {
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

func (p *PlaywrightPage) Type(selector, text string, options ...ElementOption) error {
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

func (p *PlaywrightPage) Fill(selector, text string, options ...ElementOption) error {
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

func (p *PlaywrightPage) Hover(selector string, options ...ElementOption) error {
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

func (p *PlaywrightPage) WaitForSelector(selector string, options ...WaitOption) error {
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

func (p *PlaywrightPage) WaitForURL(url string, options ...WaitOption) error {
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

func (p *PlaywrightPage) WaitForState(state string, options ...WaitOption) error {
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

func (p *PlaywrightPage) WaitForFunction(expression string, args ...interface{}) error {
	_, err := p.page.WaitForFunction(expression, args, playwright.PageWaitForFunctionOptions{})
	return err
}

func (p *PlaywrightPage) Evaluate(expression string, args ...interface{}) (interface{}, error) {
	return p.page.Evaluate(expression, args...)
}

func (p *PlaywrightPage) AddInitScript(script string) error {
	return p.page.AddInitScript(playwright.Script{Content: playwright.String(script)})
}

func (p *PlaywrightPage) QuerySelector(selector string) (Element, error) {
	el, err := p.page.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	if el == nil {
		return nil, nil
	}
	return &PlaywrightElement{el: el}, nil
}

func (p *PlaywrightPage) QuerySelectorAll(selector string) ([]Element, error) {
	els, err := p.page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	elements := make([]Element, len(els))
	for i, el := range els {
		elements[i] = &PlaywrightElement{el: el}
	}
	return elements, nil
}

func (p *PlaywrightPage) Screenshot(options ...ScreenshotOption) ([]byte, error) {
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

func (p *PlaywrightPage) GetCookies() ([]*http.Cookie, error) {
	pwCookies, err := p.browserCtx.Cookies()
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

func (p *PlaywrightPage) SetCookies(cookies []*http.Cookie) error {
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
	return p.browserCtx.AddCookies(pwCookies)
}

// PlaywrightElement implements the Element interface
type PlaywrightElement struct {
	el playwright.ElementHandle
}

func (e *PlaywrightElement) Text() (string, error) {
	return e.el.TextContent()
}

func (e *PlaywrightElement) Attribute(name string) (string, error) {
	return e.el.GetAttribute(name)
}

func (e *PlaywrightElement) InnerHTML() (string, error) {
	return e.el.InnerHTML()
}

func (e *PlaywrightElement) Click() error {
	return e.el.Click()
}

func (e *PlaywrightElement) Type(text string) error {
	return e.el.Type(text)
}

func (e *PlaywrightElement) Fill(text string) error {
	return e.el.Fill(text)
}

func (e *PlaywrightElement) Hover() error {
	return e.el.Hover()
}

func (e *PlaywrightElement) QuerySelector(selector string) (Element, error) {
	el, err := e.el.QuerySelector(selector)
	if err != nil {
		return nil, err
	}
	if el == nil {
		return nil, nil
	}
	return &PlaywrightElement{el: el}, nil
}

func (e *PlaywrightElement) QuerySelectorAll(selector string) ([]Element, error) {
	els, err := e.el.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	elements := make([]Element, len(els))
	for i, el := range els {
		elements[i] = &PlaywrightElement{el: el}
	}
	return elements, nil
}

func (e *PlaywrightElement) Screenshot(options ...ScreenshotOption) ([]byte, error) {
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
