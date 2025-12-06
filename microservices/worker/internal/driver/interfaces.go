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

// Options

type PageOptions struct {
	Timeout   time.Duration
	WaitUntil string // "load", "domcontentloaded", "networkidle"
}

type PageOption func(*PageOptions)

type ElementOption func(*ElementOptions)

type WaitOption func(*WaitOptions)

type ScreenshotOption func(*ScreenshotOptions)
