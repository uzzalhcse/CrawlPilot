package nodes

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ScrollNode handles page scrolling
type ScrollNode struct{}

func NewScrollNode() NodeExecutor {
	return &ScrollNode{}
}

func (n *ScrollNode) Type() string {
	return "scroll"
}

func (n *ScrollNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Get scroll parameters
	x := getIntParam(node.Params, "x", 0)
	y := getIntParam(node.Params, "y", 0)

	// Support scrolling by pixels or to element
	if selector, ok := node.Params["selector"].(string); ok && selector != "" {
		// Scroll to element
		logger.Info("Scrolling to element", zap.String("selector", selector))

		_, err := execCtx.Page.Evaluate(fmt.Sprintf(`
			document.querySelector('%s')?.scrollIntoView({behavior: 'smooth', block: 'center'})
		`, selector))
		if err != nil {
			return fmt.Errorf("scroll to element failed: %w", err)
		}
	} else if y != 0 || x != 0 {
		// Scroll by pixels
		logger.Info("Scrolling by offset", zap.Int("x", x), zap.Int("y", y))

		_, err := execCtx.Page.Evaluate(fmt.Sprintf("window.scrollBy(%d, %d)", x, y))
		if err != nil {
			return fmt.Errorf("scroll by offset failed: %w", err)
		}
	} else if scrollToBottom, ok := node.Params["to_bottom"].(bool); ok && scrollToBottom {
		// Scroll to bottom of page
		logger.Info("Scrolling to bottom")

		_, err := execCtx.Page.Evaluate("window.scrollTo(0, document.body.scrollHeight)")
		if err != nil {
			return fmt.Errorf("scroll to bottom failed: %w", err)
		}
	}

	// Wait after scroll if specified
	if waitAfter := getIntParam(node.Params, "wait_after", 500); waitAfter > 0 {
		time.Sleep(time.Duration(waitAfter) * time.Millisecond)
	}

	return nil
}

// HoverNode handles element hover actions
type HoverNode struct{}

func NewHoverNode() NodeExecutor {
	return &HoverNode{}
}

func (n *HoverNode) Type() string {
	return "hover"
}

func (n *HoverNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for hover node")
	}

	logger.Info("Hovering over element", zap.String("selector", selector))

	// Wait for element to be visible
	if _, err := execCtx.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	// Hover over element
	if err := execCtx.Page.Hover(selector); err != nil {
		return fmt.Errorf("hover failed: %w", err)
	}

	// Wait after hover if specified (for dropdown menus to appear)
	if waitAfter := getIntParam(node.Params, "wait_after", 300); waitAfter > 0 {
		time.Sleep(time.Duration(waitAfter) * time.Millisecond)
	}

	return nil
}

// ScreenshotNode captures page or element screenshots
type ScreenshotNode struct{}

func NewScreenshotNode() NodeExecutor {
	return &ScreenshotNode{}
}

func (n *ScreenshotNode) Type() string {
	return "screenshot"
}

func (n *ScreenshotNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Get screenshot options
	fullPage := getBoolParam(node.Params, "full_page", false)
	selector, hasSelector := node.Params["selector"].(string)

	var screenshotData []byte
	var err error

	if hasSelector && selector != "" {
		// Screenshot specific element
		logger.Info("Taking element screenshot", zap.String("selector", selector))

		element, err := execCtx.Page.QuerySelector(selector)
		if err != nil || element == nil {
			return fmt.Errorf("element not found for screenshot: %w", err)
		}

		screenshotData, err = element.Screenshot(playwright.ElementHandleScreenshotOptions{
			Type: playwright.ScreenshotTypePng,
		})
	} else {
		// Screenshot full page or viewport
		logger.Info("Taking page screenshot", zap.Bool("full_page", fullPage))

		screenshotData, err = execCtx.Page.Screenshot(playwright.PageScreenshotOptions{
			FullPage: playwright.Bool(fullPage),
			Type:     playwright.ScreenshotTypePng,
		})
	}

	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	// Store screenshot in execution context for later use
	screenshotBase64 := base64.StdEncoding.EncodeToString(screenshotData)

	// Save to disk if save_to_disk is specified
	if savePath := getStringParam(node.Params, "save_to_disk", ""); savePath != "" {
		// Expand path with task details
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("screenshot_%s_%s.png", execCtx.Task.TaskID, timestamp)
		fullPath := filepath.Join(savePath, filename)

		// Ensure directory exists
		if err := os.MkdirAll(savePath, 0755); err != nil {
			logger.Warn("Failed to create screenshot directory", zap.Error(err))
		} else {
			if err := os.WriteFile(fullPath, screenshotData, 0644); err != nil {
				logger.Warn("Failed to save screenshot to disk", zap.Error(err))
			} else {
				logger.Info("Screenshot saved to disk", zap.String("path", fullPath))
			}
		}
	}

	// Add to extracted items if save_as_item is true
	if saveAsItem := getBoolParam(node.Params, "save_as_item", false); saveAsItem {
		execCtx.ExtractedItems = append(execCtx.ExtractedItems, map[string]interface{}{
			"type":      "screenshot",
			"url":       execCtx.Task.URL,
			"data":      screenshotBase64,
			"full_page": fullPage,
			"selector":  selector,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}

	logger.Info("Screenshot captured",
		zap.Int("size_bytes", len(screenshotData)),
	)

	return nil
}
