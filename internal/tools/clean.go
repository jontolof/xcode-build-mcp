package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/internal/filter"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type XcodeCleanTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewXcodeCleanTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *XcodeCleanTool {
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
		"clean_build": map[string]interface{}{
			"type":        "boolean",
			"description": "Perform deep clean of build folder",
			"default":     false,
		},
		"output_mode": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"minimal", "standard", "verbose"},
			"description": "Output filtering level",
			"default":     "standard",
		},
	}, []string{})

	return &XcodeCleanTool{
		name:        "xcode_clean",
		description: "Clean Xcode build artifacts with support for derived data and deep cleaning",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *XcodeCleanTool) Name() string {
	return t.name
}

func (t *XcodeCleanTool) Description() string {
	return t.description
}

func (t *XcodeCleanTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *XcodeCleanTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params := &types.CleanParams{}

	// Parse basic parameters
	projectPath, _ := parseStringParam(args, "project_path", false)
	workspace, _ := parseStringParam(args, "workspace", false)
	project, _ := parseStringParam(args, "project", false)
	outputMode, _ := parseStringParam(args, "output_mode", false)

	params.ProjectPath = projectPath
	params.Workspace = workspace
	params.Project = project
	params.CleanBuild = parseBoolParam(args, "clean_build", false)

	// Validate
	if params.Workspace == "" && params.Project == "" {
		return "", fmt.Errorf("either workspace or project must be specified")
	}

	cmdArgs, err := t.executor.BuildXcodeArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command arguments: %w", err)
	}

	result, err := t.executor.ExecuteCommand(ctx, cmdArgs)
	if err != nil {
		return "", fmt.Errorf("failed to execute clean command: %w", err)
	}

	cleanResult := t.parser.ParseCleanOutput(result.Output)
	cleanResult.Duration = result.Duration
	cleanResult.ExitCode = result.ExitCode
	cleanResult.Success = result.Success()

	// Apply filtering
	if outputMode == "" {
		outputMode = "standard"
	}
	outputFilter := filter.NewFilter(filter.OutputMode(outputMode))
	filteredOutput := outputFilter.Filter(result.Output)
	cleanResult.FilteredOutput = filteredOutput

	response := map[string]interface{}{
		"success":         cleanResult.Success,
		"duration":        cleanResult.Duration.String(),
		"exit_code":       cleanResult.ExitCode,
		"filtered_output": cleanResult.FilteredOutput,
		"cleaned_count":   len(cleanResult.CleanedPaths),
	}

	if len(cleanResult.CleanedPaths) > 0 {
		response["cleaned_paths"] = cleanResult.CleanedPaths
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}