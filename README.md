# Xcode Build Server MCP

A highly optimized Model Context Protocol (MCP) server for Xcode operations that reduces token usage by 83% while maintaining full functionality for iOS/macOS development workflows.

## 🎯 Key Features

- **83% Token Reduction**: From ~47,000 to ~8,000 tokens (14 tools vs 83)
- **90%+ Output Filtering**: Intelligent filtering of verbose xcodebuild output
- **Smart Auto-Detection**: Automatically detects project types and selects appropriate simulators
- **Unified Interface**: Single tools for multiple scenarios instead of variant proliferation
- **Zero Dependencies**: Built with Go standard library for maximum reliability
- **Real-time Streaming**: Progressive output for long-running operations
- **Intelligent Caching**: LRU cache for frequently accessed project data

## 📊 Why This Project?

Popular xcode-build MCP implementation suffers from severe tool proliferation:
- 83 redundant tools consuming ~47,000 tokens
- Separate tools for project vs workspace (2x duplication)
- Separate tools for simulator name vs ID (2x duplication)  
- Separate tools for each platform (4x multiplication)
- Most developers use <10% of available tools

Our solution consolidates everything into **14 essential tools** that cover 100% of common workflows.

## 🚀 Quick Start

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

## 🛠️ The 14 Essential Tools

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

## 📈 Performance

### Token Usage Comparison

| Metric | Current Implementation | Our Implementation | Improvement |
|--------|----------------------|-------------------|-------------|
| Tool Count | 83 tools | 14 tools | 83% reduction |
| Token Usage | ~47,000 | ~8,000 | 83% reduction |
| Output Verbosity | 100% | <10% | 90%+ filtering |
| Response Time | Variable | <100ms (cached) | Consistent |

### Output Filtering Example

**Before (Raw xcodebuild output):**
```
CompileSwift normal x86_64 /Users/dev/MyApp/Sources/ViewControllers/LoginViewController.swift (in target 'MyApp' from project 'MyApp')
    cd /Users/dev/MyApp
    /Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/swift -frontend -c -primary-file...
[500+ more lines of compilation details]
```

**After (Filtered output):**
```
Building MyApp (Debug)...
✓ Compiled LoginViewController.swift
✓ Compiled AppDelegate.swift
✓ Linking MyApp.app
Build succeeded in 12.3s
Output: build/Debug-iphonesimulator/MyApp.app
```

## 🔧 Configuration

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

## 🧪 Development

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
│   ├── filter/         # Output filtering system (90%+ reduction)
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

## 📚 Documentation

- [CHANGELOG](CHANGELOG.md) - Version history and release notes
- [Architectural Decision Records](docs/adr/) - Why we made specific design decisions
- [Documentation Guide](docs/README.md) - Documentation philosophy and structure
- [Development Guidelines](CLAUDE.md) - Guidelines for working with this codebase

## 📊 Benchmarks

```bash
# Token usage benchmark
Original implementation: 46,950 tokens
Optimized implementation: 7,910 tokens
Reduction: 83.15%

# Output filtering benchmark
Input: 10,000 lines of xcodebuild output
Filtered output: 847 lines
Reduction: 91.53%
Processing time: 8.2ms

# Response time (cached operations)
discover_projects: 12ms
list_schemes: 8ms
list_simulators: 23ms
xcode_build (cached): 94ms
```

## 🔒 Security

- No sensitive data is logged or cached
- All user paths are validated
- Command injection protection
- Secure handling of build artifacts

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the need for efficient MCP servers
- Built for the iOS/macOS development community
- Optimized for AI-assisted development workflows

## 📞 Support

- [GitHub Issues](https://github.com/jontolof/xcode-build-mcp/issues) - Bug reports and feature requests
- [Discussions](https://github.com/jontolof/xcode-build-mcp/discussions) - General discussions
- [Wiki](https://github.com/jontolof/xcode-build-mcp/wiki) - Additional documentation

## 🚦 Project Status

**Current Phase**: Production Ready ✅

- [x] Design specification
- [x] Implementation plan
- [x] Core MCP server
- [x] Tool implementation (14 essential tools)
- [x] Testing & optimization
- [x] Output filtering (90%+ reduction)
- [x] Production release

---

Built with ❤️ for the iOS/macOS development community
