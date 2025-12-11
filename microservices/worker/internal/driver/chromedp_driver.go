package driver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/shared/services"
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
func NewChromedpDriverWithProfile(cfg *config.BrowserConfig, profile *models.BrowserProfile) (*ChromedpDriver, error) {
	// Use shared browser factory for consistent configuration
	factoryOpts, err := services.BuildChromedpOptions(profile, cfg.Headless)
	if err != nil {
		return nil, fmt.Errorf("failed to build chromedp options: %w", err)
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]

	// Apply settings from shared factory
	opts = append(opts,
		chromedp.Flag("headless", factoryOpts.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// Screen dimensions from factory
	if factoryOpts.WindowWidth > 0 && factoryOpts.WindowHeight > 0 {
		opts = append(opts, chromedp.WindowSize(factoryOpts.WindowWidth, factoryOpts.WindowHeight))
	}

	// Proxy from factory (already validated by EnsureScheme)
	if factoryOpts.ProxyServer != "" {
		opts = append(opts, chromedp.ProxyServer(factoryOpts.ProxyServer))
	}

	// Executable path from factory
	if factoryOpts.ExecPath != "" {
		opts = append(opts, chromedp.ExecPath(factoryOpts.ExecPath))
	}

	// Extra args from factory
	for _, arg := range factoryOpts.ExtraArgs {
		opts = append(opts, chromedp.Flag(arg, true))
	}

	// WebRTC leak prevention (from profile)
	if profile != nil && profile.DisableWebRTC {
		opts = append(opts,
			chromedp.Flag("disable-webrtc", true),
			chromedp.Flag("enforce-webrtc-ip-permission-check", true),
		)
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	return &ChromedpDriver{
		cfg:         cfg,
		allocCtx:    allocCtx,
		cancelAlloc: cancelAlloc,
	}, nil
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
	opts := &ElementOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	// Find the element
	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Nodes(selector, &nodes, chromedp.NodeVisible),
	)
	if err != nil {
		return fmt.Errorf("failed to find element for hover: %w", err)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("no element found for hover selector: %s", selector)
	}

	// Scroll element into view and hover
	return chromedp.Run(ctx, chromedp.ActionFunc(func(actx context.Context) error {
		// Resolve node for JS calls
		obj, err := dom.ResolveNode().WithNodeID(nodes[0].NodeID).Do(actx)
		if err != nil {
			return fmt.Errorf("failed to resolve node for hover: %w", err)
		}

		// Scroll into view
		_, _, err = runtime.CallFunctionOn(`function() { 
			this.scrollIntoView({behavior: 'instant', block: 'center', inline: 'center'}); 
		}`).WithObjectID(obj.ObjectID).Do(actx)
		if err != nil {
			return fmt.Errorf("failed to scroll into view: %w", err)
		}

		// Wait for scroll to complete
		time.Sleep(50 * time.Millisecond)

		// Get element's bounding box
		boxes, err := dom.GetContentQuads().WithNodeID(nodes[0].NodeID).Do(actx)
		if err != nil {
			return fmt.Errorf("failed to get element quads for hover: %w", err)
		}

		// If we have bounds, move mouse to center
		if len(boxes) > 0 && len(boxes[0]) >= 8 {
			quad := boxes[0]
			x := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
			y := (quad[1] + quad[3] + quad[5] + quad[7]) / 4
			_ = input.DispatchMouseEvent(input.MouseMoved, x, y).Do(actx)
		}

		// Always dispatch JS events to trigger CSS :hover
		_, _, err = runtime.CallFunctionOn(`function() {
			this.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true, cancelable: true }));
			this.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, cancelable: true }));
		}`).WithObjectID(obj.ObjectID).Do(actx)
		return err
	}))
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

func (p *ChromedpPage) WaitForURL(urlPattern string, options ...WaitOption) error {
	opts := &WaitOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx, cancel := context.WithTimeout(p.ctx, opts.Timeout)
	defer cancel()

	// Poll for URL match
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for URL to match: %s", urlPattern)
		case <-ticker.C:
			var currentURL string
			if err := chromedp.Run(p.ctx, chromedp.Location(&currentURL)); err != nil {
				return err
			}
			// Simple substring match (can be enhanced for regex)
			if strings.Contains(currentURL, urlPattern) || currentURL == urlPattern {
				return nil
			}
		}
	}
}

func (p *ChromedpPage) WaitForState(state string, options ...WaitOption) error {
	// Map Playwright states to Chromedp events
	// load -> page.EventLoadEventFired
	// domcontentloaded -> page.EventDomContentEventFired
	// networkidle -> network.EventLoadingFinished (complex)
	return nil // No-op for now or implement specific waits
}

func (p *ChromedpPage) WaitForFunction(expression string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	// Poll until expression returns truthy value
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for function: %s", expression)
		case <-ticker.C:
			var result bool
			if err := chromedp.Run(p.ctx, chromedp.Evaluate(expression, &result)); err != nil {
				continue // Ignore evaluation errors, keep polling
			}
			if result {
				return nil
			}
		}
	}
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
	// Check if selector contains jQuery-style :visible pseudo-class (not standard CSS)
	if strings.Contains(selector, ":visible") {
		return p.querySelectorAllWithVisibility(selector)
	}

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

