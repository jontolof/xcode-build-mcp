package tools

import (
	"context"
	"testing"

	"github.com/jontolof/xcode-build-mcp/internal/xcode"
)

func TestNewSimulatorControlTool(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewSimulatorControlTool(executor, parser, logger)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}
	if tool.Name() != "simulator_control" {
		t.Errorf("Expected name 'simulator_control', got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestSimulatorControlTool_Name(t *testing.T) {
	tool := NewSimulatorControlTool(nil, nil, nil)
	expected := "simulator_control"
	if tool.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, tool.Name())
	}
}

func TestSimulatorControlTool_Description(t *testing.T) {
	tool := NewSimulatorControlTool(nil, nil, nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(desc, "simulator") {
		t.Error("Expected description to mention simulators")
	}
}

func TestSimulatorControlTool_Schema(t *testing.T) {
	tool := NewSimulatorControlTool(nil, nil, nil)
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
	
	expectedProps := []string{"udid", "action", "timeout"}
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
	
	expectedRequired := []string{"udid", "action"}
	if len(required) != len(expectedRequired) {
		t.Errorf("Expected %d required fields, got %d", len(expectedRequired), len(required))
	}
}

func TestSimulatorControlTool_Execute_MissingParams(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}
	
	tool := NewSimulatorControlTool(executor, parser, logger)
	
	// Missing action
	params := map[string]interface{}{
		"udid": "ABC123-DEF4-5678-9ABC-DEF123456789",
	}
	
	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for missing action")
	}
	
	// Missing udid
	params = map[string]interface{}{
		"action": "boot",
	}
	
	_, err = tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for missing udid")
	}
}

func TestSimulatorControlTool_Execute_InvalidAction(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}
	
	tool := NewSimulatorControlTool(executor, parser, logger)
	
	params := map[string]interface{}{
		"udid":   "ABC123-DEF4-5678-9ABC-DEF123456789",
		"action": "invalid_action",
	}
	
	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for invalid action")
	}
}