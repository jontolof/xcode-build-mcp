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

type SimulatorControlTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewSimulatorControlTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *SimulatorControlTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"udid": map[string]interface{}{
			"type":        "string",
			"description": "UDID of the simulator to control",
		},
		"action": map[string]interface{}{
			"type":        "string",
			"description": "Action to perform on the simulator",
			"enum":        []string{"boot", "shutdown", "reset", "erase"},
		},
		"timeout": map[string]interface{}{
			"type":        "integer",
			"description": "Timeout in seconds for the operation (default: 30)",
			"minimum":     1,
			"maximum":     300,
		},
	}, []string{"udid", "action"})

	return &SimulatorControlTool{
		name:        "simulator_control",
		description: "Control iOS simulators - boot, shutdown, reset, or erase simulators",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *SimulatorControlTool) Name() string {
	return t.name
}

func (t *SimulatorControlTool) Description() string {
	return t.description
}

func (t *SimulatorControlTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *SimulatorControlTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	t.logger.Printf("Performing %s action on simulator %s", params.Action, params.UDID)

	// Get the current state before performing the action
	previousState, err := t.getSimulatorState(ctx, params.UDID)
	if err != nil {
		t.logger.Printf("Warning: Could not get initial simulator state: %v", err)
		previousState = "Unknown"
	}

	// Perform the simulator control action
	output, err := t.performSimulatorAction(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to perform %s action: %w", params.Action, err)
	}

	// Get the state after performing the action
	currentState, err := t.getSimulatorState(ctx, params.UDID)
	if err != nil {
		t.logger.Printf("Warning: Could not get final simulator state: %v", err)
		currentState = "Unknown"
	}

	duration := time.Since(start)
	t.logger.Printf("Completed %s action on simulator %s in %v", params.Action, params.UDID, duration)

	result := &types.SimulatorControlResult{
		Success:       true,
		Duration:      duration,
		Output:        output,
		PreviousState: previousState,
		CurrentState:  currentState,
	}

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

func (t *SimulatorControlTool) parseParams(args map[string]interface{}) (*types.SimulatorControlParams, error) {
	params := &types.SimulatorControlParams{
		Timeout: 30, // Default timeout of 30 seconds
	}

	// Parse UDID (required)
	udid, err := parseStringParam(args, "udid", true)
	if err != nil {
		return nil, err
	}
	params.UDID = udid

	// Parse action (required)
	action, err := parseStringParam(args, "action", true)
	if err != nil {
		return nil, err
	}

	// Validate action
	validActions := map[string]bool{
		"boot":     true,
		"shutdown": true,
		"reset":    true,
		"erase":    true,
	}

	if !validActions[action] {
		return nil, fmt.Errorf("invalid action '%s'. Valid actions are: boot, shutdown, reset, erase", action)
	}
	params.Action = action

	// Parse timeout (optional)
	if timeout, exists := args["timeout"]; exists {
		if timeoutFloat, ok := timeout.(float64); ok {
			params.Timeout = int(timeoutFloat)
		} else if timeoutInt, ok := timeout.(int); ok {
			params.Timeout = timeoutInt
		} else {
			return nil, fmt.Errorf("timeout must be a number")
		}
	}

	return params, nil
}

func (t *SimulatorControlTool) performSimulatorAction(ctx context.Context, params *types.SimulatorControlParams) (string, error) {
	var args []string

	switch params.Action {
	case "boot":
		args = []string{"xcrun", "simctl", "boot", params.UDID}
	case "shutdown":
		args = []string{"xcrun", "simctl", "shutdown", params.UDID}
	case "reset":
		// Reset is an alias for erase in simctl
		args = []string{"xcrun", "simctl", "erase", params.UDID}
	case "erase":
		args = []string{"xcrun", "simctl", "erase", params.UDID}
	default:
		return "", fmt.Errorf("unsupported action: %s", params.Action)
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	result, err := t.executor.ExecuteCommand(timeoutCtx, args)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("operation timed out after %d seconds", params.Timeout)
		}
		return "", err
	}

	if !result.Success() {
		errorOutput := result.StderrOutput
		if errorOutput == "" {
			errorOutput = result.Output
		}

		// Check for common error cases and provide better error messages
		if strings.Contains(errorOutput, "Unable to boot device") {
			return "", fmt.Errorf("unable to boot simulator (may already be booted): %s", strings.TrimSpace(errorOutput))
		} else if strings.Contains(errorOutput, "Unable to shutdown device") {
			return "", fmt.Errorf("unable to shutdown simulator (may already be shutdown): %s", strings.TrimSpace(errorOutput))
		} else if strings.Contains(errorOutput, "No device found") {
			return "", fmt.Errorf("simulator with UDID %s not found", params.UDID)
		}

		return "", fmt.Errorf("simctl command failed (exit code %d): %s", result.ExitCode, strings.TrimSpace(errorOutput))
	}

	return result.Output, nil
}

func (t *SimulatorControlTool) getSimulatorState(ctx context.Context, udid string) (string, error) {
	args := []string{"xcrun", "simctl", "list", "devices", "--json"}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return "", err
	}

	if !result.Success() {
		return "", fmt.Errorf("failed to get simulator state: %s", result.StderrOutput)
	}

	// Parse JSON output to find the simulator state
	var simctlOutput struct {
		Devices map[string][]struct {
			UDID  string `json:"udid"`
			State string `json:"state"`
		} `json:"devices"`
	}

	if err := json.Unmarshal([]byte(result.Output), &simctlOutput); err != nil {
		return "", fmt.Errorf("failed to parse simctl output: %w", err)
	}

	// Find the simulator with the given UDID
	for _, devices := range simctlOutput.Devices {
		for _, device := range devices {
			if device.UDID == udid {
				return device.State, nil
			}
		}
	}

	return "", fmt.Errorf("simulator with UDID %s not found", udid)
}
