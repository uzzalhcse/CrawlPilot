package extraction

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

type ExtractionEngine struct {
	page playwright.Page
}

type ExtractConfig struct {
	Selector     string                 `json:"selector" yaml:"selector"`
	Type         string                 `json:"type" yaml:"type"` // text, attr, html, href, src
	Attribute    string                 `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	Multiple     bool                   `json:"multiple,omitempty" yaml:"multiple,omitempty"`
	Limit        int                    `json:"limit,omitempty" yaml:"limit,omitempty"`         // Limit for multiple extraction (0 = no limit)
	Transform    interface{}            `json:"transform,omitempty" yaml:"transform,omitempty"` // Can be string or []TransformConfig
	Fields       map[string]interface{} `json:"fields,omitempty" yaml:"fields,omitempty"`
	DefaultValue interface{}            `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	Extractions  []ExtractionPair       `json:"extractions,omitempty" yaml:"extractions,omitempty"` // For independent array extraction
}

// ExtractionPair defines a pair of independent key-value selectors
type ExtractionPair struct {
	KeySelector    string      `json:"key_selector" yaml:"key_selector"`
	ValueSelector  string      `json:"value_selector" yaml:"value_selector"`
	KeyType        string      `json:"key_type" yaml:"key_type"`     // text, attr, html, href, src
	ValueType      string      `json:"value_type" yaml:"value_type"` // text, attr, html, href, src
	KeyAttribute   string      `json:"key_attribute,omitempty" yaml:"key_attribute,omitempty"`
	ValueAttribute string      `json:"value_attribute,omitempty" yaml:"value_attribute,omitempty"`
	Transform      interface{} `json:"transform,omitempty" yaml:"transform,omitempty"`
	Limit          int         `json:"limit,omitempty" yaml:"limit,omitempty"` // Limit for this extraction pair
}

type TransformConfig struct {
	Type   string                 `json:"type" yaml:"type"` // trim, lowercase, uppercase, regex, replace, split, join, parse_int, parse_float
	Params map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
}

func NewExtractionEngine(page playwright.Page) *ExtractionEngine {
	return &ExtractionEngine{
		page: page,
	}
}

// Extract extracts data based on configuration
func (ee *ExtractionEngine) Extract(config ExtractConfig) (interface{}, error) {
	// Check if this is an independent arrays extraction
	if len(config.Extractions) > 0 {
		return ee.extractFieldValueIndependentArrays(nil, config)
	}

	if config.Multiple {
		return ee.extractMultiple(config)
	}
	return ee.extractSingle(config)
}

// extractSingle extracts a single value
func (ee *ExtractionEngine) extractSingle(config ExtractConfig) (interface{}, error) {
	locator := ee.page.Locator(config.Selector)

	count, err := locator.Count()
	if err != nil || count == 0 {
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return nil, fmt.Errorf("selector not found: %s", config.Selector)
	}

	var value string
	switch config.Type {
	case "text":
		text, err := locator.First().TextContent()
		if err != nil {
			return config.DefaultValue, err
		}
		value = text
	case "attr":
		attr, err := locator.First().GetAttribute(config.Attribute)
		if err != nil {
			return config.DefaultValue, err
		}
		value = attr
	case "html":
		html, err := locator.First().InnerHTML()
		if err != nil {
			return config.DefaultValue, err
		}
		value = html
	case "href":
		href, err := locator.First().GetAttribute("href")
		if err != nil {
			return config.DefaultValue, err
		}
		value = href
	case "src":
		src, err := locator.First().GetAttribute("src")
		if err != nil {
			return config.DefaultValue, err
		}
		value = src
	default:
		text, err := locator.First().TextContent()
		if err != nil {
			return config.DefaultValue, err
		}
		value = text
	}

	// Apply transformations
	if config.Transform != nil {
		transforms := ee.parseTransforms(config.Transform)
		if len(transforms) > 0 {
			transformed, err := ee.applyTransformations(value, transforms)
			if err != nil {
				return config.DefaultValue, err
			}
			return transformed, nil
		}
	}

	return value, nil
}

