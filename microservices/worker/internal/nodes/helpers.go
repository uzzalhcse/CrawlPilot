package nodes

import "github.com/uzzalhcse/crawlify/microservices/shared/models"

// Shared helper functions for all node types

// getStringParam extracts a string parameter with a default value
func getStringParam(params map[string]interface{}, key, defaultVal string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultVal
}

// getIntParam extracts an int parameter with a default value
func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	return defaultVal
}

// getBoolParam extracts a bool parameter with a default value
func getBoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultVal
}

// parseNodeFromMap converts a map to a Node struct
func parseNodeFromMap(nodeMap map[string]interface{}) models.Node {
	node := models.Node{}

	if id, ok := nodeMap["id"].(string); ok {
		node.ID = id
	}
	if nodeType, ok := nodeMap["type"].(string); ok {
		node.Type = nodeType
	}
	if name, ok := nodeMap["name"].(string); ok {
		node.Name = name
	}
	if params, ok := nodeMap["params"].(map[string]interface{}); ok {
		node.Params = params
	}

	return node
}