// querySelectorAllWithVisibility handles jQuery-style :visible pseudo-selector
// by using JavaScript to find and tag visible elements atomically
func (p *ChromedpPage) querySelectorAllWithVisibility(selector string) ([]Element, error) {
	// Use pure JavaScript to handle :visible atomically
	// This avoids losing hover state between finding and filtering
	timestamp := time.Now().UnixNano()
	tagAttr := fmt.Sprintf("data-cdp-visible-%d", timestamp)

	jsCode := fmt.Sprintf(`
		(function() {
			// Helper to check element visibility
			function isVisible(el) {
				if (!el) return false;
				
				// Use modern checkVisibility if available
				if (typeof el.checkVisibility === 'function') {
					return el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true });
				}
				
				// Fallback visibility check
				const style = window.getComputedStyle(el);
				if (style.display === 'none' || style.visibility === 'hidden' || parseFloat(style.opacity) === 0) {
					return false;
				}
				
				// Check parent chain
				let current = el.parentElement;
				while (current) {
					const s = window.getComputedStyle(current);
					if (s.display === 'none' || s.visibility === 'hidden') {
						return false;
					}
					current = current.parentElement;
				}
				
				// Check has dimensions
				const rect = el.getBoundingClientRect();
				return rect.width > 0 && rect.height > 0;
			}
			
			const selector = %q;
			const tagAttr = %q;
			
			// Handle comma-separated selectors
			const selectorParts = selector.split(',').map(s => s.trim());
			let count = 0;
			
			for (const part of selectorParts) {
				// Check if this part has :visible
				if (part.includes(':visible')) {
					// Remove :visible and find the ancestor that should be visible
					const tokens = part.split(/\s+/);
					let cleanTokens = [];
					let visibleAncestorSelector = null;
					
					for (let i = 0; i < tokens.length; i++) {
						if (tokens[i].includes(':visible')) {
							const cleanToken = tokens[i].replace(/:visible/g, '');
							cleanTokens.push(cleanToken);
							// Build selector for the visible ancestor
							visibleAncestorSelector = cleanTokens.slice(0, i + 1).join(' ');
						} else {
							cleanTokens.push(tokens[i]);
						}
					}
					
					const cleanSelector = cleanTokens.join(' ');
					
					try {
						const elements = document.querySelectorAll(cleanSelector);
						
						elements.forEach(el => {
							// Find the ancestor that should be visible
							let ancestor = null;
							if (visibleAncestorSelector) {
								// Try to find the visible ancestor
								ancestor = el.closest(visibleAncestorSelector.split(' ').pop());
							}
							
							// Check if ancestor (or element itself) is visible
							const targetToCheck = ancestor || el;
							if (isVisible(targetToCheck)) {
								el.setAttribute(tagAttr, count.toString());
								count++;
							}
						});
					} catch (e) {
						console.error('Selector error:', e);
					}
				} else {
					// No :visible, just select and tag all
					try {
						const elements = document.querySelectorAll(part);
						elements.forEach(el => {
							el.setAttribute(tagAttr, count.toString());
							count++;
						});
					} catch (e) {
						console.error('Selector error:', e);
					}
				}
			}
			
			return count;
		})()
	`, selector, tagAttr)

	// Execute JS to tag visible elements
	var count int
	err := chromedp.Run(p.ctx, chromedp.Evaluate(jsCode, &count))
	if err != nil {
		return nil, fmt.Errorf("visibility query failed: %w", err)
	}

	if count == 0 {
		return []Element{}, nil
	}

	// Now query for the tagged elements using CDP
	taggedSelector := fmt.Sprintf("[%s]", tagAttr)

	var nodes []*cdp.Node
	err = chromedp.Run(p.ctx, chromedp.Nodes(taggedSelector, &nodes, chromedp.AtLeast(0)))
	if err != nil {
		return nil, err
	}

	// Clean up the temporary attributes
	cleanupJS := fmt.Sprintf(`
		document.querySelectorAll('[%s]').forEach(el => el.removeAttribute('%s'));
	`, tagAttr, tagAttr)
	_ = chromedp.Run(p.ctx, chromedp.Evaluate(cleanupJS, nil))

	// Convert to elements
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

// SolveCaptcha is a stub implementation for ChromeDP.
// ChromeDP / CDP lacks the shadow DOM APIs (shadowRootUnl) required for reliable
// Cloudflare CAPTCHA solving. For CAPTCHA support, use the Camoufox driver.
func (p *ChromedpPage) SolveCaptcha(opts CaptchaSolveOptions) (bool, error) {
	return false, ErrNotSupported
}

// SupportsCaptchaSolving returns false as ChromeDP cannot reliably solve CAPTCHAs.
// CDP lacks access to closed Shadow DOM elements which are used by Cloudflare.
func (p *ChromedpPage) SupportsCaptchaSolving() bool {
	return false
}

// ChromedpElement implements the Element interface
type ChromedpElement struct {
	ctx  context.Context
	node *cdp.Node
}

func (e *ChromedpElement) Text() (string, error) {
	// Use dom.GetOuterHTML and extract text content via JS
	var text string
	err := chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Get the text content using JavaScript on the node
		obj, err := dom.ResolveNode().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve node: %w", err)
		}

		result, _, err := runtime.CallFunctionOn(`function() { return this.textContent || this.innerText || ''; }`).WithObjectID(obj.ObjectID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to get text content: %w", err)
		}

		if result.Value != nil {
			// Remove quotes from JSON string
			text = string(result.Value)
			if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
				text = text[1 : len(text)-1]
			}
		}
		return nil
	}))
	return text, err
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
	var html string
	err := chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		html, err = dom.GetOuterHTML().WithNodeID(e.node.NodeID).Do(ctx)
		return err
	}))
	return html, err
}

