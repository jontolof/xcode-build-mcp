package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/jontolof/xcode-build-mcp/internal/xcode"
)

func TestNewInstallAppTool(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewInstallAppTool(executor, parser, logger)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}
	if tool.Name() != "install_app" {
		t.Errorf("Expected name 'install_app', got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestInstallAppTool_Name(t *testing.T) {
	tool := NewInstallAppTool(nil, nil, nil)
	expected := "install_app"
	if tool.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, tool.Name())
	}
}

func TestInstallAppTool_Description(t *testing.T) {
	tool := NewInstallAppTool(nil, nil, nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(strings.ToLower(desc), "install") {
		t.Error("Expected description to mention install")
	}
}

func TestInstallAppTool_Schema(t *testing.T) {
	tool := NewInstallAppTool(nil, nil, nil)
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
	
	expectedProps := []string{"app_path", "udid", "device_type", "replace"}
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
	
	expectedRequired := []string{"app_path"}
	if len(required) != len(expectedRequired) {
		t.Errorf("Expected %d required fields, got %d", len(expectedRequired), len(required))
	}
}

func TestInstallAppTool_Execute_MissingAppPath(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}
	
	tool := NewInstallAppTool(executor, parser, logger)
	
	params := map[string]interface{}{
		"udid": "ABC123-DEF4-5678-9ABC-DEF123456789",
	}
	
	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for missing app_path")
	}
}

func TestInstallAppTool_Execute_EmptyAppPath(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}
	
	tool := NewInstallAppTool(executor, parser, logger)
	
	params := map[string]interface{}{
		"app_path": "",
		"udid":     "ABC123-DEF4-5678-9ABC-DEF123456789",
	}
	
	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for empty app_path")
	}
}