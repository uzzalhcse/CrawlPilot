package nodes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// Shared helper functions for all node types

// getStringParam extracts a string parameter with a default value
func getStringParam(params map[string]interface{}, key, defaultVal string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultVal
}

// getIntParam extracts an int parameter with a default value
func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	return defaultVal
}

// getBoolParam extracts a bool parameter with a default value
func getBoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultVal
}

// parseNodeFromMap converts a map to a Node struct
func parseNodeFromMap(nodeMap map[string]interface{}) models.Node {
	node := models.Node{}

	if id, ok := nodeMap["id"].(string); ok {
		node.ID = id
	}
	if nodeType, ok := nodeMap["type"].(string); ok {
		node.Type = nodeType
	}
	if name, ok := nodeMap["name"].(string); ok {
		node.Name = name
	}
	if params, ok := nodeMap["params"].(map[string]interface{}); ok {
		node.Params = params
	}

	return node
}

// interpolateParams replaces {{variable}} placeholders in node params with actual values
// from the execution context variables. Supports simple expressions like {{loop_index + 1}}.
func interpolateParams(params map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	if params == nil || variables == nil {
		return params
	}

	result := make(map[string]interface{})
	for key, value := range params {
		result[key] = interpolateValue(value, variables)
	}
	return result
}

// interpolateValue recursively interpolates a single value
func interpolateValue(value interface{}, variables map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return interpolateString(v, variables)
	case map[string]interface{}:
		return interpolateParams(v, variables)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = interpolateValue(item, variables)
		}
		return result
	default:
		return value
	}
}

// interpolateString replaces {{variable}} patterns in a string
func interpolateString(s string, variables map[string]interface{}) string {
	// Match {{...}} patterns
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract expression inside {{...}}
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Try to evaluate the expression
		result := evaluateExpression(expr, variables)
		return result
	})
}

// evaluateExpression evaluates expressions with support for:
// - Simple variables: {{loop_index}}, {{current_url}}, {{task.marker}}
// - Arithmetic: {{loop_index + 1}}, {{count - 1}}, {{i * 2}}, {{total / 2}}, {{i % 3}}
// - Defaults: {{variable || "default"}}, {{count || 0}}
// - String concat: {{prefix ~ "_" ~ suffix}}
func evaluateExpression(expr string, variables map[string]interface{}) string {
	expr = strings.TrimSpace(expr)

	// Handle default values: "variable || default"
	if strings.Contains(expr, "||") {
		parts := strings.SplitN(expr, "||", 2)
		varExpr := strings.TrimSpace(parts[0])
		defaultVal := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		result := evaluateExpression(varExpr, variables)
		if !strings.HasPrefix(result, "{{") {
			return result
		}
		return defaultVal
	}

	// Handle string concatenation: "a ~ b ~ c"
	if strings.Contains(expr, "~") {
		parts := strings.Split(expr, "~")
		var result strings.Builder
		for _, part := range parts {
			partVal := evaluateExpression(strings.TrimSpace(part), variables)
			// Strip quotes from literal strings
			partVal = strings.Trim(partVal, `"'`)
			result.WriteString(partVal)
		}
		return result.String()
	}

	// Handle arithmetic operations: +, -, *, /, %
	for _, op := range []string{"+", "-", "*", "/", "%"} {
		if idx := strings.LastIndex(expr, op); idx > 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+1:])

			leftVal := evaluateToNumber(left, variables)
			rightVal := evaluateToNumber(right, variables)

			if leftVal != nil && rightVal != nil {
				var result float64
				switch op {
				case "+":
					result = *leftVal + *rightVal
				case "-":
					result = *leftVal - *rightVal
				case "*":
					result = *leftVal * *rightVal
				case "/":
					if *rightVal != 0 {
						result = *leftVal / *rightVal
					}
				case "%":
					if *rightVal != 0 {
						result = float64(int(*leftVal) % int(*rightVal))
					}
				}

				// Return as int if it's a whole number
				if result == float64(int(result)) {
					return strconv.Itoa(int(result))
				}
				return fmt.Sprintf("%.2f", result)
			}
		}
	}

	// Handle nested variable access: "task.marker", "item.price"
	if strings.Contains(expr, ".") {
		parts := strings.Split(expr, ".")
		var current interface{} = variables
		for _, part := range parts {
			if m, ok := current.(map[string]interface{}); ok {
				if val, exists := m[part]; exists {
					current = val
				} else {
					return "{{" + expr + "}}"
				}
			} else {
				return "{{" + expr + "}}"
			}
		}
		return fmt.Sprintf("%v", current)
	}

	// Simple variable lookup
	if val, ok := variables[expr]; ok {
		return fmt.Sprintf("%v", val)
	}

	// Check if it's a literal number
	if _, err := strconv.ParseFloat(expr, 64); err == nil {
		return expr
	}

	// Check if it's a quoted string literal
	if (strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`)) ||
		(strings.HasPrefix(expr, `'`) && strings.HasSuffix(expr, `'`)) {
		return strings.Trim(expr, `"'`)
	}

	// Return original expression if not found
	return "{{" + expr + "}}"
}

// evaluateToNumber tries to convert an expression to a number
func evaluateToNumber(expr string, variables map[string]interface{}) *float64 {
	expr = strings.TrimSpace(expr)

	// Direct number
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return &num
	}

	// Variable lookup
	if val, ok := variables[expr]; ok {
		switch v := val.(type) {
		case int:
			num := float64(v)
			return &num
		case int64:
			num := float64(v)
			return &num
		case float64:
			return &v
		case string:
			if num, err := strconv.ParseFloat(v, 64); err == nil {
				return &num
			}
		}
	}

	// Nested variable
	if strings.Contains(expr, ".") {
		result := evaluateExpression(expr, variables)
		if num, err := strconv.ParseFloat(result, 64); err == nil {
			return &num
		}
	}

	return nil
}

// toInt converts various types to int
func toInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}

// deepCopyMap creates a deep copy of a map to avoid modifying the original
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = deepCopyMap(v)
		case []interface{}:
			result[key] = deepCopySlice(v)
		default:
			result[key] = value
		}
	}
	return result
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, value := range s {
		switch v := value.(type) {
		case map[string]interface{}:
			result[i] = deepCopyMap(v)
		case []interface{}:
			result[i] = deepCopySlice(v)
		default:
			result[i] = value
		}
	}
	return result
}
