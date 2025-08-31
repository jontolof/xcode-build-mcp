package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

func TestDescribeUI_Name(t *testing.T) {
	tool := NewDescribeUI()
	if got := tool.Name(); got != "describe_ui" {
		t.Errorf("DescribeUI.Name() = %v, want %v", got, "describe_ui")
	}
}

func TestDescribeUI_Description(t *testing.T) {
	tool := NewDescribeUI()
	desc := tool.Description()
	if desc == "" {
		t.Error("DescribeUI.Description() returned empty string")
	}
	if len(desc) < 20 {
		t.Errorf("DescribeUI.Description() too short: %s", desc)
	}
}

func TestDescribeUI_Execute_InvalidParams(t *testing.T) {
	tool := NewDescribeUI()
	ctx := context.Background()

	// Test with invalid JSON
	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if result != "" {
		t.Errorf("Expected nil result for invalid params, got %+v", result)
	}
}

func TestDescribeUI_Execute_ValidParams(t *testing.T) {
	tool := NewDescribeUI()
	ctx := context.Background()

	resultStr, execErr := tool.Execute(ctx, map[string]interface{}{
		"udid":         "test-udid",
		"format":       "tree",
		"max_depth":    5,
		"include_text": true,
	})

	// Should get a result even if command fails
	if resultStr == "" {
		t.Error("Expected non-empty result")
	}

	// Parse the JSON result
	var uiResult types.UIDescribeResult
	err := json.Unmarshal([]byte(resultStr), &uiResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if uiResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// The command will likely fail in test environment, but that's expected
	if execErr != nil && uiResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestDescribeUI_Execute_DefaultValues(t *testing.T) {
	tool := NewDescribeUI()
	ctx := context.Background()

	// Test with minimal params
	resultStr, execErr := tool.Execute(ctx, map[string]interface{}{
		"udid": "test-udid",
	})

	if resultStr == "" {
		t.Error("Expected non-empty result")
	}

	// Parse the JSON result
	var uiResult types.UIDescribeResult
	err := json.Unmarshal([]byte(resultStr), &uiResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Should have applied defaults
	if uiResult.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	// Command will likely fail without real simulator, but structure should be correct
	if execErr != nil && uiResult.Success {
		t.Error("If there's an error, Success should be false")
	}
}

func TestDescribeUI_GenerateMockJSONHierarchy(t *testing.T) {
	tool := NewDescribeUI()

	// Test JSON hierarchy generation
	jsonData := tool.generateMockJSONHierarchy(false, 5)
	
	if jsonData == "" {
		t.Error("Expected non-empty JSON data")
	}

	// Verify it's valid JSON
	var hierarchy map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &hierarchy); err != nil {
		t.Errorf("Generated JSON is not valid: %v", err)
	}

	// Check basic structure
	if hierarchy["type"] != "Application" {
		t.Errorf("Expected type 'Application', got %v", hierarchy["type"])
	}

	children, ok := hierarchy["children"].([]interface{})
	if !ok || len(children) == 0 {
		t.Error("Expected children array in hierarchy")
	}
}

func TestDescribeUI_GenerateMockTextHierarchy(t *testing.T) {
	tool := NewDescribeUI()

	tests := []struct {
		format      string
		includeText bool
		expected    []string
	}{
		{
			format:   "tree",
			expected: []string{"Application", "├──", "└──"},
		},
		{
			format:   "flat",
			expected: []string{"Application", "NavigationBar", "Button"},
		},
		{
			format:      "tree",
			includeText: true,
			expected:    []string{"Application", "Text:", "Welcome"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			textData := tool.generateMockTextHierarchy(tt.format, tt.includeText, 5)
			
			if textData == "" {
				t.Error("Expected non-empty text data")
			}

			for _, expected := range tt.expected {
				if !strings.Contains(textData, expected) {
					t.Errorf("Expected text to contain '%s', got: %s", expected, textData)
				}
			}
		})
	}
}

func TestDescribeUI_CountElementsInJSON(t *testing.T) {
	tool := NewDescribeUI()

	jsonData := `{"type": "Application", "children": [{"type": "Button"}, {"type": "Label"}]}`
	count := tool.countElementsInJSON(jsonData)
	
	expected := 3 // Application, Button, Label
	if count != expected {
		t.Errorf("Expected count %d, got %d", expected, count)
	}
}

func TestDescribeUI_CountElementsInText(t *testing.T) {
	tool := NewDescribeUI()

	textData := `Application [0,0,375,812]
Button [0,0,60,30]
Text: "Hello World"
Label [0,40,100,20]`

	count := tool.countElementsInText(textData)
	
	expected := 3 // Application, Button, Label (Text: line is ignored)
	if count != expected {
		t.Errorf("Expected count %d, got %d", expected, count)
	}
}

func TestDescribeUI_AddTextToHierarchy(t *testing.T) {
	tool := NewDescribeUI()

	hierarchy := map[string]interface{}{
		"type": "Application",
		"children": []map[string]interface{}{
			{
				"type":       "Button",
				"identifier": "Submit",
			},
		},
	}

	tool.addTextToHierarchy(hierarchy)

	children := hierarchy["children"].([]map[string]interface{})
	button := children[0]
	
	if button["text"] != "Submit" {
		t.Errorf("Expected button text 'Submit', got %v", button["text"])
	}
}

func TestDescribeUI_FormatValidation(t *testing.T) {
	tool := NewDescribeUI()

	// Test unsupported format
	params := &types.UIDescribeParams{
		UDID:   "test-udid",
		Format: "xml", // Unsupported
	}

	_, err := tool.describeUI(context.Background(), params)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

func TestDescribeUI_ParameterValidation(t *testing.T) {
	tests := []struct {
		name   string
		params types.UIDescribeParams
		valid  bool
	}{
		{
			name: "Valid minimal params",
			params: types.UIDescribeParams{
				UDID: "test-udid",
			},
			valid: true,
		},
		{
			name: "Valid full params",
			params: types.UIDescribeParams{
				UDID:        "test-udid",
				Format:      "json",
				MaxDepth:    5,
				IncludeText: true,
			},
			valid: true,
		},
		{
			name: "Tree format",
			params: types.UIDescribeParams{
				UDID:   "test-udid",
				Format: "tree",
			},
			valid: true,
		},
		{
			name: "Flat format",
			params: types.UIDescribeParams{
				UDID:   "test-udid",
				Format: "flat",
			},
			valid: true,
		},
		{
			name: "No device specified",
			params: types.UIDescribeParams{
				Format: "json",
			},
			valid: true, // Auto-selection will succeed if simulators are available
		},
		{
			name: "Invalid format",
			params: types.UIDescribeParams{
				UDID:   "test-udid",
				Format: "xml",
			},
			valid: false,
		},
	}

	tool := NewDescribeUI()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{}
			if tt.params.UDID != "" {
				args["udid"] = tt.params.UDID
			}
			if tt.params.Format != "" {
				args["format"] = tt.params.Format
			}
			if tt.params.MaxDepth > 0 {
				args["max_depth"] = tt.params.MaxDepth
			}
			if tt.params.IncludeText {
				args["include_text"] = tt.params.IncludeText
			}
			
			resultStr, execErr := tool.Execute(ctx, args)

			if tt.valid {
				if resultStr == "" {
					t.Error("Expected non-empty result for valid params")
				}
			} else {
				// For invalid params, we might still get a result but with an error
				if execErr == nil && resultStr != "" {
					var uiResult types.UIDescribeResult
					if json.Unmarshal([]byte(resultStr), &uiResult) == nil && uiResult.Success {
						t.Error("Expected failure for invalid params")
					}
				}
			}
		})
	}
}