// extractMultiple extracts multiple values
func (ee *ExtractionEngine) extractMultiple(config ExtractConfig) (interface{}, error) {
	locator := ee.page.Locator(config.Selector)

	count, err := locator.Count()
	if err != nil || count == 0 {
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return []interface{}{}, nil
	}

	// Apply limit if specified (0 means no limit)
	if config.Limit > 0 && config.Limit < count {
		count = config.Limit
	}

	var results []interface{}

	for i := 0; i < count; i++ {
		element := locator.Nth(i)

		// If fields are defined, extract structured data
		if len(config.Fields) > 0 {
			item := make(map[string]interface{})
			for fieldName, fieldConfigRaw := range config.Fields {
				var fieldConfig ExtractConfig

				// Convert field config to ExtractConfig
				jsonData, err := json.Marshal(fieldConfigRaw)
				if err != nil {
					continue
				}
				if err := json.Unmarshal(jsonData, &fieldConfig); err != nil {
					continue
				}

				// Extract field value
				value, err := ee.extractFieldValue(element, fieldConfig)
				if err == nil {
					item[fieldName] = value
				}
			}
			results = append(results, item)
		} else {
			// Extract simple values
			var value string
			switch config.Type {
			case "text":
				text, err := element.TextContent()
				if err == nil {
					value = text
				}
			case "attr":
				attr, err := element.GetAttribute(config.Attribute)
				if err == nil {
					value = attr
				}
			case "html":
				html, err := element.InnerHTML()
				if err == nil {
					value = html
				}
			case "href":
				href, err := element.GetAttribute("href")
				if err == nil {
					value = href
				}
			case "src":
				src, err := element.GetAttribute("src")
				if err == nil {
					value = src
				}
			}

			// Apply transformations
			if config.Transform != nil {
				transforms := ee.parseTransforms(config.Transform)
				if len(transforms) > 0 {
					transformed, err := ee.applyTransformations(value, transforms)
					if err == nil {
						results = append(results, transformed)
					}
				} else {
					results = append(results, value)
				}
			} else {
				results = append(results, value)
			}
		}
	}

	return results, nil
}

// extractFieldValue extracts a field value from a locator
// Supports both single values and arrays (simple arrays and nested object arrays)
func (ee *ExtractionEngine) extractFieldValue(locator playwright.Locator, config ExtractConfig) (interface{}, error) {
	// Check if this is an independent arrays extraction (multiple unrelated sources)
	if len(config.Extractions) > 0 {
		return ee.extractFieldValueIndependentArrays(locator, config)
	}

	// Check if this field should extract multiple values
	if config.Multiple {
		// Check if nested fields are defined (nested object array)
		if len(config.Fields) > 0 {
			return ee.extractFieldValueNestedArray(locator, config)
		}
		// Simple array extraction
		return ee.extractFieldValueSimpleArray(locator, config)
	}

	// Single value extraction
	return ee.extractFieldValueSingle(locator, config)
}

// extractFieldValueSingle extracts a single value from a locator
func (ee *ExtractionEngine) extractFieldValueSingle(locator playwright.Locator, config ExtractConfig) (interface{}, error) {
	var subLocator playwright.Locator
	if config.Selector != "" {
		subLocator = locator.Locator(config.Selector)
	} else {
		subLocator = locator
	}

	// Check if element exists
	count, err := subLocator.Count()
	if err != nil || count == 0 {
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return nil, fmt.Errorf("selector not found: %s", config.Selector)
	}

	var value string
	switch config.Type {
	case "text":
		text, err := subLocator.First().TextContent()
		if err != nil {
			return config.DefaultValue, err
		}
		value = text
	case "attr":
		attr, err := subLocator.First().GetAttribute(config.Attribute)
		if err != nil {
			return config.DefaultValue, err
		}
		value = attr
	case "html":
		html, err := subLocator.First().InnerHTML()
		if err != nil {
			return config.DefaultValue, err
		}
		value = html
	case "href":
		href, err := subLocator.First().GetAttribute("href")
		if err != nil {
			return config.DefaultValue, err
		}
		value = href
	case "src":
		src, err := subLocator.First().GetAttribute("src")
		if err != nil {
			return config.DefaultValue, err
		}
		value = src
	default:
		text, err := subLocator.First().TextContent()
		if err != nil {
			return config.DefaultValue, err
		}
		value = text
	}

	// Apply transformations
	if config.Transform != nil {
		transforms := ee.parseTransforms(config.Transform)
		if len(transforms) > 0 {
			return ee.applyTransformations(value, transforms)
		}
	}

	return value, nil
}

