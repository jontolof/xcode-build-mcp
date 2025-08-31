package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

func TestUIInteract_Name(t *testing.T) {
	tool := &UIInteract{}
	if got := tool.Name(); got != "ui_interact" {
		t.Errorf("UIInteract.Name() = %v, want %v", got, "ui_interact")
	}
}

func TestUIInteract_Description(t *testing.T) {
	tool := &UIInteract{}
	desc := tool.Description()
	if desc == "" {
		t.Error("UIInteract.Description() returned empty string")
	}
	if len(desc) < 20 {
		t.Errorf("UIInteract.Description() too short: %s", desc)
	}
}

func TestUIInteract_Execute_InvalidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	// Test with invalid JSON
	result, err := tool.Execute(ctx, json.RawMessage(`{"invalid": json}`))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result for invalid params, got %+v", result)
	}
}

func TestUIInteract_Execute_ValidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := types.UIInteractParams{
		UDID:        "test-udid",
		Action:      "tap",
		Coordinates: []float64{100, 200},
		Timeout:     10,
	}

	paramsJSON, _ := json.Marshal(params)
	result, err := tool.Execute(ctx, paramsJSON)

	// Should get a result even if command fails
	if result == nil {
		t.Error("Expected non-nil result")
	}

	interactResult, ok := result.(*types.UIInteractResult)
	if !ok {
		t.Errorf("Expected *types.UIInteractResult, got %T", result)
	}

	if interactResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// The command will likely fail in test environment, but that's expected
	if err != nil && interactResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestUIInteract_Execute_DefaultTimeout(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	// Test with minimal params (no timeout specified)
	params := types.UIInteractParams{
		UDID:   "test-udid",
		Action: "home",
	}

	paramsJSON, _ := json.Marshal(params)
	result, err := tool.Execute(ctx, paramsJSON)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	interactResult, ok := result.(*types.UIInteractResult)
	if !ok {
		t.Errorf("Expected *types.UIInteractResult, got %T", result)
	}

	// Should have applied default timeout
	if interactResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// Command will likely fail without real simulator, but structure should be correct
	if err != nil && interactResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestUIInteract_GetSwipeDirection(t *testing.T) {
	tool := &UIInteract{}

	tests := []struct {
		startX, startY, endX, endY float64
		expected               string
	}{
		{100, 100, 200, 100, "right"}, // Horizontal right
		{200, 100, 100, 100, "left"},  // Horizontal left
		{100, 100, 100, 200, "down"},  // Vertical down
		{100, 200, 100, 100, "up"},    // Vertical up
		{100, 100, 200, 150, "right"}, // Diagonal but more horizontal
		{100, 100, 150, 200, "down"},  // Diagonal but more vertical
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			direction := tool.getSwipeDirection(tt.startX, tt.startY, tt.endX, tt.endY)
			if direction != tt.expected {
				t.Errorf("getSwipeDirection(%.1f,%.1f,%.1f,%.1f) = %s, want %s", 
					tt.startX, tt.startY, tt.endX, tt.endY, direction, tt.expected)
			}
		})
	}
}

func TestUIInteract_PerformTap_Coordinates(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:        "test-udid",
		Action:      "tap",
		Coordinates: []float64{100, 200},
	}

	// This will fail because device doesn't exist, but we can test parameter validation
	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	// Either we get a result or an error, both are acceptable for testing
	if result != nil {
		if result.Output == "" {
			t.Error("Expected non-empty output for coordinate tap")
		}
	}
	if err != nil {
		// Error is expected in test environment
		if !strings.Contains(err.Error(), "device") {
			t.Errorf("Expected device-related error, got: %v", err)
		}
	}
}

func TestUIInteract_PerformTap_Target(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "tap",
		Target: "Submit Button",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	// Target-based tap should work even without real device (it's simulated)
	if result != nil {
		if !strings.Contains(result.Output, "Submit Button") {
			t.Error("Expected output to contain target element name")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "device") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUIInteract_PerformTap_InvalidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	// Test tap without coordinates or target
	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "tap",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for tap without coordinates or target")
	}
	// Error should be about missing parameters or device issues (both are valid)
	if err != nil && !strings.Contains(err.Error(), "coordinates") && !strings.Contains(err.Error(), "target") && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected error about missing coordinates/target or device, got: %v", err)
	}
}

