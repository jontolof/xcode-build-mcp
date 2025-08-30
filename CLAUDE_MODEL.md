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

A Model Context Protocol (MCP) server for Docker Compose operations that filters verbose Docker output to essential information only, preventing AI assistant context flooding.

**Core Mission**: Reduce Docker Compose command output by 90%+ while maintaining complete operational capability.

## Quick Start

```bash
# Initialize Go module
go mod init github.com/[your-username]/docker-compose-mcp

# Build and run
go build -o docker-compose-mcp cmd/server/main.go
MCP_LOG_LEVEL=debug go run cmd/server/main.go stdio

# Run tests
go test ./...
```

## Documentation

- **[Development Guide](docs/DEVELOPMENT.md)**: Build commands, testing, code standards
- **[Architecture Guide](docs/ARCHITECTURE.md)**: System design, components, patterns
- **[MCP Tools Reference](docs/MCP_TOOLS.md)**: Available tools and their usage
- **[Configuration Guide](docs/CONFIGURATION.md)**: Environment variables and settings

## Project Structure

```
docker-compose-mcp/
├── cmd/server/          # Server entry point (main.go)
├── internal/
│   ├── mcp/            # MCP protocol implementation
│   ├── compose/        # Docker Compose logic with optimization
│   ├── filter/         # Output filtering system
│   ├── cache/          # Smart configuration caching
│   ├── parallel/       # Parallel execution engine
│   ├── metrics/        # Filtering performance metrics
│   ├── tools/          # MCP tool implementations
│   └── session/        # Session management for long-running ops
├── docs/               # Detailed documentation
└── tests/              # Test files (13 passing test cases)
```

## Implementation Status

### Current Phase: Phase 5 Complete ✅
- [x] Project specification and documentation
- [x] Go module initialization and project structure
- [x] Complete MCP server with JSON-RPC 2.0 over stdio
- [x] All 14 MCP tools implemented with intelligent filtering
- [x] Advanced optimization features (caching, parallel execution, metrics)
- [x] Comprehensive test coverage (13 passing test cases)
- [x] 90%+ context reduction achieved

### Phase 5 Achievements
1. **14 Production-Ready MCP Tools**: 83% reduction from XcodeBuildMCP's 83 tools
2. **Smart Configuration Caching**: File integrity checking with LRU eviction
3. **Parallel Execution Engine**: Worker pools for independent operations
4. **Comprehensive Metrics**: Real-time filtering performance tracking
5. **Session Management**: Long-running operations (watch, logs)
6. **Database Operations**: Migration, reset, and backup tools

### Next Phase: Phase 6 - Production Polish
1. Claude Desktop integration and testing
2. Performance profiling and optimization
3. Documentation completion and examples
4. Package for distribution
5. Usage guides and tutorials

## Key Features

- **Intelligent Filtering**: Reduces Docker output by 90%+ while preserving essential information
- **MCP Protocol**: JSON-RPC 2.0 over stdio for AI assistant integration
- **Clean Architecture**: Separated concerns with repository pattern and service layers
- **Minimal Dependencies**: Primarily uses Go standard library

## Development Workflow

1. Check [Development Guide](docs/DEVELOPMENT.md) for commands
2. Review [Architecture Guide](docs/ARCHITECTURE.md) before major changes
3. Use [MCP Tools Reference](docs/MCP_TOOLS.md) for tool implementation
4. Configure via [Configuration Guide](docs/CONFIGURATION.md)