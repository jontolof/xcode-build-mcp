# Crash Detection Implementation Design

## Problem Statement

The xcode-build-mcp server currently fails to detect and properly report when xcodebuild or the iOS Simulator crashes during execution. This causes LLMs to incorrectly believe operations succeeded when they actually crashed, leading to misleading responses and poor debugging experiences.

## Critical Gaps Identified

1. **No signal detection** - Cannot distinguish between normal exit codes and signal-based termination (SIGSEGV, SIGKILL, SIGABRT)
2. **No crash type classification** - All non-zero exits treated equally
3. **No simulator crash correlation** - Doesn't monitor DiagnosticReports for simulator crashes
4. **Silent failure detection missing** - Cannot detect cases where xcodebuild exits without proper output
5. **Inadequate error messages** - Generic failures without actionable troubleshooting steps

## Solution Design

### Phase 1: Enhanced Process Monitoring (Core)

#### 1.1 syscall.WaitStatus Integration

**Location:** `internal/xcode/executor.go:107-114`

**Current Code:**
```go
if err != nil {
    if exitError, ok := err.(*exec.ExitError); ok {
        result.ExitCode = exitError.ExitCode()
    } else {
        result.ExitCode = -1
    }
    result.Error = err
}
```

**Enhanced Implementation:**
```go
if err != nil {
    result.Error = err

    if exitErr, ok := err.(*exec.ExitError); ok {
        // Platform-specific process state (Unix/Linux/macOS)
        if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
            result.ProcessState = &types.ProcessState{
                Exited:   ws.Exited(),
                Signaled: ws.Signaled(),
            }

            if ws.Signaled() {
                signal := ws.Signal()
                result.ProcessState.Signal = int(signal)
                result.ProcessState.SignalName = signal.String()
                result.CrashType = classifySignal(signal)
                result.ExitCode = 128 + int(signal)
            } else if ws.Exited() {
                result.ExitCode = ws.ExitStatus()
                result.CrashType = classifyExitCode(result.ExitCode)
            }

            if ws.CoreDump() {
                result.ProcessState.CoreDump = true
                result.CrashType = types.CrashTypeSegmentationFault
            }
        } else {
            result.ExitCode = exitErr.ExitCode()
            result.CrashType = classifyExitCode(result.ExitCode)
        }
    } else if ctx.Err() == context.DeadlineExceeded {
        result.ExitCode = -2
        result.CrashType = types.CrashTypeTimeout
    } else if ctx.Err() == context.Canceled {
        result.ExitCode = -3
        result.CrashType = types.CrashTypeInterrupted
    } else {
        result.ExitCode = -1
        result.CrashType = types.CrashTypeUnknown
    }
} else {
    result.ExitCode = 0
    result.CrashType = types.CrashTypeNone
}
```

#### 1.2 New Types in pkg/types/tools.go

```go
type CrashType string

const (
    CrashTypeNone              CrashType = "none"
    CrashTypeSegmentationFault CrashType = "segmentation_fault"
    CrashTypeAbort             CrashType = "abort"
    CrashTypeKilled            CrashType = "killed"
    CrashTypeInterrupted       CrashType = "interrupted"
    CrashTypeTerminated        CrashType = "terminated"
    CrashTypeTimeout           CrashType = "timeout"
    CrashTypeBuildFailure      CrashType = "build_failure"
    CrashTypeTestFailure       CrashType = "test_failure"
    CrashTypeSimulatorCrash    CrashType = "simulator_crash"
    CrashTypeUnknown           CrashType = "unknown"
)

type ProcessState struct {
    Exited     bool   `json:"exited"`
    Signaled   bool   `json:"signaled"`
    Signal     int    `json:"signal,omitempty"`
    SignalName string `json:"signal_name,omitempty"`
    CoreDump   bool   `json:"core_dump,omitempty"`
}

type CommandResult struct {
    Command      string        `json:"command"`
    Output       string        `json:"output"`
    StdoutOutput string        `json:"stdout_output"`
    StderrOutput string        `json:"stderr_output"`
    Duration     time.Duration `json:"duration"`
    ExitCode     int           `json:"exit_code"`
    Error        error         `json:"-"`
    ProcessState *ProcessState `json:"process_state,omitempty"`
    CrashType    CrashType     `json:"crash_type"`
}
```

