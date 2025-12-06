package driver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
)

// ChromedpDriver implements the Driver interface using chromedp
type ChromedpDriver struct {
	cfg         *config.BrowserConfig
	allocCtx    context.Context
	cancelAlloc context.CancelFunc
}

// NewChromedpDriver creates a new ChromedpDriver
func NewChromedpDriver(cfg *config.BrowserConfig) *ChromedpDriver {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	return &ChromedpDriver{
		cfg:         cfg,
		allocCtx:    allocCtx,
		cancelAlloc: cancelAlloc,
	}
}

// NewChromedpDriverWithProfile creates a ChromedpDriver using browser profile settings
func NewChromedpDriverWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) *ChromedpDriver {
	opts := chromedp.DefaultExecAllocatorOptions[:]

	// Apply profile settings
	opts = append(opts,
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// Custom executable path from profile
	if profile.ExecutablePath != "" {
		opts = append(opts, chromedp.ExecPath(profile.ExecutablePath))
	}

	// User agent from profile
	if profile.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(profile.UserAgent))
	}

	// Screen size
	if profile.ScreenWidth > 0 && profile.ScreenHeight > 0 {
		opts = append(opts, chromedp.WindowSize(profile.ScreenWidth, profile.ScreenHeight))
	}

	// Proxy from profile
	if profile.ProxyEnabled && profile.ProxyServer != "" {
		proxyURL := profile.ProxyServer
		if profile.ProxyType != "" {
			proxyURL = profile.ProxyType + "://" + proxyURL
		}
		opts = append(opts, chromedp.ProxyServer(proxyURL))
	}

	// WebRTC leak prevention
	if profile.DisableWebRTC {
		opts = append(opts,
			chromedp.Flag("disable-webrtc", true),
			chromedp.Flag("enforce-webrtc-ip-permission-check", true),
		)
	}

	// Extra launch args from profile
	for _, arg := range profile.LaunchArgs {
		opts = append(opts, chromedp.Flag(arg, true))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	return &ChromedpDriver{
		cfg:         cfg,
		allocCtx:    allocCtx,
		cancelAlloc: cancelAlloc,
	}
}

func (d *ChromedpDriver) NewPage(ctx context.Context) (Page, error) {
	// Create a new context (tab) from the shared allocator
	// Note: We ignore the proxy config from context for now as it requires
	// a new browser process or specific proxy settings per context which
	// is complex with shared allocator.
	// TODO: Support per-page proxy with shared allocator if possible,
	// or fallback to new allocator for proxy tasks.

	// Check for proxy in context - if present, we MUST create a separate allocator
	// because proxy is set at browser level in Chrome
	if proxyCfg, ok := ctx.Value(ProxyKey).(*browser.ProxyConfig); ok && proxyCfg != nil {
		return d.newPageWithProxy(ctx, proxyCfg)
	}

	// Standard shared browser tab
	pageCtx, cancelPage := chromedp.NewContext(d.allocCtx)

	// Ensure tab is created
	if err := chromedp.Run(pageCtx); err != nil {
		cancelPage()
		return nil, fmt.Errorf("failed to create tab: %w", err)
	}

	return &ChromedpPage{
		ctx:    pageCtx,
		cancel: cancelPage,
	}, nil
}

// newPageWithProxy creates a dedicated browser instance for proxy support
func (d *ChromedpDriver) newPageWithProxy(ctx context.Context, proxyCfg *browser.ProxyConfig) (Page, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", d.cfg.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.ProxyServer(proxyCfg.Server),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	pageCtx, cancelPage := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(pageCtx); err != nil {
		cancelPage()
		cancelAlloc()
		return nil, fmt.Errorf("failed to start proxy browser: %w", err)
	}

	return &ChromedpPage{
		ctx:         pageCtx,
		cancel:      cancelPage,
		cancelAlloc: cancelAlloc, // Only set for dedicated allocators
	}, nil
}

func (d *ChromedpDriver) Close() error {
	if d.cancelAlloc != nil {
		d.cancelAlloc()
	}
	return nil
}

