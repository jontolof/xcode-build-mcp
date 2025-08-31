package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"testing"
)

func TestNewServer(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, err := NewServer(logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}
	if server.logger == nil {
		t.Fatal("Logger should not be nil")
	}
	if server.registry == nil {
		t.Fatal("Registry should not be nil")
	}
}

func TestServer_Run_UnsupportedTransport(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, _ := NewServer(logger)

	ctx := context.Background()
	err := server.Run(ctx, "unsupported")
	if err == nil {
		t.Fatal("Should return error for unsupported transport")
	}
	if err.Error() != "unsupported transport type: unsupported" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServer_HandleRequest_Initialize(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, _ := NewServer(logger)

	initParams := InitializeParams{
		ProtocolVersion: "1.0.0",
		Capabilities:    ClientCapabilities{},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	paramsJSON, _ := json.Marshal(initParams)
	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  paramsJSON,
	}

	resp := server.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("Initialize failed: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("Initialize should return a result")
	}
}

func TestServer_HandleRequest_MethodNotFound(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, _ := NewServer(logger)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	resp := server.handleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("Should return an error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Error code mismatch: got %d, want -32601", resp.Error.Code)
	}
}

func TestServer_HandleRequest_ToolsList(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, _ := NewServer(logger)

	req := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp := server.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("tools/list failed: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("tools/list should return a result")
	}
}

func TestServer_HandleRequest_ToolsCall(t *testing.T) {
	logger := log.New(bytes.NewBuffer(nil), "", 0)
	server, _ := NewServer(logger)

	// Register a test tool
	testTool := &mockTool{
		name:        "test_tool",
		description: "Test tool",
		schema:      map[string]interface{}{"type": "object"},
	}
	server.registry.Register(testTool)

	callParams := CallToolParams{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"test": "value"},
	}
	paramsJSON, _ := json.Marshal(callParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  paramsJSON,
	}

	resp := server.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("tools/call failed: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("tools/call should return a result")
	}
}

// Mock tool for testing
type mockTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) InputSchema() map[string]interface{} {
	return m.schema
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "Test result", nil
}
