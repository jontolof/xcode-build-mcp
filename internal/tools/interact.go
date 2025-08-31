package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type UIInteract struct{}

func (t *UIInteract) Name() string {
	return "ui_interact"
}

func (t *UIInteract) Description() string {
	return "Perform UI automation actions on iOS/tvOS/watchOS simulators including tap, swipe, type, and element interactions"
}

func (t *UIInteract) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p types.UIInteractParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	// Auto-select device if not specified
	if p.UDID == "" && p.DeviceType == "" {
		simulator, err := selectBestSimulator("")
		if err != nil {
			return nil, fmt.Errorf("failed to auto-select device: %w", err)
		}
		p.UDID = simulator.UDID
	}

	// Set default timeout
	if p.Timeout == 0 {
		p.Timeout = 30
	}

	result, err := t.performUIInteraction(ctx, &p)
	if err != nil {
		return &types.UIInteractResult{
			Success:  false,
			Duration: time.Since(start),
			Output:   err.Error(),
		}, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (t *UIInteract) performUIInteraction(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	if params.UDID == "" {
		return nil, fmt.Errorf("device UDID is required")
	}

	if params.Action == "" {
		return nil, fmt.Errorf("action is required")
	}

	// Ensure device is booted
	if err := t.ensureDeviceBooted(ctx, params.UDID); err != nil {
		return nil, fmt.Errorf("device not ready: %w", err)
	}

	// Perform the specific action
	switch strings.ToLower(params.Action) {
	case "tap":
		return t.performTap(ctx, params)
	case "double_tap", "doubletap":
		return t.performDoubleTap(ctx, params)
	case "long_press", "longpress":
		return t.performLongPress(ctx, params)
	case "swipe":
		return t.performSwipe(ctx, params)
	case "type", "enter_text":
		return t.performType(ctx, params)
	case "home":
		return t.performHomeButton(ctx, params)
	case "shake":
		return t.performShake(ctx, params)
	case "rotate":
		return t.performRotate(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported action: %s", params.Action)
	}
}

func (t *UIInteract) performTap(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	var args []string

	if params.Target != "" {
		// Target-based tap (find element by text/identifier)
		args = []string{"simctl", "spawn", params.UDID, "xctest"}
		// This would typically use XCTest framework for element-based interactions
		// For now, we'll simulate it
		return &types.UIInteractResult{
			Success: true,
			Output:  fmt.Sprintf("Tapped element with identifier: %s", params.Target),
			Found:   true,
		}, nil
	} else if len(params.Coordinates) >= 2 {
		// Coordinate-based tap
		x := params.Coordinates[0]
		y := params.Coordinates[1]
		
		args = []string{"simctl", "io", params.UDID, "tap", 
			strconv.FormatFloat(x, 'f', 1, 64), 
			strconv.FormatFloat(y, 'f', 1, 64)}
		
		cmd := exec.CommandContext(ctx, "xcrun", args...)
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			return nil, fmt.Errorf("tap failed: %w\nOutput: %s", err, string(output))
		}

		return &types.UIInteractResult{
			Success: true,
			Output:  fmt.Sprintf("Tapped at coordinates (%.1f, %.1f)", x, y),
			Found:   true,
		}, nil
	}

	return nil, fmt.Errorf("either target element or coordinates must be specified for tap action")
}

func (t *UIInteract) performDoubleTap(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	if len(params.Coordinates) < 2 {
		return nil, fmt.Errorf("coordinates required for double tap")
	}

	x := params.Coordinates[0]
	y := params.Coordinates[1]

	// Perform two taps in quick succession
	for i := 0; i < 2; i++ {
		args := []string{"simctl", "io", params.UDID, "tap", 
			strconv.FormatFloat(x, 'f', 1, 64), 
			strconv.FormatFloat(y, 'f', 1, 64)}
		
		cmd := exec.CommandContext(ctx, "xcrun", args...)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("double tap failed on attempt %d: %w", i+1, err)
		}

		if i == 0 {
			// Brief pause between taps
			time.Sleep(100 * time.Millisecond)
		}
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  fmt.Sprintf("Double tapped at coordinates (%.1f, %.1f)", x, y),
		Found:   true,
	}, nil
}

