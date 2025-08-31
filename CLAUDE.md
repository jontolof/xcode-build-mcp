# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT: Communication Guidelines

**NEVER reference Claude, Claude Code, Anthropic, or any AI assistant in:**
- Code comments
- Git commit messages  
- Documentation
- Error messages
- Log output
- Variable/function names
- Any project communication

This is a professional open-source project. All code and documentation should be written as if by a human developer, focusing solely on the technical implementation without mentioning AI assistance.

## Project Overview

This repository contains the design specification for an optimized Xcode Build Server MCP (Model Context Protocol) implementation. The project aims to reduce the current 83-tool implementation down to 14 essential tools, achieving an 83% reduction in token usage while maintaining full functionality for iOS/macOS development workflows.

**Core Mission**: Reduce xcodebuild command output by 90%+ while preserving essential information, preventing context flooding during development sessions.

## Repository Status

This is a **fully implemented MCP server** with all 14 essential tools operational and tested. The implementation is complete and production-ready, achieving 90%+ output reduction while maintaining full functionality.

## Key Concepts

### MCP (Model Context Protocol)
MCP servers provide tools that AI assistants can use to interact with external systems. In this case, the server would provide tools for Xcode build operations, simulator management, and iOS/macOS app testing.

### Token Optimization Strategy
The specification identifies that the current xcode-build MCP implementation uses ~47,000 tokens across 83 redundant tools. The optimized design consolidates functionality into 14 essential tools (~8,000 tokens), achieving:
- 5.9x reduction in tool count
- 83% reduction in token usage
- 100% coverage of common development workflows

## Essential Tools (Fully Implemented)

The 14 essential tools successfully implemented:

1. **xcode_build** - Universal build command (replaces 24 current tools)
2. **xcode_test** - Universal test command (replaces 12 current tools)
3. **xcode_clean** - Clean build artifacts
4. **discover_projects** - Find Xcode projects/workspaces
5. **list_schemes** - List available build schemes
6. **list_simulators** - List available simulators
7. **simulator_control** - Boot/shutdown/reset simulators
8. **install_app** - Install apps to simulators/devices
9. **launch_app** - Launch installed apps
10. **capture_logs** - Unified logging interface
11. **screenshot** - Capture simulator screenshots
12. **describe_ui** - Get UI hierarchy
13. **ui_interact** - Perform UI interactions
14. **get_app_info** - Extract app metadata

## Go Implementation Strategy

### Project Setup

```bash
# Initialize Go module
go mod init github.com/[username]/xcode-build-mcp

# Build server
go build -o xcode-build-mcp cmd/server/main.go

# Run with debug logging
MCP_LOG_LEVEL=debug go run cmd/server/main.go stdio

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...

# Vet code for issues
go vet ./...
```

### Project Structure (Go Implementation)

```
xcode-build-mcp/
├── cmd/server/          # Server entry point (main.go)
├── internal/
│   ├── mcp/            # MCP protocol implementation (JSON-RPC 2.0)
│   ├── xcode/          # Xcode command execution and parsing
│   ├── filter/         # Output filtering system (90%+ reduction)
│   ├── cache/          # Smart caching for project/scheme detection
│   ├── parallel/       # Parallel execution for independent operations
│   ├── metrics/        # Performance metrics tracking
│   ├── tools/          # MCP tool implementations (14 tools)
│   └── session/        # Session management for long-running builds
├── docs/               # Documentation
└── tests/              # Test fixtures and integration tests
```

### Implementation Phases

#### Phase 1: Core Infrastructure
1. **MCP Server Foundation**
   - JSON-RPC 2.0 protocol over stdio
   - Tool registration and dispatch
   - Error handling and logging

2. **Xcode Integration Layer**
   - Command execution wrapper
   - Output parsing utilities
   - Error detection and categorization

#### Phase 2: Essential Tools (First 7)
1. **xcode_build** - Universal build with smart detection
2. **xcode_test** - Test execution with parsed results
3. **xcode_clean** - Clean with derived data support
4. **discover_projects** - Fast project/workspace discovery
5. **list_schemes** - Scheme enumeration with caching
6. **list_simulators** - Simulator listing with filtering
7. **simulator_control** - Unified simulator management

