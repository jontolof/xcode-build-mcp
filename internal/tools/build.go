package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/internal/filter"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type XcodeBuildTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewXcodeBuildTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *XcodeBuildTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"project_path": map[string]interface{}{
			"type":        "string",
			"description": "Path to the directory containing the Xcode project or workspace",
		},
		"workspace": map[string]interface{}{
			"type":        "string",
			"description": "Name of the .xcworkspace file (relative to project_path)",
		},
		"project": map[string]interface{}{
			"type":        "string",
			"description": "Name of the .xcodeproj file (relative to project_path)",
		},
		"scheme": map[string]interface{}{
			"type":        "string",
			"description": "Build scheme to use",
		},
		"target": map[string]interface{}{
			"type":        "string",
			"description": "Build target to use (alternative to scheme)",
		},
		"configuration": map[string]interface{}{
			"type":        "string",
			"description": "Build configuration (Debug, Release, etc.)",
		},
		"sdk": map[string]interface{}{
			"type":        "string",
			"description": "SDK to build against (iphoneos, iphonesimulator, macosx, etc.)",
		},
		"destination": map[string]interface{}{
			"type":        "string",
			"description": "Build destination (platform=iOS Simulator,name=iPhone 15, etc.)",
		},
		"arch": map[string]interface{}{
			"type":        "string",
			"description": "Target architecture (arm64, x86_64, etc.)",
		},
		"output_mode": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"minimal", "standard", "verbose"},
			"description": "Output filtering level",
			"default":     "standard",
		},
		"clean": map[string]interface{}{
			"type":        "boolean",
			"description": "Perform clean build",
			"default":     false,
		},
		"archive": map[string]interface{}{
			"type":        "boolean",
			"description": "Create archive instead of regular build",
			"default":     false,
		},
		"derived_data": map[string]interface{}{
			"type":        "string",
			"description": "Path for derived data",
		},
		"environment": map[string]interface{}{
			"type":        "object",
			"description": "Environment variables for the build",
		},
		"extra_args": map[string]interface{}{
			"type":        "array",
			"items":       map[string]string{"type": "string"},
			"description": "Additional xcodebuild arguments",
		},
	}, []string{})

	return &XcodeBuildTool{
		name:        "xcode_build",
		description: "Universal Xcode build command that handles projects, workspaces, schemes, and targets with intelligent output filtering",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *XcodeBuildTool) Name() string {
	return t.name
}

func (t *XcodeBuildTool) Description() string {
	return t.description
}

func (t *XcodeBuildTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *XcodeBuildTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	t.logger.Printf("Starting Xcode build with params: %+v", params)

	// Build xcodebuild command arguments
	cmdArgs, err := t.executor.BuildXcodeArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command arguments: %w", err)
	}

	// Add environment variables if specified
	if len(params.Environment) > 0 {
		// TODO: Set environment variables for the command
		t.logger.Printf("Setting %d environment variables", len(params.Environment))
	}

	start := time.Now()

	// Execute the build command
	result, err := t.executor.ExecuteCommand(ctx, cmdArgs)
	if err != nil {
		return "", fmt.Errorf("failed to execute build command: %w", err)
	}

	duration := time.Since(start)

	// Parse the build output
	buildResult := t.parser.ParseBuildOutput(result.Output)
	buildResult.Duration = duration
	buildResult.ExitCode = result.ExitCode
	buildResult.Success = result.Success()

	// Integrate crash detection from executor
	buildResult.CrashType = result.CrashType
	buildResult.ProcessCrashed = result.ProcessState != nil && result.ProcessState.Signaled
	buildResult.ProcessState = result.ProcessState

	// Detect crash patterns in output
	buildResult.CrashIndicators = t.parser.DetectCrashIndicators(result.Output)

	// Check for silent failures
	buildResult.SilentFailure = t.parser.DetectSilentFailure(result.Output, result.ExitCode)

	// Apply output filtering
	outputMode := filter.OutputMode(params.OutputMode)
	if outputMode == "" {
		outputMode = filter.Standard
	}

	outputFilter := filter.NewFilter(outputMode)
	filteredOutput := outputFilter.Filter(result.Output)
	buildResult.FilteredOutput = filteredOutput

	// Extract build settings if present
	buildResult.BuildSettings = t.parser.ExtractBuildSettings(result.Output)

	// Format the response
	response, err := t.formatBuildResponse(buildResult, outputFilter)
	if err != nil {
		return "", fmt.Errorf("failed to format response: %w", err)
	}

	return response, nil
}