// extractFieldValueSimpleArray extracts an array of simple values (strings, numbers, etc.)
// Example: ["image1.jpg", "image2.jpg", "image3.jpg"]
func (ee *ExtractionEngine) extractFieldValueSimpleArray(locator playwright.Locator, config ExtractConfig) (interface{}, error) {
	var subLocator playwright.Locator
	if config.Selector != "" {
		subLocator = locator.Locator(config.Selector)
	} else {
		return nil, fmt.Errorf("selector required for multiple field extraction")
	}

	count, err := subLocator.Count()
	if err != nil || count == 0 {
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return []interface{}{}, nil
	}

	// Apply limit if specified
	if config.Limit > 0 && config.Limit < count {
		count = config.Limit
	}

	var results []interface{}
	for i := 0; i < count; i++ {
		element := subLocator.Nth(i)

		var value string
		switch config.Type {
		case "text":
			text, err := element.TextContent()
			if err == nil {
				value = text
			}
		case "attr":
			attr, err := element.GetAttribute(config.Attribute)
			if err == nil {
				value = attr
			}
		case "html":
			html, err := element.InnerHTML()
			if err == nil {
				value = html
			}
		case "href":
			href, err := element.GetAttribute("href")
			if err == nil {
				value = href
			}
		case "src":
			src, err := element.GetAttribute("src")
			if err == nil {
				value = src
			}
		default:
			text, err := element.TextContent()
			if err == nil {
				value = text
			}
		}

		// Apply transformations
		if config.Transform != nil {
			transforms := ee.parseTransforms(config.Transform)
			if len(transforms) > 0 {
				transformed, err := ee.applyTransformations(value, transforms)
				if err == nil {
					results = append(results, transformed)
					continue
				}
			}
		}

		results = append(results, value)
	}

	return results, nil
}

// extractFieldValueNestedArray extracts an array of objects with nested fields
// Example: [{"key": "color", "value": "black"}, {"key": "size", "value": "large"}]
func (ee *ExtractionEngine) extractFieldValueNestedArray(locator playwright.Locator, config ExtractConfig) (interface{}, error) {
	var subLocator playwright.Locator
	if config.Selector != "" {
		subLocator = locator.Locator(config.Selector)
	} else {
		return nil, fmt.Errorf("selector required for multiple field extraction")
	}

	count, err := subLocator.Count()
	if err != nil || count == 0 {
		if config.DefaultValue != nil {
			return config.DefaultValue, nil
		}
		return []interface{}{}, nil
	}

	// Apply limit if specified
	if config.Limit > 0 && config.Limit < count {
		count = config.Limit
	}

	var results []interface{}
	for i := 0; i < count; i++ {
		element := subLocator.Nth(i)
		item := make(map[string]interface{})

		// Extract each nested field
		for fieldName, fieldConfigRaw := range config.Fields {
			var fieldConfig ExtractConfig

			// Convert field config to ExtractConfig
			jsonData, err := json.Marshal(fieldConfigRaw)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(jsonData, &fieldConfig); err != nil {
				continue
			}

			// Recursively extract field value (supports nested arrays within nested arrays)
			value, err := ee.extractFieldValue(element, fieldConfig)
			if err == nil {
				item[fieldName] = value
			} else if fieldConfig.DefaultValue != nil {
				item[fieldName] = fieldConfig.DefaultValue
			}
		}

		results = append(results, item)
	}

	return results, nil
}

