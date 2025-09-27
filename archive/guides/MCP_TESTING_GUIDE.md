# MCP Server Testing Guide
*Comprehensive testing strategies for Model Context Protocol servers*

Based on learnings from the Xcode Build MCP implementation, this guide provides battle-tested patterns for creating robust, maintainable MCP servers with comprehensive test coverage.

## Table of Contents
1. [Testing Architecture Overview](#testing-architecture-overview)
2. [Test Categories & Strategies](#test-categories--strategies)
3. [MCP-Specific Testing Patterns](#mcp-specific-testing-patterns)
4. [Common Pitfalls & Solutions](#common-pitfalls--solutions)
5. [Build & CI Integration](#build--ci-integration)
6. [Performance Testing](#performance-testing)
7. [Integration Testing](#integration-testing)
8. [Docker Compose Specific Considerations](#docker-compose-specific-considerations)

## Testing Architecture Overview

### Core Testing Principles
- **Separation of Concerns**: Test protocol, business logic, and external integrations separately
- **Mock External Dependencies**: Docker daemon, container states, network conditions
- **Environment Independence**: Tests should work on any system regardless of Docker setup
- **Progressive Complexity**: Unit → Integration → End-to-End testing pyramid

### Recommended Project Structure
```
docker-compose-mcp/
├── internal/
│   ├── mcp/             # MCP protocol implementation
│   │   ├── protocol_test.go
│   │   ├── server_test.go
│   │   └── transport_test.go
│   ├── docker/          # Docker API abstraction
│   │   ├── client_test.go
│   │   └── compose_test.go
│   ├── tools/           # Tool implementations
│   │   ├── compose_up_test.go
│   │   ├── compose_down_test.go
│   │   └── helpers_test.go
│   └── filter/          # Output filtering
│       └── filter_test.go
├── cmd/server/
│   └── main_test.go     # End-to-end tests
└── test/
    ├── fixtures/        # Test docker-compose files
    ├── mocks/          # Mock implementations
    └── integration/    # Integration test suites
```

## Test Categories & Strategies

### 1. Unit Tests (`*_test.go`)

#### Protocol Layer Testing
```go
// Test MCP JSON-RPC protocol handling
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
}
```

#### Tool Implementation Testing
```go
// Test individual tool logic without external dependencies
func TestComposeTool_ValidateParams(t *testing.T) {
    tool := NewComposeTool()
    
    tests := []struct {
        name    string
        params  map[string]interface{}
        wantErr bool
    }{
        {
            name: "valid compose file",
            params: map[string]interface{}{
                "compose_file": "docker-compose.yml",
                "services":     []string{"web", "db"},
            },
            wantErr: false,
        },
        {
            name: "missing compose file",
            params: map[string]interface{}{
                "services": []string{"web"},
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tool.validateParams(tt.params)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Integration Tests

#### Mock Docker Client Pattern
```go
// Create comprehensive mocks for Docker operations
type MockDockerClient struct {
    containers map[string]*Container
    networks   map[string]*Network
    volumes    map[string]*Volume
    
    // Control test behavior
    shouldFailOn map[string]error
    callHistory  []string
}

func (m *MockDockerClient) ComposeUp(ctx context.Context, project string) error {
    m.callHistory = append(m.callHistory, "ComposeUp")
    
    if err, exists := m.shouldFailOn["ComposeUp"]; exists {
        return err
    }
    
    // Simulate successful container creation
    m.containers[project+"_web_1"] = &Container{
        ID:     "abc123",
        Name:   project + "_web_1",
        State:  "running",
        Status: "Up 2 seconds",
    }
    
    return nil
}
```

#### Test External Command Execution
```go
// Test command construction and output parsing
func TestCompose_BuildCommand(t *testing.T) {
    tests := []struct {
        name     string
        params   ComposeParams
        expected []string
    }{
        {
            name: "basic up command",
            params: ComposeParams{
                ComposeFile: "docker-compose.yml",
                Services:    []string{"web", "db"},
                Detached:    true,
            },
            expected: []string{
                "docker-compose", "-f", "docker-compose.yml",
                "up", "-d", "web", "db",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := buildComposeCommand(tt.params)
            if !reflect.DeepEqual(cmd, tt.expected) {
                t.Errorf("buildComposeCommand() = %v, want %v", cmd, tt.expected)
            }
        })
    }
}
```

### 3. End-to-End Tests

#### Full MCP Protocol Flow
```go
// Test complete request/response cycle
func TestE2E_ComposeUpDown(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    
    // Setup test environment with real docker-compose file
    testDir := setupTestEnvironment(t)
    defer cleanup(testDir)
    
    // Start MCP server
    server := startMCPServer(t)
    defer server.Stop()
    
    // Test compose up
    upResult := sendMCPRequest(t, server, "compose_up", map[string]interface{}{
        "compose_file": filepath.Join(testDir, "docker-compose.test.yml"),
        "detached":     true,
    })
    
    assert.NoError(t, upResult.Error)
    assert.Contains(t, upResult.Content, "Container started")
    
    // Verify containers are running
    containers := listRunningContainers(t)
    assert.GreaterOrEqual(t, len(containers), 1)
    
    // Test compose down
    downResult := sendMCPRequest(t, server, "compose_down", map[string]interface{}{
        "compose_file": filepath.Join(testDir, "docker-compose.test.yml"),
    })
    
    assert.NoError(t, downResult.Error)
    assert.Contains(t, downResult.Content, "Container stopped")
}
```

## MCP-Specific Testing Patterns

### Tool Registration and Discovery
```go
func TestToolRegistry_Registration(t *testing.T) {
    registry := NewRegistry()
    
    // Register multiple tools
    tools := []Tool{
        NewComposeUpTool(),
        NewComposeDownTool(),
        NewComposeLogsTool(),
    }
    
    for _, tool := range tools {
        registry.Register(tool)
    }
    
    // Test tool listing
    listed := registry.List()
    assert.Len(t, listed, len(tools))
    
    // Test tool retrieval by name
    upTool := registry.Get("compose_up")
    assert.NotNil(t, upTool)
    assert.Equal(t, "compose_up", upTool.Name())
}
```

### JSON Schema Validation
```go
func TestTool_InputSchemaValidation(t *testing.T) {
    tool := NewComposeUpTool()
    schema := tool.InputSchema()
    
    // Validate schema structure
    assert.Equal(t, "object", schema["type"])
    
    properties, ok := schema["properties"].(map[string]interface{})
    assert.True(t, ok)
    
    // Required properties
    required, ok := schema["required"].([]string)
    assert.True(t, ok)
    assert.Contains(t, required, "compose_file")
    
    // Property types
    composeFile, ok := properties["compose_file"].(map[string]interface{})
    assert.True(t, ok)
    assert.Equal(t, "string", composeFile["type"])
}
```

### Error Handling Patterns
```go
func TestMCP_ErrorHandling(t *testing.T) {
    tests := []struct {
        name           string
        request        *Request
        expectedCode   int
        expectedMsg    string
    }{
        {
            name: "method not found",
            request: &Request{
                JSONRPC: "2.0",
                ID:      1,
                Method:  "unknown/method",
            },
            expectedCode: -32601,
            expectedMsg:  "Method not found",
        },
        {
            name: "invalid params",
            request: &Request{
                JSONRPC: "2.0",
                ID:      2,
                Method:  "tools/call",
                Params:  json.RawMessage(`{"invalid": true}`),
            },
            expectedCode: -32602,
            expectedMsg:  "Invalid params",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := server.handleRequest(context.Background(), tt.request)
            assert.NotNil(t, resp.Error)
            assert.Equal(t, tt.expectedCode, resp.Error.Code)
            assert.Contains(t, resp.Error.Message, tt.expectedMsg)
        })
    }
}
```

## Common Pitfalls & Solutions

### 1. **Pitfall: Testing with Real Docker Daemon**
```go
// ❌ BAD: Requires Docker running, pollutes system
func TestCompose_RealDocker(t *testing.T) {
    exec.Command("docker-compose", "up", "-d").Run()
    // Test logic...
    exec.Command("docker-compose", "down").Run()
}

// ✅ GOOD: Mock external dependencies
func TestCompose_MockDocker(t *testing.T) {
    mockClient := &MockDockerClient{}
    service := NewComposeService(mockClient)
    
    err := service.Up(context.Background(), "test-project")
    assert.NoError(t, err)
    assert.Contains(t, mockClient.callHistory, "ComposeUp")
}
```

### 2. **Pitfall: Brittle Output Parsing Tests**
```go
// ❌ BAD: Fragile string matching
func TestParseLogs(t *testing.T) {
    output := "web_1  | Server started on port 3000"
    service := parseLogLine(output)
    assert.Equal(t, "web", service)  // Breaks if format changes
}

// ✅ GOOD: Regex-based parsing with multiple test cases
func TestParseLogs(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"web_1  | Server started", "web"},
        {"database_1 | Connection ready", "database"},
        {"service-name_1 | Log message", "service-name"},
    }
    
    for _, tt := range tests {
        result := parseLogLine(tt.input)
        assert.Equal(t, tt.expected, result)
    }
}
```

### 3. **Pitfall: Race Conditions in Async Operations**
```go
// ❌ BAD: No synchronization
func TestAsyncOperation(t *testing.T) {
    go performAsyncWork()
    time.Sleep(100 * time.Millisecond)  // Flaky timing
    assert.True(t, workCompleted)
}

// ✅ GOOD: Proper synchronization
func TestAsyncOperation(t *testing.T) {
    done := make(chan bool, 1)
    
    go func() {
        performAsyncWork()
        done <- true
    }()
    
    select {
    case <-done:
        assert.True(t, workCompleted)
    case <-time.After(5 * time.Second):
        t.Fatal("Operation timed out")
    }
}
```

### 4. **Pitfall: Poor Error Message Testing**
```go
// ❌ BAD: Generic error checking
func TestError(t *testing.T) {
    err := someOperation()
    assert.Error(t, err)  // Too generic
}

// ✅ GOOD: Specific error validation
func TestError(t *testing.T) {
    err := someOperation()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "docker-compose.yml not found")
    assert.IsType(t, &ComposeFileNotFoundError{}, err)
}
```

## Build & CI Integration

### Makefile Targets
```makefile
# Test automation
.PHONY: test test-unit test-integration test-e2e test-coverage

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	@go test -short -race -v ./internal/...

test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration -v ./test/integration/...

test-e2e:
	@echo "Running end-to-end tests..."
	@docker-compose -f test/docker-compose.test.yml up -d
	@go test -tags=e2e -v ./test/e2e/...
	@docker-compose -f test/docker-compose.test.yml down

test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
```

### CI Pipeline Example (GitHub Actions)
```yaml
# .github/workflows/test.yml
name: Test Suite

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: make test-unit

  integration-tests:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:20.10-dind
        options: --privileged
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Install docker-compose
        run: |
          sudo apt-get update
          sudo apt-get install -y docker-compose
      
      - name: Run integration tests
        run: make test-integration

  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Generate coverage report
        run: make test-coverage
      
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
```

## Performance Testing

### Benchmark Tests
```go
func BenchmarkOutputFilter(b *testing.B) {
    filter := NewFilter(Standard)
    testInput := generateLargeDockerComposeOutput(1000) // 1000 lines
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        filter.Filter(testInput)
    }
}

func BenchmarkToolExecution(b *testing.B) {
    tool := NewComposeUpTool()
    params := map[string]interface{}{
        "compose_file": "test-compose.yml",
        "detached":     true,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        tool.Execute(context.Background(), params)
    }
}
```

### Load Testing
```go
func TestConcurrentToolCalls(t *testing.T) {
    server := setupTestServer(t)
    concurrency := 10
    iterations := 100
    
    var wg sync.WaitGroup
    errors := make(chan error, concurrency*iterations)
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < iterations; j++ {
                err := callTool(server, "compose_ps", map[string]interface{}{})
                if err != nil {
                    errors <- err
                }
            }
        }()
    }
    
    wg.Wait()
    close(errors)
    
    errorCount := 0
    for err := range errors {
        errorCount++
        t.Logf("Concurrent call error: %v", err)
    }
    
    if errorCount > iterations*concurrency*0.05 { // 5% error rate threshold
        t.Errorf("Too many errors: %d/%d", errorCount, iterations*concurrency)
    }
}
```

## Integration Testing

### Docker Environment Setup
```go
// test/integration/setup.go
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
    testDir, err := os.MkdirTemp("", "mcp-test-*")
    require.NoError(t, err)
    
    // Create test docker-compose.yml
    composeContent := `
version: '3.8'
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb
      POSTGRES_PASSWORD: testpass
`
    
    composeFile := filepath.Join(testDir, "docker-compose.yml")
    err = os.WriteFile(composeFile, []byte(composeContent), 0644)
    require.NoError(t, err)
    
    return &TestEnvironment{
        TempDir:     testDir,
        ComposeFile: composeFile,
        cleanup: func() {
            os.RemoveAll(testDir)
        },
    }
}

