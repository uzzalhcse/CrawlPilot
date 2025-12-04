package browser

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
)

// InteractionEngine handles browser interactions
type InteractionEngine struct {
	browserCtx *BrowserContext
}

func NewInteractionEngine(browserCtx *BrowserContext) *InteractionEngine {
	return &InteractionEngine{
		browserCtx: browserCtx,
	}
}

// Click clicks on an element
func (ie *InteractionEngine) Click(selector string, options ...playwright.PageClickOptions) error {
	var opts playwright.PageClickOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = playwright.PageClickOptions{
			Timeout: playwright.Float(30000),
		}
	}

	err := ie.browserCtx.Page.Click(selector, opts)
	if err != nil {
		return fmt.Errorf("failed to click on '%s': %w", selector, err)
	}
	return nil
}

// Type types text into an element
func (ie *InteractionEngine) Type(selector, text string, delay time.Duration) error {
	err := ie.browserCtx.Page.Fill(selector, "")
	if err != nil {
		return fmt.Errorf("failed to clear field '%s': %w", selector, err)
	}

	opts := playwright.PageTypeOptions{
		Timeout: playwright.Float(30000),
	}
	if delay > 0 {
		opts.Delay = playwright.Float(float64(delay.Milliseconds()))
	}

	err = ie.browserCtx.Page.Type(selector, text, opts)
	if err != nil {
		return fmt.Errorf("failed to type into '%s': %w", selector, err)
	}
	return nil
}

// Scroll scrolls the page
func (ie *InteractionEngine) Scroll(x, y int) error {
	_, err := ie.browserCtx.Page.Evaluate(fmt.Sprintf("window.scrollBy(%d, %d)", x, y))
	if err != nil {
		return fmt.Errorf("failed to scroll: %w", err)
	}
	return nil
}

// ScrollToElement scrolls to an element
func (ie *InteractionEngine) ScrollToElement(selector string) error {
	_, err := ie.browserCtx.Page.Evaluate(fmt.Sprintf(`
		document.querySelector('%s').scrollIntoView({behavior: 'smooth', block: 'center'})
	`, selector))
	if err != nil {
		return fmt.Errorf("failed to scroll to element '%s': %w", selector, err)
	}
	return nil
}

// Hover hovers over an element
func (ie *InteractionEngine) Hover(selector string) error {
	err := ie.browserCtx.Page.Hover(selector, playwright.PageHoverOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("failed to hover over '%s': %w", selector, err)
	}
	return nil
}

// Wait waits for a specified duration
func (ie *InteractionEngine) Wait(duration time.Duration) error {
	time.Sleep(duration)
	return nil
}

// WaitForSelector waits for a selector to appear
func (ie *InteractionEngine) WaitForSelector(selector string, timeout time.Duration, state string) error {
	var waitState *playwright.WaitForSelectorState
	switch state {
	case "visible":
		waitState = playwright.WaitForSelectorStateVisible
	case "hidden":
		waitState = playwright.WaitForSelectorStateHidden
	case "attached":
		waitState = playwright.WaitForSelectorStateAttached
	default:
		waitState = playwright.WaitForSelectorStateVisible
	}

	_, err := ie.browserCtx.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(timeout.Milliseconds())),
		State:   waitState,
	})
	if err != nil {
		return fmt.Errorf("failed to wait for selector '%s': %w", selector, err)
	}
	return nil
}

// WaitForNavigation waits for navigation to complete
func (ie *InteractionEngine) WaitForNavigation(timeout time.Duration) error {
	err := ie.browserCtx.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(float64(timeout.Milliseconds())),
	})
	if err != nil {
		return fmt.Errorf("failed to wait for navigation: %w", err)
	}
	return nil
}

// Select selects an option from a dropdown
func (ie *InteractionEngine) Select(selector string, values ...string) error {
	_, err := ie.browserCtx.Page.SelectOption(selector, playwright.SelectOptionValues{
		Values: &values,
	})
	if err != nil {
		return fmt.Errorf("failed to select option in '%s': %w", selector, err)
	}
	return nil
}

// Check checks a checkbox
func (ie *InteractionEngine) Check(selector string) error {
	err := ie.browserCtx.Page.Check(selector, playwright.PageCheckOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("failed to check '%s': %w", selector, err)
	}
	return nil
}

// Uncheck unchecks a checkbox
func (ie *InteractionEngine) Uncheck(selector string) error {
	err := ie.browserCtx.Page.Uncheck(selector, playwright.PageUncheckOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("failed to uncheck '%s': %w", selector, err)
	}
	return nil
}

// Press presses a key
func (ie *InteractionEngine) Press(selector, key string) error {
	err := ie.browserCtx.Page.Press(selector, key, playwright.PagePressOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("failed to press key '%s' on '%s': %w", key, selector, err)
	}
	return nil
}

// ExecuteScript executes JavaScript in the page context
func (ie *InteractionEngine) ExecuteScript(script string) (interface{}, error) {
	result, err := ie.browserCtx.Page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}
	return result, nil
}

// GetAttribute gets an attribute value from an element
func (ie *InteractionEngine) GetAttribute(selector, attribute string) (string, error) {
	value, err := ie.browserCtx.Page.GetAttribute(selector, attribute, playwright.PageGetAttributeOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get attribute '%s' from '%s': %w", attribute, selector, err)
	}
	return value, nil
}

// GetText gets text content from an element
func (ie *InteractionEngine) GetText(selector string) (string, error) {
	text, err := ie.browserCtx.Page.TextContent(selector, playwright.PageTextContentOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get text from '%s': %w", selector, err)
	}
	return text, nil
}

// IsVisible checks if an element is visible
func (ie *InteractionEngine) IsVisible(selector string) (bool, error) {
	visible, err := ie.browserCtx.Page.IsVisible(selector, playwright.PageIsVisibleOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		return false, nil
	}
	return visible, nil
}

// IsEnabled checks if an element is enabled
func (ie *InteractionEngine) IsEnabled(selector string) (bool, error) {
	enabled, err := ie.browserCtx.Page.IsEnabled(selector, playwright.PageIsEnabledOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		return false, nil
	}
	return enabled, nil
}