// extractFieldValueIndependentArrays extracts arrays from multiple independent sources and merges them
// Example: Extract keys from one selector and values from another, then pair them by index
func (ee *ExtractionEngine) extractFieldValueIndependentArrays(locator playwright.Locator, config ExtractConfig) (interface{}, error) {
	var allResults []interface{}

	// Use page as base if locator is nil, otherwise use the provided locator
	var baseLocator playwright.Page
	if locator == nil {
		baseLocator = ee.page
	}

	// Process each extraction pair
	for _, extraction := range config.Extractions {
		// Extract keys - use page directly if no parent locator
		var keyLocator playwright.Locator
		if locator != nil {
			keyLocator = locator.Locator(extraction.KeySelector)
		} else {
			keyLocator = baseLocator.Locator(extraction.KeySelector)
		}

		keyCount, err := keyLocator.Count()
		if err != nil || keyCount == 0 {
			continue
		}

		// Extract values
		var valueLocator playwright.Locator
		if locator != nil {
			valueLocator = locator.Locator(extraction.ValueSelector)
		} else {
			valueLocator = baseLocator.Locator(extraction.ValueSelector)
		}

		valueCount, err := valueLocator.Count()
		if err != nil || valueCount == 0 {
			continue
		}

		// Use the minimum count to avoid index out of bounds
		count := keyCount
		if valueCount < count {
			count = valueCount
		}

		// Apply limit if specified
		if extraction.Limit > 0 && extraction.Limit < count {
			count = extraction.Limit
		}

		// Extract and pair key-value items
		for i := 0; i < count; i++ {
			keyElement := keyLocator.Nth(i)
			valueElement := valueLocator.Nth(i)

			// Extract key
			var key string
			switch extraction.KeyType {
			case "text", "":
				text, err := keyElement.TextContent()
				if err == nil {
					key = text
				}
			case "attr":
				attr, err := keyElement.GetAttribute(extraction.KeyAttribute)
				if err == nil {
					key = attr
				}
			case "html":
				html, err := keyElement.InnerHTML()
				if err == nil {
					key = html
				}
			case "href":
				href, err := keyElement.GetAttribute("href")
				if err == nil {
					key = href
				}
			case "src":
				src, err := keyElement.GetAttribute("src")
				if err == nil {
					key = src
				}
			}

			// Extract value
			var value string
			switch extraction.ValueType {
			case "text", "":
				text, err := valueElement.TextContent()
				if err == nil {
					value = text
				}
			case "attr":
				attr, err := valueElement.GetAttribute(extraction.ValueAttribute)
				if err == nil {
					value = attr
				}
			case "html":
				html, err := valueElement.InnerHTML()
				if err == nil {
					value = html
				}
			case "href":
				href, err := valueElement.GetAttribute("href")
				if err == nil {
					value = href
				}
			case "src":
				src, err := valueElement.GetAttribute("src")
				if err == nil {
					value = src
				}
			}

			// Apply transformations if specified
			if extraction.Transform != nil {
				transforms := ee.parseTransforms(extraction.Transform)
				if len(transforms) > 0 {
					if transformedKey, err := ee.applyTransformations(key, transforms); err == nil {
						if str, ok := transformedKey.(string); ok {
							key = str
						}
					}
					if transformedValue, err := ee.applyTransformations(value, transforms); err == nil {
						if str, ok := transformedValue.(string); ok {
							value = str
						}
					}
				}
			}

			// Create key-value pair
			item := map[string]interface{}{
				"key":   key,
				"value": value,
			}
			allResults = append(allResults, item)
		}
	}

	return allResults, nil
}

