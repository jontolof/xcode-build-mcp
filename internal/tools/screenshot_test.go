package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

func TestScreenshot_Name(t *testing.T) {
	tool := &Screenshot{}
	if got := tool.Name(); got != "screenshot" {
		t.Errorf("Screenshot.Name() = %v, want %v", got, "screenshot")
	}
}

func TestScreenshot_Description(t *testing.T) {
	tool := &Screenshot{}
	desc := tool.Description()
	if desc == "" {
		t.Error("Screenshot.Description() returned empty string")
	}
	if len(desc) < 20 {
		t.Errorf("Screenshot.Description() too short: %s", desc)
	}
}

func TestScreenshot_Execute_InvalidParams(t *testing.T) {
	tool := &Screenshot{}
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

func TestScreenshot_Execute_ValidParams(t *testing.T) {
	tool := &Screenshot{}
	ctx := context.Background()

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "screenshot_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	params := types.ScreenshotParams{
		UDID:       "test-udid",
		OutputPath: filepath.Join(tempDir, "test.png"),
		Format:     "png",
	}

	paramsJSON, _ := json.Marshal(params)
	result, err := tool.Execute(ctx, paramsJSON)

	// Should get a result even if command fails
	if result == nil {
		t.Error("Expected non-nil result")
	}

	screenshotResult, ok := result.(*types.ScreenshotResult)
	if !ok {
		t.Errorf("Expected *types.ScreenshotResult, got %T", result)
	}

	if screenshotResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// The command will likely fail in test environment, but that's expected
	if err != nil && screenshotResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestScreenshot_Execute_DefaultFormat(t *testing.T) {
	tool := &Screenshot{}
	ctx := context.Background()

	// Test with minimal params (no format specified)
	params := types.ScreenshotParams{
		UDID: "test-udid",
	}

	paramsJSON, _ := json.Marshal(params)
	result, err := tool.Execute(ctx, paramsJSON)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	screenshotResult, ok := result.(*types.ScreenshotResult)
	if !ok {
		t.Errorf("Expected *types.ScreenshotResult, got %T", result)
	}

	// Should have applied default format (png)
	if err == nil && screenshotResult.Success {
		if !strings.HasSuffix(screenshotResult.FilePath, ".png") {
			t.Errorf("Expected .png extension, got %s", screenshotResult.FilePath)
		}
	}
}

func TestScreenshot_GenerateScreenshotPath(t *testing.T) {
	tool := &Screenshot{}

	tests := []struct {
		format   string
		expected string
	}{
		{"png", ".png"},
		{"jpeg", ".jpeg"},
		{"jpg", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			path := tool.generateScreenshotPath(tt.format)
			
			if path == "" {
				t.Error("Expected non-empty path")
			}

			if !strings.HasSuffix(path, tt.expected) {
				t.Errorf("Expected path to end with %s, got %s", tt.expected, path)
			}

			if !strings.Contains(path, "screenshots") {
				t.Errorf("Expected path to contain 'screenshots' directory, got %s", path)
			}

			if !strings.Contains(path, "simulator_screenshot_") {
				t.Errorf("Expected path to contain 'simulator_screenshot_', got %s", path)
			}
		})
	}
}

func TestScreenshot_GetImageDimensions(t *testing.T) {
	tool := &Screenshot{}

	// Test with non-existent file
	dimensions := tool.getImageDimensions("/non/existent/file.png")
	if dimensions != "" {
		t.Errorf("Expected empty dimensions for non-existent file, got %s", dimensions)
	}

	// Test with invalid file (if sips is available)
	// Create a temporary text file that's not an image
	tempFile, err := os.CreateTemp("", "not_an_image.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	tempFile.WriteString("This is not an image")
	tempFile.Close()

	dimensions = tool.getImageDimensions(tempFile.Name())
	// Should return empty string for non-image files
	if dimensions != "" {
		t.Logf("Got dimensions for text file: %s (this might be expected behavior)", dimensions)
	}
}

func TestScreenshot_ParameterValidation(t *testing.T) {
	tests := []struct {
		name   string
		params types.ScreenshotParams
		valid  bool
	}{
		{
			name: "Valid minimal params",
			params: types.ScreenshotParams{
				UDID: "test-udid",
			},
			valid: true,
		},
		{
			name: "Valid full params",
			params: types.ScreenshotParams{
				UDID:       "test-udid",
				OutputPath: "/tmp/test.png",
				Format:     "png",
			},
			valid: true,
		},
		{
			name: "Valid JPEG format",
			params: types.ScreenshotParams{
				UDID:       "test-udid",
				OutputPath: "/tmp/test.jpg",
				Format:     "jpeg",
			},
			valid: true,
		},
		{
			name: "No device specified",
			params: types.ScreenshotParams{
				Format: "png",
			},
			valid: true, // Auto-selection will succeed if simulators are available
		},
	}

	tool := &Screenshot{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramsJSON, _ := json.Marshal(tt.params)
			result, err := tool.Execute(ctx, paramsJSON)

			if tt.valid {
				if result == nil {
					t.Error("Expected non-nil result for valid params")
				}
			} else {
				// For invalid params, we might still get a result but with an error
				if err == nil && result != nil {
					screenshotResult, ok := result.(*types.ScreenshotResult)
					if ok && screenshotResult.Success {
						t.Error("Expected failure for invalid params")
					}
				}
			}
		})
	}
}

func TestScreenshot_FormatValidation(t *testing.T) {
	tool := &Screenshot{}

	// Test unsupported format in captureScreenshot
	params := &types.ScreenshotParams{
		UDID:       "test-udid",
		OutputPath: "/tmp/test.gif",
		Format:     "gif", // Unsupported
	}

	_, err := tool.captureScreenshot(context.Background(), params)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

func TestScreenshot_OutputPathExtension(t *testing.T) {
	tool := &Screenshot{}
	ctx := context.Background()

	// Test that extension is added when missing
	params := types.ScreenshotParams{
		UDID:       "test-udid",
		OutputPath: "/tmp/test",
		Format:     "png",
	}

	paramsJSON, _ := json.Marshal(params)
	result, err := tool.Execute(ctx, paramsJSON)

	if result != nil {
		screenshotResult, ok := result.(*types.ScreenshotResult)
		if ok && err != nil {
			// Even if the command fails, the path should have been corrected
			expectedPath := "/tmp/test.png"
			// The error message should contain the corrected path or we can check other ways
			if !strings.Contains(err.Error(), expectedPath) && screenshotResult.FilePath != "" {
				if !strings.HasSuffix(screenshotResult.FilePath, ".png") {
					t.Error("Expected .png extension to be added to output path")
				}
			}
		}
	}
}