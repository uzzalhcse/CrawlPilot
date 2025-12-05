package nodes

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
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

			childNode := n.parseNode(nodeMap)
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
func (n *ConditionalNode) evaluateCondition(page playwright.Page, condition, selector, value string) (bool, error) {
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

func (n *ConditionalNode) elementExists(page playwright.Page, selector string) (bool, error) {
	count, err := page.Locator(selector).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (n *ConditionalNode) elementVisible(page playwright.Page, selector string) (bool, error) {
	locator := page.Locator(selector).First()
	return locator.IsVisible()
}

func (n *ConditionalNode) elementCount(page playwright.Page, selector string) (int, error) {
	return page.Locator(selector).Count()
}

func (n *ConditionalNode) getElementText(page playwright.Page, selector string) (string, error) {
	locator := page.Locator(selector).First()
	return locator.TextContent()
}

// parseNode converts a map to a Node struct
func (n *ConditionalNode) parseNode(nodeMap map[string]interface{}) models.Node {
	node := models.Node{}

	if id, ok := nodeMap["id"].(string); ok {
		node.ID = id
	}
	if nodeType, ok := nodeMap["type"].(string); ok {
		node.Type = nodeType
	}
	if params, ok := nodeMap["params"].(map[string]interface{}); ok {
		node.Params = params
	}

	return node
}