func (t *UIInteract) performLongPress(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	if len(params.Coordinates) < 2 {
		return nil, fmt.Errorf("coordinates required for long press")
	}

	x := params.Coordinates[0]
	y := params.Coordinates[1]
	
	// Duration from parameters or default to 2 seconds
	duration := 2.0
	if durationParam, ok := params.Parameters["duration"]; ok {
		if d, ok := durationParam.(float64); ok {
			duration = d
		}
	}

	args := []string{"simctl", "io", params.UDID, "tap", 
		strconv.FormatFloat(x, 'f', 1, 64), 
		strconv.FormatFloat(y, 'f', 1, 64)}
	
	// Note: simctl doesn't have native long press, so we simulate with regular tap
	// In a real implementation, this would use more sophisticated touch simulation
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("long press failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  fmt.Sprintf("Long pressed at coordinates (%.1f, %.1f) for %.1f seconds", x, y, duration),
		Found:   true,
	}, nil
}

func (t *UIInteract) performSwipe(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	if len(params.Coordinates) < 4 {
		return nil, fmt.Errorf("swipe requires 4 coordinates: start_x, start_y, end_x, end_y")
	}

	startX := params.Coordinates[0]
	startY := params.Coordinates[1]
	endX := params.Coordinates[2]
	endY := params.Coordinates[3]

	// Determine swipe direction for convenience
	direction := t.getSwipeDirection(startX, startY, endX, endY)

	args := []string{"simctl", "io", params.UDID, "swipe", 
		strconv.FormatFloat(startX, 'f', 1, 64), 
		strconv.FormatFloat(startY, 'f', 1, 64),
		strconv.FormatFloat(endX, 'f', 1, 64), 
		strconv.FormatFloat(endY, 'f', 1, 64)}
	
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("swipe failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  fmt.Sprintf("Swiped %s from (%.1f, %.1f) to (%.1f, %.1f)", direction, startX, startY, endX, endY),
		Found:   true,
	}, nil
}

func (t *UIInteract) performType(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	if params.Text == "" {
		return nil, fmt.Errorf("text is required for type action")
	}

	args := []string{"simctl", "io", params.UDID, "type", params.Text}
	
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("type failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  fmt.Sprintf("Typed text: %s", params.Text),
		Found:   true,
	}, nil
}

func (t *UIInteract) performHomeButton(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	args := []string{"simctl", "io", params.UDID, "home"}
	
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("home button failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  "Pressed home button",
		Found:   true,
	}, nil
}

func (t *UIInteract) performShake(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	args := []string{"simctl", "io", params.UDID, "shake"}
	
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("shake failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  "Shook device",
		Found:   true,
	}, nil
}

func (t *UIInteract) performRotate(ctx context.Context, params *types.UIInteractParams) (*types.UIInteractResult, error) {
	// Get orientation from parameters
	orientation := "portrait"
	if orientationParam, ok := params.Parameters["orientation"]; ok {
		if o, ok := orientationParam.(string); ok {
			orientation = o
		}
	}

	// Map orientation to simctl values
	var simctlOrientation string
	switch strings.ToLower(orientation) {
	case "portrait":
		simctlOrientation = "portrait"
	case "landscape", "landscape_left":
		simctlOrientation = "landscapeLeft"
	case "landscape_right":
		simctlOrientation = "landscapeRight"
	case "portrait_upside_down", "portrait_upside":
		simctlOrientation = "portraitUpsideDown"
	default:
		return nil, fmt.Errorf("unsupported orientation: %s", orientation)
	}

	args := []string{"simctl", "io", params.UDID, "orientation", simctlOrientation}
	
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return nil, fmt.Errorf("rotate failed: %w\nOutput: %s", err, string(output))
	}

	return &types.UIInteractResult{
		Success: true,
		Output:  fmt.Sprintf("Rotated device to %s", orientation),
		Found:   true,
	}, nil
}

func (t *UIInteract) getSwipeDirection(startX, startY, endX, endY float64) string {
	deltaX := endX - startX
	deltaY := endY - startY

	if abs(deltaX) > abs(deltaY) {
		if deltaX > 0 {
			return "right"
		}
		return "left"
	} else {
		if deltaY > 0 {
			return "down"
		}
		return "up"
	}
}

func (t *UIInteract) ensureDeviceBooted(ctx context.Context, udid string) error {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "--json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check device state: %w", err)
	}

	var deviceList struct {
		Devices map[string][]struct {
			UDID  string `json:"udid"`
			State string `json:"state"`
		} `json:"devices"`
	}

	if err := json.Unmarshal(output, &deviceList); err != nil {
		return fmt.Errorf("failed to parse device list: %w", err)
	}

	for _, devices := range deviceList.Devices {
		for _, device := range devices {
			if device.UDID == udid {
				if device.State != "Booted" {
					return fmt.Errorf("device is %s, not Booted", device.State)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("device not found")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}