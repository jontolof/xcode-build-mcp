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

func TestGetAppInfo_Name(t *testing.T) {
	tool := NewGetAppInfo()
	if got := tool.Name(); got != "get_app_info" {
		t.Errorf("GetAppInfo.Name() = %v, want %v", got, "get_app_info")
	}
}

func TestGetAppInfo_Description(t *testing.T) {
	tool := NewGetAppInfo()
	desc := tool.Description()
	if desc == "" {
		t.Error("GetAppInfo.Description() returned empty string")
	}
	if len(desc) < 20 {
		t.Errorf("GetAppInfo.Description() too short: %s", desc)
	}
}

func TestGetAppInfo_Execute_InvalidParams(t *testing.T) {
	tool := NewGetAppInfo()
	ctx := context.Background()

	// Test with empty params (no app_path or bundle_id)
	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty params, got nil")
	}
	if result == "" {
		t.Error("Expected non-empty result string even on error")
	}
	// Result should be valid JSON even on error
	var resultData map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(result), &resultData); jsonErr != nil {
		t.Errorf("Result should be valid JSON: %v", jsonErr)
	}
}

func TestGetAppInfo_Execute_ValidParams(t *testing.T) {
	tool := NewGetAppInfo()
	ctx := context.Background()

	// Create a temporary app bundle for testing
	tempDir, err := os.MkdirTemp("", "test_app")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	appPath := filepath.Join(tempDir, "TestApp.app")
	if err := os.Mkdir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	args := map[string]interface{}{
		"app_path": appPath,
	}

	result, err := tool.Execute(ctx, args)

	// Should get a result string even if extraction partially fails
	if result == "" {
		t.Error("Expected non-empty result string")
	}

	// Parse JSON result
	var appInfoResult types.AppInfoResult
	if jsonErr := json.Unmarshal([]byte(result), &appInfoResult); jsonErr != nil {
		t.Errorf("Failed to parse result JSON: %v", jsonErr)
	}

	if appInfoResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestGetAppInfo_Execute_MissingParams(t *testing.T) {
	tool := NewGetAppInfo()
	ctx := context.Background()

	// Test with no app path or bundle ID
	args := map[string]interface{}{}

	result, err := tool.Execute(ctx, args)

	if err == nil {
		t.Error("Expected error for missing parameters")
	}

	if !strings.Contains(err.Error(), "app_path or bundle_id must be specified") {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result string even for errors")
	}

	// Parse JSON result
	var appInfoResult types.AppInfoResult
	if jsonErr := json.Unmarshal([]byte(result), &appInfoResult); jsonErr != nil {
		t.Errorf("Failed to parse result JSON: %v", jsonErr)
	}

	if appInfoResult.Success {
		t.Errorf("Expected *types.AppInfoResult, got %T", result)
	}

	if appInfoResult.Success {
		t.Error("Expected Success to be false for error case")
	}
}

func TestGetAppInfo_ExtractFromLocalBundle_NonexistentPath(t *testing.T) {
	tool := &GetAppInfo{}
	ctx := context.Background()

	params := &types.AppInfoParams{
		AppPath: "/nonexistent/path/App.app",
	}

	result, err := tool.extractAppInfo(ctx, params); _ = result

	if err == nil {
		t.Error("Expected error for nonexistent app path")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetAppInfo_ExtractFromLocalBundle_InvalidFormat(t *testing.T) {
	tool := &GetAppInfo{}
	ctx := context.Background()

	// Create a temporary file with wrong extension
	tempFile, err := os.CreateTemp("", "notanapp.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	params := &types.AppInfoParams{
		AppPath: tempFile.Name(),
	}

	result, err := tool.extractAppInfo(ctx, params); _ = result

	if err == nil {
		t.Error("Expected error for invalid app bundle format")
	}

	if !strings.Contains(err.Error(), "invalid app bundle format") {
		t.Errorf("Expected format error, got: %v", err)
	}
}

func TestGetAppInfo_FindIconFiles(t *testing.T) {
	tool := &GetAppInfo{}

	// Create a temporary app directory
	tempDir, err := os.MkdirTemp("", "icon_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test icon files
	iconFiles := []string{
		"AppIcon60x60@2x.png",
		"Icon.png",
		"Icon@2x.png",
	}

	for _, iconFile := range iconFiles {
		iconPath := filepath.Join(tempDir, iconFile)
		file, err := os.Create(iconPath)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	// Test with empty plist data (should find common icons)
	plistData := map[string]interface{}{}
	iconPaths := tool.findIconFiles(tempDir, plistData)

	if len(iconPaths) == 0 {
		t.Error("Expected to find some icon files")
	}

	// Check that found paths exist
	for _, iconPath := range iconPaths {
		if _, err := os.Stat(iconPath); os.IsNotExist(err) {
			t.Errorf("Icon path should exist: %s", iconPath)
		}
	}
}

func TestGetAppInfo_ExtractIconPathsFromData(t *testing.T) {
	tool := &GetAppInfo{}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "icon_extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test icon file
	iconPath := filepath.Join(tempDir, "test-icon.png")
	file, err := os.Create(iconPath)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	tests := []struct {
		name     string
		iconData interface{}
		expected int
	}{
		{
			name:     "String icon",
			iconData: "test-icon.png",
			expected: 1,
		},
		{
			name:     "Array of icons",
			iconData: []interface{}{"test-icon.png", "nonexistent.png"},
			expected: 1, // Only existing file should be included
		},
		{
			name: "Icon dictionary",
			iconData: map[string]interface{}{
				"CFBundlePrimaryIcon": map[string]interface{}{
					"CFBundleIconFiles": []interface{}{"test-icon.png"},
				},
			},
			expected: 1,
		},
		{
			name:     "Invalid data type",
			iconData: 123,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := tool.extractIconPathsFromData(tt.iconData, tempDir)
			if len(paths) != tt.expected {
				t.Errorf("Expected %d paths, got %d", tt.expected, len(paths))
			}
		})
	}
}

func TestGetAppInfo_FindExecutable(t *testing.T) {
	tool := &GetAppInfo{}

	// Create temporary app directory
	tempDir, err := os.MkdirTemp("", "executable_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	appPath := filepath.Join(tempDir, "TestApp.app")
	if err := os.Mkdir(appPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a mock Info.plist with executable name
	infoPlistPath := filepath.Join(appPath, "Info.plist")
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>TestApp</string>
</dict>
</plist>`

	if err := os.WriteFile(infoPlistPath, []byte(plistContent), 0644); err != nil {
		t.Fatal(err)
	}

	executable := tool.findExecutable(appPath)
	if executable != "TestApp" {
		t.Errorf("Expected executable 'TestApp', got '%s'", executable)
	}
}

func TestGetAppInfo_ParseAppInfoText(t *testing.T) {
	tool := &GetAppInfo{}

	testOutput := `Application Information:
CFBundleShortVersionString = "1.2.3"
CFBundleVersion = "456"
CFBundleDisplayName = "Test App"
CFBundleIdentifier = "com.example.testapp"`

	result, err := tool.parseAppInfoText(testOutput, "com.example.testapp")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.BundleID != "com.example.testapp" {
		t.Errorf("Expected bundle ID 'com.example.testapp', got '%s'", result.BundleID)
	}

	if result.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", result.Version)
	}

	if result.BuildNumber != "456" {
		t.Errorf("Expected build number '456', got '%s'", result.BuildNumber)
	}

	if result.DisplayName != "Test App" {
		t.Errorf("Expected display name 'Test App', got '%s'", result.DisplayName)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
}

func TestGetAppInfo_ParameterValidation(t *testing.T) {
	tests := []struct {
		name   string
		params types.AppInfoParams
		valid  bool
	}{
		{
			name: "Valid app path",
			params: types.AppInfoParams{
				AppPath: "/path/to/App.app",
			},
			valid: true,
		},
		{
			name: "Valid bundle ID with device",
			params: types.AppInfoParams{
				BundleID: "com.example.app",
				UDID:     "test-udid",
			},
			valid: true,
		},
		{
			name: "Bundle ID without device (should auto-select)",
			params: types.AppInfoParams{
				BundleID: "com.example.testapp",
			},
			valid: true, // Auto-selection succeeds and simctl appinfo returns basic info even for non-existent apps
		},
		{
			name:   "No parameters",
			params: types.AppInfoParams{},
			valid:  false,
		},
	}

	tool := NewGetAppInfo()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert params to args map
			args := map[string]interface{}{}
			if tt.params.AppPath != "" {
				args["app_path"] = tt.params.AppPath
			}
			if tt.params.BundleID != "" {
				args["bundle_id"] = tt.params.BundleID
			}
			if tt.params.UDID != "" {
				args["udid"] = tt.params.UDID
			}
			if tt.params.DeviceType != "" {
				args["device_type"] = tt.params.DeviceType
			}

			result, err := tool.Execute(ctx, args)

			if tt.valid {
				// For valid params, we should get a result string
				if result == "" {
					t.Error("Expected non-empty result for valid params")
				}
			} else {
				// For invalid params, we expect an error
				if err == nil {
					t.Errorf("Expected error for invalid params (%s), but got none", tt.name)
				}
				// Should still return a result string
				if result == "" {
					t.Error("Expected non-empty result even for errors")
				} else {
					// Parse JSON result
					var appInfoResult types.AppInfoResult
					if jsonErr := json.Unmarshal([]byte(result), &appInfoResult); jsonErr != nil {
						t.Errorf("Failed to parse result JSON: %v", jsonErr)
						return
					}
					if appInfoResult.Success {
						t.Errorf("Expected Success to be false for invalid params (%s)", tt.name)
					}
				}
			}
		})
	}
}