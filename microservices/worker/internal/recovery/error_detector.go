package recovery

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// ErrorDetector detects and classifies errors into patterns
type ErrorDetector struct {
	patterns map[ErrorPattern]*patternMatcher
}

type patternMatcher struct {
	substrings  []string
	regexes     []*regexp.Regexp
	statusCodes []int
}

// NewErrorDetector creates a new error detector with predefined patterns
func NewErrorDetector() *ErrorDetector {
	d := &ErrorDetector{
		patterns: make(map[ErrorPattern]*patternMatcher),
	}

	// Blocked patterns
	d.patterns[PatternBlocked] = &patternMatcher{
		substrings: []string{
			"access denied",
			"forbidden",
			"blocked",
			"not allowed",
			"permission denied",
			"ip banned",
			"your ip has been",
			"temporarily banned",
			"you have been blocked",
			"cf-error",
			"cloudflare",
			"ddos protection",
			"bot detected",
			"automated access",
			"unusual traffic",
		},
		statusCodes: []int{403, 406, 451},
	}

	// Rate limited patterns
	d.patterns[PatternRateLimited] = &patternMatcher{
		substrings: []string{
			"too many requests",
			"rate limit",
			"ratelimit",
			"rate_limit",
			"throttled",
			"slow down",
			"quota exceeded",
			"limit exceeded",
			"try again later",
			"request limit",
		},
		statusCodes: []int{429},
	}

	// Captcha patterns
	d.patterns[PatternCaptcha] = &patternMatcher{
		substrings: []string{
			"captcha",
			"recaptcha",
			"hcaptcha",
			"challenge",
			"verify you are human",
			"are you a robot",
			"prove you're not a robot",
			"security check",
			"bot verification",
		},
	}

	// Timeout patterns
	d.patterns[PatternTimeout] = &patternMatcher{
		substrings: []string{
			"timeout",
			"timed out",
			"deadline exceeded",
			"context deadline",
			"connection timeout",
			"read timeout",
			"write timeout",
			"operation timed out",
		},
		statusCodes: []int{408, 504, 524},
	}

	// Connection error patterns
	d.patterns[PatternConnectionErr] = &patternMatcher{
		substrings: []string{
			"connection refused",
			"connection reset",
			"no such host",
			"network is unreachable",
			"host unreachable",
			"dns lookup",
			"dial tcp",
			"connect: connection",
			"eof",
			"broken pipe",
			"connection closed",
		},
	}

	// Layout changed patterns
	d.patterns[PatternLayoutChanged] = &patternMatcher{
		substrings: []string{
			"element not found",
			"selector not found",
			"no element matches",
			"could not find",
			"waiting for selector",
			"element not visible",
			"stale element",
		},
	}

	// Auth required patterns
	d.patterns[PatternAuthRequired] = &patternMatcher{
		substrings: []string{
			"login required",
			"sign in",
			"authentication required",
			"session expired",
			"please log in",
			"unauthorized",
		},
		statusCodes: []int{401},
	}

	// Not found patterns
	d.patterns[PatternNotFound] = &patternMatcher{
		substrings: []string{
			"not found",
			"page not found",
			"404",
			"does not exist",
			"no longer available",
		},
		statusCodes: []int{404, 410},
	}

	// Server error patterns
	d.patterns[PatternServerError] = &patternMatcher{
		substrings: []string{
			"internal server error",
			"server error",
			"service unavailable",
			"bad gateway",
			"under maintenance",
			"try again",
		},
		statusCodes: []int{500, 502, 503},
	}

	return d
}

// Detect analyzes an error and returns a DetectedError with pattern classification
func (d *ErrorDetector) Detect(err error, pageURL string, statusCode int, pageContent string) *DetectedError {
	if err == nil && statusCode == 0 && pageContent == "" {
		return nil
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	// Extract domain from URL
	domain := extractDomain(pageURL)

	detected := &DetectedError{
		Pattern:     PatternUnknown,
		Confidence:  0.0,
		RawError:    errStr,
		Domain:      domain,
		URL:         pageURL,
		StatusCode:  statusCode,
		PageContent: truncateContent(pageContent, 1000),
		DetectedAt:  time.Now(),
	}

	// Try to match patterns
	searchText := strings.ToLower(errStr + " " + pageContent)

	var bestMatch ErrorPattern
	var bestConfidence float64

	for pattern, matcher := range d.patterns {
		confidence := d.matchPattern(searchText, statusCode, matcher)
		if confidence > bestConfidence {
			bestConfidence = confidence
			bestMatch = pattern
		}
	}

	if bestConfidence > 0 {
		detected.Pattern = bestMatch
		detected.Confidence = bestConfidence
	}

	logger.Debug("Error detected",
		zap.String("pattern", string(detected.Pattern)),
		zap.Float64("confidence", detected.Confidence),
		zap.String("domain", detected.Domain),
		zap.Int("status_code", statusCode),
	)

	return detected
}

// matchPattern calculates a confidence score for a pattern match
func (d *ErrorDetector) matchPattern(text string, statusCode int, matcher *patternMatcher) float64 {
	var confidence float64

	// Check status codes (high confidence)
	for _, code := range matcher.statusCodes {
		if statusCode == code {
			confidence = 0.9
			break
		}
	}

	// Check substrings
	matchCount := 0
	for _, substr := range matcher.substrings {
		if strings.Contains(text, substr) {
			matchCount++
		}
	}

	if matchCount > 0 {
		// More matches = higher confidence
		substringConfidence := 0.5 + (float64(matchCount) * 0.1)
		if substringConfidence > 0.95 {
			substringConfidence = 0.95
		}
		if substringConfidence > confidence {
			confidence = substringConfidence
		}
	}

	// Check regexes
	for _, re := range matcher.regexes {
		if re.MatchString(text) {
			if confidence < 0.8 {
				confidence = 0.8
			}
		}
	}

	return confidence
}

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return parsed.Host
}

// truncateContent truncates content to a maximum length
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// GetPatternDescription returns a human-readable description of a pattern
func GetPatternDescription(pattern ErrorPattern) string {
	descriptions := map[ErrorPattern]string{
		PatternBlocked:       "Request blocked by target site (IP/bot detection)",
		PatternRateLimited:   "Rate limit exceeded - too many requests",
		PatternCaptcha:       "CAPTCHA challenge detected",
		PatternTimeout:       "Request timed out",
		PatternConnectionErr: "Network connection error",
		PatternLayoutChanged: "Page layout or selectors changed",
		PatternAuthRequired:  "Authentication or login required",
		PatternNotFound:      "Page not found (404)",
		PatternServerError:   "Server error (5xx)",
		PatternUnknown:       "Unknown error pattern",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}
	return "Unknown error"
}

// GetRecommendedAction returns a default action for a pattern
func GetRecommendedAction(pattern ErrorPattern) ActionType {
	recommendations := map[ErrorPattern]ActionType{
		PatternBlocked:       ActionSwitchProxy,
		PatternRateLimited:   ActionAddDelay,
		PatternCaptcha:       ActionSendToDLQ,
		PatternTimeout:       ActionRetry,
		PatternConnectionErr: ActionSwitchProxy,
		PatternLayoutChanged: ActionSendToDLQ,
		PatternAuthRequired:  ActionSendToDLQ,
		PatternNotFound:      ActionSendToDLQ,
		PatternServerError:   ActionRetry,
		PatternUnknown:       ActionRetry,
	}

	if action, ok := recommendations[pattern]; ok {
		return action
	}
	return ActionRetry
}
