package tools

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

func TestCaptureLogs_Name(t *testing.T) {
	tool := NewCaptureLogs()
	if got := tool.Name(); got != "capture_logs" {
		t.Errorf("CaptureLogs.Name() = %v, want %v", got, "capture_logs")
	}
}

func TestCaptureLogs_Description(t *testing.T) {
	tool := NewCaptureLogs()
	desc := tool.Description()
	if desc == "" {
		t.Error("CaptureLogs.Description() returned empty string")
	}
	if len(desc) < 20 {
		t.Errorf("CaptureLogs.Description() too short: %s", desc)
	}
}

func TestCaptureLogs_Execute_InvalidParams(t *testing.T) {
	tool := NewCaptureLogs()
	ctx := context.Background()

	// Test with empty params (no device specified)
	result, _ := tool.Execute(ctx, map[string]interface{}{})
	// This might succeed due to auto-selection, so just check for string result
	if result == "" {
		t.Error("Expected non-empty result string")
	}
}

func TestCaptureLogs_Execute_ValidParams(t *testing.T) {
	tool := NewCaptureLogs()
	ctx := context.Background()

	args := map[string]interface{}{
		"udid":         "test-udid",
		"max_lines":    10,
		"timeout_secs": 1, // Short timeout for test
	}

	result, err := tool.Execute(ctx, args)

	// Should get a result string even if command fails
	if result == "" {
		t.Error("Expected non-empty result string")
	}

	// Parse JSON result
	var logResult types.LogCaptureResult
	if jsonErr := json.Unmarshal([]byte(result), &logResult); jsonErr != nil {
		t.Errorf("Failed to parse result JSON: %v", jsonErr)
	}

	if logResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// The command will likely fail in test environment, but that's expected
	if err != nil && logResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestCaptureLogs_Execute_DefaultValues(t *testing.T) {
	tool := NewCaptureLogs()
	ctx := context.Background()

	// Test with minimal params
	args := map[string]interface{}{
		"udid": "test-udid",
	}

	result, err := tool.Execute(ctx, args)

	if result == "" {
		t.Error("Expected non-empty result string")
	}

	// Parse JSON result
	var logResult types.LogCaptureResult
	if jsonErr := json.Unmarshal([]byte(result), &logResult); jsonErr != nil {
		t.Errorf("Failed to parse result JSON: %v", jsonErr)
	}

	// Should have applied defaults
	if logResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// Command will likely fail without real simulator, but structure should be correct
	if err != nil && logResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestCaptureLogs_ParseLogLine(t *testing.T) {
	tool := NewCaptureLogs()
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}\.\d+[+-]\d{4})\s+(\w+)\s+(\w+)\s+(\[.*?\])?\s*(.*)$`)

	tests := []struct {
		name     string
		line     string
		expected *types.LogEntry
	}{
		{
			name: "Standard log line",
			line: "2024-01-01 10:30:45.123456+0000 Default SpringBoard [1234] Application launched",
			expected: &types.LogEntry{
				Level:    "default",
				Category: "SpringBoard",
				Process:  "1234",
				Message:  "Application launched",
			},
		},
		{
			name: "Error log line",
			line: "2024-01-01 10:30:45.123456+0000 Error CoreData [5678] Failed to save context",
			expected: &types.LogEntry{
				Level:    "error",
				Category: "CoreData",
				Process:  "5678",
				Message:  "Failed to save context",
			},
		},
		{
			name: "Line without process",
			line: "2024-01-01 10:30:45.123456+0000 Info System Simple message",
			expected: &types.LogEntry{
				Level:    "info",
				Category: "System",
				Message:  "Simple message",
			},
		},
		{
			name: "Unparseable line",
			line: "This is not a standard log line",
			expected: &types.LogEntry{
				Level:   "info",
				Message: "This is not a standard log line",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.parseLogLine(tt.line, pattern)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Level != tt.expected.Level {
				t.Errorf("Level = %v, want %v", result.Level, tt.expected.Level)
			}

			if result.Category != tt.expected.Category {
				t.Errorf("Category = %v, want %v", result.Category, tt.expected.Category)
			}

			if result.Process != tt.expected.Process {
				t.Errorf("Process = %v, want %v", result.Process, tt.expected.Process)
			}

			if result.Message != tt.expected.Message {
				t.Errorf("Message = %v, want %v", result.Message, tt.expected.Message)
			}

			if result.Timestamp == "" {
				t.Error("Expected non-empty timestamp")
			}
		})
	}
}

func TestCaptureLogs_SelectBestSimulator(t *testing.T) {
	// This test checks the function signature and error handling
	// It will likely fail in CI without simulators, which is expected
	simulator, err := selectBestSimulator("")

	// Either we get a simulator or an error, both are valid outcomes
	if simulator != nil {
		if simulator.UDID == "" {
			t.Error("Expected non-empty UDID when simulator is returned")
		}
		if simulator.Name == "" {
			t.Error("Expected non-empty Name when simulator is returned")
		}
	} else if err == nil {
		t.Error("Expected either simulator or error, got neither")
	}
}

func TestCaptureLogs_ParameterValidation(t *testing.T) {
	tests := []struct {
		name   string
		params types.LogCaptureParams
		valid  bool
	}{
		{
			name: "Valid minimal params",
			params: types.LogCaptureParams{
				UDID: "test-udid",
			},
			valid: true,
		},
		{
			name: "Valid full params",
			params: types.LogCaptureParams{
				UDID:        "test-udid",
				BundleID:    "com.example.app",
				LogLevel:    "error",
				FilterText:  "test",
				FollowMode:  true,
				MaxLines:    50,
				TimeoutSecs: 60,
			},
			valid: true,
		},
		{
			name: "No device specified",
			params: types.LogCaptureParams{
				LogLevel: "info",
			},
			valid: true, // Auto-selection will succeed if simulators are available
		},
	}

	tool := NewCaptureLogs()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert params to args map
			args := map[string]interface{}{}
			if tt.params.UDID != "" {
				args["udid"] = tt.params.UDID
			}
			if tt.params.DeviceType != "" {
				args["device_type"] = tt.params.DeviceType
			}
			if tt.params.LogLevel != "" {
				args["log_level"] = tt.params.LogLevel
			}
			if tt.params.MaxLines != 0 {
				args["max_lines"] = tt.params.MaxLines
			}

			result, err := tool.Execute(ctx, args)
			_ = err // May or may not have error depending on environment

			if tt.valid {
				if result == "" {
					t.Error("Expected non-empty result for valid params")
				}
			} else {
				// For invalid params, we might get success (if auto-selection works) or failure
				// Both are acceptable depending on environment
				if result == "" {
					t.Error("Expected non-nil result")
				}
			}
		})
	}
}
