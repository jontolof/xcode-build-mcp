package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/jontolof/xcode-build-mcp/internal/xcode"
)

func TestNewLaunchAppTool(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewLaunchAppTool(executor, parser, logger)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}
	if tool.Name() != "launch_app" {
		t.Errorf("Expected name 'launch_app', got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestLaunchAppTool_Name(t *testing.T) {
	tool := NewLaunchAppTool(nil, nil, nil)
	expected := "launch_app"
	if tool.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, tool.Name())
	}
}

func TestLaunchAppTool_Description(t *testing.T) {
	tool := NewLaunchAppTool(nil, nil, nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(strings.ToLower(desc), "launch") {
		t.Error("Expected description to mention launch")
	}
}

func TestLaunchAppTool_Schema(t *testing.T) {
	tool := NewLaunchAppTool(nil, nil, nil)
	schema := tool.InputSchema()

	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	schemaType, ok := schema["type"].(string)
	if !ok || schemaType != "object" {
		t.Error("Expected schema type to be 'object'")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	expectedProps := []string{"bundle_id", "udid", "device_type", "arguments", "environment", "wait_for_exit"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected property %s in schema", prop)
		}
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required fields in schema")
	}

	expectedRequired := []string{"bundle_id"}
	if len(required) != len(expectedRequired) {
		t.Errorf("Expected %d required fields, got %d", len(expectedRequired), len(required))
	}
}

func TestLaunchAppTool_Execute_MissingBundleID(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewLaunchAppTool(executor, parser, logger)

	params := map[string]interface{}{
		"udid": "ABC123-DEF4-5678-9ABC-DEF123456789",
	}

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for missing bundle_id")
	}
}

func TestLaunchAppTool_Execute_EmptyBundleID(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewLaunchAppTool(executor, parser, logger)

	params := map[string]interface{}{
		"bundle_id": "",
		"udid":      "ABC123-DEF4-5678-9ABC-DEF123456789",
	}

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for empty bundle_id")
	}
}
