package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/xcode"
)

// testLogger implements common.Logger for testing
type testLogger struct {
	logs []string
}

func (l *testLogger) Printf(format string, v ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf(format, v...))
}

func (l *testLogger) Println(v ...interface{}) {
	l.logs = append(l.logs, fmt.Sprint(v...))
}

func (l *testLogger) Logs() []string {
	return l.logs
}

// mockExecutor implements a mock xcode.Executor for testing
type mockExecutor struct {
	result    *xcode.CommandResult
	results   []*xcode.CommandResult
	callCount int
}

func (e *mockExecutor) Execute(ctx context.Context, args []string) (*xcode.CommandResult, error) {
	e.callCount++

	if len(e.results) > 0 {
		if e.callCount <= len(e.results) {
			return e.results[e.callCount-1], nil
		}
		return e.results[len(e.results)-1], nil
	}

	if e.result != nil {
		return e.result, nil
	}

	// Default success result
	return &xcode.CommandResult{
		ExitCode: 0,
		Output:   "",
		Error:    nil,
		Duration: time.Millisecond * 10,
	}, nil
}

func (e *mockExecutor) FindXcodeCommand() (string, error) {
	return "/usr/bin/xcodebuild", nil
}

func (e *mockExecutor) BuildXcodeArgs(command string, params map[string]interface{}) ([]string, error) {
	args := []string{command}

	if workspace, ok := params["workspace"]; ok {
		args = append(args, "-workspace", workspace.(string))
	}
	if project, ok := params["project"]; ok {
		args = append(args, "-project", project.(string))
	}
	if scheme, ok := params["scheme"]; ok {
		args = append(args, "-scheme", scheme.(string))
	}
	if destination, ok := params["destination"]; ok {
		args = append(args, "-destination", destination.(string))
	}

	return args, nil
}

func (e *mockExecutor) CallCount() int {
	return e.callCount
}

// Mock data for testing
const mockSimulatorListJSON = `{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-17-0": [
      {
        "dataPath": "/Users/test/Library/Developer/CoreSimulator/Devices/ABC123/data",
        "dataPathSize": 1234567890,
        "logPath": "/Users/test/Library/Logs/CoreSimulator/ABC123",
        "udid": "ABC123-DEF4-5678-9ABC-DEF123456789",
        "isAvailable": true,
        "logPathSize": 123456,
        "deviceTypeIdentifier": "com.apple.CoreSimulator.SimDeviceType.iPhone-15",
        "state": "Shutdown",
        "name": "iPhone 15"
      }
    ]
  }
}`

const mockProjectListOutput = `Information about project "MyApp":
    Targets:
        MyApp
        MyAppTests

    Build Configurations:
        Debug
        Release

    If no build configuration is specified and -scheme is not passed then "Release" is used.

    Schemes:
        MyApp
        MyAppTests`

// Test utility functions
func newMockExecutorWithResults(results []*xcode.CommandResult) *mockExecutor {
	return &mockExecutor{results: results}
}

func newMockExecutorWithResult(result *xcode.CommandResult) *mockExecutor {
	return &mockExecutor{result: result}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
