package driver

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
)

// HttpDriver implements the Driver interface using net/http and goquery
type HttpDriver struct {
	client *http.Client
}

// NewHttpDriver creates a new HttpDriver
func NewHttpDriver() *HttpDriver {
	jar, _ := cookiejar.New(nil)
	return &HttpDriver{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

func (d *HttpDriver) NewPage(ctx context.Context) (Page, error) {
	// Start with default client
	client := d.client

	// Check for proxy in context
	if proxyCfg, ok := ctx.Value(ProxyKey).(*browser.ProxyConfig); ok && proxyCfg != nil {
		// Create custom transport with proxy
		proxyURL, err := url.Parse(proxyCfg.Server)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		// Handle authentication if provided
		if proxyCfg.Username != "" {
			auth := proxyCfg.Username + ":" + proxyCfg.Password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			transport.ProxyConnectHeader = http.Header{}
			transport.ProxyConnectHeader.Add("Proxy-Authorization", basicAuth)
		}

		// Create new client with proxy transport
		client = &http.Client{
			Transport: transport,
			Timeout:   d.client.Timeout,
		}
	}

	return &HttpPage{
		client: client,
		ctx:    ctx,
	}, nil
}

func (d *HttpDriver) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

func (d *HttpDriver) Name() string {
	return "http"
}

// HttpPage implements the Page interface
type HttpPage struct {
	client *http.Client
	ctx    context.Context
	doc    *goquery.Document
	url    string
	body   string
}

func (p *HttpPage) Goto(url string, options ...PageOption) error {
	opts := &PageOptions{
		Timeout: 30 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}

	req, err := http.NewRequestWithContext(p.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set common headers to look like a browser
	// Set common headers to look like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	p.body = string(bodyBytes)
	p.url = url

	// Parse DOM
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}
	p.doc = doc

	return nil
}

func (p *HttpPage) Content() (string, error) {
	if p.body == "" {
		return "", fmt.Errorf("no content loaded")
	}
	return p.body, nil
}

func (p *HttpPage) Title() (string, error) {
	if p.doc == nil {
		return "", fmt.Errorf("page not loaded")
	}
	return p.doc.Find("title").Text(), nil
}

func (p *HttpPage) URL() (string, error) {
	return p.url, nil
}

func (p *HttpPage) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	return nil, ErrNotSupported
}

func (p *HttpPage) Click(selector string, options ...ElementOption) error {
	return ErrNotSupported
}

func (p *HttpPage) Type(selector, text string, options ...ElementOption) error {
	return ErrNotSupported
}

func (p *HttpPage) Fill(selector, text string, options ...ElementOption) error {
	return ErrNotSupported
}

func (p *HttpPage) Hover(selector string, options ...ElementOption) error {
	return ErrNotSupported
}

func (p *HttpPage) WaitForSelector(selector string, options ...WaitOption) error {
	if p.doc == nil {
		return fmt.Errorf("page not loaded")
	}

	// Static check: if element exists, we are good. If not, it will never appear.
	if p.doc.Find(selector).Length() > 0 {
		return nil
	}

	// Check if we should wait for "hidden" or "detached" state
	opts := &WaitOptions{}
	for _, opt := range options {
		opt(opts)
	}

	if opts.State == "hidden" || opts.State == "detached" {
		return nil // It's not there, so it's hidden/detached
	}

	return ErrElementNotFound
}

func (p *HttpPage) WaitForURL(url string, options ...WaitOption) error {
	if p.url == url {
		return nil
	}
	// Simple check if current URL matches (or contains)
	if strings.Contains(p.url, url) {
		return nil
	}
	return fmt.Errorf("timeout waiting for URL: %s", url)
}

func (p *HttpPage) WaitForState(state string, options ...WaitOption) error {
	// HttpPage is always "loaded" after Goto returns
	return nil
}

func (p *HttpPage) WaitForFunction(expression string, args ...interface{}) error {
	return ErrNotSupported
}

func (p *HttpPage) Evaluate(expression string, args ...interface{}) (interface{}, error) {
	return nil, ErrNotSupported
}

func (p *HttpPage) AddInitScript(script string) error {
	return ErrNotSupported
}

func (p *HttpPage) QuerySelector(selector string) (Element, error) {
	if p.doc == nil {
		return nil, fmt.Errorf("page not loaded")
	}
	sel := p.doc.Find(selector).First()
	if sel.Length() == 0 {
		return nil, nil
	}
	return &HttpElement{sel: sel}, nil
}

func (p *HttpPage) QuerySelectorAll(selector string) ([]Element, error) {
	if p.doc == nil {
		return nil, fmt.Errorf("page not loaded")
	}
	var elements []Element
	p.doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		elements = append(elements, &HttpElement{sel: s})
	})
	return elements, nil
}

func (p *HttpPage) Close() error {
	p.doc = nil
	p.body = ""
	return nil
}

func (p *HttpPage) GetCookies() ([]*http.Cookie, error) {
	if p.client.Jar == nil {
		return []*http.Cookie{}, nil
	}
	u, err := url.Parse(p.url)
	if err != nil {
		return nil, err
	}
	return p.client.Jar.Cookies(u), nil
}

func (p *HttpPage) SetCookies(cookies []*http.Cookie) error {
	if p.client.Jar == nil {
		return fmt.Errorf("cookie jar not initialized")
	}
	u, err := url.Parse(p.url)
	if err != nil {
		return p.setCookiesByDomain(cookies)
	}
	p.client.Jar.SetCookies(u, cookies)
	return nil
}

func (p *HttpPage) setCookiesByDomain(cookies []*http.Cookie) error {
	for _, c := range cookies {
		domain := c.Domain
		if strings.HasPrefix(domain, ".") {
			domain = domain[1:]
		}
		scheme := "http"
		if c.Secure {
			scheme = "https"
		}
		u := &url.URL{Scheme: scheme, Host: domain, Path: c.Path}
		p.client.Jar.SetCookies(u, []*http.Cookie{c})
	}
	return nil
}

// HttpElement implements the Element interface
type HttpElement struct {
	sel *goquery.Selection
}

func (e *HttpElement) Text() (string, error) {
	return e.sel.Text(), nil
}

func (e *HttpElement) Attribute(name string) (string, error) {
	val, exists := e.sel.Attr(name)
	if !exists {
		return "", nil // Return empty string if attribute not found, or error? Playwright returns empty string usually or null.
	}
	return val, nil
}

func (e *HttpElement) InnerHTML() (string, error) {
	return e.sel.Html()
}

func (e *HttpElement) Screenshot(options ...ScreenshotOption) ([]byte, error) {
	return nil, ErrNotSupported
}

func (e *HttpElement) Click() error {
	return ErrNotSupported
}

func (e *HttpElement) Type(text string) error {
	return ErrNotSupported
}

func (e *HttpElement) Fill(text string) error {
	return ErrNotSupported
}

func (e *HttpElement) Hover() error {
	return ErrNotSupported
}

func (e *HttpElement) QuerySelector(selector string) (Element, error) {
	sel := e.sel.Find(selector).First()
	if sel.Length() == 0 {
		return nil, nil
	}
	return &HttpElement{sel: sel}, nil
}

func (e *HttpElement) QuerySelectorAll(selector string) ([]Element, error) {
	var elements []Element
	e.sel.Find(selector).Each(func(i int, s *goquery.Selection) {
		elements = append(elements, &HttpElement{sel: s})
	})
	return elements, nil
}
