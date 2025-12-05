package browser

import (
	"crawer-agent/exp/v2/internal/dom"
	"errors"
	"fmt"
	"runtime"

	"github.com/kbinani/screenshot"
)

type BrowserConfig map[string]interface{}

func NewBrowserConfig() BrowserConfig {
	return BrowserConfig{
		"headless":         false,
		"disable_security": false,
		"is_mobile":        false,
		"has_touch":        false,
	}
}

type TabInfo struct {
	PageID       int
	URL          string
	Title        string
	ParentPageID *int
}

func (ti *TabInfo) String() string {
	if ti.ParentPageID == nil {
		return fmt.Sprintf("Tab(page_id=%d, url=%s, title=%s)", ti.PageID, ti.URL, ti.Title)
	}
	return fmt.Sprintf("Tab(page_id=%d, url=%s, title=%s, parent_page_id=%d)", ti.PageID, ti.URL, ti.Title, *ti.ParentPageID)
}

type BrowserState struct {
	URL           string
	Title         string
	Tabs          []*TabInfo
	Screenshot    *string
	PixelAbove    int
	PixelBelow    int
	BrowserErrors []string
	ElementTree   *dom.DOMElementNode
	SelectorMap   *dom.SelectorMap
}

type BrowserStateHistory struct {
	URL               string
	Title             string
	Tabs              []*TabInfo
	InteractedElement []*dom.DOMElementNode
}

type BrowserError struct {
	Message string
}

func (e *BrowserError) Error() string {
	return e.Message
}

type URLNotAllowedError struct {
	BrowserError
}

func NewURLNotAllowedError(url string) error {
	return &URLNotAllowedError{
		BrowserError: BrowserError{
			Message: "URL not allowed: " + url,
		},
	}
}

func getScreenResolution() map[string]int {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return map[string]int{"width": 1920, "height": 1080}
	}
	b := screenshot.GetDisplayBounds(0)
	return map[string]int{
		"width":  b.Dx(),
		"height": b.Dy(),
	}
}

func getWindowAdjustments() (int, int) {
	switch runtime.GOOS {
	case "darwin":
		return -4, 24
	case "windows":
		return -8, 0
	default:
		return 0, 0
	}
}

func ParseNumberToInt(value any) (int, error) {
	if value == nil {
		return 0, nil
	}
	if v, ok := value.(int); ok {
		return v, nil
	}
	if v, ok := value.(float64); ok {
		return int(v), nil
	}
	return 0, errors.New("value is not a number")
}
