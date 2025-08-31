package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

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

// selectBestSimulator is a shared helper function to auto-select a booted simulator
// with proper timeout handling to prevent hanging in test environments
func selectBestSimulator(platform string) (*types.SimulatorInfo, error) {
	// Add timeout to prevent hanging in test environments
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list simulators (timeout or command failed): %w", err)
	}

	var simList struct {
		Devices map[string][]struct {
			UDID         string `json:"udid"`
			Name         string `json:"name"`
			State        string `json:"state"`
			DeviceTypeID string `json:"deviceTypeIdentifier"`
			IsAvailable  bool   `json:"isAvailable"`
		} `json:"devices"`
	}

	if err := json.Unmarshal(output, &simList); err != nil {
		return nil, fmt.Errorf("failed to parse simulator list: %w", err)
	}

	// Look for booted simulators first
	for runtime, devices := range simList.Devices {
		for _, device := range devices {
			if device.State == "Booted" && device.IsAvailable {
				return &types.SimulatorInfo{
					UDID:       device.UDID,
					Name:       device.Name,
					State:      device.State,
					DeviceType: device.DeviceTypeID,
					Runtime:    runtime,
					Available:  device.IsAvailable,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no booted simulators found (this is expected in test environments)")
}