# ADR-0001: Crash Detection Architecture

**Status:** Accepted
**Date:** 2025-11-30
**Deciders:** Project Team
**Tags:** architecture, error-handling, reliability

---

## Context

The xcode-build-mcp server failed to properly detect and report when xcodebuild or the iOS Simulator crashed during execution. This caused LLMs to incorrectly believe operations succeeded when they actually crashed, leading to misleading responses and poor debugging experiences.

### Critical Gaps

1. **No signal detection** - Cannot distinguish between normal exit codes and signal-based termination (SIGSEGV, SIGKILL, SIGABRT)
2. **No crash type classification** - All non-zero exits treated equally
3. **No simulator crash correlation** - Doesn't monitor DiagnosticReports for simulator crashes
4. **Silent failure detection missing** - Cannot detect cases where xcodebuild exits without proper output
5. **Inadequate error messages** - Generic failures without actionable troubleshooting steps

## Decision

Implement comprehensive crash detection using:
1. **syscall.WaitStatus** for signal-based crash detection
2. **Crash type classification** with specific enum types
3. **Simulator crash correlation** via DiagnosticReports
4. **Silent failure detection** based on exit code + output analysis
5. **Swift fatal error detection** via pattern matching

## Alternatives Considered

### Option 1: Exit Code Only (Status Quo)

**Pros:**
- Simple implementation
- Cross-platform compatible
- Minimal code

**Cons:**
- Cannot distinguish crash types (SIGSEGV vs SIGABRT vs exit 1)
- No signal information
- Poor error messages
- Cannot detect simulator crashes
- Cannot detect silent failures

### Option 2: Polling DiagnosticReports

**Pros:**
- Can detect all crashes after the fact
- Platform-native approach

**Cons:**
- High latency (crash reports written asynchronously)
- Complex file monitoring
- Race conditions
- Doesn't help with process signals

### Option 3: syscall.WaitStatus + Pattern Matching (Chosen)

**Pros:**
- Immediate crash detection
- Accurate signal classification
- Can detect Swift fatal errors
- Can correlate with simulator crashes
- Provides detailed error context

**Cons:**
- Platform-specific code (Unix-only)
- More complex implementation
- Requires pattern matching for output

## Consequences

### Positive

- **Accurate crash reporting** - LLMs now receive correct failure information
- **Better debugging** - Clear crash types (segfault, abort, timeout, etc.)
- **Swift fatal error detection** - Catches `fatalError()`, `preconditionFailure()`, etc.
- **Simulator crash correlation** - Links crashes to DiagnosticReports
- **Silent failure detection** - Catches exit code 65 with no failure output

### Negative

- **Platform dependency** - `syscall.WaitStatus` is Unix-specific (acceptable for macOS-only tool)
- **Code complexity** - More error handling paths to maintain
- **Pattern matching fragility** - Swift error patterns may change with compiler versions

### Neutral

- **Additional types** - New `CrashType` enum and `ProcessState` struct
- **Tool description changes** - MCP tool descriptions now document crash detection

## Implementation Notes

### Key Components

1. **Enhanced executor.go** - Process state capture using `syscall.WaitStatus`
2. **Crash type classification** - Signal → CrashType mapping
3. **Pattern matchers** - Swift fatal error detection in output
4. **Simulator crash detector** - DiagnosticReports monitoring
5. **Silent failure detector** - Exit code vs output validation

### Exit Code Mapping

- `0` → No crash
- `1` → Build failure
- `65` → Test failure
- `128+N` → Killed by signal N
- `-1` → Unknown error
- `-2` → Timeout
- `-3` → Interrupted

### Testing Strategy

- Unit tests for crash type classification
- Integration tests with actual crashes
- Mock simulator crash reports
- Swift fatal error test cases

## References

- Original issue: Test failures reported as successes
- Implementation commits: e09c41c, 1f87765, 15d4b34
- Related: ADR-0002 (Output Filtering)
