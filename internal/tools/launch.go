package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type LaunchAppTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewLaunchAppTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *LaunchAppTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"bundle_id": map[string]interface{}{
			"type":        "string",
			"description": "Bundle identifier of the app to launch (e.g., com.example.MyApp)",
		},
		"udid": map[string]interface{}{
			"type":        "string",
			"description": "UDID of the target simulator or device (optional for auto-detection)",
		},
		"device_type": map[string]interface{}{
			"type":        "string",
			"description": "Device type filter for auto-selection if UDID not provided",
		},
		"arguments": map[string]interface{}{
			"type":        "array",
			"description": "Command line arguments to pass to the app",
			"items": map[string]interface{}{
				"type": "string",
			},
		},
		"environment": map[string]interface{}{
			"type":        "object",
			"description": "Environment variables to set for the app",
			"additionalProperties": map[string]interface{}{
				"type": "string",
			},
		},
		"wait_for_exit": map[string]interface{}{
			"type":        "boolean",
			"description": "Wait for the app to exit and capture its exit code (default: false)",
		},
	}, []string{"bundle_id"})

	return &LaunchAppTool{
		name:        "launch_app",
		description: "Launch iOS/tvOS/watchOS apps on simulators or devices",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *LaunchAppTool) Name() string {
	return t.name
}

func (t *LaunchAppTool) Description() string {
	return t.description
}

func (t *LaunchAppTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *LaunchAppTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	t.logger.Printf("Launching app %s", params.BundleID)

	// Resolve target device if not provided
	targetUDID := params.UDID
	if targetUDID == "" {
		detectedUDID, err := t.selectBestDevice(ctx, params.DeviceType)
		if err != nil {
			return "", fmt.Errorf("failed to auto-detect device: %w", err)
		}
		targetUDID = detectedUDID
		t.logger.Printf("Auto-selected device: %s", targetUDID)
	}

	// Launch the app
	result, err := t.launchApp(ctx, params, targetUDID)
	if err != nil {
		return "", fmt.Errorf("failed to launch app: %w", err)
	}

	duration := time.Since(start)
	result.Duration = duration
	
	t.logger.Printf("Successfully launched app %s in %v", params.BundleID, duration)

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

func (t *LaunchAppTool) parseParams(args map[string]interface{}) (*types.AppLaunchParams, error) {
	params := &types.AppLaunchParams{
		Arguments:   []string{},
		Environment: make(map[string]string),
		WaitForExit: false,
	}

	// Parse bundle_id (required)
	bundleID, err := parseStringParam(args, "bundle_id", true)
	if err != nil {
		return nil, err
	}
	params.BundleID = bundleID

	// Parse udid (optional)
	if udid, err := parseStringParam(args, "udid", false); err != nil {
		return nil, err
	} else if udid != "" {
		params.UDID = udid
	}

	// Parse device_type (optional)
	if deviceType, err := parseStringParam(args, "device_type", false); err != nil {
		return nil, err
	} else if deviceType != "" {
		params.DeviceType = deviceType
	}

	// Parse wait_for_exit (optional)
	params.WaitForExit = parseBoolParam(args, "wait_for_exit", false)

	// Parse arguments (optional)
	if arguments, err := parseArrayParam(args, "arguments"); err != nil {
		return nil, err
	} else if arguments != nil {
		for _, arg := range arguments {
			if strArg, ok := arg.(string); ok {
				params.Arguments = append(params.Arguments, strArg)
			}
		}
	}

	// Parse environment variables (optional)
	if env, exists := args["environment"]; exists {
		if envMap, ok := env.(map[string]interface{}); ok {
			for k, v := range envMap {
				if strVal, ok := v.(string); ok {
					params.Environment[k] = strVal
				}
			}
		}
	}

	return params, nil
}

func (t *LaunchAppTool) selectBestDevice(ctx context.Context, deviceTypeFilter string) (string, error) {
	// Get list of available booted simulators (similar to install_app)
	args := []string{"xcrun", "simctl", "list", "devices", "--json"}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return "", err
	}

	if !result.Success() {
		return "", fmt.Errorf("failed to list devices: %s", result.StderrOutput)
	}

	// Parse JSON output (same structure as install_app)
	var simctlOutput struct {
		Devices map[string][]struct {
			UDID         string `json:"udid"`
			Name         string `json:"name"`
			State        string `json:"state"`
			IsAvailable  bool   `json:"isAvailable"`
			DeviceTypeId string `json:"deviceTypeIdentifier"`
		} `json:"devices"`
	}

	if err := json.Unmarshal([]byte(result.Output), &simctlOutput); err != nil {
		return "", fmt.Errorf("failed to parse device list: %w", err)
	}

	// Find booted devices (we can only launch apps on booted devices)
	var candidates []struct {
		UDID       string
		Name       string
		DeviceType string
		Score      int
	}

	for runtime, devices := range simctlOutput.Devices {
		platform := "iOS"
		if strings.Contains(strings.ToLower(runtime), "watchos") {
			platform = "watchOS"
		} else if strings.Contains(strings.ToLower(runtime), "tvos") {
			platform = "tvOS"
		}

		for _, device := range devices {
			if !device.IsAvailable || device.State != "Booted" {
				continue // Only consider booted devices
			}

			deviceType := extractDeviceTypeFromId(device.DeviceTypeId)
			
			// Apply device type filter if provided
			if deviceTypeFilter != "" && !strings.Contains(strings.ToLower(deviceType), strings.ToLower(deviceTypeFilter)) {
				continue
			}

			score := 0
			
			// Prefer iOS devices
			if platform == "iOS" {
				score += 100
			} else if platform == "tvOS" {
				score += 50
			}
			
			// Prefer iPhone over iPad
			if strings.Contains(deviceType, "iPhone") {
				score += 30
			} else if strings.Contains(deviceType, "iPad") {
				score += 20
			}

			candidates = append(candidates, struct {
				UDID       string
				Name       string
				DeviceType string
				Score      int
			}{
				UDID:       device.UDID,
				Name:       device.Name,
				DeviceType: deviceType,
				Score:      score,
			})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no suitable booted devices found")
	}

	// Sort by score (highest first)
	bestCandidate := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Score > bestCandidate.Score {
			bestCandidate = candidate
		}
	}

	t.logger.Printf("Selected device: %s (%s)", bestCandidate.Name, bestCandidate.DeviceType)
	return bestCandidate.UDID, nil
}