func (d *ChromedpDriver) Name() string {
	return "chromedp"
}

// ChromedpPage implements the Page interface
type ChromedpPage struct {
	ctx         context.Context
	cancel      context.CancelFunc
	cancelAlloc context.CancelFunc // Optional, for dedicated browsers
}

func (p *ChromedpPage) Close() error {
	p.cancel()
	if p.cancelAlloc != nil {
		p.cancelAlloc()
	}
	return nil
}

func (p *ChromedpPage) DriverName() string {
	return "chromedp"
}

func (p *ChromedpPage) Goto(url string, options ...PageOption) error {
	opts := &PageOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	// Create a timeout context for the navigation
	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Navigate(url))
}

func (p *ChromedpPage) Content() (string, error) {
	var html string
	err := chromedp.Run(p.ctx, chromedp.OuterHTML("html", &html))
	return html, err
}

func (p *ChromedpPage) Title() (string, error) {
	var title string
	err := chromedp.Run(p.ctx, chromedp.Title(&title))
	return title, err
}

func (p *ChromedpPage) URL() (string, error) {
	var url string
	err := chromedp.Run(p.ctx, chromedp.Location(&url))
	return url, err
}

func (p *ChromedpPage) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	opts := &ScreenshotOptions{}
	for _, opt := range options {
		opt(opts)
	}

	var buf []byte
	var err error

	if opts.FullPage {
		err = chromedp.Run(p.ctx, chromedp.FullScreenshot(&buf, opts.Quality))
	} else {
		err = chromedp.Run(p.ctx, chromedp.CaptureScreenshot(&buf))
	}

	return buf, err
}

func (p *ChromedpPage) Click(selector string, options ...ElementOption) error {
	opts := &ElementOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Click(selector, chromedp.NodeVisible))
}

func (p *ChromedpPage) Type(selector, text string, options ...ElementOption) error {
	opts := &ElementOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.SendKeys(selector, text, chromedp.NodeVisible))
}

func (p *ChromedpPage) Fill(selector, text string, options ...ElementOption) error {
	// Chromedp doesn't have a direct Fill, usually SetValue or SendKeys
	// SetValue is closer to Fill (clears and sets)
	opts := &ElementOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.SetValue(selector, text, chromedp.NodeVisible))
}

func (p *ChromedpPage) Hover(selector string, options ...ElementOption) error {
	// Chromedp doesn't have a direct Hover action exposed easily in high level API
	// But we can use MouseMove
	// Or execute JS
	// Actually chromedp has MouseMoveNode
	return fmt.Errorf("hover not fully implemented for chromedp yet")
}

func (p *ChromedpPage) WaitForSelector(selector string, options ...WaitOption) error {
	opts := &WaitOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	if opts.State == "hidden" || opts.State == "detached" {
		return chromedp.Run(ctx, chromedp.WaitNotPresent(selector))
	}
	return chromedp.Run(ctx, chromedp.WaitVisible(selector))
}

func (p *ChromedpPage) WaitForURL(url string, options ...WaitOption) error {
	// Simple wait for URL to contain string
	// Chromedp doesn't have a direct WaitForURL like Playwright
	// We can poll Location
	return fmt.Errorf("WaitForURL not implemented for chromedp yet")
}

func (p *ChromedpPage) WaitForState(state string, options ...WaitOption) error {
	// Map Playwright states to Chromedp events
	// load -> page.EventLoadEventFired
	// domcontentloaded -> page.EventDomContentEventFired
	// networkidle -> network.EventLoadingFinished (complex)
	return nil // No-op for now or implement specific waits
}

func (p *ChromedpPage) WaitForFunction(expression string, args ...interface{}) error {
	// Evaluate expression until true
	// chromedp.WaitFunc?
	return fmt.Errorf("WaitForFunction not implemented for chromedp yet")
}

func (p *ChromedpPage) Evaluate(expression string, args ...interface{}) (interface{}, error) {
	var res interface{}
	err := chromedp.Run(p.ctx, chromedp.Evaluate(expression, &res))
	return res, err
}

