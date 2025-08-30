package tools

import "fmt"

// Helper functions for parameter parsing
func parseStringParam(args map[string]interface{}, key string, required bool) (string, error) {
	value, exists := args[key]
	if !exists {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", key)
	}

	return str, nil
}

func parseBoolParam(args map[string]interface{}, key string, defaultValue bool) bool {
	value, exists := args[key]
	if !exists {
		return defaultValue
	}

	boolVal, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return boolVal
}

func parseArrayParam(args map[string]interface{}, key string) ([]interface{}, error) {
	value, exists := args[key]
	if !exists {
		return nil, nil
	}

	array, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an array", key)
	}

	return array, nil
}

func createJSONSchema(schemaType string, properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       schemaType,
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}