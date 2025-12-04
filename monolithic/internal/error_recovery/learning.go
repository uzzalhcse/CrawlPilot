package error_recovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// SolutionStats tracks the success of a solution
type SolutionStats struct {
	Solution     *Solution
	SuccessCount int
	FailCount    int
	Domains      map[string]int
	Contexts     []*ExecutionContext
}

// LearningEngine converts successful AI solutions into reusable rules
type LearningEngine struct {
	successTracker map[string]*SolutionStats
	minSuccessRate float64
	minusageCount  int
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine(minSuccessRate float64, minUsageCount int) *LearningEngine {
	return &LearningEngine{
		successTracker: make(map[string]*SolutionStats),
		minSuccessRate: minSuccessRate,
		minusageCount:  minUsageCount,
	}
}

// TrackSuccess records a successful solution
func (l *LearningEngine) TrackSuccess(solution *Solution, ctx *ExecutionContext) {
	fingerprint := l.generateFingerprint(solution)

	if _, exists := l.successTracker[fingerprint]; !exists {
		l.successTracker[fingerprint] = &SolutionStats{
			Solution:     solution,
			SuccessCount: 0,
			FailCount:    0,
			Domains:      make(map[string]int),
			Contexts:     []*ExecutionContext{},
		}
	}

	stats := l.successTracker[fingerprint]
	stats.SuccessCount++
	stats.Domains[ctx.Domain]++
	stats.Contexts = append(stats.Contexts, ctx)
}

// TrackFailure records a failed solution
func (l *LearningEngine) TrackFailure(solution *Solution) {
	fingerprint := l.generateFingerprint(solution)

	if stats, exists := l.successTracker[fingerprint]; exists {
		stats.FailCount++
	}
}

// ConvertToContextAwareRule converts a successful solution into a rule
func (l *LearningEngine) ConvertToContextAwareRule(solution *Solution, ctx *ExecutionContext) *ContextAwareRule {
	fingerprint := l.generateFingerprint(solution)
	stats, exists := l.successTracker[fingerprint]

	if !exists {
		// First time seeing this solution, track it
		l.TrackSuccess(solution, ctx)
		return nil
	}

	// Need at least minimum successful uses
	if stats.SuccessCount < l.minusageCount {
		return nil
	}

	successRate := float64(stats.SuccessCount) / float64(stats.SuccessCount+stats.FailCount)

	// Need minimum success rate
	if successRate < l.minSuccessRate {
		return nil
	}

	// Analyze contexts to extract patterns
	patterns := l.analyzeContextPatterns(stats.Contexts)

	// Generate context-aware rule
	rule := NewContextAwareRule(fmt.Sprintf("learned_%s", solution.RuleName))
	rule.Description = fmt.Sprintf("Learned from %d AI solutions", stats.SuccessCount)
	rule.Priority = 5 // Medium priority for learned rules
	rule.Conditions = l.extractConditions(patterns)
	rule.Context.DomainPattern = l.extractDomainPattern(stats.Domains)
	rule.Context.Variables = l.extractVariables(patterns)
	rule.Actions = solution.Actions
	rule.Confidence = successRate
	rule.SuccessRate = successRate
	rule.UsageCount = stats.SuccessCount
	rule.CreatedBy = "learned"

	return rule
}

// generateFingerprint creates a unique fingerprint for a solution
func (l *LearningEngine) generateFingerprint(solution *Solution) string {
	data, _ := json.Marshal(solution.Actions)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// analyzeContextPatterns analyzes common patterns in contexts
func (l *LearningEngine) analyzeContextPatterns(contexts []*ExecutionContext) map[string]interface{} {
	patterns := make(map[string]interface{})

	// Analyze common error types
	errorTypes := make(map[string]int)
	statusCodes := make(map[int]int)

	for _, ctx := range contexts {
		errorTypes[ctx.Error.Type]++
		statusCodes[ctx.Response.StatusCode]++
	}

	// Extract dominant patterns
	patterns["dominant_error"] = findMostCommonString(errorTypes)
	patterns["dominant_status"] = findMostCommonInt(statusCodes)

	return patterns
}

// extractConditions extracts conditions from patterns
func (l *LearningEngine) extractConditions(patterns map[string]interface{}) []Condition {
	conditions := []Condition{}

	if dominantError, ok := patterns["dominant_error"].(string); ok && dominantError != "" {
		conditions = append(conditions, Condition{
			Field:    "error_type",
			Operator: "contains",
			Value:    dominantError,
		})
	}

	if dominantStatus, ok := patterns["dominant_status"].(int); ok && dominantStatus != 0 {
		conditions = append(conditions, Condition{
			Field:    "status_code",
			Operator: "equals",
			Value:    dominantStatus,
		})
	}

	return conditions
}

// extractDomainPattern extracts the most common domain pattern
func (l *LearningEngine) extractDomainPattern(domains map[string]int) string {
	if len(domains) == 1 {
		// Single domain
		for domain := range domains {
			return domain
		}
	}

	// Multiple domains - use wildcard
	return "*"
}

// extractVariables extracts variables from patterns
func (l *LearningEngine) extractVariables(patterns map[string]interface{}) map[string]interface{} {
	// For now, return empty - could be enhanced to extract dynamic variables
	return make(map[string]interface{})
}

func findMostCommonString(m map[string]int) string {
	maxCount := 0
	result := ""
	for key, count := range m {
		if count > maxCount {
			maxCount = count
			result = key
		}
	}
	return result
}

func findMostCommonInt(m map[int]int) int {
	maxCount := 0
	result := 0
	for key, count := range m {
		if count > maxCount {
			maxCount = count
			result = key
		}
	}
	return result
}
