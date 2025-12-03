# Xcode Build Server MCP

A lightweight Model Context Protocol (MCP) server for Xcode operations, designed with token efficiency and simplicity in mind.

## Key Features

- **14 Unified Tools** - Complete Xcode workflow coverage with minimal tool count
- **Intelligent Output Filtering** - Reduces verbose xcodebuild output by 80-95% while preserving errors and failures
- **Failure-Aware** - Two-pass filtering guarantees test failures and build errors are never hidden
- **Smart Auto-Detection** - Automatically detects project types and selects appropriate simulators
- **Zero Dependencies** - Built entirely with Go standard library
- **Crash Detection** - Identifies segfaults, Swift fatal errors, and silent failures
- **Skipped Test Tracking** - Reports which tests are skipped with class names and reasons

## Design Philosophy

This server takes a minimalist approach:

- **Unified tools** instead of separate variants for project/workspace, simulator name/ID, etc.
- **Filtered output** that removes compilation noise while keeping actionable information
- **Simple configuration** with sensible defaults

## Quick Start

### Prerequisites

- Go 1.24.4 or higher
- Xcode 14.0 or higher
- macOS 12.0 or higher

### Installation

```bash
# Clone the repository
git clone https://github.com/jontolof/xcode-build-mcp.git
cd xcode-build-mcp

# Build the server
make build

# Or build manually
# go build -o bin/xcode-build-mcp cmd/server/main.go

# Install to PATH (optional)
sudo cp bin/xcode-build-mcp /usr/local/bin/
```

### Basic Usage

```bash
# Run the MCP server
xcode-build-mcp

# Run with debug logging
MCP_LOG_LEVEL=debug xcode-build-mcp
```

