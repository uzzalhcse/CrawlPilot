package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ExtractNode handles data extraction from web pages
type ExtractNode struct{}

func NewExtractNode() NodeExecutor {
	return &ExtractNode{}
}

func (n *ExtractNode) Type() string {
	return "extract"
}

func (n *ExtractNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Get schema name
	schemaName, ok := node.Params["schema"].(string)
	if !ok || schemaName == "" {
		return fmt.Errorf("schema is required for extract node")
	}

	// Get fields configuration
	fields, ok := node.Params["fields"].(map[string]interface{})
	if !ok || len(fields) == 0 {
		return fmt.Errorf("fields are required for extract node")
	}

	logger.Info("Extracting data",
		zap.String("schema", schemaName),
		zap.Int("field_count", len(fields)),
	)

	// Extract all fields
	extractedData := make(map[string]interface{})

	for fieldName, fieldConfig := range fields {
		value, err := n.extractField(ctx, execCtx, fieldName, fieldConfig)
		if err != nil {
			logger.Warn("Failed to extract field",
				zap.String("field", fieldName),
				zap.Error(err),
			)
			// Check for default value
			if configMap, ok := fieldConfig.(map[string]interface{}); ok {
				if defaultVal, hasDefault := configMap["default"]; hasDefault {
					extractedData[fieldName] = defaultVal
					continue
				}
			}
			continue
		}

		if value != nil {
			extractedData[fieldName] = value
		}
	}

	// Store extracted data
	if execCtx.Variables == nil {
		execCtx.Variables = make(map[string]interface{})
	}

	// Add to extracted_items array
	if _, ok := execCtx.Variables["extracted_items"]; !ok {
		execCtx.Variables["extracted_items"] = []map[string]interface{}{}
	}

	items := execCtx.Variables["extracted_items"].([]map[string]interface{})
	items = append(items, extractedData)
	execCtx.Variables["extracted_items"] = items

	logger.Info("Data extracted successfully",
		zap.String("schema", schemaName),
		zap.Int("fields_extracted", len(extractedData)),
		zap.Int("total_items", len(items)),
	)

	return nil
}

// extractField extracts a single field based on its configuration
func (n *ExtractNode) extractField(ctx context.Context, execCtx *ExecutionContext, fieldName string, config interface{}) (interface{}, error) {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid field config for: %s", fieldName)
	}

	// Check for nested extractions (like attributes field)
	if extractions, hasExtractions := configMap["extractions"]; hasExtractions {
		return n.extractNested(ctx, execCtx, configMap, extractions)
	}

	// Standard single field extraction
	selector, ok := configMap["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required for field: %s", fieldName)
	}

	extractType := "text"
	if t, ok := configMap["type"].(string); ok {
		extractType = t
	}

	// Check if multiple values
	isMultiple := false
	if m, ok := configMap["multiple"].(bool); ok {
		isMultiple = m
	}

	// Extract value(s)
	var rawValue interface{}
	var err error

	if isMultiple {
		rawValue, err = n.extractMultipleValues(execCtx, selector, extractType, configMap)
	} else {
		rawValue, err = n.extractSingleValue(execCtx, selector, extractType, configMap)
	}

	if err != nil {
		return nil, err
	}

	// Apply transformation if specified
	if transform, ok := configMap["transform"].(string); ok && transform != "" {
		rawValue = n.applyTransform(rawValue, transform)
	}

	return rawValue, nil
}