#### 1.3 Signal Classification Helper

**Location:** `internal/xcode/executor.go` (new functions)

```go
func classifySignal(sig syscall.Signal) types.CrashType {
    switch sig {
    case syscall.SIGSEGV:
        return types.CrashTypeSegmentationFault
    case syscall.SIGABRT:
        return types.CrashTypeAbort
    case syscall.SIGKILL:
        return types.CrashTypeKilled
    case syscall.SIGINT:
        return types.CrashTypeInterrupted
    case syscall.SIGTERM:
        return types.CrashTypeTerminated
    default:
        return types.CrashTypeUnknown
    }
}

func classifyExitCode(exitCode int) types.CrashType {
    switch exitCode {
    case 0:
        return types.CrashTypeNone
    case 65:
        return types.CrashTypeBuildFailure
    case 70:
        return types.CrashTypeBuildFailure
    default:
        if exitCode > 128 && exitCode < 160 {
            return classifySignal(syscall.Signal(exitCode - 128))
        }
        return types.CrashTypeUnknown
    }
}
```

### Phase 2: Output Pattern Detection

#### 2.1 Crash Pattern Recognition

**Location:** `internal/xcode/parser.go` (add new regexes and struct)

```go
var (
    // Existing regexes...

    // Crash detection patterns
    testRunnerCrashedRegex      = regexp.MustCompile(`Test runner.*crashed|Testing failed.*crashed`)
    connectionInterruptedRegex  = regexp.MustCompile(`Connection interrupted|Connection with the remote side was unexpectedly closed`)
    earlyExitRegex              = regexp.MustCompile(`Early unexpected exit|operation never finished bootstrapping`)
    neverBeganTestingRegex      = regexp.MustCompile(`Test runner never began executing tests`)
    failedToLoadBundleRegex     = regexp.MustCompile(`Failed to load the test bundle`)
    simulatorBootTimeoutRegex   = regexp.MustCompile(`Simulator.*timed out|Failed to boot simulator`)
    testProcessCrashedRegex     = regexp.MustCompile(`Test process crashed`)
)

type CrashIndicators struct {
    TestRunnerCrashed     bool `json:"test_runner_crashed"`
    ConnectionInterrupted bool `json:"connection_interrupted"`
    EarlyExit             bool `json:"early_exit"`
    NeverBeganTesting     bool `json:"never_began_testing"`
    BundleLoadFailed      bool `json:"bundle_load_failed"`
    SimulatorBootTimeout  bool `json:"simulator_boot_timeout"`
    TestProcessCrashed    bool `json:"test_process_crashed"`
}

func (p *Parser) DetectCrashIndicators(output string) types.CrashIndicators {
    indicators := types.CrashIndicators{}

    scanner := bufio.NewScanner(strings.NewReader(output))
    for scanner.Scan() {
        line := scanner.Text()

        if testRunnerCrashedRegex.MatchString(line) {
            indicators.TestRunnerCrashed = true
        }
        if connectionInterruptedRegex.MatchString(line) {
            indicators.ConnectionInterrupted = true
        }
        if earlyExitRegex.MatchString(line) {
            indicators.EarlyExit = true
        }
        if neverBeganTestingRegex.MatchString(line) {
            indicators.NeverBeganTesting = true
        }
        if failedToLoadBundleRegex.MatchString(line) {
            indicators.BundleLoadFailed = true
        }
        if simulatorBootTimeoutRegex.MatchString(line) {
            indicators.SimulatorBootTimeout = true
        }
        if testProcessCrashedRegex.MatchString(line) {
            indicators.TestProcessCrashed = true
        }
    }

    return indicators
}
```

#### 2.2 Silent Failure Detection

**Location:** `internal/xcode/parser.go`