### Integration with MCP Clients

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "xcode-build": {
      "type": "stdio",
      "command": "/usr/local/bin/xcode-build-mcp",
      "args": [],
      "env": {
        "MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```

## The 14 Tools

### Build & Test Tools

#### 1. `xcode_build`
Universal build command that auto-detects project type and simulator.
```json
{
  "tool": "xcode_build",
  "parameters": {
    "project_path": ".",
    "project": "MyApp.xcodeproj",
    "scheme": "MyApp",
    "configuration": "Debug"
  }
}
```

#### 2. `xcode_test`
Universal test execution with parsed results.
```json
{
  "tool": "xcode_test",
  "parameters": {
    "project_path": ".",
    "project": "MyApp.xcodeproj",
    "scheme": "MyAppTests"
  }
}
```

#### 3. `xcode_clean`
Clean build artifacts and derived data.
```json
{
  "tool": "xcode_clean",
  "parameters": {
    "project_path": ".",
    "project": "MyApp.xcodeproj",
    "clean_build": true
  }
}
```

### Discovery Tools

#### 4. `discover_projects`
Find all Xcode projects in directory tree.
```json
{
  "tool": "discover_projects",
  "parameters": {
    "root_path": ".",
    "max_depth": 3
  }
}
```

#### 5. `list_schemes`
List available build schemes.
```json
{
  "tool": "list_schemes",
  "parameters": {
    "project_path": ".",
    "project": "MyApp.xcodeproj"
  }
}
```

#### 6. `list_simulators`
List available iOS/macOS simulators.
```json
{
  "tool": "list_simulators",
  "parameters": {
    "platform": "iOS",
    "available": true
  }
}
```

### Runtime Tools

#### 7. `simulator_control`
Boot, shutdown, or reset simulators.
```json
{
  "tool": "simulator_control",
  "parameters": {
    "udid": "SIMULATOR-UDID-HERE",
    "action": "boot"
  }
}
```

#### 8. `install_app`
Install apps to simulators or devices.
```json
{
  "tool": "install_app",
  "parameters": {
    "app_path": "build/MyApp.app",
    "device_type": "iPhone"
  }
}
```

#### 9. `launch_app`
Launch installed apps with optional arguments.
```json
{
  "tool": "launch_app",
  "parameters": {
    "bundle_id": "com.example.myapp",
    "device_type": "iPhone",
    "arguments": ["--debug", "--mock-data"]
  }
}
```

### Debug Tools

#### 10. `capture_logs`
Capture and filter device/simulator logs.
```json
{
  "tool": "capture_logs",
  "parameters": {
    "device_type": "iPhone",
    "bundle_id": "com.example.myapp",
    "max_lines": 100
  }
}
```

#### 11. `screenshot`
Capture simulator screenshots.
```json
{
  "tool": "screenshot",
  "parameters": {
    "device_type": "iPhone",
    "output_path": "screenshots/login.png"
  }
}
```

#### 12. `describe_ui`
Get UI element hierarchy for testing.
```json
{
  "tool": "describe_ui",
  "parameters": {
    "device_type": "iPhone",
    "output_format": "tree"
  }
}
```

### Automation Tools

#### 13. `ui_interact`
Perform UI interactions (tap, swipe, type).
```json
{
  "tool": "ui_interact",
  "parameters": {
    "device_type": "iPhone",
    "action": "tap",
    "x": 100,
    "y": 200
  }
}
```

#### 14. `get_app_info`
Extract app metadata and information.
```json
{
  "tool": "get_app_info",
  "parameters": {
    "app_path": "build/MyApp.app",
    "include_entitlements": true
  }
}
```

## Output Filtering

Raw xcodebuild output can be extremely verbose (100K+ characters for a typical test run). This server filters output to show what matters:

### Output Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `minimal` | Errors and final result only | Quick status checks |
| `standard` | Errors, warnings, test summaries | Normal development (default) |
| `verbose` | Full output with reduced noise | Debugging build issues |

### What Gets Filtered

**Removed** (noise):
- Compilation command details
- Framework loading messages
- SwiftDriver internal output
- Code signing verbose logs

**Preserved** (important):
- Build/test success or failure
- Error messages with file/line info
- Test failure details
- Skipped test details (class name, reason)
- Warnings
- Final summaries

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |

### Tool Parameters

Output verbosity is controlled per-tool using the `output_mode` parameter:

```json
{
  "tool": "xcode_build",
  "parameters": {
    "scheme": "MyApp",
    "output_mode": "minimal"
  }
}
```

Available modes:
- `minimal` - Errors and critical information only
- `standard` - Errors, warnings, and summary (default)
- `verbose` - Full build output

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/jontolof/xcode-build-mcp.git
cd xcode-build-mcp

# Install dependencies (minimal)
go mod download

# Build
go build -o xcode-build-mcp cmd/server/main.go

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -cover ./...
```

### Project Structure

```
xcode-build-mcp/
├── cmd/server/          # Server entry point (main.go)
├── internal/
│   ├── mcp/            # MCP protocol implementation (JSON-RPC 2.0)
│   ├── xcode/          # Xcode command execution and parsing
│   ├── filter/         # Output filtering system
│   ├── cache/          # Smart caching for project/scheme detection
│   ├── tools/          # MCP tool implementations (14 tools)
│   ├── common/         # Shared interfaces and utilities
│   ├── metrics/        # Performance metrics tracking
│   └── session/        # Session management
├── pkg/types/          # Shared types and error handling
├── tests/              # Test fixtures and integration tests
│   ├── fixtures/       # Test data and mock files
│   ├── integration/    # Integration tests
│   └── mocks/          # Mock implementations
└── docs/               # Documentation and ADRs
    └── adr/            # Architectural Decision Records
```

### Testing

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run benchmarks
make bench

# Check coverage
make coverage
```

## Documentation

- [CHANGELOG](CHANGELOG.md) - Version history and release notes
- [Architectural Decision Records](docs/adr/) - Design decisions and rationale
- [Contributing](CONTRIBUTING.md) - How to contribute to this project

## Security

- No sensitive data is logged or cached
- All user paths are validated
- Command injection protection
- Secure handling of build artifacts

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- [GitHub Issues](https://github.com/jontolof/xcode-build-mcp/issues) - Bug reports and feature requests
- [Discussions](https://github.com/jontolof/xcode-build-mcp/discussions) - General discussions
- [Wiki](https://github.com/jontolof/xcode-build-mcp/wiki) - Additional documentation

## Project Status

This server is stable and actively maintained. All 14 tools are implemented and tested.

See the [CHANGELOG](CHANGELOG.md) for recent updates.
