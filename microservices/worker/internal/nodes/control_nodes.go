package nodes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// WaitForNode waits for specific conditions (selector, text, network idle)
type WaitForNode struct{}

func NewWaitForNode() NodeExecutor {
	return &WaitForNode{}
}

func (n *WaitForNode) Type() string {
	return "wait_for"
}

func (n *WaitForNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	condition := getStringParam(node.Params, "condition", "selector")
	timeout := getIntParam(node.Params, "timeout", 30000)

	logger.Info("Waiting for condition",
		zap.String("condition", condition),
		zap.Int("timeout", timeout),
	)

	switch condition {
	case "selector":
		selector := getStringParam(node.Params, "selector", "")
		if selector == "" {
			return fmt.Errorf("selector is required for wait_for with condition=selector")
		}

		state := "visible"
		if stateStr := getStringParam(node.Params, "state", "visible"); stateStr != "" {
			state = stateStr
		}

		err := execCtx.Page.WaitForSelector(selector,
			driver.WithState(state),
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		)
		return err

	case "text":
		text := getStringParam(node.Params, "text", "")
		if text == "" {
			return fmt.Errorf("text is required for wait_for with condition=text")
		}

		// Wait for text to appear on page
		err := execCtx.Page.WaitForFunction(fmt.Sprintf(`
			() => document.body.innerText.includes('%s')
		`, strings.ReplaceAll(text, "'", "\\'"))) // TODO: Add timeout option to WaitForFunction if needed, or rely on context/default
		return err

	case "network_idle":
		return execCtx.Page.WaitForState("networkidle",
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		)

	case "load":
		return execCtx.Page.WaitForState("load",
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		)

	case "domcontentloaded":
		return execCtx.Page.WaitForState("domcontentloaded",
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		)

	case "url":
		urlPattern := getStringParam(node.Params, "url", "")
		if urlPattern == "" {
			return fmt.Errorf("url is required for wait_for with condition=url")
		}

		return execCtx.Page.WaitForURL(urlPattern,
			driver.WithWaitTimeout(time.Duration(timeout)*time.Millisecond),
		)

	default:
		return fmt.Errorf("unknown wait_for condition: %s", condition)
	}
}

// InputNode fills input fields directly (unlike TypeNode which simulates keystrokes)
type InputNode struct{}

func NewInputNode() NodeExecutor {
	return &InputNode{}
}

func (n *InputNode) Type() string {
	return "input"
}

func (n *InputNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for input node")
	}

	value, ok := node.Params["value"].(string)
	if !ok {
		return fmt.Errorf("value is required for input node")
	}

	logger.Info("Filling input",
		zap.String("selector", selector),
	)

	// Wait for element
	if err := execCtx.Page.WaitForSelector(selector,
		driver.WithState("visible"),
	); err != nil {
		return fmt.Errorf("input element not found: %w", err)
	}

	// Fill directly (faster than Type, sets value instantly)
	if err := execCtx.Page.Fill(selector, value); err != nil {
		return fmt.Errorf("fill failed: %w", err)
	}

	return nil
}

// LoopNode iterates over elements and executes child nodes for each
type LoopNode struct{}

func NewLoopNode() NodeExecutor {
	return &LoopNode{}
}

func (n *LoopNode) Type() string {
	return "loop"
}

func (n *LoopNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector, ok := node.Params["selector"].(string)
	if !ok || selector == "" {
		return fmt.Errorf("selector is required for loop node")
	}

	maxIterations := getIntParam(node.Params, "max_iterations", 100)

	logger.Info("Starting loop",
		zap.String("selector", selector),
		zap.Int("max_iterations", maxIterations),
	)

	// Get all matching elements
	elements, err := execCtx.Page.QuerySelectorAll(selector)
	if err != nil {
		return fmt.Errorf("failed to find elements: %w", err)
	}

	count := len(elements)

	if count == 0 {
		logger.Info("No elements found for loop")
		return nil
	}

	// Limit iterations
	if count > maxIterations {
		count = maxIterations
	}

	logger.Info("Loop will iterate",
		zap.Int("count", count),
	)

	// Get child nodes to execute for each element
	childNodes, ok := node.Params["nodes"].([]interface{})
	if !ok || len(childNodes) == 0 {
		logger.Warn("No child nodes specified for loop")
		return nil
	}

	// Iterate over elements
	for i := 0; i < count; i++ {
		// Set current index in variables for child nodes
		execCtx.Variables["loop_index"] = i

		// Get nth element handle
		element := elements[i]

		// Store element for child nodes to use
		execCtx.Variables["loop_element"] = element

		// Parse and queue child nodes for execution
		for _, childNodeData := range childNodes {
			childNodeMap, ok := childNodeData.(map[string]interface{})
			if !ok {
				continue
			}

			childNode := parseNodeFromMap(childNodeMap)
			if childNode.Type == "" {
				continue
			}

			// Add to branch nodes for execution
			execCtx.BranchNodes = append(execCtx.BranchNodes, childNode)
		}
	}

	return nil
}

// InfiniteScrollNode scrolls to load all content (lazy loading)
type InfiniteScrollNode struct{}

func NewInfiniteScrollNode() NodeExecutor {
	return &InfiniteScrollNode{}
}

func (n *InfiniteScrollNode) Type() string {
	return "infinite_scroll"
}

func (n *InfiniteScrollNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	maxScrolls := getIntParam(node.Params, "max_scrolls", 10)
	waitBetweenScrolls := getIntParam(node.Params, "wait_between", 1000)
	endSelector := getStringParam(node.Params, "end_selector", "")

	logger.Info("Starting infinite scroll",
		zap.Int("max_scrolls", maxScrolls),
	)

	var previousHeight int64 = 0

	for i := 0; i < maxScrolls; i++ {
		// Check for end marker
		if endSelector != "" {
			elements, _ := execCtx.Page.QuerySelectorAll(endSelector)
			if len(elements) > 0 {
				logger.Info("End selector found, stopping scroll", zap.Int("iteration", i))
				break
			}
		}

		// Scroll to bottom FIRST
		_, err := execCtx.Page.Evaluate("window.scrollTo(0, document.body.scrollHeight)")
		if err != nil {
			return fmt.Errorf("scroll failed: %w", err)
		}

		logger.Debug("Scrolled to bottom", zap.Int("iteration", i))

		// Wait for new content to load
		time.Sleep(time.Duration(waitBetweenScrolls) * time.Millisecond)

		// Get new scroll height AFTER scrolling
		heightResult, err := execCtx.Page.Evaluate("document.body.scrollHeight")
		if err != nil {
			return fmt.Errorf("failed to get scroll height: %w", err)
		}

		currentHeight, ok := heightResult.(float64)
		if !ok {
			// Try int
			if ch, ok := heightResult.(int); ok {
				currentHeight = float64(ch)
			} else {
				continue
			}
		}

		// Check if height changed (new content loaded)
		if int64(currentHeight) == previousHeight && i > 0 {
			logger.Info("No more content to load, stopping scroll", zap.Int("iteration", i))
			break
		}
		previousHeight = int64(currentHeight)
	}

	logger.Info("Infinite scroll complete", zap.Int("max_scrolls", maxScrolls))
	return nil
}
