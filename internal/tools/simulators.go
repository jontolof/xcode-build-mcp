package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type ListSimulatorsTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewListSimulatorsTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *ListSimulatorsTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"platform": map[string]interface{}{
			"type":        "string",
			"description": "Platform to filter simulators (iOS, watchOS, tvOS)",
		},
		"device_type": map[string]interface{}{
			"type":        "string",
			"description": "Device type to filter (e.g., iPhone, iPad, Apple Watch)",
		},
		"runtime": map[string]interface{}{
			"type":        "string",
			"description": "Runtime version to filter (e.g., iOS 17.0)",
		},
		"available": map[string]interface{}{
			"type":        "boolean",
			"description": "Filter by availability (true for available only)",
		},
		"state": map[string]interface{}{
			"type":        "string",
			"description": "Filter by simulator state (Booted, Shutdown)",
			"enum":        []string{"Booted", "Shutdown", "Shutting Down", "Creating", "Booting"},
		},
	}, []string{})

	return &ListSimulatorsTool{
		name:        "list_simulators",
		description: "List available iOS, watchOS, and tvOS simulators with filtering options",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *ListSimulatorsTool) Name() string {
	return t.name
}

func (t *ListSimulatorsTool) Description() string {
	return t.description
}

func (t *ListSimulatorsTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *ListSimulatorsTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	t.logger.Printf("Listing simulators with filters: platform=%s, device_type=%s, runtime=%s, available=%v, state=%s", 
		params.Platform, params.DeviceType, params.Runtime, params.Available, params.State)

	// Execute xcrun simctl list command
	simulators, err := t.listSimulators(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to list simulators: %w", err)
	}

	duration := time.Since(start)
	t.logger.Printf("Listed %d simulators in %v", len(simulators), duration)

	result := &types.SimulatorListResult{
		Simulators: simulators,
		Duration:   duration,
	}

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

func (t *ListSimulatorsTool) parseParams(args map[string]interface{}) (*types.SimulatorListParams, error) {
	params := &types.SimulatorListParams{}

	// Parse platform
	if platform, err := parseStringParam(args, "platform", false); err != nil {
		return nil, err
	} else if platform != "" {
		params.Platform = platform
	}

	// Parse device_type
	if deviceType, err := parseStringParam(args, "device_type", false); err != nil {
		return nil, err
	} else if deviceType != "" {
		params.DeviceType = deviceType
	}

	// Parse runtime
	if runtime, err := parseStringParam(args, "runtime", false); err != nil {
		return nil, err
	} else if runtime != "" {
		params.Runtime = runtime
	}

	// Parse state
	if state, err := parseStringParam(args, "state", false); err != nil {
		return nil, err
	} else if state != "" {
		params.State = state
	}

	// Parse available boolean
	if available, exists := args["available"]; exists {
		if availableBool, ok := available.(bool); ok {
			params.Available = &availableBool
		} else {
			return nil, fmt.Errorf("available must be a boolean")
		}
	}

	return params, nil
}

func (t *ListSimulatorsTool) listSimulators(ctx context.Context, params *types.SimulatorListParams) ([]types.SimulatorInfo, error) {
	// Execute xcrun simctl list devices command
	args := []string{"xcrun", "simctl", "list", "devices", "--json"}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute simctl command: %w", err)
	}

	if !result.Success() {
		return nil, fmt.Errorf("simctl command failed with exit code %d: %s", result.ExitCode, result.StderrOutput)
	}

	// Parse the JSON output
	simulators, err := t.parseSimulatorListOutput(result.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse simulator list output: %w", err)
	}

	// Apply filters
	filteredSimulators := t.filterSimulators(simulators, params)

	return filteredSimulators, nil
}

func (t *ListSimulatorsTool) parseSimulatorListOutput(output string) ([]types.SimulatorInfo, error) {
	var simctlOutput struct {
		Devices map[string][]struct {
			UDID         string `json:"udid"`
			IsAvailable  bool   `json:"isAvailable"`
			DeviceTypeId string `json:"deviceTypeIdentifier"`
			State        string `json:"state"`
			Name         string `json:"name"`
		} `json:"devices"`
	}

	if err := json.Unmarshal([]byte(output), &simctlOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal simctl output: %w", err)
	}

	var simulators []types.SimulatorInfo

	// Parse each runtime and its devices
	for runtime, devices := range simctlOutput.Devices {
		platform := t.extractPlatformFromRuntime(runtime)
		
		for _, device := range devices {
			simulator := types.SimulatorInfo{
				UDID:       device.UDID,
				Name:       device.Name,
				DeviceType: t.extractDeviceTypeFromId(device.DeviceTypeId),
				Runtime:    runtime,
				State:      device.State,
				Available:  device.IsAvailable,
				Platform:   platform,
			}
			simulators = append(simulators, simulator)
		}
	}

	return simulators, nil
}

func (t *ListSimulatorsTool) extractPlatformFromRuntime(runtime string) string {
	runtime = strings.ToLower(runtime)
	if strings.Contains(runtime, "ios") {
		return "iOS"
	} else if strings.Contains(runtime, "watchos") {
		return "watchOS"
	} else if strings.Contains(runtime, "tvos") {
		return "tvOS"
	}
	return "Unknown"
}

func (t *ListSimulatorsTool) extractDeviceTypeFromId(deviceTypeId string) string {
	// Extract device type from identifier like "com.apple.CoreSimulator.SimDeviceType.iPhone-15-Pro"
	parts := strings.Split(deviceTypeId, ".")
	if len(parts) > 0 {
		deviceType := parts[len(parts)-1]
		// Convert from "iPhone-15-Pro" to "iPhone 15 Pro"
		deviceType = strings.ReplaceAll(deviceType, "-", " ")
		return deviceType
	}
	return deviceTypeId
}

func (t *ListSimulatorsTool) filterSimulators(simulators []types.SimulatorInfo, params *types.SimulatorListParams) []types.SimulatorInfo {
	var filtered []types.SimulatorInfo

	for _, simulator := range simulators {
		// Apply platform filter
		if params.Platform != "" {
			if !strings.EqualFold(simulator.Platform, params.Platform) {
				continue
			}
		}

		// Apply device type filter (case-insensitive partial match)
		if params.DeviceType != "" {
			if !strings.Contains(strings.ToLower(simulator.DeviceType), strings.ToLower(params.DeviceType)) {
				continue
			}
		}

		// Apply runtime filter (case-insensitive partial match)
		if params.Runtime != "" {
			if !strings.Contains(strings.ToLower(simulator.Runtime), strings.ToLower(params.Runtime)) {
				continue
			}
		}

		// Apply state filter
		if params.State != "" {
			if !strings.EqualFold(simulator.State, params.State) {
				continue
			}
		}

		// Apply availability filter
		if params.Available != nil {
			if simulator.Available != *params.Available {
				continue
			}
		}

		filtered = append(filtered, simulator)
	}

	return filtered
}