```go
func (p *Parser) DetectSilentFailure(output string, exitCode int) bool {
    // If exit code indicates failure but output is suspiciously small
    if exitCode != 0 && len(output) < 500 {
        return true
    }

    // If no success/failure markers found
    hasSuccessMarker := buildSuccessRegex.MatchString(output) ||
                        testSuccessRegex.MatchString(output)
    hasFailureMarker := buildFailedRegex.MatchString(output) ||
                        testFailedRegex.MatchString(output)

    if !hasSuccessMarker && !hasFailureMarker && exitCode != 0 {
        return true
    }

    return false
}
```

### Phase 3: Simulator Crash Detection

#### 3.1 Crash Report Monitor

**Location:** `internal/xcode/crash_detector.go` (new file)

```go
package xcode

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/jontolof/xcode-build-mcp/pkg/types"
)

type SimulatorCrashDetector struct {
    diagnosticPath string
    startTime      time.Time
}

func NewSimulatorCrashDetector() *SimulatorCrashDetector {
    homeDir, _ := os.UserHomeDir()
    return &SimulatorCrashDetector{
        diagnosticPath: filepath.Join(homeDir, "Library/Logs/DiagnosticReports"),
        startTime:      time.Now(),
    }
}

func (d *SimulatorCrashDetector) CheckForCrashes(processName string) ([]types.CrashReport, error) {
    var crashes []types.CrashReport

    files, err := filepath.Glob(filepath.Join(d.diagnosticPath, "*.ips"))
    if err != nil {
        return nil, err
    }

    for _, file := range files {
        info, err := os.Stat(file)
        if err != nil {
            continue
        }

        // Only check crashes that occurred during our execution
        if info.ModTime().Before(d.startTime) {
            continue
        }

        // Check if related to simulator or test processes
        baseName := filepath.Base(file)
        if strings.Contains(baseName, "Simulator") ||
           strings.Contains(baseName, "testmanagerd") ||
           strings.Contains(baseName, "xctest") ||
           (processName != "" && strings.Contains(baseName, processName)) {

            crash, err := d.parseCrashReport(file)
            if err == nil {
                crashes = append(crashes, crash)
            }
        }
    }

    return crashes, nil
}

func (d *SimulatorCrashDetector) parseCrashReport(path string) (types.CrashReport, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return types.CrashReport{}, err
    }

    var report map[string]interface{}
    if err := json.Unmarshal(data, &report); err != nil {
        return types.CrashReport{}, err
    }

    crash := types.CrashReport{
        FilePath:  path,
        Timestamp: time.Now(),
    }

    if procName, ok := report["procName"].(string); ok {
        crash.ProcessName = procName
    }

    if procPath, ok := report["procPath"].(string); ok {
        crash.ProcessPath = procPath
    }

    if exception, ok := report["exception"].(map[string]interface{}); ok {
        if exType, ok := exception["type"].(string); ok {
            crash.ExceptionType = exType
        }
    }

    if termination, ok := report["termination"].(map[string]interface{}); ok {
        if code, ok := termination["code"].(float64); ok {
            crash.Signal = int(code)
        }
    }

    return crash, nil
}
```

#### 3.2 CrashReport Type

**Location:** `pkg/types/tools.go`

```go
type CrashReport struct {
    ProcessName   string    `json:"process_name"`
    ProcessPath   string    `json:"process_path"`
    ExceptionType string    `json:"exception_type"`
    Signal        int       `json:"signal"`
    Timestamp     time.Time `json:"timestamp"`
    FilePath      string    `json:"file_path"`
}
```

### Phase 4: Enhanced Result Types

#### 4.1 Update TestResult and BuildResult

**Location:** `pkg/types/tools.go`

