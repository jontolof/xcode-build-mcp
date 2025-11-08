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

type XcodeTestTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewXcodeTestTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *XcodeTestTool {
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
			"description": "Test scheme to use",
		},
		"destination": map[string]interface{}{
			"type":        "string",
			"description": "Test destination (platform=iOS Simulator,name=iPhone 15, etc.)",
		},
		"output_mode": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"minimal", "standard", "verbose"},
			"description": "Output filtering level",
			"default":     "standard",
		},
	}, []string{})

	return &XcodeTestTool{
		name:        "xcode_test",
		description: "Universal Xcode test command that runs tests with detailed results and intelligent output filtering",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *XcodeTestTool) Name() string {
	return t.name
}

func (t *XcodeTestTool) Description() string {
	return t.description
}

func (t *XcodeTestTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *XcodeTestTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params := &types.TestParams{
		OutputMode:  "standard",
		Environment: make(map[string]string),
		ExtraArgs:   []string{},
	}

	// Parse basic parameters
	projectPath, _ := parseStringParam(args, "project_path", false)
	workspace, _ := parseStringParam(args, "workspace", false)
	project, _ := parseStringParam(args, "project", false)
	scheme, _ := parseStringParam(args, "scheme", false)
	destination, _ := parseStringParam(args, "destination", false)
	outputMode, _ := parseStringParam(args, "output_mode", false)

	params.ProjectPath = projectPath
	params.Workspace = workspace
	params.Project = project
	params.Scheme = scheme
	params.Destination = destination
	if outputMode != "" {
		params.OutputMode = outputMode
	}

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
		return "", fmt.Errorf("failed to execute test command: %w", err)
	}

	testResult := t.parser.ParseTestOutput(result.Output)
	testResult.Duration = result.Duration
	testResult.ExitCode = result.ExitCode
	testResult.Success = result.Success()

	// Apply filtering
	outputFilter := filter.NewFilter(filter.OutputMode(params.OutputMode))
	filteredOutput := outputFilter.Filter(result.Output)
	testResult.FilteredOutput = filteredOutput

	// Convert test bundles to map format for JSON response
	testBundles := make([]map[string]interface{}, 0, len(testResult.TestSummary.TestBundles))
	for _, bundle := range testResult.TestSummary.TestBundles {
		testBundles = append(testBundles, map[string]interface{}{
			"name":       bundle.Name,
			"type":       bundle.Type,
			"executed":   bundle.Executed,
			"status":     bundle.Status,
			"test_count": bundle.TestCount,
			"duration":   bundle.Duration.String(),
		})
	}

	response := map[string]interface{}{
		"success":         testResult.Success,
		"duration":        testResult.Duration.String(),
		"exit_code":       testResult.ExitCode,
		"filtered_output": testResult.FilteredOutput,
		"test_summary": map[string]interface{}{
			"total_tests":  testResult.TestSummary.TotalTests,
			"passed_tests": testResult.TestSummary.PassedTests,
			"failed_tests": testResult.TestSummary.FailedTests,
		},
		"test_bundles": testBundles,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