func (e *ChromedpElement) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	var buf []byte
	err := chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Get element's bounding box
		boxes, err := dom.GetContentQuads().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to get element quads: %w", err)
		}

		if len(boxes) == 0 || len(boxes[0]) < 8 {
			return fmt.Errorf("element has no visible bounds")
		}

		// Calculate bounding box from quad
		quad := boxes[0]
		minX := quad[0]
		maxX := quad[0]
		minY := quad[1]
		maxY := quad[1]
		for i := 0; i < 8; i += 2 {
			if quad[i] < minX {
				minX = quad[i]
			}
			if quad[i] > maxX {
				maxX = quad[i]
			}
			if quad[i+1] < minY {
				minY = quad[i+1]
			}
			if quad[i+1] > maxY {
				maxY = quad[i+1]
			}
		}

		clip := &page.Viewport{
			X:      minX,
			Y:      minY,
			Width:  maxX - minX,
			Height: maxY - minY,
			Scale:  1.0,
		}

		buf, err = page.CaptureScreenshot().
			WithClip(clip).
			WithFormat(page.CaptureScreenshotFormatPng).
			Do(ctx)
		return err
	}))
	return buf, err
}

func (e *ChromedpElement) Click() error {
	return chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Get element's center coordinates
		boxes, err := dom.GetContentQuads().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to get element quads: %w", err)
		}

		if len(boxes) == 0 || len(boxes[0]) < 8 {
			return fmt.Errorf("element has no visible bounds")
		}

		// Calculate center from quad points (8 floats: x1,y1, x2,y2, x3,y3, x4,y4)
		quad := boxes[0]
		x := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
		y := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

		// Click requires MousePressed + MouseReleased
		if err := input.DispatchMouseEvent(input.MousePressed, x, y).
			WithButton(input.Left).
			WithClickCount(1).
			Do(ctx); err != nil {
			return err
		}
		return input.DispatchMouseEvent(input.MouseReleased, x, y).
			WithButton(input.Left).
			WithClickCount(1).
			Do(ctx)
	}))
}

