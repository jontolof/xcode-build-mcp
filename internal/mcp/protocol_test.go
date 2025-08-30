package mcp

import (
	"encoding/json"
	"testing"
)

func TestRequest_JSON(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  json.RawMessage(`{"filter": "xcode"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal Request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Request: %v", err)
	}

	if decoded.JSONRPC != req.JSONRPC {
		t.Errorf("JSONRPC mismatch: got %s, want %s", decoded.JSONRPC, req.JSONRPC)
	}
	if decoded.Method != req.Method {
		t.Errorf("Method mismatch: got %s, want %s", decoded.Method, req.Method)
	}
	
	// Compare IDs as floats since JSON unmarshaling converts numbers to float64
	if decodedID, ok := decoded.ID.(float64); !ok || int(decodedID) != req.ID.(int) {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, req.ID)
	}
}

func TestResponse_Success(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"status": "success"}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal Response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Response: %v", err)
	}

	if decoded.Error != nil {
		t.Error("Error should be nil for success response")
	}
	if decoded.Result == nil {
		t.Error("Result should not be nil for success response")
	}
}

func TestResponse_Error(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Error: &Error{
			Code:    -32601,
			Message: "Method not found",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal Response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Response: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Error should not be nil for error response")
	}
	if decoded.Error.Code != -32601 {
		t.Errorf("Error code mismatch: got %d, want %d", decoded.Error.Code, -32601)
	}
	if decoded.Result != nil {
		t.Error("Result should be nil for error response")
	}
}

func TestInitializeParams(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "1.0.0",
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{},
		},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal InitializeParams: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InitializeParams: %v", err)
	}

	if decoded.ProtocolVersion != params.ProtocolVersion {
		t.Errorf("ProtocolVersion mismatch: got %s, want %s", decoded.ProtocolVersion, params.ProtocolVersion)
	}
	if decoded.ClientInfo.Name != params.ClientInfo.Name {
		t.Errorf("ClientInfo.Name mismatch: got %s, want %s", decoded.ClientInfo.Name, params.ClientInfo.Name)
	}
	if decoded.Capabilities.Roots == nil {
		t.Error("Roots capability should not be nil")
	}
}

func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "1.0.0",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
			Prompts: &PromptsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "xcode-build-mcp",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal InitializeResult: %v", err)
	}

	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InitializeResult: %v", err)
	}

	if decoded.ServerInfo.Name != result.ServerInfo.Name {
		t.Errorf("ServerInfo.Name mismatch: got %s, want %s", decoded.ServerInfo.Name, result.ServerInfo.Name)
	}
	if decoded.Capabilities.Tools == nil {
		t.Error("Tools capability should not be nil")
	}
	if decoded.Capabilities.Prompts == nil {
		t.Error("Prompts capability should not be nil")
	}
}

func TestToolDefinition(t *testing.T) {
	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
			"required": []string{"param1"},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal ToolDefinition: %v", err)
	}

	var decoded ToolDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolDefinition: %v", err)
	}

	if decoded.Name != tool.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tool.Name)
	}
	if decoded.Description != tool.Description {
		t.Errorf("Description mismatch: got %s, want %s", decoded.Description, tool.Description)
	}
	if decoded.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}

func TestCallToolParams(t *testing.T) {
	params := CallToolParams{
		Name: "xcode_build",
		Arguments: map[string]interface{}{
			"project": "MyApp.xcodeproj",
			"scheme":  "MyApp",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal CallToolParams: %v", err)
	}

	var decoded CallToolParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CallToolParams: %v", err)
	}

	if decoded.Name != params.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, params.Name)
	}
	if decoded.Arguments == nil {
		t.Fatal("Arguments should not be nil")
	}
	if decoded.Arguments["project"] != params.Arguments["project"] {
		t.Errorf("Project argument mismatch")
	}
}

func TestCallToolResult(t *testing.T) {
	isError := false
	result := CallToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: "Build succeeded",
			},
		},
		IsError: &isError,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CallToolResult: %v", err)
	}

	var decoded CallToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CallToolResult: %v", err)
	}

	if len(decoded.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(decoded.Content))
	}
	if decoded.Content[0].Type != "text" {
		t.Errorf("Content type mismatch: got %s, want text", decoded.Content[0].Type)
	}
	if decoded.IsError == nil || *decoded.IsError != false {
		t.Error("IsError should be false")
	}
}

func TestContent(t *testing.T) {
	tests := []struct {
		name    string
		content Content
	}{
		{
			name: "text content",
			content: Content{
				Type: "text",
				Text: "Hello, world!",
			},
		},
		{
			name: "data content",
			content: Content{
				Type:     "resource",
				Data:     "base64encodeddata",
				MimeType: "image/png",
			},
		},
		{
			name: "content with meta",
			content: Content{
				Type: "text",
				Text: "Metadata example",
				Meta: map[string]interface{}{
					"timestamp": "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("Failed to marshal Content: %v", err)
			}

			var decoded Content
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal Content: %v", err)
			}

			if decoded.Type != tt.content.Type {
				t.Errorf("Type mismatch: got %s, want %s", decoded.Type, tt.content.Type)
			}
			if decoded.Text != tt.content.Text {
				t.Errorf("Text mismatch: got %s, want %s", decoded.Text, tt.content.Text)
			}
			if decoded.Data != tt.content.Data {
				t.Errorf("Data mismatch: got %s, want %s", decoded.Data, tt.content.Data)
			}
			if decoded.MimeType != tt.content.MimeType {
				t.Errorf("MimeType mismatch: got %s, want %s", decoded.MimeType, tt.content.MimeType)
			}
		})
	}
}