```go
type TestResult struct {
    Success         bool              `json:"success"`
    Duration        time.Duration     `json:"duration"`
    Output          string            `json:"output"`
    FilteredOutput  string            `json:"filtered_output"`
    TestSummary     TestSummary       `json:"test_summary"`
    Coverage        *Coverage         `json:"coverage,omitempty"`
    ExitCode        int               `json:"exit_code"`

    // Crash detection fields
    CrashType        CrashType         `json:"crash_type"`
    ProcessCrashed   bool              `json:"process_crashed"`
    ProcessState     *ProcessState     `json:"process_state,omitempty"`
    CrashIndicators  CrashIndicators   `json:"crash_indicators,omitempty"`
    SimulatorCrashes []CrashReport     `json:"simulator_crashes,omitempty"`
    SilentFailure    bool              `json:"silent_failure"`
}

type BuildResult struct {
    Success        bool                   `json:"success"`
    Duration       time.Duration          `json:"duration"`
    Output         string                 `json:"output"`
    FilteredOutput string                 `json:"filtered_output"`
    Errors         []BuildError           `json:"errors,omitempty"`
    Warnings       []BuildWarning         `json:"warnings,omitempty"`
    ArtifactPaths  []string               `json:"artifact_paths,omitempty"`
    BuildSettings  map[string]interface{} `json:"build_settings,omitempty"`
    ExitCode       int                    `json:"exit_code"`

    // Crash detection fields
    CrashType       CrashType       `json:"crash_type"`
    ProcessCrashed  bool            `json:"process_crashed"`
    ProcessState    *ProcessState   `json:"process_state,omitempty"`
    CrashIndicators CrashIndicators `json:"crash_indicators,omitempty"`
    SilentFailure   bool            `json:"silent_failure"`
}
```

### Phase 5: Integration and Error Messages

#### 5.1 Test Tool Integration

**Location:** `internal/tools/test.go`

```go
func (t *XcodeTestTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // ... existing parameter parsing ...

    // Initialize crash detector
    crashDetector := xcode.NewSimulatorCrashDetector()

    // Execute command
    result, err := t.executor.ExecuteCommand(ctx, cmdArgs)
    if err != nil {
        return "", err
    }

    // Parse test output
    testResult := t.parser.ParseTestOutput(result.Output)
    testResult.ExitCode = result.ExitCode
    testResult.CrashType = result.CrashType
    testResult.ProcessCrashed = result.ProcessState != nil && result.ProcessState.Signaled
    testResult.ProcessState = result.ProcessState

    // Detect crash patterns in output
    testResult.CrashIndicators = t.parser.DetectCrashIndicators(result.Output)

    // Check for silent failures
    testResult.SilentFailure = t.parser.DetectSilentFailure(result.Output, result.ExitCode)

    // Check for simulator crashes
    crashes, _ := crashDetector.CheckForCrashes("Simulator")
    if len(crashes) > 0 {
        testResult.SimulatorCrashes = crashes
        if testResult.CrashType == types.CrashTypeNone {
            testResult.CrashType = types.CrashTypeSimulatorCrash
        }
    }

    // Generate user-friendly error message if crashed
    if testResult.CrashType != types.CrashTypeNone || testResult.ProcessCrashed || testResult.SilentFailure {
        errorMsg := formatCrashErrorMessage(testResult)
        testResult.FilteredOutput = errorMsg + "\n\n" + testResult.FilteredOutput
    }

    // ... rest of response building ...
}
```

#### 5.2 Error Message Formatter

**Location:** `internal/tools/error_messages.go` (new file)

