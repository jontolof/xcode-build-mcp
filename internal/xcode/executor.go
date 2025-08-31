package xcode

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Read stdout
	go func() {
		defer func() { outputChan <- "" }()
		scanner := bufio.NewScanner(stdout)
		var output strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line)
			output.WriteString("\n")
		}
		outputChan <- output.String()
	}()

	// Read stderr
	go func() {
		defer func() { outputChan <- "" }()
		scanner := bufio.NewScanner(stderr)
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
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err
	}

	e.logger.Printf("Command completed in %v with exit code %d", duration, result.ExitCode)

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
}

func (r *CommandResult) Success() bool {
	return r.ExitCode == 0 && r.Error == nil
}

func (r *CommandResult) HasOutput() bool {
	return strings.TrimSpace(r.Output) != ""
}