type TestEnvironment struct {
    TempDir     string
    ComposeFile string
    cleanup     func()
}

func (te *TestEnvironment) Cleanup() {
    te.cleanup()
}
```

### MCP Server Testing
```go
func TestMCPServer_FullWorkflow(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup()
    
    server := startMCPServer(t)
    defer server.Stop()
    
    // Test initialization
    initResp := sendMCPRequest(t, server, "initialize", map[string]interface{}{
        "protocolVersion": "1.0.0",
        "clientInfo": map[string]string{
            "name":    "test-client",
            "version": "1.0.0",
        },
    })
    require.NoError(t, initResp.Error)
    
    // Test tool listing
    listResp := sendMCPRequest(t, server, "tools/list", nil)
    require.NoError(t, listResp.Error)
    
    tools := parseToolsList(t, listResp.Content)
    require.Contains(t, tools, "compose_up")
    require.Contains(t, tools, "compose_down")
    
    // Test compose up
    upResp := sendMCPRequest(t, server, "tools/call", map[string]interface{}{
        "name": "compose_up",
        "arguments": map[string]interface{}{
            "compose_file": env.ComposeFile,
            "detached":     true,
        },
    })
    require.NoError(t, upResp.Error)
    
    // Verify containers are running
    psResp := sendMCPRequest(t, server, "tools/call", map[string]interface{}{
        "name": "compose_ps",
        "arguments": map[string]interface{}{
            "compose_file": env.ComposeFile,
        },
    })
    require.NoError(t, psResp.Error)
    require.Contains(t, psResp.Content, "Up")
    
    // Test compose down
    downResp := sendMCPRequest(t, server, "tools/call", map[string]interface{}{
        "name": "compose_down",
        "arguments": map[string]interface{}{
            "compose_file": env.ComposeFile,
        },
    })
    require.NoError(t, downResp.Error)
}
```

## Docker Compose Specific Considerations

### 1. **Service Dependency Testing**
```go
func TestServiceDependencies(t *testing.T) {
    // Test that services start in correct order
    composeContent := `
version: '3.8'
services:
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: app
  web:
    image: nginx
    depends_on:
      - db
`
    
    env := setupComposeEnvironment(t, composeContent)
    defer env.Cleanup()
    
    result := callComposeTool(t, "compose_up", map[string]interface{}{
        "compose_file": env.ComposeFile,
    })
    
    // Verify DB started before web
    assert.Contains(t, result.Content, "Starting db")
    assert.Contains(t, result.Content, "Starting web")
    
    lines := strings.Split(result.Content, "\n")
    dbIndex := findLineIndex(lines, "db")
    webIndex := findLineIndex(lines, "web")
    assert.Less(t, dbIndex, webIndex)
}
```

### 2. **Network and Volume Testing**
```go
func TestNetworksAndVolumes(t *testing.T) {
    composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    volumes:
      - ./data:/usr/share/nginx/html
    networks:
      - webnet

networks:
  webnet:
    driver: bridge

volumes:
  data:
`
    
    env := setupComposeEnvironment(t, composeContent)
    defer env.Cleanup()
    
    // Test network creation
    result := callComposeTool(t, "compose_up", map[string]interface{}{
        "compose_file": env.ComposeFile,
    })
    
    assert.Contains(t, result.Content, "Creating network")
    assert.Contains(t, result.Content, "webnet")
    
    // Test volume creation
    assert.Contains(t, result.Content, "Creating volume")
}
```

### 3. **Environment Variable Handling**
```go
func TestEnvironmentVariables(t *testing.T) {
    // Set test environment variables
    os.Setenv("TEST_DB_HOST", "localhost")
    os.Setenv("TEST_DB_PORT", "5432")
    defer os.Unsetenv("TEST_DB_HOST")
    defer os.Unsetenv("TEST_DB_PORT")
    
    composeContent := `
version: '3.8'
services:
  app:
    image: alpine
    environment:
      - DB_HOST=${TEST_DB_HOST}
      - DB_PORT=${TEST_DB_PORT}
`
    
    env := setupComposeEnvironment(t, composeContent)
    defer env.Cleanup()
    
    result := callComposeTool(t, "compose_config", map[string]interface{}{
        "compose_file": env.ComposeFile,
    })
    
    // Verify environment variables are resolved
    assert.Contains(t, result.Content, "DB_HOST=localhost")
    assert.Contains(t, result.Content, "DB_PORT=5432")
}
```

### 4. **Output Filtering for Docker Compose**
```go
func TestComposeOutputFiltering(t *testing.T) {
    filter := NewComposeFilter(Standard)
    
    dockerComposeOutput := `
Pulling db (postgres:13)...
13: Pulling from library/postgres
e756f3fdd6a3: Pull complete
bf168a674899: Pull complete
Creating network "test_default" with the default driver
Creating volume "test_db_data" with default driver  
Creating test_db_1 ... done
Creating test_web_1 ... done
Attaching to test_db_1, test_web_1
db_1   | The files belonging to this database system will be owned by user "postgres"
web_1  | /docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to run files
web_1  | 2023-01-01 12:00:00 [info] Server ready on port 3000
`
    
    filtered := filter.Filter(dockerComposeOutput)
    
    // Should keep essential info
    assert.Contains(t, filtered, "Creating test_db_1 ... done")
    assert.Contains(t, filtered, "Server ready on port 3000")
    
    // Should filter verbose output
    assert.NotContains(t, filtered, "Pull complete")
    assert.NotContains(t, filtered, "docker-entrypoint.sh")
}
```

### 5. **Health Check Testing**
```go
func TestServiceHealthChecks(t *testing.T) {
    composeContent := `
version: '3.8'
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
`
    
    env := setupComposeEnvironment(t, composeContent)
    defer env.Cleanup()
    
    // Start service
    callComposeTool(t, "compose_up", map[string]interface{}{
        "compose_file": env.ComposeFile,
        "wait":         true,
    })
    
    // Check health status
    result := callComposeTool(t, "compose_ps", map[string]interface{}{
        "compose_file": env.ComposeFile,
        "format":       "json",
    })
    
    services := parseServicesJSON(t, result.Content)
    webService := findService(services, "web")
    assert.Equal(t, "healthy", webService.Health)
}
```

## Test Data Management

### Fixture Management
```go
// test/fixtures/compose_files.go
var ComposeFixtures = map[string]string{
    "simple_web": `
version: '3.8'
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`,
    
    "web_with_db": `
version: '3.8'
services:
  web:
    image: nginx:alpine
    depends_on:
      - db
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb
`,
    
    "complex_stack": `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  redis:
    image: redis:alpine
  worker:
    build: .
    command: worker
    depends_on:
      - redis
`,
}

func GetComposeFixture(name string) string {
    content, exists := ComposeFixtures[name]
    if !exists {
        panic(fmt.Sprintf("Fixture %s not found", name))
    }
    return content
}
```

## Summary

This guide provides a comprehensive framework for testing MCP servers with specific focus on Docker Compose implementations. Key takeaways:

1. **Layer your testing**: Unit tests for logic, integration tests for Docker interactions, E2E tests for full workflows
2. **Mock external dependencies**: Don't rely on Docker daemon availability in unit tests
3. **Test error conditions**: Network failures, permission issues, invalid configurations
4. **Validate output filtering**: Ensure verbose Docker output is properly filtered
5. **Use proper CI/CD**: Automate testing across different environments
6. **Performance matters**: Test concurrent operations and large-scale deployments

The patterns shown here have been proven effective in the Xcode Build MCP implementation and translate well to other MCP servers dealing with external tools and complex output parsing.