```go
package tools

import (
    "fmt"
    "strings"

    "github.com/jontolof/xcode-build-mcp/pkg/types"
)

func formatCrashErrorMessage(result *types.TestResult) string {
    var msg strings.Builder

    msg.WriteString("═══════════════════════════════════════════════════════\n")
    msg.WriteString("⚠️  CRASH DETECTED\n")
    msg.WriteString("═══════════════════════════════════════════════════════\n\n")

    // Crash type specific messages
    switch result.CrashType {
    case types.CrashTypeSegmentationFault:
        msg.WriteString("Type: Segmentation Fault (SIGSEGV)\n")
        msg.WriteString("Cause: Invalid memory access in xcodebuild or test code\n")
        msg.WriteString("Action: Check crash reports in ~/Library/Logs/DiagnosticReports/\n")

    case types.CrashTypeAbort:
        msg.WriteString("Type: Abort Signal (SIGABRT)\n")
        msg.WriteString("Cause: Assertion failure or unhandled exception\n")
        msg.WriteString("Action: Review test logs for assertion failures\n")

    case types.CrashTypeKilled:
        msg.WriteString("Type: Forcefully Killed (SIGKILL)\n")
        msg.WriteString("Cause: Out of memory, CI timeout, or manual termination\n")
        msg.WriteString("Action: Check system resources, increase timeouts\n")

    case types.CrashTypeTimeout:
        msg.WriteString("Type: Execution Timeout\n")
        msg.WriteString("Cause: Operation did not complete within time limit\n")
        msg.WriteString("Action: Increase timeout or investigate hung tests\n")

    case types.CrashTypeSimulatorCrash:
        msg.WriteString("Type: Simulator Crash\n")
        msg.WriteString(fmt.Sprintf("Found %d crash report(s):\n", len(result.SimulatorCrashes)))
        for _, crash := range result.SimulatorCrashes {
            msg.WriteString(fmt.Sprintf("  • %s: %s (Signal %d)\n",
                crash.ProcessName, crash.ExceptionType, crash.Signal))
            msg.WriteString(fmt.Sprintf("    Report: %s\n", crash.FilePath))
        }
    }

    // Pattern-based indicators
    if result.CrashIndicators.TestRunnerCrashed {
        msg.WriteString("\n⚠️  Test runner crashed during execution\n")
        msg.WriteString("Try: Disable parallel testing (-parallel-testing-enabled NO)\n")
    }

    if result.CrashIndicators.ConnectionInterrupted {
        msg.WriteString("\n⚠️  Connection to test runner interrupted\n")
        msg.WriteString("Try: Reboot simulator, check system resources\n")
    }

    if result.CrashIndicators.SimulatorBootTimeout {
        msg.WriteString("\n⚠️  Simulator failed to boot within timeout\n")
        msg.WriteString("Try: Kill all simulators and retry\n")
    }

    if result.SilentFailure {
        msg.WriteString("\n⚠️  Silent failure detected (minimal output)\n")
        msg.WriteString("Try: Run with verbose output, verify project configuration\n")
    }

    // Exit code information
    if result.ExitCode != 0 {
        msg.WriteString(fmt.Sprintf("\nExit Code: %d", result.ExitCode))
        switch result.ExitCode {
        case 65:
            msg.WriteString(" (Code signing, simulator timeout, or dependency issue)")
        case 70:
            msg.WriteString(" (Target not found, version mismatch, or device issue)")
        }
        msg.WriteString("\n")
    }

    msg.WriteString("\n═══════════════════════════════════════════════════════\n")

    return msg.String()
}

func formatBuildCrashErrorMessage(result *types.BuildResult) string {
    // Similar implementation for build crashes
    var msg strings.Builder

    msg.WriteString("═══════════════════════════════════════════════════════\n")
    msg.WriteString("⚠️  BUILD CRASH DETECTED\n")
    msg.WriteString("═══════════════════════════════════════════════════════\n\n")

    // Similar crash type handling as test results...

    return msg.String()
}
```

## Implementation Plan

### Step 1: Type Definitions (Day 1)
- [ ] Add CrashType enum to pkg/types/tools.go
- [ ] Add ProcessState struct
- [ ] Add CrashIndicators struct
- [ ] Add CrashReport struct
- [ ] Update CommandResult with crash fields
- [ ] Update TestResult with crash fields
- [ ] Update BuildResult with crash fields

### Step 2: Executor Enhancement (Day 1)
- [ ] Update ExecuteCommand to inspect syscall.WaitStatus
- [ ] Implement classifySignal helper
- [ ] Implement classifyExitCode helper
- [ ] Add logging for crash detection
- [ ] Test with simulated crashes

### Step 3: Parser Enhancement (Day 2)
- [ ] Add crash detection regex patterns
- [ ] Implement DetectCrashIndicators method
- [ ] Implement DetectSilentFailure method
- [ ] Add tests for pattern detection