func (p *ChromedpPage) AddInitScript(script string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return err
	}))
}

func (p *ChromedpPage) QuerySelector(selector string) (Element, error) {
	var nodes []*cdp.Node
	err := chromedp.Run(p.ctx, chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return &ChromedpElement{ctx: p.ctx, node: nodes[0]}, nil
}

func (p *ChromedpPage) QuerySelectorAll(selector string) ([]Element, error) {
	var nodes []*cdp.Node
	err := chromedp.Run(p.ctx, chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return nil, err
	}

	elements := make([]Element, len(nodes))
	for i, node := range nodes {
		elements[i] = &ChromedpElement{ctx: p.ctx, node: node}
	}
	return elements, nil
}

func (p *ChromedpPage) GetCookies() ([]*http.Cookie, error) {
	var cookies []*network.Cookie
	err := chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return nil, err
	}

	httpCookies := make([]*http.Cookie, len(cookies))
	for i, c := range cookies {
		httpCookies[i] = &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  time.Unix(int64(c.Expires), 0),
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
			// SameSite mapping needed
		}
		switch c.SameSite {
		case network.CookieSameSiteStrict:
			httpCookies[i].SameSite = http.SameSiteStrictMode
		case network.CookieSameSiteLax:
			httpCookies[i].SameSite = http.SameSiteLaxMode
		case network.CookieSameSiteNone:
			httpCookies[i].SameSite = http.SameSiteNoneMode
		}
	}
	return httpCookies, nil
}

func (p *ChromedpPage) SetCookies(cookies []*http.Cookie) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		for _, c := range cookies {
			sameSite := network.CookieSameSiteNone
			switch c.SameSite {
			case http.SameSiteStrictMode:
				sameSite = network.CookieSameSiteStrict
			case http.SameSiteLaxMode:
				sameSite = network.CookieSameSiteLax
			}

			// Expiration
			expires := cdp.TimeSinceEpoch(c.Expires)

			err := network.SetCookie(c.Name, c.Value).
				WithDomain(c.Domain).
				WithPath(c.Path).
				WithSecure(c.Secure).
				WithHTTPOnly(c.HttpOnly).
				WithSameSite(sameSite).
				WithExpires(&expires).
				Do(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

// ChromedpElement implements the Element interface
type ChromedpElement struct {
	ctx  context.Context
	node *cdp.Node
}

func (e *ChromedpElement) Text() (string, error) {
	// This might need a selector or node ID.
	// chromedp.Text usually takes a selector.
	// To get text of a specific node without selector is harder.
	// We can use Javascript or specific CDP query.
	// For now, let's try to use Javascript on the node.
	// Actually, we need to resolve the node to a remote object ID to use with DOM.
	// Or we can construct a unique selector (XPath) for this node.
	return "", fmt.Errorf("Text() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Attribute(name string) (string, error) {
	// Similar issue, need to operate on the specific node.
	// We can iterate attributes from e.node.Attributes if populated?
	// cdp.Node has Attributes []string (key, value pairs)
	for i := 0; i < len(e.node.Attributes); i += 2 {
		if e.node.Attributes[i] == name {
			return e.node.Attributes[i+1], nil
		}
	}
	return "", nil
}

func (e *ChromedpElement) InnerHTML() (string, error) {
	return "", fmt.Errorf("InnerHTML() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	return nil, fmt.Errorf("Screenshot() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Click() error {
	// Need to resolve node to something clickable
	return fmt.Errorf("Click() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Type(text string) error {
	_ = text
	return fmt.Errorf("Type() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Fill(text string) error {
	_ = text
	return fmt.Errorf("Fill() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) Hover() error {
	return fmt.Errorf("Hover() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) QuerySelector(selector string) (Element, error) {
	return nil, fmt.Errorf("QuerySelector() on element not fully implemented for chromedp")
}

func (e *ChromedpElement) QuerySelectorAll(selector string) ([]Element, error) {
	return nil, fmt.Errorf("QuerySelectorAll() on element not fully implemented for chromedp")
}
