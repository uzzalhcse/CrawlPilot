package error_recovery

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

// ContextAwareRulesEngine matches errors to rules and executes actions
type ContextAwareRulesEngine struct {
	rules          []ContextAwareRule
	variableEngine *DynamicVariableEngine
}

// NewContextAwareRulesEngine creates a new rules engine
func NewContextAwareRulesEngine(rules []ContextAwareRule) *ContextAwareRulesEngine {
	return &ContextAwareRulesEngine{
		rules:          rules,
		variableEngine: NewDynamicVariableEngine(),
	}
}

// FindSolution finds a matching rule and returns the solution
func (e *ContextAwareRulesEngine) FindSolution(err error, ctx *ExecutionContext) (*Solution, error) {
	// Sort rules by priority
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority > e.rules[j].Priority
	})

	// Try each rule
	for _, rule := range e.rules {
		// Check if all conditions match
		if !e.evaluateConditions(rule.Conditions, err, ctx) {
			continue
		}

		// Check domain pattern
		if !e.matchesDomain(rule.Context.DomainPattern, ctx.Domain) {
			continue
		}

		logger.Info("Rule matched",
			zap.String("rule", rule.Name),
			zap.Float64("confidence", rule.Confidence))

		// Resolve dynamic variables
		resolvedVariables := e.variableEngine.Resolve(rule.Context.Variables, ctx)

		// Execute actions with variable substitution
		actions := e.substituteVariables(rule.Actions, resolvedVariables)

		return &Solution{
			RuleName:   rule.Name,
			Actions:    actions,
			Confidence: rule.Confidence,
			Context:    resolvedVariables,
			Type:       "rule",
		}, nil
	}

	return nil, fmt.Errorf("no rule matched")
}

// evaluateConditions checks if all conditions match
func (e *ContextAwareRulesEngine) evaluateConditions(conditions []Condition, err error, ctx *ExecutionContext) bool {
	for _, cond := range conditions {
		value := e.extractFieldValue(cond.Field, err, ctx)

		if !e.evaluateOperator(cond.Operator, value, cond.Value) {
			return false
		}
	}
	return true
}

// extractFieldValue extracts a field value from error or context
func (e *ContextAwareRulesEngine) extractFieldValue(field string, err error, ctx *ExecutionContext) interface{} {
	switch field {
	case "error_type":
		if err != nil {
			return err.Error()
		}
		return ctx.Error.Type
	case "status_code":
		return ctx.Response.StatusCode
	case "domain":
		return ctx.Domain
	case "response_body":
		return ctx.Response.Body
	case "response_headers":
		return ctx.Response.Header
	default:
		return nil
	}
}

// evaluateOperator evaluates a condition operator
func (e *ContextAwareRulesEngine) evaluateOperator(operator string, actual interface{}, expected interface{}) bool {
	switch operator {
	case "equals":
		// Handle type coercion for numeric comparisons (e.g., status_code)
		// This is needed because frontend may save status codes as strings
		if actualInt, ok := actual.(int); ok {
			// Actual is int, try to convert expected
			if expectedInt, ok := expected.(int); ok {
				return actualInt == expectedInt
			}
			// Try string to int conversion
			if expectedStr, ok := expected.(string); ok {
				if convertedInt, err := strconv.Atoi(expectedStr); err == nil {
					return actualInt == convertedInt
				}
			}
			// Try float64 to int conversion (JSON unmarshaling)
			if expectedFloat, ok := expected.(float64); ok {
				return actualInt == int(expectedFloat)
			}
		}
		// Fallback to direct comparison
		return actual == expected

	case "contains":
		actualStr, ok1 := actual.(string)
		expectedStr, ok2 := expected.(string)
		if ok1 && ok2 {
			return strings.Contains(actualStr, expectedStr)
		}
		return false

	case "regex":
		actualStr, ok1 := actual.(string)
		pattern, ok2 := expected.(string)
		if ok1 && ok2 {
			matched, err := regexp.MatchString(pattern, actualStr)
			return err == nil && matched
		}
		return false

	case "gt":
		actualInt, ok1 := actual.(int)
		expectedInt, ok2 := expected.(int)
		if ok1 && ok2 {
			return actualInt > expectedInt
		}
		return false

	case "lt":
		actualInt, ok1 := actual.(int)
		expectedInt, ok2 := expected.(int)
		if ok1 && ok2 {
			return actualInt < expectedInt
		}
		return false

	default:
		return false
	}
}

// matchesDomain checks if a domain matches a pattern
func (e *ContextAwareRulesEngine) matchesDomain(pattern, domain string) bool {
	if pattern == "*" {
		return true
	}

	// Try exact match
	if pattern == domain {
		return true
	}

	// Try glob pattern match
	matched, _ := filepath.Match(pattern, domain)
	return matched
}

// substituteVariables replaces {{variable}} placeholders in actions
func (e *ContextAwareRulesEngine) substituteVariables(actions []Action, variables map[string]interface{}) []Action {
	result := make([]Action, len(actions))

	for i, action := range actions {
		newAction := Action{
			Type:       action.Type,
			Parameters: make(map[string]interface{}),
			Condition:  action.Condition,
		}

		// Substitute variables in parameters
		for key, value := range action.Parameters {
			if strValue, ok := value.(string); ok {
				// Check if it's a variable placeholder
				if strings.HasPrefix(strValue, "{{") && strings.HasSuffix(strValue, "}}") {
					varName := strings.TrimSuffix(strings.TrimPrefix(strValue, "{{"), "}}")
					if resolvedValue, exists := variables[varName]; exists {
						newAction.Parameters[key] = resolvedValue
					} else {
						newAction.Parameters[key] = value
					}
				} else {
					newAction.Parameters[key] = value
				}
			} else {
				newAction.Parameters[key] = value
			}
		}

		result[i] = newAction
	}

	return result
}

// AddRule adds a new rule to the engine
func (e *ContextAwareRulesEngine) AddRule(rule ContextAwareRule) {
	e.rules = append(e.rules, rule)
}

// UpdateRule updates an existing rule
func (e *ContextAwareRulesEngine) UpdateRule(rule ContextAwareRule) error {
	for i, r := range e.rules {
		if r.ID == rule.ID {
			e.rules[i] = rule
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}

// RemoveRule removes a rule by ID
func (e *ContextAwareRulesEngine) RemoveRule(id string) error {
	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule not found")
}
