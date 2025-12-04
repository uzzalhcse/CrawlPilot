package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// NavigateNode handles page navigation
type NavigateNode struct{}

// NewNavigateNode creates a new navigate node executor
func NewNavigateNode() NodeExecutor {
	return &NavigateNode{}
}

func (n *NavigateNode) Type() string {
	return "navigate"
}

func (n *NavigateNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	url := execCtx.Task.URL

	// Get timeout from params or use default
	timeout := 60000.0 // 60 seconds
	if timeoutVal, ok := node.Params["timeout"].(float64); ok {
		timeout = timeoutVal
	}

	logger.Info("Navigating to URL",
		zap.String("url", url),
		zap.Float64("timeout", timeout),
	)

	// Navigate to URL
	response, err := execCtx.Page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(timeout),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})

	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	if response != nil {
		logger.Info("Navigation complete",
			zap.String("url", url),
			zap.Int("status", response.Status()),
		)
	}

	// Optional: Wait for specific selector
	if waitSelector, ok := node.Params["wait_selector"].(string); ok && waitSelector != "" {
		logger.Debug("Waiting for selector", zap.String("selector", waitSelector))

		if _, err := execCtx.Page.WaitForSelector(waitSelector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(timeout),
		}); err != nil {
			return fmt.Errorf("wait for selector failed: %w", err)
		}
	}

	return nil
}

// ClickNode handles element clicks
type ClickNode struct{}

func NewClickNode() NodeExecutor {
	return &ClickNode{}
}

func (n *ClickNode) Type() string {
	return "click"
}

func (n *ClickNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for click node")
	}

	logger.Info("Clicking element", zap.String("selector", selector))

	// Wait for element to be visible
	if _, err := execCtx.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	// Click element
	if err := execCtx.Page.Click(selector); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	// Optional: Wait after click
	if waitAfter, ok := node.Params["wait_after"].(float64); ok && waitAfter > 0 {
		time.Sleep(time.Duration(waitAfter) * time.Millisecond)
	}

	return nil
}

// TypeNode handles text input
type TypeNode struct{}

func NewTypeNode() NodeExecutor {
	return &TypeNode{}
}

func (n *TypeNode) Type() string {
	return "type"
}

func (n *TypeNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for type node")
	}

	text, ok := node.Params["text"].(string)
	if !ok {
		return fmt.Errorf("text is required for type node")
	}

	logger.Info("Typing text",
		zap.String("selector", selector),
		zap.Int("text_length", len(text)),
	)

	// Wait for input field
	if _, err := execCtx.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("input field not found: %w", err)
	}

	// Clear existing text if specified
	if clear, ok := node.Params["clear"].(bool); ok && clear {
		if err := execCtx.Page.Fill(selector, ""); err != nil {
			return fmt.Errorf("failed to clear field: %w", err)
		}
	}

	// Type text
	if err := execCtx.Page.Type(selector, text); err != nil {
		return fmt.Errorf("typing failed: %w", err)
	}

	return nil
}

// WaitNode handles explicit waits
type WaitNode struct{}

func NewWaitNode() NodeExecutor {
	return &WaitNode{}
}

func (n *WaitNode) Type() string {
	return "wait"
}

func (n *WaitNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Wait by duration
	if duration, ok := node.Params["duration"].(float64); ok && duration > 0 {
		logger.Info("Waiting", zap.Float64("duration_ms", duration))
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil
	}

	// Wait for selector
	if selector, ok := node.Params["selector"].(string); ok && selector != "" {
		timeout := 30000.0
		if t, ok := node.Params["timeout"].(float64); ok {
			timeout = t
		}

		logger.Info("Waiting for selector",
			zap.String("selector", selector),
			zap.Float64("timeout", timeout),
		)

		if _, err := execCtx.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(timeout),
		}); err != nil {
			return fmt.Errorf("wait for selector failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("either duration or selector must be specified for wait node")
}
