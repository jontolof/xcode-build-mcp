package mcp

import (
	"context"
	"errors"
	"testing"
)

// Test tool implementation
type testTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executeFunc func(ctx context.Context, args map[string]interface{}) (string, error)
}

func (t *testTool) Name() string {
	return t.name
}

func (t *testTool) Description() string {
	return t.description
}

func (t *testTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *testTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, args)
	}
	return "default result", nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("Registry should not be nil")
	}
	if registry.tools == nil {
		t.Fatal("Tools map should be initialized")
	}
	if registry.Count() != 0 {
		t.Errorf("New registry should have 0 tools, got %d", registry.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tool := &testTool{
		name:        "test_tool",
		description: "A test tool",
		schema:      map[string]interface{}{"type": "object"},
	}

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Registry should have 1 tool, got %d", registry.Count())
	}

	// Test registering duplicate tool
	err = registry.Register(tool)
	if err == nil {
		t.Error("Should return error when registering duplicate tool")
	}

	// Test registering tool with empty name
	emptyTool := &testTool{
		name:        "",
		description: "Empty name tool",
	}
	err = registry.Register(emptyTool)
	if err == nil {
		t.Error("Should return error when registering tool with empty name")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	tool := &testTool{
		name:        "test_tool",
		description: "A test tool",
	}

	registry.Register(tool)
	if registry.Count() != 1 {
		t.Fatal("Tool should be registered")
	}

	registry.Unregister("test_tool")
	if registry.Count() != 0 {
		t.Error("Tool should be unregistered")
	}

	// Unregistering non-existent tool should not panic
	registry.Unregister("non_existent")
}

func TestRegistry_GetTool(t *testing.T) {
	registry := NewRegistry()

	tool := &testTool{
		name:        "test_tool",
		description: "A test tool",
		schema:      map[string]interface{}{"type": "object"},
	}

	registry.Register(tool)

	// Test getting existing tool
	gotTool := registry.GetTool("test_tool")
	if gotTool == nil {
		t.Fatal("Should find registered tool")
	}
	if gotTool.Name() != "test_tool" {
		t.Errorf("Tool name mismatch: got %s, want test_tool", gotTool.Name())
	}

	// Test getting non-existent tool
	gotTool = registry.GetTool("non_existent")
	if gotTool != nil {
		t.Error("Should return nil for non-existent tool")
	}
}

func TestRegistry_ListTools(t *testing.T) {
	registry := NewRegistry()

	tool1 := &testTool{
		name:        "tool1",
		description: "First tool",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool2 := &testTool{
		name:        "tool2",
		description: "Second tool",
		schema:      map[string]interface{}{"type": "object"},
	}

	registry.Register(tool1)
	registry.Register(tool2)

	tools := registry.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// Check that both tools are in the list
	foundTool1 := false
	foundTool2 := false
	for _, tool := range tools {
		if tool.Name == "tool1" {
			foundTool1 = true
			if tool.Description != "First tool" {
				t.Errorf("Tool1 description mismatch: got %s", tool.Description)
			}
		}
		if tool.Name == "tool2" {
			foundTool2 = true
			if tool.Description != "Second tool" {
				t.Errorf("Tool2 description mismatch: got %s", tool.Description)
			}
		}
	}

	if !foundTool1 {
		t.Error("tool1 should be in the list")
	}
	if !foundTool2 {
		t.Error("tool2 should be in the list")
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	tool := &testTool{
		name:        "echo_tool",
		description: "Echoes input",
		schema:      map[string]interface{}{"type": "object"},
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			if msg, ok := args["message"].(string); ok {
				return "Echo: " + msg, nil
			}
			return "", errors.New("message not found")
		},
	}

	registry.Register(tool)

	// Test executing the tool
	ctx := context.Background()
	args := map[string]interface{}{"message": "Hello"}

	gotTool := registry.GetTool("echo_tool")
	if gotTool == nil {
		t.Fatal("Tool should be registered")
	}

	result, err := gotTool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "Echo: Hello" {
		t.Errorf("Unexpected result: %s", result)
	}

	// Test with missing message
	args = map[string]interface{}{}
	_, err = gotTool.Execute(ctx, args)
	if err == nil {
		t.Error("Should return error when message is missing")
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registration and access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tool := &testTool{
				name:        "concurrent_tool",
				description: "Test concurrent access",
				schema:      map[string]interface{}{"type": "object"},
			}
			registry.Register(tool)
			registry.Unregister("concurrent_tool")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			registry.ListTools()
			registry.GetTool("concurrent_tool")
			registry.Count()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Registry should be in a consistent state
	if registry.Count() < 0 {
		t.Error("Registry count should not be negative")
	}
}