// extractSingleValue extracts a single value from the page
func (n *ExtractNode) extractSingleValue(execCtx *ExecutionContext, selector string, extractType string, config map[string]interface{}) (interface{}, error) {
	element := execCtx.Page.Locator(selector).First()

	// Check if element exists
	count, err := execCtx.Page.Locator(selector).Count()
	if err != nil || count == 0 {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	switch extractType {
	case "text":
		return element.TextContent()

	case "html":
		return element.InnerHTML()

	case "attr":
		attrName, ok := config["attribute"].(string)
		if !ok || attrName == "" {
			return nil, fmt.Errorf("attribute name is required for attr type")
		}
		return element.GetAttribute(attrName)

	default:
		return nil, fmt.Errorf("unknown extract type: %s", extractType)
	}
}

// extractMultipleValues extracts multiple values from the page
func (n *ExtractNode) extractMultipleValues(execCtx *ExecutionContext, selector string, extractType string, config map[string]interface{}) (interface{}, error) {
	elements := execCtx.Page.Locator(selector)
	count, err := elements.Count()
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return []string{}, nil
	}

	results := make([]string, 0, count)

	for i := 0; i < count; i++ {
		element := elements.Nth(i)
		var value string

		switch extractType {
		case "text":
			value, err = element.TextContent()
		case "html":
			value, err = element.InnerHTML()
		case "attr":
			attrName, ok := config["attribute"].(string)
			if !ok || attrName == "" {
				continue
			}
			value, err = element.GetAttribute(attrName)
		default:
			continue
		}

		if err != nil {
			logger.Debug("Failed to extract element",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		if value != "" {
			results = append(results, value)
		}
	}

	return results, nil
}

// extractNested handles nested extractions (like key-value pairs for attributes)
func (n *ExtractNode) extractNested(ctx context.Context, execCtx *ExecutionContext, config map[string]interface{}, extractions interface{}) (interface{}, error) {
	extractionsList, ok := extractions.([]interface{})
	if !ok || len(extractionsList) == 0 {
		return nil, fmt.Errorf("invalid extractions configuration")
	}

	// For now, support the first extraction config
	extractConfig, ok := extractionsList[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid extraction config")
	}

	keySelector, ok := extractConfig["key_selector"].(string)
	if !ok || keySelector == "" {
		return nil, fmt.Errorf("key_selector is required")
	}

	valueSelector, ok := extractConfig["value_selector"].(string)
	if !ok || valueSelector == "" {
		return nil, fmt.Errorf("value_selector is required")
	}

	// Extract key-value pairs
	keys := execCtx.Page.Locator(keySelector)
	values := execCtx.Page.Locator(valueSelector)

	keyCount, err := keys.Count()
	if err != nil {
		return nil, err
	}

	// Get key and value types
	keyType := "text"
	if kt, ok := extractConfig["key_type"].(string); ok {
		keyType = kt
	}

	valueType := "text"
	if vt, ok := extractConfig["value_type"].(string); ok {
		valueType = vt
	}

	// Get output format (default: array)
	outputFormat := "array"
	if format, ok := config["output_format"].(string); ok {
		outputFormat = format
	}

	// Extract pairs
	result := make([]map[string]string, 0, keyCount)

	for i := 0; i < keyCount; i++ {
		keyElement := keys.Nth(i)
		valueElement := values.Nth(i)

		var key, value string

		// Extract key
		if keyType == "text" {
			key, err = keyElement.TextContent()
		} else {
			key, err = keyElement.InnerHTML()
		}
		if err != nil {
			continue
		}

		// Extract value
		if valueType == "text" {
			value, err = valueElement.TextContent()
		} else {
			value, err = valueElement.InnerHTML()
		}
		if err != nil {
			continue
		}

		// Apply transformation if specified
		if transform, ok := extractConfig["transform"].(string); ok && transform != "" {
			key = n.applyTransformString(key, transform)
			value = n.applyTransformString(value, transform)
		}

		if key != "" && value != "" {
			result = append(result, map[string]string{
				"key":   key,
				"value": value,
			})
		}
	}

	// Return in requested format
	if outputFormat == "object" {
		obj := make(map[string]string)
		for _, pair := range result {
			obj[pair["key"]] = pair["value"]
		}
		return obj, nil
	}

	return result, nil // Default: array format
}

// applyTransform applies transformation to extracted value
func (n *ExtractNode) applyTransform(value interface{}, transform string) interface{} {
	switch v := value.(type) {
	case string:
		return n.applyTransformString(v, transform)
	case []string:
		transformed := make([]string, len(v))
		for i, str := range v {
			transformed[i] = n.applyTransformString(str, transform)
		}
		return transformed
	default:
		return value
	}
}

// applyTransformString applies transformation to a string value
func (n *ExtractNode) applyTransformString(value string, transform string) string {
	switch transform {
	case "trim":
		return strings.TrimSpace(value)

	case "lowercase":
		return strings.ToLower(value)

	case "uppercase":
		return strings.ToUpper(value)

	case "clean_html":
		// Remove HTML tags and extra whitespace
		re := regexp.MustCompile(`<[^>]*>`)
		cleaned := re.ReplaceAllString(value, " ")
		// Remove multiple spaces
		re = regexp.MustCompile(`\s+`)
		cleaned = re.ReplaceAllString(cleaned, " ")
		return strings.TrimSpace(cleaned)

	case "remove_whitespace":
		re := regexp.MustCompile(`\s+`)
		return re.ReplaceAllString(value, "")

	default:
		return value
	}
}

// DiscoverLinksNode discovers URLs from the page
type DiscoverLinksNode struct{}

func NewDiscoverLinksNode() NodeExecutor {
	return &DiscoverLinksNode{}
}

func (n *DiscoverLinksNode) Type() string {
	return "discover_links"
}

func (n *DiscoverLinksNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	selector := "a[href]"
	if s, ok := node.Params["selector"].(string); ok && s != "" {
		selector = s
	}

	logger.Info("Discovering links", zap.String("selector", selector))

	// Find all links
	links := execCtx.Page.Locator(selector)
	count, err := links.Count()
	if err != nil {
		return fmt.Errorf("failed to count links: %w", err)
	}

	discoveredURLs := make([]string, 0)

	for i := 0; i < count; i++ {
		link := links.Nth(i)
		href, err := link.GetAttribute("href")
		if err != nil {
			continue
		}

		if href != "" {
			discoveredURLs = append(discoveredURLs, href)
		}
	}

	// Store discovered URLs in execution context
	if execCtx.Variables == nil {
		execCtx.Variables = make(map[string]interface{})
	}
	execCtx.Variables["discovered_urls"] = discoveredURLs

	logger.Info("Links discovered",
		zap.Int("count", len(discoveredURLs)),
	)

	return nil
}

// ScriptNode executes custom JavaScript
type ScriptNode struct{}

func NewScriptNode() NodeExecutor {
	return &ScriptNode{}
}

func (n *ScriptNode) Type() string {
	return "script"
}

func (n *ScriptNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	script, ok := node.Params["code"].(string)
	if !ok || script == "" {
		return fmt.Errorf("code is required for script node")
	}

	logger.Info("Executing custom script")

	result, err := execCtx.Page.Evaluate(script)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Store result if needed
	if storeAs, ok := node.Params["store_as"].(string); ok && storeAs != "" {
		if execCtx.Variables == nil {
			execCtx.Variables = make(map[string]interface{})
		}

		// Convert result to proper type
		var resultData interface{}
		if resultBytes, ok := result.([]byte); ok {
			json.Unmarshal(resultBytes, &resultData)
		} else {
			resultData = result
		}

		execCtx.Variables[storeAs] = resultData
		logger.Debug("Script result stored", zap.String("variable", storeAs))
	}

	return nil
}
