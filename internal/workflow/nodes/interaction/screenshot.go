package interaction

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ScreenshotExecutor handles screenshot capturing
type ScreenshotExecutor struct {
	nodes.BaseNodeExecutor
}

// NewScreenshotExecutor creates a new screenshot executor
func NewScreenshotExecutor() *ScreenshotExecutor {
	return &ScreenshotExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ScreenshotExecutor) Type() models.NodeType {
	return models.NodeTypeScreenshot
}

// Validate validates the node parameters
func (e *ScreenshotExecutor) Validate(params map[string]interface{}) error {
	// Filename is optional - will auto-generate if not provided
	return nil
}

// Execute captures a screenshot
func (e *ScreenshotExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	selector := nodes.GetStringParam(input.Params, "selector")
	filename := nodes.GetStringParam(input.Params, "filename")
	path := nodes.GetStringParam(input.Params, "path")
	fullPage := nodes.GetBoolParam(input.Params, "full_page", true)

	// Generate default filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("screenshot_%s.png", timestamp)
	}

	// Ensure filename has .png extension
	if filepath.Ext(filename) == "" {
		filename = filename + ".png"
	}

	// Build full path
	var screenshotPath string
	if path != "" {
		screenshotPath = filepath.Join(path, filename)
	} else {
		// Default to screenshots directory relative to execution
		screenshotPath = filepath.Join("screenshots", filename)
	}

	// Screenshot options
	options := playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(fullPage),
	}

	var screenshotBytes []byte
	var err error

	if selector != "" {
		// Screenshot specific element
		locator := input.BrowserContext.Page.Locator(selector)
		screenshotBytes, err = locator.Screenshot(playwright.LocatorScreenshotOptions{
			Path: playwright.String(screenshotPath),
		})
	} else {
		// Screenshot full page or viewport
		screenshotBytes, err = input.BrowserContext.Page.Screenshot(options)
	}

	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	result := map[string]interface{}{
		"screenshot_path": screenshotPath,
		"filename":        filename,
		"size_bytes":      len(screenshotBytes),
	}

	if selector != "" {
		result["selector"] = selector
	}

	return &nodes.ExecutionOutput{
		Result: result,
		Metadata: map[string]interface{}{
			"screenshot_captured": true,
		},
	}, nil
}