func (t *LaunchAppTool) launchApp(ctx context.Context, params *types.AppLaunchParams, udid string) (*types.AppLaunchResult, error) {
	args := []string{"xcrun", "simctl", "launch"}
	
	// Add wait flag if requested
	if params.WaitForExit {
		args = append(args, "--wait-for-debugger")
	}
	
	// Add device UDID
	args = append(args, udid)
	
	// Add bundle ID
	args = append(args, params.BundleID)
	
	// Add arguments if provided
	if len(params.Arguments) > 0 {
		args = append(args, params.Arguments...)
	}

	result := &types.AppLaunchResult{
		Success: false,
	}

	// Set environment variables if provided
	if len(params.Environment) > 0 {
		// Note: simctl launch doesn't directly support environment variables
		// This would require a more complex approach using simctl spawn
		t.logger.Printf("Warning: Environment variables not fully supported in launch command")
	}

	cmdResult, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		result.Output = fmt.Sprintf("Command execution error: %v", err)
		return result, err
	}

	result.Output = cmdResult.Output
	if cmdResult.StderrOutput != "" {
		result.Output += "\n" + cmdResult.StderrOutput
	}

	if !cmdResult.Success() {
		errorOutput := cmdResult.StderrOutput
		if errorOutput == "" {
			errorOutput = cmdResult.Output
		}
		
		// Check for common error cases and provide better error messages
		if strings.Contains(errorOutput, "Unable to launch") {
			return result, fmt.Errorf("unable to launch app: %s", strings.TrimSpace(errorOutput))
		} else if strings.Contains(errorOutput, "device is not booted") {
			return result, fmt.Errorf("target device is not booted")
		} else if strings.Contains(errorOutput, "App is not installed") {
			return result, fmt.Errorf("app with bundle ID %s is not installed on the device", params.BundleID)
		}
		
		return result, fmt.Errorf("launch failed (exit code %d): %s", cmdResult.ExitCode, strings.TrimSpace(errorOutput))
	}

	result.Success = true
	
	// Try to extract process ID from output
	processID, err := t.extractProcessID(cmdResult.Output)
	if err == nil {
		result.ProcessID = processID
	}

	// If waiting for exit, the exit code should be available
	if params.WaitForExit {
		exitCode := cmdResult.ExitCode
		result.ExitCode = &exitCode
	}

	return result, nil
}

func (t *LaunchAppTool) extractProcessID(output string) (int, error) {
	// Look for patterns like "Process ID: 12345" or similar
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Try different patterns for process ID
		if strings.Contains(line, "Process ID:") || strings.Contains(line, "PID:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "ID:" || part == "PID:") && i+1 < len(parts) {
					if pid, err := strconv.Atoi(parts[i+1]); err == nil {
						return pid, nil
					}
				}
			}
		}
		
		// Try to find standalone numbers that might be PIDs
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				pidStr := strings.TrimSpace(parts[len(parts)-1])
				if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
					return pid, nil
				}
			}
		}
	}
	
	return 0, fmt.Errorf("process ID not found in output")
}

// extractDeviceTypeFromId is defined in install.go