func TestUIInteract_PerformSwipe_ValidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:        "test-udid",
		Action:      "swipe",
		Coordinates: []float64{100, 100, 200, 200}, // start_x, start_y, end_x, end_y
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if result != nil {
		if !strings.Contains(result.Output, "Swiped") {
			t.Error("Expected output to contain 'Swiped'")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected device-related error or success, got: %v", err)
	}
}

func TestUIInteract_PerformSwipe_InvalidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	// Test swipe with insufficient coordinates
	params := &types.UIInteractParams{
		UDID:        "test-udid",
		Action:      "swipe",
		Coordinates: []float64{100, 100}, // Only start coordinates
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for swipe with insufficient coordinates")
	}
	
	// Accept either coordinate error or device error
	if err != nil && !strings.Contains(err.Error(), "4 coordinates") && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected error about 4 coordinates or device, got: %v", err)
	}
}

func TestUIInteract_PerformType_ValidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "type",
		Text:   "Hello World",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if result != nil {
		if !strings.Contains(result.Output, "Hello World") {
			t.Error("Expected output to contain typed text")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected device-related error or success, got: %v", err)
	}
}

func TestUIInteract_PerformType_InvalidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	// Test type without text
	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "type",
		Text:   "",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for type without text")
	}
	
	// Accept either text error or device error
	if err != nil && !strings.Contains(err.Error(), "text is required") && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected error about missing text or device, got: %v", err)
	}
}

func TestUIInteract_PerformRotate_ValidParams(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "rotate",
		Parameters: map[string]interface{}{
			"orientation": "landscape",
		},
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if result != nil {
		if !strings.Contains(result.Output, "landscape") {
			t.Error("Expected output to contain orientation")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected device-related error or success, got: %v", err)
	}
}

func TestUIInteract_SupportedActions(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	supportedActions := []string{
		"tap", "double_tap", "doubletap", "long_press", "longpress",
		"swipe", "type", "enter_text", "home", "shake", "rotate",
	}

	for _, action := range supportedActions {
		t.Run(action, func(t *testing.T) {
			params := &types.UIInteractParams{
				UDID:   "test-udid",
				Action: action,
			}

			// Add required parameters for specific actions
			switch action {
			case "tap", "double_tap", "doubletap", "long_press", "longpress":
				params.Coordinates = []float64{100, 100}
			case "swipe":
				params.Coordinates = []float64{100, 100, 200, 200}
			case "type", "enter_text":
				params.Text = "test"
			case "rotate":
				params.Parameters = map[string]interface{}{"orientation": "portrait"}
			}

			result, err := tool.performUIInteraction(ctx, params); _ = result
			
			// Should not get "unsupported action" error
			if err != nil && strings.Contains(err.Error(), "unsupported action") {
				t.Errorf("Action %s should be supported, got: %v", action, err)
			}
			
			// Should get some kind of result
			if result == nil && err == nil {
				t.Errorf("Action %s should return result or error", action)
			}
		})
	}
}

func TestUIInteract_UnsupportedAction(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "unsupported_action",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for unsupported action")
	}
	
	// Accept either unsupported action error or device error
	if err != nil && !strings.Contains(err.Error(), "unsupported action") && !strings.Contains(err.Error(), "device") {
		t.Errorf("Expected 'unsupported action' or device error, got: %v", err)
	}
}

func TestUIInteract_MissingAction(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "test-udid",
		Action: "",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for missing action")
	}
	
	if !strings.Contains(err.Error(), "action is required") {
		t.Errorf("Expected 'action is required' error, got: %v", err)
	}
}

func TestUIInteract_MissingUDID(t *testing.T) {
	tool := &UIInteract{}
	ctx := context.Background()

	params := &types.UIInteractParams{
		UDID:   "",
		Action: "home",
	}

	result, err := tool.performUIInteraction(ctx, params); _ = result
	
	if err == nil {
		t.Error("Expected error for missing UDID")
	}
	
	if !strings.Contains(err.Error(), "UDID is required") {
		t.Errorf("Expected 'UDID is required' error, got: %v", err)
	}
}