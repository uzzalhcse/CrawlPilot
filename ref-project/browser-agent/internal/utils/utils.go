package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/encoder"
)

func FileExists(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return !stat.IsDir()
}

func ConvertToStringMap(inputMap map[string]any) map[string]string {
	resultMap := make(map[string]string)
	if inputMap == nil {
		return resultMap
	}
	for key, value := range inputMap {
		if strValue, ok := value.(string); ok {
			resultMap[key] = strValue
		}
	}
	return resultMap
}

func GetDefaultValue[T any](data map[string]any, key string, defaultValue T) T {
	if value, ok := data[key]; ok {
		if value, ok := value.(T); ok {
			return value
		}
	}
	return defaultValue
}

func ConvertToSliceOfInt(input any) ([]int, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	slice, ok := input.([]any)
	if !ok {
		return nil, fmt.Errorf("input is not a slice")
	}
	result := make([]int, 0, len(slice))

	for i, elem := range slice {
		if elem == nil {
			return nil, fmt.Errorf("element at index %d is nil", i)
		}
		if strValue, ok := elem.(string); ok {
			value, err := strconv.Atoi(strValue)
			if err != nil {
				return nil, fmt.Errorf("failed to convert string to int: %s", err)
			}
			result = append(result, value)
			continue
		}
		value, ok := elem.(int)
		if !ok {
			if floatValue, ok := elem.(float64); ok {
				result = append(result, int(floatValue))
				continue
			}
			return nil, fmt.Errorf("element at index %d is not of type int", i)
		}
		result = append(result, value)
	}
	return result, nil
}

func ConvertToOptional[T any](value any) *T {
	if value, ok := value.(T); ok {
		return &value
	}
	return nil
}

func StringifyJSON(v any) (string, error) {
	b, e := encoder.Encode(v, encoder.EscapeHTML)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

func ParseJSON(data string, v any) error {
	return sonic.UnmarshalString(data, v)
}

func ModelDump(v any) (map[string]interface{}, error) {
	b, e := encoder.Encode(v, encoder.EscapeHTML)
	if e != nil {
		return nil, e
	}
	var dict map[string]interface{}
	e = sonic.Unmarshal(b, &dict)
	if e != nil {
		return nil, e
	}
	return dict, nil
}

// Add this new function to utils.go
func ConvertToInt(value interface{}) int {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	case int32:
		return int(v)
	default:
		return 0
	}
}
