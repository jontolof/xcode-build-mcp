package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jontolof/xcode-build-mcp/internal/tools"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
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
	// Only log in debug mode
	if os.Getenv("MCP_LOG_LEVEL") == "debug" {
		s.logger.Println("MCP server starting...")
	}

	for {
		select {
		case <-ctx.Done():
			// Only log in debug mode
			if os.Getenv("MCP_LOG_LEVEL") == "debug" {
				s.logger.Println("Server context cancelled")
			}
			return nil
		default:
			request, err := s.transport.ReadRequest()
			if err != nil {
				// Check if the connection was closed - this is normal behavior
				if err.Error() == "connection closed" {
					// Don't log in normal operation, just exit gracefully
					return nil
				}
				s.logger.Printf("Failed to read request: %v", err)
				// For other errors, return to avoid infinite loop
				return err
			}

			response := s.handleRequest(ctx, request)
			if err := s.transport.WriteResponse(response); err != nil {
				s.logger.Printf("Failed to write response: %v", err)
				return err
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	// Only log in debug mode
	if os.Getenv("MCP_LOG_LEVEL") == "debug" {
		s.logger.Printf("Handling request: %s (ID: %v)", req.Method, req.ID)
	}

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
	// Create xcode components
	executor := xcode.NewExecutor(s.logger)
	parser := xcode.NewParser()

	// Register build tool
	buildTool := tools.NewXcodeBuildTool(executor, parser, s.logger)
	if err := s.registry.Register(buildTool); err != nil {
		return fmt.Errorf("failed to register xcode_build tool: %w", err)
	}

	// Register test tool
	testTool := tools.NewXcodeTestTool(executor, parser, s.logger)
	if err := s.registry.Register(testTool); err != nil {
		return fmt.Errorf("failed to register xcode_test tool: %w", err)
	}

	// Register clean tool
	cleanTool := tools.NewXcodeCleanTool(executor, parser, s.logger)
	if err := s.registry.Register(cleanTool); err != nil {
		return fmt.Errorf("failed to register xcode_clean tool: %w", err)
	}

	// Register discover projects tool
	discoverTool := tools.NewDiscoverProjectsTool(executor, parser, s.logger)
	if err := s.registry.Register(discoverTool); err != nil {
		return fmt.Errorf("failed to register discover_projects tool: %w", err)
	}

	// Register list simulators tool
	listSimulatorsTool := tools.NewListSimulatorsTool(executor, parser, s.logger)
	if err := s.registry.Register(listSimulatorsTool); err != nil {
		return fmt.Errorf("failed to register list_simulators tool: %w", err)
	}

	// Register simulator control tool
	simulatorControlTool := tools.NewSimulatorControlTool(executor, parser, s.logger)
	if err := s.registry.Register(simulatorControlTool); err != nil {
		return fmt.Errorf("failed to register simulator_control tool: %w", err)
	}

	// Register install app tool
	installAppTool := tools.NewInstallAppTool(executor, parser, s.logger)
	if err := s.registry.Register(installAppTool); err != nil {
		return fmt.Errorf("failed to register install_app tool: %w", err)
	}

	// Register launch app tool
	launchAppTool := tools.NewLaunchAppTool(executor, parser, s.logger)
	if err := s.registry.Register(launchAppTool); err != nil {
		return fmt.Errorf("failed to register launch_app tool: %w", err)
	}

	// Register list schemes tool
	listSchemesTool := tools.NewListSchemes()
	if err := s.registry.Register(listSchemesTool); err != nil {
		return fmt.Errorf("failed to register list_schemes tool: %w", err)
	}

	// Register capture logs tool
	captureLogsTool := tools.NewCaptureLogs()
	if err := s.registry.Register(captureLogsTool); err != nil {
		return fmt.Errorf("failed to register capture_logs tool: %w", err)
	}

	// Register screenshot tool
	screenshotTool := tools.NewScreenshot()
	if err := s.registry.Register(screenshotTool); err != nil {
		return fmt.Errorf("failed to register screenshot tool: %w", err)
	}

	// Register describe UI tool
	describeUITool := tools.NewDescribeUI()
	if err := s.registry.Register(describeUITool); err != nil {
		return fmt.Errorf("failed to register describe_ui tool: %w", err)
	}

	// Register UI interact tool
	uiInteractTool := tools.NewUIInteract()
	if err := s.registry.Register(uiInteractTool); err != nil {
		return fmt.Errorf("failed to register ui_interact tool: %w", err)
	}

	// Register get app info tool
	getAppInfoTool := tools.NewGetAppInfo()
	if err := s.registry.Register(getAppInfoTool); err != nil {
		return fmt.Errorf("failed to register get_app_info tool: %w", err)
	}

	// Only log in debug mode
	if os.Getenv("MCP_LOG_LEVEL") == "debug" {
		s.logger.Printf("Registered %d tools successfully", s.registry.Count())
	}
	return nil
}
