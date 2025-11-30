package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
		description: "Universal Xcode test command that runs tests with detailed results and intelligent output filtering. Returns comprehensive crash detection including: crash_type (segmentation_fault, abort, killed, timeout, fatal_error, test_crash, etc.), process_crashed (bool), crash_indicators (test_runner_crashed, fatal_error_detected, swift_runtime_crash, connection_interrupted, simulator_boot_timeout, etc.), simulator_crashes (array of crash reports), and silent_failure detection. Always check crash_type field - if not 'none', the test execution crashed rather than failed normally.",
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

// fixMisleadingSummary replaces misleading "passed" summaries with accurate failure counts
// This handles silent test failures that only appear in xcresult bundles
func fixMisleadingSummary(output string, totalTests, passedTests, failedTests, skippedTests int) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))
	foundSummary := false

	for scanner.Scan() {
		line := scanner.Text()

		// Look for the final "All tests" summary line - this is what we need to fix
		if strings.Contains(line, "Test Suite 'All tests'") && strings.Contains(line, "passed at") {
			// Replace with accurate summary
			timestamp := ""
			if idx := strings.Index(line, "at "); idx != -1 {
				timestamp = line[idx+3:]
			}
			if timestamp != "" {
				result.WriteString(fmt.Sprintf("Test Suite 'All tests' finished at %s\n", timestamp))
			} else {
				result.WriteString("Test Suite 'All tests' finished.\n")
			}
			foundSummary = true
			continue
		}

		// Look for "Executed X tests, with 0 failures" right after "All tests" line
		// This is the misleading summary we need to replace
		if foundSummary && strings.Contains(line, "Executed") && strings.Contains(line, "with 0 failures") {
			// Replace with accurate counts, including skipped tests if any
			if skippedTests > 0 {
				result.WriteString(fmt.Sprintf("\t Executed %d tests, with %d failures (%d passed, %d skipped)\n",
					totalTests, failedTests, passedTests, skippedTests))
			} else {
				result.WriteString(fmt.Sprintf("\t Executed %d tests, with %d failures (%d passed)\n",
					totalTests, failedTests, passedTests))
			}
			foundSummary = false // Reset flag
			continue
		}

		// Keep all other lines as-is
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
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

	// Generate a temporary result bundle path for accurate test result parsing
	// This provides structured JSON results instead of text parsing
	resultBundlePath := xcode.GenerateResultBundlePath()
	defer xcode.CleanupResultBundle(resultBundlePath)

	// Add result bundle path to params if not already specified
	if params.ResultBundle == "" {
		params.ResultBundle = resultBundlePath
	}

	cmdArgs, err := t.executor.BuildXcodeArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to build command arguments: %w", err)
	}

	// Initialize crash detector before execution
	crashDetector := xcode.NewSimulatorCrashDetector()

	result, err := t.executor.ExecuteCommand(ctx, cmdArgs)
	if err != nil {
		return "", fmt.Errorf("failed to execute test command: %w", err)
	}

	// Debug logging: save raw output for troubleshooting parsing issues
	debugEnabled := os.Getenv("MCP_DEBUG_TEST_OUTPUT") == "true"
	if debugEnabled {
		debugDir := os.Getenv("MCP_DEBUG_OUTPUT_DIR")
		if debugDir == "" {
			debugDir = os.TempDir()
		}
		timestamp := time.Now().Unix()
		debugPath := fmt.Sprintf("%s/xcode_test_output_%d.txt", debugDir, timestamp)
		if err := os.WriteFile(debugPath, []byte(result.Output), 0644); err == nil {
			t.logger.Printf("Debug: raw test output saved to %s (%d bytes)", debugPath, len(result.Output))
		}
	}

	testResult := t.parser.ParseTestOutput(result.Output)
	testResult.Duration = result.Duration
	testResult.ExitCode = result.ExitCode
	testResult.Success = result.Success()

	// Debug: Log initial parsing results
	if debugEnabled {
		t.logger.Printf("Text parsing results: %d total, %d passed, %d failed",
			testResult.TestSummary.TotalTests,
			testResult.TestSummary.PassedTests,
			testResult.TestSummary.FailedTests)
		if len(testResult.TestSummary.FailedTestsDetails) > 0 {
			t.logger.Printf("Failed tests from text parsing:")
			for _, failure := range testResult.TestSummary.FailedTestsDetails {
				t.logger.Printf("  - %s.%s: %s", failure.ClassName, failure.Name, failure.Message)
			}
		}
	}

	// CRITICAL: Validate test results against exit code
	// This catches cases where exit code 65 indicates failures but parsing found none
	t.parser.ValidateTestResults(testResult, result.ExitCode)

	// Parse xcresult bundle for accurate results (if available)
	// This is the most reliable source of test results
	xcresultParser := xcode.NewXCResultParser()
	xcresultSummary, xcresultErr := xcresultParser.ParseResultBundle(resultBundlePath)

	// Debug: Log xcresult parsing attempt
	if debugEnabled {
		if xcresultErr != nil {
			t.logger.Printf("XCResult parsing failed: %v", xcresultErr)
		} else {
			t.logger.Printf("XCResult parsing succeeded: %d total, %d passed, %d failed, %d skipped",
				xcresultSummary.TotalTests,
				xcresultSummary.PassedTests,
				xcresultSummary.FailedTestCount,
				xcresultSummary.SkippedTests)
			if len(xcresultSummary.FailedTestDetails) > 0 {
				t.logger.Printf("Failed tests from xcresult:")
				for _, failure := range xcresultSummary.FailedTestDetails {
					t.logger.Printf("  - %s.%s: %s", failure.ClassName, failure.Name, failure.Message)
				}
			}
		}
	}

	if xcresultErr == nil {
		// Use xcresult data to validate and correct text-parsed results
		if xcresultSummary.TotalTests > 0 {
			// Trust xcresult counts over text parsing
			if testResult.TestSummary.TotalTests != xcresultSummary.TotalTests ||
				testResult.TestSummary.FailedTests != xcresultSummary.FailedTestCount {
				// Log discrepancy for debugging
				t.logger.Printf("Test result discrepancy detected: text parsed %d/%d (pass/total), xcresult shows %d/%d (pass/total), %d failed",
					testResult.TestSummary.PassedTests, testResult.TestSummary.TotalTests,
					xcresultSummary.PassedTests, xcresultSummary.TotalTests,
					xcresultSummary.FailedTestCount)

				// Correct the counts using xcresult (authoritative source)
				testResult.TestSummary.TotalTests = xcresultSummary.TotalTests
				testResult.TestSummary.PassedTests = xcresultSummary.PassedTests
				testResult.TestSummary.FailedTests = xcresultSummary.FailedTestCount
				testResult.TestSummary.SkippedTests = xcresultSummary.SkippedTests

				// Update test bundles if xcresult has them
				if len(xcresultSummary.TestBundles) > 0 {
					testResult.TestSummary.TestBundles = xcresultSummary.TestBundles
				}

				// Add failed test details
				if len(xcresultSummary.FailedTestDetails) > 0 {
					testResult.TestSummary.FailedTestsDetails = xcresultSummary.FailedTestDetails
				}

				// Clear parsing warning since we now have accurate data
				if testResult.TestSummary.UnparsedFailures && xcresultSummary.FailedTestCount > 0 {
					testResult.TestSummary.ParsingWarning = ""
					testResult.TestSummary.UnparsedFailures = false
				}
			}

			// Update success based on actual failure count
			// BUT: don't override if unparsed failures were detected by ValidateTestResults
			if !testResult.TestSummary.UnparsedFailures {
				testResult.Success = xcresultSummary.FailedTestCount == 0
			}
			// If xcresult shows 0 failures but we have unparsed failures warning,
			// keep Success = false (set by ValidateTestResults)
		}
	} else {
		// xcresult parsing failed - log but continue with text-parsed results
		t.logger.Printf("Warning: failed to parse xcresult bundle: %v", xcresultErr)

		// Add warning to test summary if xcresult parsing failed
		if testResult.TestSummary.ParsingWarning == "" {
			testResult.TestSummary.ParsingWarning = fmt.Sprintf("XCResult parsing failed: %v. Using text-based parsing only.", xcresultErr)
		} else {
			testResult.TestSummary.ParsingWarning += fmt.Sprintf(" Additionally, XCResult parsing failed: %v", xcresultErr)
		}
	}

	// Integrate crash detection from executor
	testResult.CrashType = result.CrashType
	testResult.ProcessCrashed = result.ProcessState != nil && result.ProcessState.Signaled
	testResult.ProcessState = result.ProcessState

	// Detect crash patterns in output
	testResult.CrashIndicators = t.parser.DetectCrashIndicators(result.Output)

	// Check for silent failures
	testResult.SilentFailure = t.parser.DetectSilentFailure(result.Output, result.ExitCode)

	// Context-aware crash type upgrade based on indicators
	if testResult.CrashIndicators.FatalErrorDetected {
		// Swift fatal error takes precedence - this is a definite crash
		testResult.CrashType = types.CrashTypeFatalError
		testResult.ProcessCrashed = true
	} else if testResult.CrashIndicators.SwiftRuntimeCrash {
		// Other Swift runtime crashes (precondition, assertion, etc.)
		testResult.CrashType = types.CrashTypeTestCrash
		testResult.ProcessCrashed = true
	} else if testResult.CrashIndicators.TestProcessCrashed || testResult.CrashIndicators.TestRunnerCrashed {
		// If exit code 65 with test crash indicator, it's a test crash not build failure
		if testResult.CrashType == types.CrashTypeBuildFailure {
			testResult.CrashType = types.CrashTypeTestCrash
		}
		testResult.ProcessCrashed = true
	}

	// Check for simulator crashes
	crashes, _ := crashDetector.CheckForCrashes("Simulator")
	if len(crashes) > 0 {
		testResult.SimulatorCrashes = crashes
		// Upgrade crash type if we found simulator crashes but no other crash detected
		if testResult.CrashType == types.CrashTypeNone || testResult.CrashType == types.CrashTypeUnknown {
			testResult.CrashType = types.CrashTypeSimulatorCrash
		}
	}

	// Apply filtering
	outputFilter := filter.NewFilter(filter.OutputMode(params.OutputMode))
	filteredOutput := outputFilter.Filter(result.Output)

	// IMPORTANT: Handle silent test failures - fix misleading output
	// Some test failures (especially ViewInspector tests) don't appear in xcodebuild text output
	// but are present in the .xcresult bundle. We must show these to the LLM accurately.
	if len(testResult.TestSummary.FailedTestsDetails) > 0 {
		// Check if failures are already visible in filtered output
		hasFailuresInOutput := false
		for _, failure := range testResult.TestSummary.FailedTestsDetails {
			if strings.Contains(filteredOutput, failure.Name) {
				hasFailuresInOutput = true
				break
			}
		}

		// If failures aren't in the filtered output, we have silent failures
		if !hasFailuresInOutput {
			// Replace misleading "passed" summary with accurate one
			filteredOutput = fixMisleadingSummary(filteredOutput, testResult.TestSummary.TotalTests,
				testResult.TestSummary.PassedTests, testResult.TestSummary.FailedTests,
				testResult.TestSummary.SkippedTests)

			// Append detailed failure list
			var failureSection strings.Builder
			failureSection.WriteString("\n=== Failed Tests ===\n")
			for _, failure := range testResult.TestSummary.FailedTestsDetails {
				failureSection.WriteString(fmt.Sprintf("❌ %s.%s\n", failure.ClassName, failure.Name))
				if failure.Message != "" {
					failureSection.WriteString(fmt.Sprintf("   %s\n", failure.Message))
				}
				if failure.Duration > 0 {
					failureSection.WriteString(fmt.Sprintf("   Duration: %v\n", failure.Duration))
				}
			}
			filteredOutput += failureSection.String()
		}
	}

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

	// Build test summary with optional warning fields
	testSummaryMap := map[string]interface{}{
		"total_tests":   testResult.TestSummary.TotalTests,
		"passed_tests":  testResult.TestSummary.PassedTests,
		"failed_tests":  testResult.TestSummary.FailedTests,
		"skipped_tests": testResult.TestSummary.SkippedTests,
	}
	// Include warning fields if set (indicates parsing issues)
	if testResult.TestSummary.ParsingWarning != "" {
		testSummaryMap["parsing_warning"] = testResult.TestSummary.ParsingWarning
	}
	if testResult.TestSummary.UnparsedFailures {
		testSummaryMap["unparsed_failures"] = true
	}

	response := map[string]interface{}{
		"success":         testResult.Success,
		"duration":        testResult.Duration.String(),
		"exit_code":       testResult.ExitCode,
		"filtered_output": testResult.FilteredOutput,
		"test_summary":    testSummaryMap,
		"test_bundles":    testBundles,
		// Crash detection fields
		"crash_type":        testResult.CrashType,
		"process_crashed":   testResult.ProcessCrashed,
		"silent_failure":    testResult.SilentFailure,
		"crash_indicators":  testResult.CrashIndicators,
		"process_state":     testResult.ProcessState,
		"simulator_crashes": testResult.SimulatorCrashes,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonData), nil
}
