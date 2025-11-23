package extraction

import (
	"context"
	"encoding/json"
	"fmt"

	extraction_engine "github.com/uzzalhcse/crawlify/internal/extraction"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// ExtractExecutor handles data extraction
type ExtractExecutor struct {
	nodes.BaseNodeExecutor
}

// NewExtractExecutor creates a new extract executor
func NewExtractExecutor() *ExtractExecutor {
	return &ExtractExecutor{
		BaseNodeExecutor: nodes.BaseNodeExecutor{},
	}
}

// Type returns the node type
func (e *ExtractExecutor) Type() models.NodeType {
	return models.NodeTypeExtract
}

// Validate validates the node parameters
func (e *ExtractExecutor) Validate(params map[string]interface{}) error {
	// Either fields or selector must be present
	fields := nodes.GetMapParam(params, "fields")
	selector := nodes.GetStringParam(params, "selector")

	if len(fields) == 0 && selector == "" {
		return fmt.Errorf("either fields or selector must be provided for extract node")
	}
	return nil
}

// Execute performs data extraction
func (e *ExtractExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	engine := extraction_engine.NewExtractionEngine(input.BrowserContext.Page)

	// Convert params to ExtractConfig
	var config extraction_engine.ExtractConfig
	configBytes, err := json.Marshal(input.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle field-based extraction (when fields are defined but no top-level selector)
	if len(config.Fields) > 0 && config.Selector == "" {
		// For field-based extraction, we extract individual fields and combine results
		fieldResults := make(map[string]interface{})

		for fieldName, fieldConfigRaw := range config.Fields {
			var fieldConfig extraction_engine.ExtractConfig

			// Convert field config to ExtractConfig
			fieldConfigBytes, err := json.Marshal(fieldConfigRaw)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(fieldConfigBytes, &fieldConfig); err != nil {
				continue
			}

			// Check if this is a key-value extraction
			if len(fieldConfig.Extractions) > 0 {
				result, err := engine.Extract(fieldConfig)
				if err == nil {
					// Apply output format transformation if specified
					outputFormat := getOutputFormat(fieldConfigRaw)
					fieldResults[fieldName] = transformKeyValueOutput(result, outputFormat)
				}
			} else {
				// Normal field extraction
				fieldValue, err := engine.Extract(fieldConfig)
				if err != nil {
					// Use default value if extraction fails and it's defined
					if fieldConfig.DefaultValue != nil {
						fieldResults[fieldName] = fieldConfig.DefaultValue
					} else if defaultVal, ok := fieldConfigRaw.(map[string]interface{})["default"]; ok {
						fieldResults[fieldName] = defaultVal
					}
				} else {
					fieldResults[fieldName] = fieldValue
				}
			}
		}

		// Store in execution context
		extractedFieldNames := make([]string, 0, len(fieldResults))
		for fieldName, fieldValue := range fieldResults {
			input.ExecutionContext.Set(fieldName, fieldValue)
			extractedFieldNames = append(extractedFieldNames, fieldName)
		}
		// Set marker to indicate these fields should be saved
		input.ExecutionContext.Set("__extracted_fields__", extractedFieldNames)

		return &nodes.ExecutionOutput{
			Result: fieldResults,
		}, nil
	}

	// Normal extraction with top-level selector
	result, err := engine.Extract(config)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return &nodes.ExecutionOutput{
		Result: result,
	}, nil
}

// getOutputFormat extracts output_format from field config
func getOutputFormat(fieldConfigRaw interface{}) string {
	if configMap, ok := fieldConfigRaw.(map[string]interface{}); ok {
		if format, ok := configMap["output_format"].(string); ok {
			return format
		}
	}
	return "array" // default
}

// transformKeyValueOutput transforms key-value extraction results based on output format
func transformKeyValueOutput(result interface{}, format string) interface{} {
	pairs, ok := result.([]interface{})
	if !ok {
		return result
	}

	switch format {
	case "object":
		obj := make(map[string]interface{})
		for _, pair := range pairs {
			if p, ok := pair.(map[string]interface{}); ok {
				if key, ok := p["key"].(string); ok {
					obj[key] = p["value"]
				}
			}
		}
		return obj
	case "array_of_arrays":
		result := make([][]interface{}, 0, len(pairs))
		for _, pair := range pairs {
			if p, ok := pair.(map[string]interface{}); ok {
				result = append(result, []interface{}{p["key"], p["value"]})
			}
		}
		return result
	default:
		return pairs // array of objects
	}
}
