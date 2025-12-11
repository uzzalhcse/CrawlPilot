package headers

import (
	"strings"
)

// PascalizeUpper are header names that should be fully uppercased
var pascalizeUpper = map[string]bool{
	"dnt": true,
	"rtt": true,
	"ect": true,
}

// GetUserAgent extracts the User-Agent from headers (case-insensitive)
func GetUserAgent(headers map[string]string) string {
	if ua, ok := headers["User-Agent"]; ok {
		return ua
	}
	if ua, ok := headers["user-agent"]; ok {
		return ua
	}
	return ""
}

// GetBrowser determines the browser name from a User-Agent string
func GetBrowser(userAgent string) string {
	if strings.Contains(userAgent, "Firefox") || strings.Contains(userAgent, "FxiOS") {
		return "firefox"
	}
	if strings.Contains(userAgent, "Chrome") || strings.Contains(userAgent, "CriOS") {
		return "chrome"
	}
	if strings.Contains(userAgent, "Safari") {
		return "safari"
	}
	if strings.Contains(userAgent, "Edge") || strings.Contains(userAgent, "EdgA") ||
		strings.Contains(userAgent, "Edg") || strings.Contains(userAgent, "EdgiOS") {
		return "edge"
	}
	return ""
}

// Pascalize converts a header name to Pascal-Case (title case)
func Pascalize(name string) string {
	// Don't modify pseudo-headers or sec-ch-ua- headers
	if strings.HasPrefix(name, ":") || strings.HasPrefix(name, "sec-ch-ua") {
		return name
	}

	lower := strings.ToLower(name)
	if pascalizeUpper[lower] {
		return strings.ToUpper(name)
	}

	// Title case each hyphen-separated word
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "-")
}

// PascalizeHeaders converts all header names in a map to Pascal-Case
func PascalizeHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		result[Pascalize(key)] = value
	}
	return result
}

// ToSlice converts a single string or slice to a string slice
func ToSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// ContainsString checks if a string slice contains a string
func ContainsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
