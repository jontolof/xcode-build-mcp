# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive crash detection system (ADR-0001)
  - Signal-based crash detection (SIGSEGV, SIGABRT, SIGKILL, etc.)
  - Swift fatal error detection (`fatalError()`, `preconditionFailure()`, etc.)
  - Simulator crash correlation via DiagnosticReports
  - Silent failure detection
  - Detailed crash type classification
- Two-pass failure-aware output filtering (ADR-0002)
  - Guarantees test failures are never truncated
  - 90%+ output reduction while preserving all failures
  - XCResult bundle parsing as authoritative test source
  - Silent test failure handling (ViewInspector tests)
- Skipped test tracking
  - Complete test accounting (passed + failed + skipped = total)
  - Unknown test status handling with debug warnings
- Test bundle detection and tracking
- Architectural Decision Records (ADR) system

### Changed
- Increased output limits for reliable failure reporting (ADR-0003)
  - Standard mode: 5K → 40K characters
  - Prevents truncation of test failures in large test suites
- Improved test failure detection
  - Exit code validation
  - XCResult + text parsing dual approach
  - Accurate test counts in all scenarios
- Enhanced debug logging
  - Test result discrepancies
  - XCResult parsing details
  - Filter statistics

### Fixed
- Test failure detection bugs
  - Fixed scanner buffer overflow issues
  - Fixed exit code 65 handling
  - Fixed silent test failures not being reported
  - Fixed misleading "All tests passed" when failures exist
- Token overflow in large test runs
- Output filtering edge cases
- MCP server connection stability

### Documentation
- Restructured to ADR approach
- Added comprehensive ADRs for major decisions
- Removed obsolete implementation guides
- Improved project documentation structure

## [0.1.0] - 2025-11-15

### Added
- Initial MCP server implementation
- 14 essential Xcode tools
  - `xcode_build` - Universal build command
  - `xcode_test` - Universal test command
  - `xcode_clean` - Clean build artifacts
  - `discover_projects` - Find Xcode projects/workspaces
  - `list_schemes` - List available build schemes
  - `list_simulators` - List available simulators
  - `simulator_control` - Boot/shutdown/reset simulators
  - `install_app` - Install apps to simulators/devices
  - `launch_app` - Launch installed apps
  - `capture_logs` - Unified logging interface
  - `screenshot` - Capture simulator screenshots
  - `describe_ui` - Get UI hierarchy
  - `ui_interact` - Perform UI interactions
  - `get_app_info` - Extract app metadata
- Intelligent output filtering system
- Smart project/workspace detection
- Simulator auto-selection
- LRU caching for project data
- Zero-dependency Go implementation

### Performance
- 83% token reduction (47K → 8K tokens)
- 90%+ output filtering of xcodebuild verbose output

## Version History Notes

- **Unreleased**: Current development (crash detection, filtering improvements)
- **0.1.0**: Initial release with 14 essential tools

---

## How to Update This Changelog

When preparing a release:
1. Move items from [Unreleased] to a new version section
2. Add the release date
3. Update version links at bottom
4. Create a git tag: `git tag v0.2.0`
5. Push tags: `git push --tags`

## Categories

Use these standard categories:
- **Added** - New features
- **Changed** - Changes in existing functionality
- **Deprecated** - Soon-to-be-removed features
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Vulnerability fixes
- **Performance** - Performance improvements
- **Documentation** - Documentation changes

[Unreleased]: https://github.com/jontolof/xcode-build-mcp/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/jontolof/xcode-build-mcp/releases/tag/v0.1.0
