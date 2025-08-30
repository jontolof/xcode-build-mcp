package xcode

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type testLogger struct{}

func (l *testLogger) Printf(format string, v ...interface{}) {}
func (l *testLogger) Println(v ...interface{})              {}

func TestNewExecutor(t *testing.T) {
	logger := &testLogger{}
	executor := NewExecutor(logger)
	
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}
	
	if executor.logger != logger {
		t.Error("Logger not properly set")
	}
}

func TestExecutor_FindXcodeCommand(t *testing.T) {
	logger := &testLogger{}
	executor := NewExecutor(logger)
	
	// This test will vary by system, but should not panic
	path, err := executor.FindXcodeCommand()
	
	// On systems without Xcode, this should return an error
	// On systems with Xcode, this should return a valid path
	if err != nil {
		t.Logf("Xcode not found (expected on non-Mac systems): %v", err)
	} else {
		t.Logf("Found xcodebuild at: %s", path)
		if path == "" {
			t.Error("FindXcodeCommand returned empty path without error")
		}
	}
}

func TestExecutor_BuildXcodeArgs_Build(t *testing.T) {
	logger := &testLogger{}
	executor := NewExecutor(logger)
	
	params := &types.BuildParams{
		Project:       "MyProject.xcodeproj",
		Scheme:        "MyScheme",
		Configuration: "Debug",
		SDK:           "iphonesimulator",
	}
	
	// Mock finding xcodebuild for testing
	args, err := executor.buildBuildArgs([]string{"xcodebuild"}, params)
	if err != nil {
		t.Fatalf("buildBuildArgs failed: %v", err)
	}
	
	expected := []string{
		"xcodebuild", "build",
		"-project", "MyProject.xcodeproj",
		"-scheme", "MyScheme",
		"-configuration", "Debug",
		"-sdk", "iphonesimulator",
	}
	
	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}
	
	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("Expected arg[%d] = %q, got %q", i, arg, args[i])
		}
	}
}

func TestExecutor_BuildXcodeArgs_Test(t *testing.T) {
	logger := &testLogger{}
	executor := NewExecutor(logger)
	
	params := &types.TestParams{
		Workspace:   "MyWorkspace.xcworkspace",
		Scheme:      "MyScheme",
		Destination: "platform=iOS Simulator,name=iPhone 15",
	}
	
	args, err := executor.buildTestArgs([]string{"xcodebuild"}, params)
	if err != nil {
		t.Fatalf("buildTestArgs failed: %v", err)
	}
	
	expected := []string{
		"xcodebuild", "test",
		"-workspace", "MyWorkspace.xcworkspace",
		"-scheme", "MyScheme",
		"-destination", "platform=iOS Simulator,name=iPhone 15",
		"-parallel-testing-enabled", "NO",
	}
	
	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}
	
	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("Expected arg[%d] = %q, got %q", i, arg, args[i])
		}
	}
}

func TestExecutor_BuildXcodeArgs_Clean(t *testing.T) {
	logger := &testLogger{}
	executor := NewExecutor(logger)
	
	params := &types.CleanParams{
		Project: "MyProject.xcodeproj",
		Target:  "MyTarget",
	}
	
	args, err := executor.buildCleanArgs([]string{"xcodebuild"}, params)
	if err != nil {
		t.Fatalf("buildCleanArgs failed: %v", err)
	}
	
	expected := []string{
		"xcodebuild", "clean",
		"-project", "MyProject.xcodeproj",
		"-target", "MyTarget",
	}
	
	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}
	
	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("Expected arg[%d] = %q, got %q", i, arg, args[i])
		}
	}
}

func TestExecutor_ExecuteCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}
	
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)
	executor := NewExecutor(logger)
	
	// Test with a simple command that should work on most systems
	ctx := context.Background()
	result, err := executor.ExecuteCommand(ctx, []string{"echo", "hello world"})
	
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("ExecuteCommand returned nil result")
	}
	
	if !result.Success() {
		t.Errorf("Expected command to succeed, got exit code %d", result.ExitCode)
	}
	
	if result.Output == "" {
		t.Error("Expected output from echo command")
	}
	
	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestCommandResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		result   CommandResult
		expected bool
	}{
		{
			name:     "successful command",
			result:   CommandResult{ExitCode: 0, Error: nil},
			expected: true,
		},
		{
			name:     "failed command with exit code",
			result:   CommandResult{ExitCode: 1, Error: nil},
			expected: false,
		},
		{
			name:     "failed command with error",
			result:   CommandResult{ExitCode: 0, Error: os.ErrNotExist},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Success(); got != tt.expected {
				t.Errorf("Success() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCommandResult_HasOutput(t *testing.T) {
	tests := []struct {
		name     string
		result   CommandResult
		expected bool
	}{
		{
			name:     "has output",
			result:   CommandResult{Output: "some output"},
			expected: true,
		},
		{
			name:     "empty output",
			result:   CommandResult{Output: ""},
			expected: false,
		},
		{
			name:     "whitespace only",
			result:   CommandResult{Output: "   \n\t  "},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasOutput(); got != tt.expected {
				t.Errorf("HasOutput() = %v, expected %v", got, tt.expected)
			}
		})
	}
}