package error_recovery

import (
	"container/ring"
	"fmt"
	"time"
)

// Result represents the result of a request execution
type Result struct {
	Error     error
	Context   *ExecutionContext
	Timestamp time.Time
	Success   bool
}

// AnalyzerConfig configures the error pattern analyzer
type AnalyzerConfig struct {
	WindowSize            int     // Analyze last N requests
	ErrorRateThreshold    float64 // 0.10 = 10%
	ConsecutiveErrorLimit int     // 5 consecutive
	SameErrorThreshold    int     // 10 same errors
	DomainErrorThreshold  float64 // 0.20 per domain
}

// ErrorPatternAnalyzer detects systematic issues vs random errors
type ErrorPatternAnalyzer struct {
	recentResults *ring.Ring
	config        AnalyzerConfig
}

// NewErrorPatternAnalyzer creates a new analyzer
func NewErrorPatternAnalyzer(config AnalyzerConfig) *ErrorPatternAnalyzer {
	return &ErrorPatternAnalyzer{
		recentResults: ring.New(config.WindowSize),
		config:        config,
	}
}

// ShouldActivate determines if error recovery should be activated
func (a *ErrorPatternAnalyzer) ShouldActivate(err error, ctx *ExecutionContext) ActivationDecision {
	// Add result to ring buffer
	a.recentResults.Value = Result{
		Error:     err,
		Context:   ctx,
		Timestamp: time.Now(),
		Success:   err == nil,
	}
	a.recentResults = a.recentResults.Next()

	// Critical errors activate immediately
	if err != nil && isCriticalError(err) {
		return ActivationDecision{
			ShouldActivate: true,
			Reason:         "critical_error_type",
			ErrorPattern:   ErrorPattern{Type: "critical"},
		}
	}

	// Check error rate
	errorRate := a.calculateErrorRate()
	if errorRate >= a.config.ErrorRateThreshold {
		return ActivationDecision{
			ShouldActivate: true,
			Reason:         fmt.Sprintf("error_rate_%.1f%%", errorRate*100),
			ErrorPattern: ErrorPattern{
				Type:      "rate_spike",
				ErrorRate: errorRate,
			},
		}
	}

	// Check consecutive errors
	consecutive := a.countConsecutiveErrors()
	if consecutive >= a.config.ConsecutiveErrorLimit {
		return ActivationDecision{
			ShouldActivate: true,
			Reason:         fmt.Sprintf("%d_consecutive_errors", consecutive),
			ErrorPattern: ErrorPattern{
				Type:             "consecutive",
				ConsecutiveCount: consecutive,
			},
		}
	}

	// Check same error repeating
	dominantError := a.findDominantError()
	if dominantError.Count >= a.config.SameErrorThreshold {
		return ActivationDecision{
			ShouldActivate: true,
			Reason:         "systematic_error",
			ErrorPattern: ErrorPattern{
				Type:          "systematic",
				DominantError: dominantError.Type,
			},
		}
	}

	return ActivationDecision{ShouldActivate: false}
}

func (a *ErrorPatternAnalyzer) calculateErrorRate() float64 {
	total := 0
	errors := 0

	a.recentResults.Do(func(v interface{}) {
		if v != nil {
			result := v.(Result)
			total++
			if !result.Success {
				errors++
			}
		}
	})

	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total)
}

func (a *ErrorPatternAnalyzer) countConsecutiveErrors() int {
	consecutive := 0
	maxConsecutive := 0

	a.recentResults.Do(func(v interface{}) {
		if v != nil {
			result := v.(Result)
			if !result.Success {
				consecutive++
				if consecutive > maxConsecutive {
					maxConsecutive = consecutive
				}
			} else {
				consecutive = 0
			}
		}
	})

	return maxConsecutive
}

type ErrorCount struct {
	Type  string
	Count int
}

func (a *ErrorPatternAnalyzer) findDominantError() ErrorCount {
	errorCounts := make(map[string]int)

	a.recentResults.Do(func(v interface{}) {
		if v != nil {
			result := v.(Result)
			if !result.Success && result.Error != nil {
				errorType := result.Error.Error()
				errorCounts[errorType]++
			}
		}
	})

	dominant := ErrorCount{}
	for errType, count := range errorCounts {
		if count > dominant.Count {
			dominant.Type = errType
			dominant.Count = count
		}
	}

	return dominant
}

func isCriticalError(err error) bool {
	// Define critical error types
	criticalPatterns := []string{
		"panic",
		"fatal",
		"out of memory",
		"database connection",
	}

	errMsg := err.Error()
	for _, pattern := range criticalPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(findSubstring(s, substr) != -1))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
