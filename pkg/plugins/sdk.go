package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"go.uber.org/zap"
)

// SDK provides helper functions for plugin developers
type SDK struct {
	logger *zap.Logger
}

// NewSDK creates a new plugin SDK instance
func NewSDK(logger *zap.Logger) *SDK {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SDK{
		logger: logger,
	}
}

// BrowserHelpers provides simplified browser interaction methods
type BrowserHelpers struct {
	ctx *browser.BrowserContext
}

// NewBrowserHelpers creates browser helper utilities
func (s *SDK) NewBrowserHelpers(ctx *browser.BrowserContext) *BrowserHelpers {
	return &BrowserHelpers{ctx: ctx}
}

// WaitForSelector waits for element to appear
func (bh *BrowserHelpers) WaitForSelector(selector string, timeoutMs int) error {
	return bh.ctx.WaitForSelector(selector, time.Duration(timeoutMs)*time.Millisecond)
}

// Click clicks on element matching selector
func (bh *BrowserHelpers) Click(selector string) error {
	page := bh.ctx.Page
	return page.Click(selector)
}

// Type types text into element
func (bh *BrowserHelpers) Type(selector, text string) error {
	page := bh.ctx.Page
	return page.Type(selector, text)
}

// GetText extracts text content from element
func (bh *BrowserHelpers) GetText(selector string) (string, error) {
	page := bh.ctx.Page
	return page.TextContent(selector)
}

// GetAttribute gets attribute value from element
func (bh *BrowserHelpers) GetAttribute(selector, attribute string) (string, error) {
	page := bh.ctx.Page
	return page.GetAttribute(selector, attribute)
}

// GetAllText extracts text from all matching elements
func (bh *BrowserHelpers) GetAllText(selector string) ([]string, error) {
	page := bh.ctx.Page
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	var texts []string
	for _, elem := range elements {
		text, err := elem.TextContent()
		if err == nil && text != "" {
			texts = append(texts, strings.TrimSpace(text))
		}
	}
	return texts, nil
}

// ExtractLinks extracts all href attributes from matching elements
func (bh *BrowserHelpers) ExtractLinks(selector string) ([]string, error) {
	page := bh.ctx.Page
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}

	var links []string
	for _, elem := range elements {
		href, err := elem.GetAttribute("href")
		if err == nil && href != "" {
			links = append(links, href)
		}
	}
	return links, nil
}

// ScrollToBottom scrolls page to bottom
func (bh *BrowserHelpers) ScrollToBottom(waitMs int) error {
	page := bh.ctx.Page
	_, err := page.Evaluate(`
		window.scrollTo(0, document.body.scrollHeight);
	`)
	if err != nil {
		return err
	}
	if waitMs > 0 {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
	}
	return nil
}

// URLHelpers provides URL processing utilities
type URLHelpers struct{}

// NewURLHelpers creates URL helper utilities
func (s *SDK) NewURLHelpers() *URLHelpers {
	return &URLHelpers{}
}

// NormalizeURL normalizes a URL (removes fragment, sorts params, etc.)
func (uh *URLHelpers) NormalizeURL(rawURL string) (string, error) {
	// Remove fragments
	if idx := strings.Index(rawURL, "#"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	// Trim whitespace
	return strings.TrimSpace(rawURL), nil
}

// IsAbsoluteURL checks if URL is absolute
func (uh *URLHelpers) IsAbsoluteURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// JoinURL joins base URL with relative path
func (uh *URLHelpers) JoinURL(baseURL, relativeURL string) (string, error) {
	if uh.IsAbsoluteURL(relativeURL) {
		return relativeURL, nil
	}

	// Simple join - in production would use url.Parse and url.ResolveReference
	baseURL = strings.TrimSuffix(baseURL, "/")
	relativeURL = strings.TrimPrefix(relativeURL, "/")
	return fmt.Sprintf("%s/%s", baseURL, relativeURL), nil
}

// MatchesPattern checks if URL matches regex pattern
func (uh *URLHelpers) MatchesPattern(url, pattern string) (bool, error) {
	return regexp.MatchString(pattern, url)
}

// DataHelpers provides data extraction and transformation utilities
type DataHelpers struct{}

// NewDataHelpers creates data helper utilities
func (s *SDK) NewDataHelpers() *DataHelpers {
	return &DataHelpers{}
}

// ParseHTML parses HTML string and returns goquery document
func (dh *DataHelpers) ParseHTML(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

// ExtractWithCSS extracts data using CSS selector
func (dh *DataHelpers) ExtractWithCSS(doc *goquery.Document, selector string) []string {
	var results []string
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		results = append(results, strings.TrimSpace(s.Text()))
	})
	return results
}

// ExtractAttribute extracts attribute values using CSS selector
func (dh *DataHelpers) ExtractAttribute(doc *goquery.Document, selector, attr string) []string {
	var results []string
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr(attr); exists {
			results = append(results, val)
		}
	})
	return results
}

// CleanText cleans and normalizes text
func (dh *DataHelpers) CleanText(text string) string {
	// Remove extra whitespace
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return text
}

// ExtractNumbers extracts all numbers from text
func (dh *DataHelpers) ExtractNumbers(text string) []string {
	re := regexp.MustCompile(`\d+\.?\d*`)
	return re.FindAllString(text, -1)
}

// Logger provides structured logging for plugins
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a logger for plugins
func (s *SDK) NewLogger(pluginID string) *Logger {
	return &Logger{
		logger: s.logger.With(zap.String("plugin_id", pluginID)),
	}
}

// Info logs info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Error logs error message
func (l *Logger) Error(msg string, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	l.logger.Error(msg, fields...)
}

// Debug logs debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Warn logs warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// ConfigHelpers provides configuration parsing utilities
type ConfigHelpers struct{}

// NewConfigHelpers creates config helper utilities
func (s *SDK) NewConfigHelpers() *ConfigHelpers {
	return &ConfigHelpers{}
}

// GetString safely retrieves string from config
func (ch *ConfigHelpers) GetString(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return defaultValue
}

// GetInt safely retrieves int from config
func (ch *ConfigHelpers) GetInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key].(float64); ok {
		return int(val)
	}
	if val, ok := config[key].(int); ok {
		return val
	}
	return defaultValue
}

// GetBool safely retrieves bool from config
func (ch *ConfigHelpers) GetBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key].(bool); ok {
		return val
	}
	return defaultValue
}

// GetStringSlice safely retrieves string slice from config
func (ch *ConfigHelpers) GetStringSlice(config map[string]interface{}, key string) []string {
	if val, ok := config[key].([]interface{}); ok {
		var result []string
		for _, v := range val {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

// GetMap safely retrieves map from config
func (ch *ConfigHelpers) GetMap(config map[string]interface{}, key string) map[string]interface{} {
	if val, ok := config[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

// RequireString returns string or error if not found
func (ch *ConfigHelpers) RequireString(config map[string]interface{}, key string) (string, error) {
	if val, ok := config[key].(string); ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("required config key '%s' not found or empty", key)
}
