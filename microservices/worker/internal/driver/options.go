package driver

import "time"

// Default timeouts
const (
	DefaultTimeout = 30 * time.Second
)

// Page Options

func WithPageTimeout(d time.Duration) PageOption {
	return func(o *PageOptions) {
		o.Timeout = d
	}
}

func WithWaitUntil(state string) PageOption {
	return func(o *PageOptions) {
		o.WaitUntil = state
	}
}

// Element Options

func WithElementTimeout(d time.Duration) ElementOption {
	return func(o *ElementOptions) {
		o.Timeout = d
	}
}

func WithVisible(visible bool) ElementOption {
	return func(o *ElementOptions) {
		o.Visible = visible
	}
}

// Wait Options

func WithWaitTimeout(d time.Duration) WaitOption {
	return func(o *WaitOptions) {
		o.Timeout = d
	}
}

func WithState(state string) WaitOption {
	return func(o *WaitOptions) {
		o.State = state
	}
}

type ElementOptions struct {
	Timeout time.Duration
	Delay   time.Duration
	Visible bool
}

type WaitOptions struct {
	Timeout   time.Duration
	State     string
	WaitUntil string
}

type ScreenshotOptions struct {
	FullPage bool
	Type     string
	Quality  int
}

// Screenshot Options

func WithFullPage(fullPage bool) ScreenshotOption {
	return func(o *ScreenshotOptions) {
		o.FullPage = fullPage
	}
}

func WithScreenshotType(t string) ScreenshotOption {
	return func(o *ScreenshotOptions) {
		o.Type = t
	}
}

func WithQuality(quality int) ScreenshotOption {
	return func(o *ScreenshotOptions) {
		o.Quality = quality
	}
}
