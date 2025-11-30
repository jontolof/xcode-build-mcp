package xcode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type Executor struct {
	logger common.Logger
}

func NewExecutor(logger common.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

func (e *Executor) ExecuteCommand(ctx context.Context, args []string) (*CommandResult, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command arguments provided")
	}

	e.logger.Printf("Executing command: %s %s", args[0], strings.Join(args[1:], " "))

	start := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Set up pipes for capturing output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Capture output
	outputChan := make(chan string, 2)

	// Read stdout - use safe scanner to handle lines >64KB
	go func() {
		defer func() { outputChan <- "" }()
		scanner := newSafeScanner(stdout)
		var output strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line)
			output.WriteString("\n")
		}
		outputChan <- output.String()
	}()

	// Read stderr - use safe scanner to handle lines >64KB
	go func() {
		defer func() { outputChan <- "" }()
		scanner := newSafeScanner(stderr)
		var output strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line)
			output.WriteString("\n")
		}
		outputChan <- output.String()
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	duration := time.Since(start)

	// Get outputs
	stdoutOutput := <-outputChan
	stderrOutput := <-outputChan

	var combinedOutput strings.Builder
	if stdoutOutput != "" {
		combinedOutput.WriteString(stdoutOutput)
	}
	if stderrOutput != "" {
		combinedOutput.WriteString(stderrOutput)
	}

	result := &CommandResult{
		Command:      strings.Join(args, " "),
		Output:       combinedOutput.String(),
		StdoutOutput: stdoutOutput,
		StderrOutput: stderrOutput,
		Duration:     duration,
		ExitCode:     0,
		CrashType:    types.CrashTypeNone,
	}

	// Enhanced crash detection
	if err != nil {
		result.Error = err

		if exitErr, ok := err.(*exec.ExitError); ok {
			// Get platform-specific process state (Unix/Linux/macOS)
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ProcessState = &types.ProcessState{
					Exited:   ws.Exited(),
					Signaled: ws.Signaled(),
				}

				if ws.Signaled() {
					// Process was killed by a signal
					signal := ws.Signal()
					result.ProcessState.Signal = int(signal)
					result.ProcessState.SignalName = signal.String()
					result.CrashType = classifySignal(signal)
					result.ExitCode = 128 + int(signal)

					e.logger.Printf("Command terminated by signal: %s (%d)",
						signal.String(), signal)
				} else if ws.Exited() {
					// Normal exit with exit code
					result.ExitCode = ws.ExitStatus()
					result.CrashType = classifyExitCode(result.ExitCode)

					e.logger.Printf("Command exited with code: %d", result.ExitCode)
				}

				if ws.CoreDump() {
					result.ProcessState.CoreDump = true
					result.CrashType = types.CrashTypeSegmentationFault
					e.logger.Printf("Command produced core dump")
				}
			} else {
				// Fallback for non-Unix systems or when casting fails
				result.ExitCode = exitErr.ExitCode()
				result.CrashType = classifyExitCode(result.ExitCode)
			}
		} else if ctx.Err() == context.DeadlineExceeded {
			// Timeout
			result.ExitCode = -2
			result.CrashType = types.CrashTypeTimeout
			e.logger.Printf("Command timed out after %v", duration)
		} else if ctx.Err() == context.Canceled {
			// Canceled
			result.ExitCode = -3
			result.CrashType = types.CrashTypeInterrupted
			e.logger.Printf("Command was canceled")
		} else {
			// Other errors (failed to start, etc.)
			result.ExitCode = -1
			result.CrashType = types.CrashTypeUnknown
			e.logger.Printf("Command failed with unknown error: %v", err)
		}
	}

	e.logger.Printf("Command completed in %v with exit code %d (crash type: %s)",
		duration, result.ExitCode, result.CrashType)

	return result, nil
}