func (t *XcodeBuildTool) parseParams(args map[string]interface{}) (*types.BuildParams, error) {
	params := &types.BuildParams{
		OutputMode:  "standard",
		Environment: make(map[string]string),
		ExtraArgs:   []string{},
	}

	// Parse required and optional parameters
	if projectPath, err := parseStringParam(args, "project_path", false); err != nil {
		return nil, err
	} else if projectPath != "" {
		params.ProjectPath = projectPath
	}

	if workspace, err := parseStringParam(args, "workspace", false); err != nil {
		return nil, err
	} else if workspace != "" {
		params.Workspace = workspace
	}

	if project, err := parseStringParam(args, "project", false); err != nil {
		return nil, err
	} else if project != "" {
		params.Project = project
	}

	if scheme, err := parseStringParam(args, "scheme", false); err != nil {
		return nil, err
	} else if scheme != "" {
		params.Scheme = scheme
	}

	if target, err := parseStringParam(args, "target", false); err != nil {
		return nil, err
	} else if target != "" {
		params.Target = target
	}

	if configuration, err := parseStringParam(args, "configuration", false); err != nil {
		return nil, err
	} else if configuration != "" {
		params.Configuration = configuration
	}

	if sdk, err := parseStringParam(args, "sdk", false); err != nil {
		return nil, err
	} else if sdk != "" {
		params.SDK = sdk
	}

	if destination, err := parseStringParam(args, "destination", false); err != nil {
		return nil, err
	} else if destination != "" {
		params.Destination = destination
	}

	if arch, err := parseStringParam(args, "arch", false); err != nil {
		return nil, err
	} else if arch != "" {
		params.Arch = arch
	}

	if outputMode, err := parseStringParam(args, "output_mode", false); err != nil {
		return nil, err
	} else if outputMode != "" {
		params.OutputMode = outputMode
	}

	if derivedData, err := parseStringParam(args, "derived_data", false); err != nil {
		return nil, err
	} else if derivedData != "" {
		params.DerivedData = derivedData
	}

	params.Clean = parseBoolParam(args, "clean", false)
	params.Archive = parseBoolParam(args, "archive", false)

	// Parse environment variables
	if env, exists := args["environment"]; exists {
		if envMap, ok := env.(map[string]interface{}); ok {
			for k, v := range envMap {
				if strVal, ok := v.(string); ok {
					params.Environment[k] = strVal
				}
			}
		}
	}

	// Parse extra arguments
	if extraArgs, err := parseArrayParam(args, "extra_args"); err != nil {
		return nil, err
	} else if extraArgs != nil {
		for _, arg := range extraArgs {
			if strArg, ok := arg.(string); ok {
				params.ExtraArgs = append(params.ExtraArgs, strArg)
			}
		}
	}

	// Validate parameters
	if params.Workspace == "" && params.Project == "" {
		return nil, fmt.Errorf("either workspace or project must be specified")
	}

	if params.Scheme == "" && params.Target == "" {
		return nil, fmt.Errorf("either scheme or target must be specified")
	}

	return params, nil
}

func (t *XcodeBuildTool) formatBuildResponse(result *types.BuildResult, outputFilter *filter.Filter) (string, error) {
	response := map[string]interface{}{
		"success":         result.Success,
		"duration":        result.Duration.String(),
		"exit_code":       result.ExitCode,
		"filtered_output": result.FilteredOutput,
		// Crash detection fields
		"crash_type":       result.CrashType,
		"process_crashed":  result.ProcessCrashed,
		"silent_failure":   result.SilentFailure,
		"crash_indicators": result.CrashIndicators,
		"process_state":    result.ProcessState,
	}

	// ALWAYS include a summary message
	if result.FilteredOutput == "" || len(result.FilteredOutput) < 100 {
		if result.Success {
			response["summary"] = "Build completed successfully (output was filtered)"
		} else {
			response["summary"] = fmt.Sprintf("Build FAILED with exit code %d (check errors in output)", result.ExitCode)
		}
	}
	
	// If verbose mode and output is suspiciously small, add a note
	stats := outputFilter.GetStats()
	if len(result.FilteredOutput) < 1000 && stats.TotalLines > 50 {
		response["note"] = "Output appears truncated. Enable MCP_FILTER_DEBUG=true to investigate"
	}

	// Add debug hint when builds seem to fail silently
	if !result.Success && len(result.FilteredOutput) < 500 {
		response["debug_hint"] = "Build failed with minimal output. Try: 1) Check if project exists, 2) Verify scheme name, 3) Set MCP_DEBUG=true for details"
	}

	// Add filtering statistics
	response["filtering_stats"] = map[string]interface{}{
		"total_lines":       stats.TotalLines,
		"filtered_lines":    stats.FilteredLines,
		"kept_lines":        stats.KeptLines,
		"reduction_percent": outputFilter.ReductionPercentage(),
	}

	// Add errors if any
	if len(result.Errors) > 0 {
		response["errors"] = result.Errors
		response["error_count"] = len(result.Errors)
	}

	// Add warnings if any
	if len(result.Warnings) > 0 {
		response["warnings"] = result.Warnings
		response["warning_count"] = len(result.Warnings)
	}

	// Add artifact paths if any
	if len(result.ArtifactPaths) > 0 {
		response["artifact_paths"] = result.ArtifactPaths
	}

	// Add build settings if present
	if len(result.BuildSettings) > 0 {
		response["build_settings"] = result.BuildSettings
	}

	// NEVER include full output - it defeats the entire purpose of filtering!
	// The filtered output already contains all critical information including errors

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