// parseTransforms converts various transform formats to []TransformConfig
func (ee *ExtractionEngine) parseTransforms(transform interface{}) []TransformConfig {
	switch t := transform.(type) {
	case string:
		// Handle single string transform like "trim", "clean_html", "extract_price"
		return []TransformConfig{{Type: t}}
	case []interface{}:
		// Handle array of transform configs
		var transforms []TransformConfig
		for _, item := range t {
			if transformMap, ok := item.(map[string]interface{}); ok {
				var tc TransformConfig
				if transformType, ok := transformMap["type"].(string); ok {
					tc.Type = transformType
					if params, ok := transformMap["params"].(map[string]interface{}); ok {
						tc.Params = params
					}
					transforms = append(transforms, tc)
				}
			}
		}
		return transforms
	case []TransformConfig:
		// Already in correct format
		return t
	default:
		return []TransformConfig{}
	}
}

// applyTransformations applies a chain of transformations to a value
func (ee *ExtractionEngine) applyTransformations(value string, transforms []TransformConfig) (interface{}, error) {
	var result interface{} = value

	for _, transform := range transforms {
		switch transform.Type {
		case "trim":
			if str, ok := result.(string); ok {
				result = strings.TrimSpace(str)
			}
		case "lowercase":
			if str, ok := result.(string); ok {
				result = strings.ToLower(str)
			}
		case "uppercase":
			if str, ok := result.(string); ok {
				result = strings.ToUpper(str)
			}
		case "regex":
			if str, ok := result.(string); ok {
				pattern, _ := transform.Params["pattern"].(string)
				replacement, _ := transform.Params["replacement"].(string)
				re := regexp.MustCompile(pattern)
				result = re.ReplaceAllString(str, replacement)
			}
		case "replace":
			if str, ok := result.(string); ok {
				old, _ := transform.Params["old"].(string)
				new, _ := transform.Params["new"].(string)
				result = strings.ReplaceAll(str, old, new)
			}
		case "split":
			if str, ok := result.(string); ok {
				delimiter, _ := transform.Params["delimiter"].(string)
				result = strings.Split(str, delimiter)
			}
		case "join":
			if arr, ok := result.([]string); ok {
				delimiter, _ := transform.Params["delimiter"].(string)
				result = strings.Join(arr, delimiter)
			}
		case "parse_int":
			if str, ok := result.(string); ok {
				i, err := strconv.ParseInt(strings.TrimSpace(str), 10, 64)
				if err == nil {
					result = i
				}
			}
		case "parse_float":
			if str, ok := result.(string); ok {
				f, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
				if err == nil {
					result = f
				}
			}
		case "clean_html":
			if str, ok := result.(string); ok {
				// Remove HTML tags and clean up text
				re := regexp.MustCompile(`<[^>]*>`)
				cleaned := re.ReplaceAllString(str, " ")
				// Clean up multiple spaces and newlines
				re = regexp.MustCompile(`\s+`)
				cleaned = re.ReplaceAllString(cleaned, " ")
				result = strings.TrimSpace(cleaned)
			}
		case "extract_price":
			if str, ok := result.(string); ok {
				// Extract price from strings like "¥123,456" or "$99.99"
				re := regexp.MustCompile(`[\d,]+\.?\d*`)
				matches := re.FindString(str)
				if matches != "" {
					// Remove commas and parse as float
					priceStr := strings.ReplaceAll(matches, ",", "")
					if f, err := strconv.ParseFloat(priceStr, 64); err == nil {
						result = f
					}
				}
			}
		}
	}

	return result, nil
}

// ExtractLinks extracts all links from the page with optional limit
func (ee *ExtractionEngine) ExtractLinks(selector string, limit int) ([]string, error) {
	content, err := ee.page.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var links []string
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		// If limit is set and we've reached it, stop extracting
		if limit > 0 && len(links) >= limit {
			return
		}

		href, exists := s.Attr("href")
		if exists && href != "" {
			links = append(links, href)
		}
	})

	return links, nil
}

// ExtractJSON extracts JSON data from a script tag or embedded JSON
func (ee *ExtractionEngine) ExtractJSON(selector string) (map[string]interface{}, error) {
	content, err := ee.page.Locator(selector).First().InnerHTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get element content: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}
