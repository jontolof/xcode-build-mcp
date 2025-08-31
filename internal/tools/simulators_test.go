package tools

import (
	"testing"

	"github.com/jontolof/xcode-build-mcp/internal/xcode"
)

func TestNewListSimulatorsTool(t *testing.T) {
	executor := xcode.NewExecutor(&testLogger{})
	parser := xcode.NewParser()
	logger := &testLogger{}

	tool := NewListSimulatorsTool(executor, parser, logger)

	if tool == nil {
		t.Fatal("Expected non-nil tool")
	}
	if tool.Name() != "list_simulators" {
		t.Errorf("Expected name 'list_simulators', got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestListSimulatorsTool_Name(t *testing.T) {
	tool := NewListSimulatorsTool(nil, nil, nil)
	expected := "list_simulators"
	if tool.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, tool.Name())
	}
}

func TestListSimulatorsTool_Description(t *testing.T) {
	tool := NewListSimulatorsTool(nil, nil, nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(desc, "simulator") {
		t.Error("Expected description to mention simulators")
	}
}

func TestListSimulatorsTool_Schema(t *testing.T) {
	tool := NewListSimulatorsTool(nil, nil, nil)
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
	
	expectedProps := []string{"platform", "device_type", "runtime", "available", "state"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected property %s in schema", prop)
		}
	}
}