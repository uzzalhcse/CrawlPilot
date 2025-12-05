package dom

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func ConvertSimpleXpathToCSSSelector(xpath string) string {
	if xpath == "" {
		return ""
	}

	xpath = strings.TrimPrefix(xpath, "/")
	parts := strings.Split(xpath, "/")
	var cssParts []string

	for _, part := range parts {
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") && !strings.Contains(part, "[") {
			basePart := strings.Replace(part, ":", `\:`, -1)
			cssParts = append(cssParts, basePart)
			continue
		}

		if strings.Contains(part, "[") {
			basePart := part[:strings.Index(part, "[")]
			basePart = strings.Replace(basePart, ":", `\:`, -1)
			indexPart := part[strings.Index(part, "["):]

			indices := strings.Split(indexPart, "]")
			indices = indices[:len(indices)-1]

			for _, idx := range indices {
				idx = strings.Trim(idx, "[]")
				if idxNum, err := strconv.Atoi(idx); err == nil {
					basePart += fmt.Sprintf(":nth-of-type(%d)", idxNum)
				} else if idx == "last()" {
					basePart += ":last-of-type"
				} else if strings.Contains(idx, "position()>1") {
					basePart += ":nth-of-type(n+2)"
				}
			}
			cssParts = append(cssParts, basePart)
		} else {
			cssParts = append(cssParts, part)
		}
	}

	return strings.Join(cssParts, " > ")
}

func EnhancedCSSSelector(element *DOMElementNode, includeDynamicAttributes bool) string {
	cssSelector := ConvertSimpleXpathToCSSSelector(element.Xpath)

	if classAttr := element.Attributes["class"]; classAttr != "" && includeDynamicAttributes {
		validPattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)$`)
		classes := strings.Split(classAttr, " ")

		for _, className := range classes {
			if strings.TrimSpace(className) == "" {
				continue
			}
			if validPattern.MatchString(className) {
				cssSelector += fmt.Sprintf(".%s", className)
			}
		}
	}

	safeAttributes := []string{
		"id", "name", "type", "placeholder",
		"aria-label", "aria-labelledby", "aria-describedby", "role",
		"for", "autocomplete", "required", "readonly",
		"alt", "title", "src", "href", "target",
	}

	if includeDynamicAttributes {
		safeAttributes = append(safeAttributes,
			"data-id", "data-qa", "data-cy", "data-testid")
	}

	var keys []string
	for k := range element.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, attribute := range keys {
		value := element.Attributes[attribute]
		if attribute == "class" || strings.TrimSpace(attribute) == "" {
			continue
		}

		if !slices.Contains(safeAttributes, attribute) {
			continue
		}

		safeAttr := strings.Replace(attribute, ":", "\\:", -1)

		if value == "" {
			cssSelector += fmt.Sprintf("[%s]", safeAttr)
		} else if strings.ContainsAny(value, "\"'<>`\n\r\t") {
			if strings.Contains(value, "\n") {
				value = strings.Split(value, "\n")[0]
			}
			re := regexp.MustCompile(`\s+`)
			collapsedValue := re.ReplaceAllString(value, " ")
			safeValue := strings.Replace(collapsedValue, "\"", "\\\"", -1)
			cssSelector += fmt.Sprintf("[%s*=\"%s\"]", safeAttr, safeValue)
		} else {
			cssSelector += fmt.Sprintf("[%s=\"%s\"]", safeAttr, value)
		}
	}

	return cssSelector
}