### Step 4: Simulator Crash Detection (Day 2-3)
- [ ] Create internal/xcode/crash_detector.go
- [ ] Implement SimulatorCrashDetector
- [ ] Implement parseCrashReport method
- [ ] Test with real crash reports
- [ ] Add unit tests

### Step 5: Tool Integration (Day 3)
- [ ] Update test.go Execute method
- [ ] Update build.go Execute method
- [ ] Create error_messages.go
- [ ] Implement formatCrashErrorMessage
- [ ] Implement formatBuildCrashErrorMessage

### Step 6: Testing (Day 4)
- [ ] Create test cases for signal detection
- [ ] Create test cases for crash pattern detection
- [ ] Create test cases for simulator crash detection
- [ ] Integration tests with real xcodebuild crashes
- [ ] Validate error message formatting

### Step 7: Documentation (Day 4)
- [ ] Update README with crash detection capabilities
- [ ] Document exit codes and their meanings
- [ ] Create troubleshooting guide
- [ ] Add examples of crash output

## Testing Strategy

### Unit Tests
```go
// internal/xcode/executor_test.go
func TestExecutor_DetectSegmentationFault(t *testing.T)
func TestExecutor_DetectKilledProcess(t *testing.T)
func TestExecutor_DetectTimeout(t *testing.T)
func TestExecutor_ClassifySignals(t *testing.T)
func TestExecutor_ClassifyExitCodes(t *testing.T)

// internal/xcode/parser_test.go
func TestParser_DetectTestRunnerCrashed(t *testing.T)
func TestParser_DetectConnectionInterrupted(t *testing.T)
func TestParser_DetectSilentFailure(t *testing.T)

// internal/xcode/crash_detector_test.go
func TestSimulatorCrashDetector_FindCrashes(t *testing.T)
func TestSimulatorCrashDetector_ParseCrashReport(t *testing.T)
```

### Integration Tests
- Test with actual xcodebuild failures
- Test with simulated crashes (kill -11 PID)
- Test with simulator boot timeouts
- Test with real crash reports

## Exit Code Reference

| Code | Name | Meaning | Common Causes |
|------|------|---------|---------------|
| 0 | Success | Build/test succeeded | - |
| 64 | EX_USAGE | Command usage error | Malformed arguments |
| 65 | EX_DATAERR | Data error | Code signing, simulator timeout, dependencies |
| 66 | EX_NOINPUT | Input missing | Project/workspace not found |
| 70 | EX_SOFTWARE | Internal error | Target not found, version mismatch |
| 74 | EX_IOERR | I/O error | Cannot read/write files |
| 130 | SIGINT | Interrupted | Ctrl+C |
| 134 | SIGABRT | Abort | Assertion failure |
| 137 | SIGKILL | Killed | OOM, timeout, manual kill |
| 139 | SIGSEGV | Segmentation fault | Crash |
| 143 | SIGTERM | Terminated | Graceful shutdown request |

## Signal Reference

| Signal | Number | Exit Code | Meaning |
|--------|--------|-----------|---------|
| SIGINT | 2 | 130 | Interrupt (Ctrl+C) |
| SIGABRT | 6 | 134 | Abort signal |
| SIGKILL | 9 | 137 | Kill (uncatchable) |
| SIGSEGV | 11 | 139 | Segmentation fault |
| SIGTERM | 15 | 143 | Termination request |

## Success Criteria

1. **All crashes detected**: No crash goes unreported
2. **Clear error messages**: Users understand what failed and why
3. **Actionable guidance**: Error messages include troubleshooting steps
4. **Type classification**: Each crash type properly identified
5. **Simulator correlation**: Simulator crashes linked to test failures
6. **Silent failure detection**: Minimal output failures caught
7. **LLM clarity**: LLMs never misinterpret crashes as success

## Future Enhancements

1. **Automatic retry logic** for transient failures
2. **Crash analytics** and trend reporting
3. **Proactive health checks** before test execution
4. **Crash report symbolication** for stack traces
5. **Performance profiling** of crash detection overhead