#### Phase 3: Runtime Tools (Next 7)
8. **install_app** - Universal app installation
9. **launch_app** - App launching with output capture
10. **capture_logs** - Filtered log streaming
11. **screenshot** - Simulator screenshot capture
12. **describe_ui** - UI hierarchy extraction
13. **ui_interact** - Consolidated UI automation
14. **get_app_info** - Metadata extraction

#### Phase 4: Optimization Features
- **Smart Caching**: LRU cache for project/scheme data
- **Parallel Execution**: Worker pools for independent operations
- **Output Filtering**: 90%+ reduction in verbose output
- **Session Management**: Long-running operation tracking
- **Metrics Collection**: Performance and filtering statistics

### Key Go Implementation Patterns

```go
// Use standard library whenever possible
import (
    "encoding/json"
    "os/exec"
    "strings"
    "sync"
    "time"
)

// Minimal external dependencies
// Prefer standard library over third-party packages

// Clean architecture with separated concerns
type XcodeService interface {
    Build(ctx context.Context, params BuildParams) (*BuildResult, error)
}

// Repository pattern for data access
type ProjectRepository interface {
    FindProjects(root string, maxDepth int) ([]Project, error)
}

// Intelligent output filtering
type OutputFilter struct {
    rules []FilterRule
    cache *lru.Cache
}
```

## Architecture Principles

1. **Unified Interfaces**: Each tool should handle multiple scenarios (project/workspace, simulator/device) through smart parameter detection
2. **Minimal Token Usage**: Tool descriptions should be concise while remaining clear
3. **Progressive Output**: Show filtered, relevant output by default with full output available on request
4. **Error Recovery**: Provide clear, actionable error messages that help users resolve issues
5. **Minimal Dependencies**: Primarily use Go standard library, avoid unnecessary third-party packages
6. **Clean Architecture**: Separated concerns with repository pattern and service layers
7. **Intelligent Filtering**: Reduce output by 90%+ while preserving essential information

## Output Filtering Strategy

### What to Keep (Essential Information)
- Build/test success or failure status
- Error messages and warnings
- File paths for generated artifacts
- Test results summary
- Critical configuration issues
- Progress indicators for long operations

### What to Filter (Redundant Noise)
- Verbose compilation details
- Repetitive framework messages
- Internal xcodebuild diagnostics
- Duplicate warnings
- Detailed dependency resolution logs
- Non-critical status updates

### Filtering Implementation
```go
// Example filter rules
type FilterRule struct {
    Pattern  string
    Action   FilterAction // KEEP, REMOVE, SUMMARIZE
    Priority int
}

// Progressive disclosure
type OutputMode string
const (
    Minimal  OutputMode = "minimal"  // Errors only
    Standard OutputMode = "standard" // Errors + warnings + summary
    Verbose  OutputMode = "verbose"  // Everything
)
```

## Testing Approach

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestXcodeBuild ./internal/xcode

# Benchmark performance
go test -bench=. ./internal/filter
```

### Test Categories
- **Unit tests**: Core logic, filtering rules, parsing utilities
- **Integration tests**: Actual xcodebuild command execution
- **Mock tests**: Simulated xcodebuild responses for CI/CD
- **Performance tests**: Filtering efficiency and speed
- **End-to-end tests**: Full MCP protocol flow with all 14 tools

## Performance Optimization

### Caching Strategy
- Cache project/workspace detection results (5-minute TTL)
- Cache scheme listings (invalidate on file changes)
- Cache simulator listings (30-second TTL)
- Use file modification times for cache invalidation

### Parallel Execution
- Run independent operations concurrently
- Use worker pools for multiple simulator operations
- Stream output as it's generated (don't wait for completion)
- Implement context cancellation for long-running operations

### Memory Management
- Stream large outputs instead of buffering
- Use sync.Pool for frequently allocated objects
- Implement circular buffers for log capture
- Clean up completed sessions promptly