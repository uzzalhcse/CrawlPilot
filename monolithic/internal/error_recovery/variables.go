package error_recovery

import (
	"path/filepath"
	"strconv"
	"strings"
)

// DynamicVariableEngine resolves context-aware variables at runtime
type DynamicVariableEngine struct{}

// DynamicVariable represents a variable that is computed at runtime
type DynamicVariable struct {
	Type  string                 `json:"type"` // "conditional", "calculated", "domain_based"
	Logic map[string]interface{} `json:"logic"`
}

// NewDynamicVariableEngine creates a new variable engine
func NewDynamicVariableEngine() *DynamicVariableEngine {
	return &DynamicVariableEngine{}
}

// Resolve resolves all variables in the map
func (v *DynamicVariableEngine) Resolve(variables map[string]interface{}, ctx *ExecutionContext) map[string]interface{} {
	resolved := make(map[string]interface{})

	for key, value := range variables {
		switch val := value.(type) {
		case map[string]interface{}:
			// Try to parse as DynamicVariable
			if varType, ok := val["type"].(string); ok {
				if logic, ok := val["logic"].(map[string]interface{}); ok {
					dv := DynamicVariable{Type: varType, Logic: logic}
					resolved[key] = v.resolveDynamicVariable(dv, ctx)
					continue
				}
			}
			resolved[key] = value
		default:
			resolved[key] = value
		}
	}

	return resolved
}

// resolveDynamicVariable resolves a single dynamic variable
func (v *DynamicVariableEngine) resolveDynamicVariable(dv DynamicVariable, ctx *ExecutionContext) interface{} {
	switch dv.Type {
	case "conditional":
		return v.resolveConditional(dv.Logic, ctx)
	case "calculated":
		return v.resolveCalculated(dv.Logic, ctx)
	case "domain_based":
		return v.resolveDomainBased(dv.Logic, ctx)
	default:
		return nil
	}
}

// resolveConditional resolves conditional logic
func (v *DynamicVariableEngine) resolveConditional(logic map[string]interface{}, ctx *ExecutionContext) interface{} {
	responseBody := ctx.Response.Body

	if ifContains, ok := logic["if_contains"].(string); ok {
		if strings.Contains(responseBody, ifContains) {
			return logic["then"]
		}
	}

	if elseIfContains, ok := logic["else_if_contains"].(string); ok {
		if strings.Contains(responseBody, elseIfContains) {
			return logic["then"]
		}
	}

	return logic["else"]
}

// resolveCalculated resolves calculated values
func (v *DynamicVariableEngine) resolveCalculated(logic map[string]interface{}, ctx *ExecutionContext) interface{} {
	// Example: Extract from response header
	if source, ok := logic["source"].(string); ok {
		parts := strings.Split(source, ":")
		if parts[0] == "response_header" && len(parts) > 1 {
			headerValue := ctx.Response.Header.Get(parts[1])
			if headerValue != "" {
				if parsed, err := strconv.Atoi(headerValue); err == nil {
					if multiplier, ok := logic["multiplier"].(float64); ok {
						return int(float64(parsed) * multiplier)
					}
					return parsed
				}
			}
		}
	}

	// Fallback
	if fallback, ok := logic["fallback"]; ok {
		return fallback
	}

	return nil
}

// resolveDomainBased resolves domain-based values
func (v *DynamicVariableEngine) resolveDomainBased(logic map[string]interface{}, ctx *ExecutionContext) interface{} {
	domain := ctx.Domain

	// Try exact match
	if value, ok := logic[domain]; ok {
		return value
	}

	// Try pattern match
	for pattern, value := range logic {
		if pattern == "default" {
			continue
		}

		if matched, _ := filepath.Match(pattern, domain); matched {
			return value
		}
	}

	// Default
	return logic["default"]
}
