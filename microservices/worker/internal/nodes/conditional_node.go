package nodes

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/driver"
	"go.uber.org/zap"
)

// ConditionalNode executes different branches based on conditions
type ConditionalNode struct{}

func NewConditionalNode() NodeExecutor {
	return &ConditionalNode{}
}

func (n *ConditionalNode) Type() string {
	return "conditional"
}

func (n *ConditionalNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Get condition type: "exists", "not_exists", "contains", "equals", "matches"
	condition := getStringParam(node.Params, "condition", "exists")
	selector := getStringParam(node.Params, "selector", "")
	value := getStringParam(node.Params, "value", "")

	logger.Info("Evaluating condition",
		zap.String("condition", condition),
		zap.String("selector", selector),
	)

	// Evaluate the condition
	result, err := n.evaluateCondition(execCtx.Page, condition, selector, value)
	if err != nil {
		return fmt.Errorf("condition evaluation failed: %w", err)
	}

	logger.Info("Condition result", zap.Bool("result", result))

	// Get the nodes to execute based on result
	var nodesToExecute []interface{}
	if result {
		if thenNodes, ok := node.Params["then"].([]interface{}); ok {
			nodesToExecute = thenNodes
		}
	} else {
		if elseNodes, ok := node.Params["else"].([]interface{}); ok {
			nodesToExecute = elseNodes
		}
	}

	// Execute the branch nodes
	if len(nodesToExecute) > 0 {
		for _, nodeData := range nodesToExecute {
			nodeMap, ok := nodeData.(map[string]interface{})
			if !ok {
				continue
			}

			childNode := parseNodeFromMap(nodeMap)
			if childNode.Type == "" {
				continue
			}

			// Execute through the registry (need to pass registry reference)
			// For now, store the nodes to execute in context
			execCtx.BranchNodes = append(execCtx.BranchNodes, childNode)
		}
	}

	return nil
}

// evaluateCondition evaluates the specified condition
func (n *ConditionalNode) evaluateCondition(page driver.Page, condition, selector, value string) (bool, error) {
	switch condition {
	case "exists":
		return n.elementExists(page, selector)

	case "not_exists":
		exists, err := n.elementExists(page, selector)
		return !exists, err

	case "visible":
		return n.elementVisible(page, selector)

	case "contains":
		text, err := n.getElementText(page, selector)
		if err != nil {
			return false, err
		}
		return strings.Contains(strings.ToLower(text), strings.ToLower(value)), nil

	case "equals":
		text, err := n.getElementText(page, selector)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(text) == value, nil

	case "matches":
		text, err := n.getElementText(page, selector)
		if err != nil {
			return false, err
		}
		re, err := regexp.Compile(value)
		if err != nil {
			return false, fmt.Errorf("invalid regex: %w", err)
		}
		return re.MatchString(text), nil

	case "count_gt":
		count, err := n.elementCount(page, selector)
		if err != nil {
			return false, err
		}
		threshold := 0
		fmt.Sscanf(value, "%d", &threshold)
		return count > threshold, nil

	case "count_lt":
		count, err := n.elementCount(page, selector)
		if err != nil {
			return false, err
		}
		threshold := 0
		fmt.Sscanf(value, "%d", &threshold)
		return count < threshold, nil

	default:
		return false, fmt.Errorf("unknown condition type: %s", condition)
	}
}

func (n *ConditionalNode) elementExists(page driver.Page, selector string) (bool, error) {
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return false, err
	}
	return len(elements) > 0, nil
}

func (n *ConditionalNode) elementVisible(page driver.Page, selector string) (bool, error) {
	// Check visibility using WaitForSelector with immediate timeout?
	// Or Evaluate?
	// Let's use WaitForSelector with short timeout and Visible option
	err := page.WaitForSelector(selector,
		driver.WithWaitTimeout(100*time.Millisecond),
		driver.WithState("visible"),
	)
	return err == nil, nil
}

func (n *ConditionalNode) elementCount(page driver.Page, selector string) (int, error) {
	elements, err := page.QuerySelectorAll(selector)
	if err != nil {
		return 0, err
	}
	return len(elements), nil
}

func (n *ConditionalNode) getElementText(page driver.Page, selector string) (string, error) {
	element, err := page.QuerySelector(selector)
	if err != nil {
		return "", err
	}
	if element == nil {
		return "", fmt.Errorf("element not found: %s", selector)
	}
	return element.Text()
}
