package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type Server struct {
	logger    *log.Logger
	registry  *Registry
	transport Transport
}

func NewServer(logger *log.Logger) (*Server, error) {
	registry := NewRegistry()
	
	server := &Server{
		logger:   logger,
		registry: registry,
	}

	if err := server.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return server, nil
}

func (s *Server) Run(ctx context.Context, transportType string) error {
	var transport Transport
	var err error

	switch transportType {
	case "stdio":
		transport, err = NewStdioTransport(s.logger)
	default:
		return fmt.Errorf("unsupported transport type: %s", transportType)
	}

	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	s.transport = transport
	return s.serve(ctx)
}

func (s *Server) serve(ctx context.Context) error {
	s.logger.Println("MCP server starting...")

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("Server context cancelled")
			return nil
		default:
			request, err := s.transport.ReadRequest()
			if err != nil {
				s.logger.Printf("Failed to read request: %v", err)
				continue
			}

			response := s.handleRequest(ctx, request)
			if err := s.transport.WriteResponse(response); err != nil {
				s.logger.Printf("Failed to write response: %v", err)
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	s.logger.Printf("Handling request: %s (ID: %v)", req.Method, req.ID)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(ctx, req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err)
	}

	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "xcode-build-mcp",
			Version: "0.1.0",
		},
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleListTools(req *Request) *Response {
	tools := s.registry.ListTools()
	result := ListToolsResult{Tools: tools}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleCallTool(ctx context.Context, req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err)
	}

	tool := s.registry.GetTool(params.Name)
	if tool == nil {
		return s.errorResponse(req.ID, -32601, "Tool not found", fmt.Errorf("tool %s not found", params.Name))
	}

	result, err := tool.Execute(ctx, params.Arguments)
	if err != nil {
		return s.errorResponse(req.ID, -32603, "Tool execution failed", err)
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: result,
			}},
		},
	}
}

func (s *Server) errorResponse(id interface{}, code int, message string, err error) *Response {
	data := make(map[string]interface{})
	if err != nil {
		data["error"] = err.Error()
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func (s *Server) registerTools() error {
	return nil
}