package driver

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	// ErrNotSupported is returned when a driver doesn't support an operation
	ErrNotSupported = errors.New("operation not supported by this driver")
	// ErrElementNotFound is returned when a selector finds no elements
	ErrElementNotFound = errors.New("element not found")
)

// Driver defines the capability to create pages/contexts
type Driver interface {
	// NewPage creates a new page/context for execution
	NewPage(ctx context.Context) (Page, error)
	// Close cleans up driver resources
	Close() error
	// Name returns the driver name (e.g., "playwright", "http")
	Name() string
}

// Page defines the common interactions required by nodes
type Page interface {
	// Navigation
	Goto(url string, options ...PageOption) error

	// Content retrieval
	Content() (string, error)
	Title() (string, error)
	URL() (string, error)
	Screenshot(options ...ScreenshotOption) ([]byte, error)

	// Interaction
	Click(selector string, options ...ElementOption) error
	Type(selector, text string, options ...ElementOption) error
	Fill(selector, text string, options ...ElementOption) error
	Hover(selector string, options ...ElementOption) error
	WaitForSelector(selector string, options ...WaitOption) error
	WaitForURL(url string, options ...WaitOption) error
	WaitForState(state string, options ...WaitOption) error
	WaitForFunction(expression string, args ...interface{}) error

	// Script Execution
	Evaluate(expression string, args ...interface{}) (interface{}, error)
	AddInitScript(script string) error

	// Extraction
	QuerySelector(selector string) (Element, error)
	QuerySelectorAll(selector string) ([]Element, error)

	// State Management
	GetCookies() ([]*http.Cookie, error)
	SetCookies(cookies []*http.Cookie) error

	// Lifecycle
	Close() error

	// Driver identification
	DriverName() string
}

// Element defines interaction with DOM elements
type Element interface {
	// Data retrieval
	Text() (string, error)
	Attribute(name string) (string, error)
	InnerHTML() (string, error)
	Screenshot(options ...ScreenshotOption) ([]byte, error)

	// Interaction (scoped to element)
	Click() error
	Type(text string) error
	Fill(text string) error
	Hover() error

	// Traversal
	QuerySelector(selector string) (Element, error)
	QuerySelectorAll(selector string) ([]Element, error)
}

// CaptchaSolver is an optional interface that Page implementations can support
// for automatic CAPTCHA detection and solving
type CaptchaSolver interface {
	// SolveCaptcha attempts to detect and solve CAPTCHA on the current page
	// Returns (solved bool, err error) where solved indicates if a CAPTCHA was found and solved
	SolveCaptcha(opts CaptchaSolveOptions) (bool, error)

	// SupportsCaptchaSolving returns true if this driver supports CAPTCHA solving
	SupportsCaptchaSolving() bool
}

// CaptchaSolveOptions configures CAPTCHA solving behavior
type CaptchaSolveOptions struct {
	// CaptchaType is the CAPTCHA provider (e.g., "cloudflare")
	CaptchaType string
	// ChallengeType is the specific challenge type (e.g., "interstitial", "turnstile")
	ChallengeType string
	// ExpectedContentSelector is a CSS selector to verify page content is accessible after solving
	ExpectedContentSelector string
	// SolveAttempts is the maximum number of solve attempts (default: 3)
	SolveAttempts int
	// Debug enables debug logging
	Debug bool
	// Timeout is the maximum time to wait for solving
	Timeout time.Duration
}

// DefaultCaptchaSolveOptions returns sensible defaults for CAPTCHA solving
func DefaultCaptchaSolveOptions() CaptchaSolveOptions {
	return CaptchaSolveOptions{
		CaptchaType:   "cloudflare",
		ChallengeType: "interstitial",
		SolveAttempts: 3,
		Timeout:       60 * time.Second,
	}
}

// Options

type PageOptions struct {
	Timeout   time.Duration
	WaitUntil string // "load", "domcontentloaded", "networkidle"
}

type PageOption func(*PageOptions)

type ElementOption func(*ElementOptions)

type WaitOption func(*WaitOptions)

type ScreenshotOption func(*ScreenshotOptions)