func (e *Executor) FindXcodeCommand() (string, error) {
	// Try to find xcodebuild in common locations
	paths := []string{
		"/usr/bin/xcodebuild",
		"/Applications/Xcode.app/Contents/Developer/usr/bin/xcodebuild",
	}

	// Also check PATH
	if path, err := exec.LookPath("xcodebuild"); err == nil {
		paths = append([]string{path}, paths...)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			e.logger.Printf("Found xcodebuild at: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("xcodebuild not found in any expected location")
}

func (e *Executor) BuildXcodeArgs(params interface{}) ([]string, error) {
	xcodeCmd, err := e.FindXcodeCommand()
	if err != nil {
		return nil, err
	}

	args := []string{xcodeCmd}

	switch p := params.(type) {
	case *types.BuildParams:
		return e.buildBuildArgs(args, p)
	case *types.TestParams:
		return e.buildTestArgs(args, p)
	case *types.CleanParams:
		return e.buildCleanArgs(args, p)
	default:
		return nil, fmt.Errorf("unsupported parameter type: %T", params)
	}
}

func (e *Executor) buildBuildArgs(baseArgs []string, params *types.BuildParams) ([]string, error) {
	args := append(baseArgs, "build")

	// Workspace or project
	if params.Workspace != "" {
		if !filepath.IsAbs(params.Workspace) && params.ProjectPath != "" {
			params.Workspace = filepath.Join(params.ProjectPath, params.Workspace)
		}
		args = append(args, "-workspace", params.Workspace)
	} else if params.Project != "" {
		if !filepath.IsAbs(params.Project) && params.ProjectPath != "" {
			params.Project = filepath.Join(params.ProjectPath, params.Project)
		}
		args = append(args, "-project", params.Project)
	}

	// Scheme or target
	if params.Scheme != "" {
		args = append(args, "-scheme", params.Scheme)
	} else if params.Target != "" {
		args = append(args, "-target", params.Target)
	}

	// Configuration
	if params.Configuration != "" {
		args = append(args, "-configuration", params.Configuration)
	}

	// SDK
	if params.SDK != "" {
		args = append(args, "-sdk", params.SDK)
	}

	// Destination
	if params.Destination != "" {
		args = append(args, "-destination", params.Destination)
	}

	// Architecture
	if params.Arch != "" {
		args = append(args, "-arch", params.Arch)
	}

	// Derived data
	if params.DerivedData != "" {
		args = append(args, "-derivedDataPath", params.DerivedData)
	}

	// Archive flag
	if params.Archive {
		args = append(args, "archive")
	}

	// Clean flag
	if params.Clean {
		args = append(args, "clean")
	}

	// Extra arguments
	args = append(args, params.ExtraArgs...)

	return args, nil
}

func (e *Executor) buildTestArgs(baseArgs []string, params *types.TestParams) ([]string, error) {
	args := append(baseArgs, "test")

	// Workspace or project
	if params.Workspace != "" {
		if !filepath.IsAbs(params.Workspace) && params.ProjectPath != "" {
			params.Workspace = filepath.Join(params.ProjectPath, params.Workspace)
		}
		args = append(args, "-workspace", params.Workspace)
	} else if params.Project != "" {
		if !filepath.IsAbs(params.Project) && params.ProjectPath != "" {
			params.Project = filepath.Join(params.ProjectPath, params.Project)
		}
		args = append(args, "-project", params.Project)
	}

	// Scheme or target
	if params.Scheme != "" {
		args = append(args, "-scheme", params.Scheme)
	} else if params.Target != "" {
		args = append(args, "-target", params.Target)
	}

	// Test plan
	if params.TestPlan != "" {
		args = append(args, "-testPlan", params.TestPlan)
	}

	// SDK
	if params.SDK != "" {
		args = append(args, "-sdk", params.SDK)
	}

	// Destination
	if params.Destination != "" {
		args = append(args, "-destination", params.Destination)
	}

	// Only testing
	for _, test := range params.OnlyTesting {
		args = append(args, "-only-testing", test)
	}

	// Skip testing
	for _, test := range params.SkipTesting {
		args = append(args, "-skip-testing", test)
	}

	// Parallel testing
	if params.Parallel {
		args = append(args, "-parallel-testing-enabled", "YES")
	} else {
		args = append(args, "-parallel-testing-enabled", "NO")
	}

	// Code coverage
	if params.Coverage {
		args = append(args, "-enableCodeCoverage", "YES")
	}

	// Result bundle
	if params.ResultBundle != "" {
		args = append(args, "-resultBundlePath", params.ResultBundle)
	}

	// Derived data
	if params.DerivedData != "" {
		args = append(args, "-derivedDataPath", params.DerivedData)
	}

	// Extra arguments
	args = append(args, params.ExtraArgs...)

	return args, nil
}

func (e *Executor) buildCleanArgs(baseArgs []string, params *types.CleanParams) ([]string, error) {
	args := append(baseArgs, "clean")

	// Workspace or project
	if params.Workspace != "" {
		if !filepath.IsAbs(params.Workspace) && params.ProjectPath != "" {
			params.Workspace = filepath.Join(params.ProjectPath, params.Workspace)
		}
		args = append(args, "-workspace", params.Workspace)
	} else if params.Project != "" {
		if !filepath.IsAbs(params.Project) && params.ProjectPath != "" {
			params.Project = filepath.Join(params.ProjectPath, params.Project)
		}
		args = append(args, "-project", params.Project)
	}

	// Target
	if params.Target != "" {
		args = append(args, "-target", params.Target)
	}

	// Derived data
	if params.DerivedData != "" {
		args = append(args, "-derivedDataPath", params.DerivedData)
	}

	return args, nil
}

type CommandResult struct {
	Command      string
	Output       string
	StdoutOutput string
	StderrOutput string
	Duration     time.Duration
	ExitCode     int
	Error        error
	ProcessState *types.ProcessState
	CrashType    types.CrashType
}

func (r *CommandResult) Success() bool {
	return r.ExitCode == 0 && r.Error == nil
}

func (r *CommandResult) HasOutput() bool {
	return strings.TrimSpace(r.Output) != ""
}

func (r *CommandResult) IsCrash() bool {
	return r.ProcessState != nil && r.ProcessState.Signaled
}

// classifySignal classifies a signal into a CrashType
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

// classifyExitCode classifies an exit code into a CrashType
func classifyExitCode(exitCode int) types.CrashType {
	switch exitCode {
	case 0:
		return types.CrashTypeNone
	case 65:
		// Exit code 65 from xcodebuild specifically indicates test failures occurred.
		// This is NOT a build failure - tests compiled and ran, but some assertions failed.
		return types.CrashTypeTestFailure
	case 66:
		// Build failed (compilation errors, linking errors, etc.)
		return types.CrashTypeBuildFailure
	case 70:
		// Target not found, version mismatch, or device issues
		return types.CrashTypeBuildFailure
	default:
		if exitCode > 128 && exitCode < 160 {
			// Likely signal-based exit (128 + signal number)
			return classifySignal(syscall.Signal(exitCode - 128))
		}
		return types.CrashTypeUnknown
	}
}
