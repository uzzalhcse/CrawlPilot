package browser

import (
	"github.com/uzzalhcse/crawlify/internal/extraction"
)

// ConvertWorkflowFieldsToSelectedFields converts workflow extraction fields to SelectedField format
// This is used when editing existing workflows to pre-populate the visual selector overlay
func ConvertWorkflowFieldsToSelectedFields(fields map[string]interface{}) []SelectedField {
	var selectedFields []SelectedField

	for fieldName, fieldConfig := range fields {
		if extractConfig, ok := fieldConfig.(extraction.ExtractConfig); ok {
			// Handle key-value pairs mode
			if len(extractConfig.Extractions) > 0 {
				selectedField := SelectedField{
					Name: fieldName,
					Mode: "key-value-pairs",
					Attributes: &FieldAttributesConfig{
						Extractions: make([]ExtractionPairConfig, 0),
					},
				}

				for _, pair := range extractConfig.Extractions {
					transform := ""
					if pair.Transform != nil {
						if t, ok := pair.Transform.(string); ok {
							transform = t
						}
					}
					selectedField.Attributes.Extractions = append(selectedField.Attributes.Extractions, ExtractionPairConfig{
						KeySelector:    pair.KeySelector,
						ValueSelector:  pair.ValueSelector,
						KeyType:        pair.KeyType,
						ValueType:      pair.ValueType,
						KeyAttribute:   pair.KeyAttribute,
						ValueAttribute: pair.ValueAttribute,
						Transform:      transform,
					})
				}

				selectedFields = append(selectedFields, selectedField)
			} else {
				// Handle regular fields (single/list)
				mode := "single"
				if extractConfig.Multiple {
					mode = "list"
				}

				selectedField := SelectedField{
					Name:      fieldName,
					Selector:  extractConfig.Selector,
					Type:      extractConfig.Type,
					Attribute: extractConfig.Attribute,
					Multiple:  extractConfig.Multiple,
					Mode:      mode,
				}

				selectedFields = append(selectedFields, selectedField)
			}
		}
	}

	return selectedFields
}