func (e *ChromedpElement) Type(text string) error {
	return chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Focus the element first
		if err := dom.Focus().WithNodeID(e.node.NodeID).Do(ctx); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}

		// Type the text using keyboard input
		for _, char := range text {
			if err := input.DispatchKeyEvent(input.KeyDown).WithText(string(char)).Do(ctx); err != nil {
				return err
			}
			if err := input.DispatchKeyEvent(input.KeyUp).WithText(string(char)).Do(ctx); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (e *ChromedpElement) Fill(text string) error {
	return chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Focus the element
		if err := dom.Focus().WithNodeID(e.node.NodeID).Do(ctx); err != nil {
			return fmt.Errorf("failed to focus element: %w", err)
		}

		// Resolve to remote object and set value via JS
		obj, err := dom.ResolveNode().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve node: %w", err)
		}

		// Clear and set value
		_, _, err = runtime.CallFunctionOn(fmt.Sprintf(`function() { 
			this.value = ''; 
			this.value = %q;
			this.dispatchEvent(new Event('input', { bubbles: true }));
			this.dispatchEvent(new Event('change', { bubbles: true }));
		}`, text)).WithObjectID(obj.ObjectID).Do(ctx)
		return err
	}))
}

func (e *ChromedpElement) Hover() error {
	return chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// First, scroll element into view
		obj, err := dom.ResolveNode().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve node for hover: %w", err)
		}

		// Scroll into view using JavaScript
		_, _, err = runtime.CallFunctionOn(`function() { 
			this.scrollIntoView({behavior: 'instant', block: 'center', inline: 'center'}); 
		}`).WithObjectID(obj.ObjectID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to scroll into view: %w", err)
		}

		// Wait a moment for scroll to complete
		time.Sleep(50 * time.Millisecond)

		// Get element's center coordinates for mouse move
		boxes, err := dom.GetContentQuads().WithNodeID(e.node.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to get element quads: %w", err)
		}

		if len(boxes) == 0 || len(boxes[0]) < 8 {
			// Fallback: use JS to trigger hover events directly
			_, _, err = runtime.CallFunctionOn(`function() {
				this.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true, cancelable: true }));
				this.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, cancelable: true }));
			}`).WithObjectID(obj.ObjectID).Do(ctx)
			return err
		}

		// Calculate center from quad points
		quad := boxes[0]
		x := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
		y := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

		// Move mouse to element center
		if err := input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx); err != nil {
			return err
		}

		// Also dispatch JS events to ensure CSS :hover is triggered in all cases
		_, _, err = runtime.CallFunctionOn(`function() {
			this.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true, cancelable: true }));
			this.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, cancelable: true }));
		}`).WithObjectID(obj.ObjectID).Do(ctx)
		return err
	}))
}

func (e *ChromedpElement) QuerySelector(selector string) (Element, error) {
	var childNode *cdp.Node
	err := chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Query for child nodes within this element
		nodeID, err := dom.QuerySelector(e.node.NodeID, selector).Do(ctx)
		if err != nil {
			return err
		}
		if nodeID == 0 {
			return nil // No match found
		}

		// Get the node details
		node, err := dom.DescribeNode().WithNodeID(nodeID).Do(ctx)
		if err != nil {
			return err
		}
		childNode = node
		childNode.NodeID = nodeID
		return nil
	}))

	if err != nil {
		return nil, err
	}
	if childNode == nil {
		return nil, nil
	}
	return &ChromedpElement{ctx: e.ctx, node: childNode}, nil
}

func (e *ChromedpElement) QuerySelectorAll(selector string) ([]Element, error) {
	var elements []Element
	err := chromedp.Run(e.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Query for all matching child nodes
		nodeIDs, err := dom.QuerySelectorAll(e.node.NodeID, selector).Do(ctx)
		if err != nil {
			return err
		}

		for _, nodeID := range nodeIDs {
			node, err := dom.DescribeNode().WithNodeID(nodeID).Do(ctx)
			if err != nil {
				continue
			}
			node.NodeID = nodeID
			elements = append(elements, &ChromedpElement{ctx: e.ctx, node: node})
		}
		return nil
	}))

	return elements, err
}
