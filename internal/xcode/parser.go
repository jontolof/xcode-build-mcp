package xcode

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// Regular expressions for parsing xcodebuild output
var (
	// Build errors and warnings
	errorRegex  = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(error|warning):\s*(.+)$`)
	errorRegex2 = regexp.MustCompile(`^(.+?):\s*(error|warning):\s*(.+)$`)

	// Build success/failure
	buildSuccessRegex = regexp.MustCompile(`\*\* BUILD SUCCEEDED \*\*`)
	buildFailedRegex  = regexp.MustCompile(`\*\* BUILD FAILED \*\*`)

	// Test results
	testSuccessRegex = regexp.MustCompile(`\*\* TEST SUCCEEDED \*\*`)
	testFailedRegex  = regexp.MustCompile(`\*\* TEST FAILED \*\*`)
	testCaseRegex    = regexp.MustCompile(`Test Case '(.+?)' (passed|failed|started) \((\d+\.\d+) seconds\)`)
	testSuiteRegex   = regexp.MustCompile(`Test Suite '(.+?)' (passed|failed|started)`)
	testSuiteCountRegex = regexp.MustCompile(`Executed (\d+) tests?, with (\d+) failures? .* in ([\d.]+) seconds`)

	// Archive/export paths
	archiveRegex = regexp.MustCompile(`Archive path: (.+\.xcarchive)`)
	exportRegex  = regexp.MustCompile(`Export path: (.+)`)

	// Clean results
	cleanSuccessRegex = regexp.MustCompile(`\*\* CLEAN SUCCEEDED \*\*`)
	cleanFailedRegex  = regexp.MustCompile(`\*\* CLEAN FAILED \*\*`)

	// Crash detection patterns
	testRunnerCrashedRegex      = regexp.MustCompile(`Test runner.*crashed|Testing failed.*crashed`)
	connectionInterruptedRegex  = regexp.MustCompile(`Connection interrupted|Connection with the remote side was unexpectedly closed`)
	earlyExitRegex              = regexp.MustCompile(`Early unexpected exit|operation never finished bootstrapping`)
	neverBeganTestingRegex      = regexp.MustCompile(`Test runner never began executing tests`)
	failedToLoadBundleRegex     = regexp.MustCompile(`Failed to load the test bundle`)
	simulatorBootTimeoutRegex   = regexp.MustCompile(`Simulator.*timed out|Failed to boot simulator`)
	testProcessCrashedRegex     = regexp.MustCompile(`Test process crashed`)

	// Swift runtime crash patterns
	swiftFatalErrorRegex        = regexp.MustCompile(`Fatal error:`)
	swiftPreconditionRegex      = regexp.MustCompile(`Precondition failed:`)
	swiftAssertionRegex         = regexp.MustCompile(`Assertion failed:`)
	swiftForceUnwrapRegex       = regexp.MustCompile(`Unexpectedly found nil while (unwrapping|implicitly unwrapping)`)
	swiftIndexOutOfBoundsRegex  = regexp.MustCompile(`Index out of (range|bounds)`)
)

func (p *Parser) ParseBuildOutput(output string) *types.BuildResult {
	result := &types.BuildResult{
		Output:        output,
		Errors:        []types.BuildError{},
		Warnings:      []types.BuildWarning{},
		ArtifactPaths: []string{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check for build success/failure
		if buildSuccessRegex.MatchString(line) {
			result.Success = true
		} else if buildFailedRegex.MatchString(line) {
			result.Success = false
		}

		// Parse errors and warnings
		if matches := errorRegex.FindStringSubmatch(line); matches != nil {
			file := matches[1]
			lineNum, _ := strconv.Atoi(matches[2])
			column, _ := strconv.Atoi(matches[3])
			severity := matches[4]
			message := matches[5]

			if severity == "error" {
				result.Errors = append(result.Errors, types.BuildError{
					File:     file,
					Line:     lineNum,
					Column:   column,
					Message:  message,
					Severity: severity,
				})
			} else if severity == "warning" {
				result.Warnings = append(result.Warnings, types.BuildWarning{
					File:    file,
					Line:    lineNum,
					Column:  column,
					Message: message,
				})
			}
		} else if matches := errorRegex2.FindStringSubmatch(line); matches != nil {
			file := matches[1]
			severity := matches[2]
			message := matches[3]

			if severity == "error" {
				result.Errors = append(result.Errors, types.BuildError{
					File:     file,
					Message:  message,
					Severity: severity,
				})
			} else if severity == "warning" {
				result.Warnings = append(result.Warnings, types.BuildWarning{
					File:    file,
					Message: message,
				})
			}
		}

		// Parse artifact paths
		if matches := archiveRegex.FindStringSubmatch(line); matches != nil {
			result.ArtifactPaths = append(result.ArtifactPaths, matches[1])
		}
		if matches := exportRegex.FindStringSubmatch(line); matches != nil {
			result.ArtifactPaths = append(result.ArtifactPaths, matches[1])
		}
	}

	return result
}

func (p *Parser) ParseTestOutput(output string) *types.TestResult {
	result := &types.TestResult{
		Output: output,
		TestSummary: types.TestSummary{
			TestResults:        []types.TestCase{},
			FailedTestsDetails: []types.TestCase{},
			TestBundles:        []types.TestBundle{},
		},
	}

	var currentTest *types.TestCase
	testBundles := make(map[string]*types.TestBundle)
	var lastBundleName string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check for test success/failure
		if testSuccessRegex.MatchString(line) {
			result.Success = true
		} else if testFailedRegex.MatchString(line) {
			result.Success = false
		}

		// Parse test suites/bundles
		if matches := testSuiteRegex.FindStringSubmatch(line); matches != nil {
			suiteName := matches[1]
			status := matches[2]

			// Skip "All tests" suite as it's just a container
			if suiteName == "All tests" {
				lastBundleName = "" // Clear to prevent summary line from being assigned to wrong bundle
				continue
			}

			if status == "started" {
				// Create new test bundle
				bundleType := p.detectBundleType(suiteName)
				testBundles[suiteName] = &types.TestBundle{
					Name:     suiteName,
					Type:     bundleType,
					Executed: true,
					Status:   "started",
				}
				lastBundleName = suiteName
			} else if status == "passed" || status == "failed" {
				// Update existing bundle
				if bundle, exists := testBundles[suiteName]; exists {
					bundle.Status = status
					lastBundleName = suiteName
				}
			}
		}

		// Parse test suite summary (to get test count and duration)
		if matches := testSuiteCountRegex.FindStringSubmatch(line); matches != nil && lastBundleName != "" {
			testCount, _ := strconv.Atoi(matches[1])
			duration, _ := strconv.ParseFloat(matches[3], 64)

			if bundle, exists := testBundles[lastBundleName]; exists {
				bundle.TestCount = testCount
				bundle.Duration = time.Duration(duration * float64(time.Second))
			}
		}

		// Parse test cases
		if matches := testCaseRegex.FindStringSubmatch(line); matches != nil {
			testName := matches[1]
			status := matches[2]
			duration, _ := strconv.ParseFloat(matches[3], 64)

			// Extract class and method names
			parts := strings.Split(testName, ".")
			className := ""
			methodName := testName
			if len(parts) >= 2 {
				className = strings.Join(parts[:len(parts)-1], ".")
				methodName = parts[len(parts)-1]
			}

			testCase := types.TestCase{
				Name:      methodName,
				ClassName: className,
				Status:    status,
				Duration:  time.Duration(duration * float64(time.Second)),
			}

			if status == "started" {
				currentTest = &testCase
			} else {
				if currentTest != nil && currentTest.Name == methodName {
					currentTest.Status = status
					currentTest.Duration = testCase.Duration
					testCase = *currentTest
					currentTest = nil
				}

				result.TestSummary.TestResults = append(result.TestSummary.TestResults, testCase)
				result.TestSummary.TotalTests++

				switch status {
				case "passed":
					result.TestSummary.PassedTests++
				case "failed":
					result.TestSummary.FailedTests++
					result.TestSummary.FailedTestsDetails = append(result.TestSummary.FailedTestsDetails, testCase)
				}
			}
		}
	}

	// Convert map to slice
	for _, bundle := range testBundles {
		result.TestSummary.TestBundles = append(result.TestSummary.TestBundles, *bundle)
	}

	return result
}

func (p *Parser) detectBundleType(suiteName string) string {
	nameLower := strings.ToLower(suiteName)

	if strings.Contains(nameLower, "uitest") || strings.Contains(nameLower, "ui-test") {
		return "ui"
	}
	if strings.Contains(nameLower, "performance") || strings.Contains(nameLower, "perf") {
		return "performance"
	}
	if strings.Contains(nameLower, "integration") {
		return "integration"
	}

	return "unit"
}

func (p *Parser) ParseCleanOutput(output string) *types.CleanResult {
	result := &types.CleanResult{
		Output:       output,
		CleanedPaths: []string{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check for clean success/failure
		if cleanSuccessRegex.MatchString(line) {
			result.Success = true
		} else if cleanFailedRegex.MatchString(line) {
			result.Success = false
		}

		// Parse cleaned paths (simple heuristic - lines starting with "Removed" or "Cleaning")
		if strings.HasPrefix(line, "Removed ") || strings.HasPrefix(line, "Cleaning ") {
			// Extract path from the line
			parts := strings.Fields(line)
			if len(parts) > 1 {
				path := parts[len(parts)-1]
				// Clean up quotes and other formatting
				path = strings.Trim(path, `"'`)
				result.CleanedPaths = append(result.CleanedPaths, path)
			}
		}
	}

	return result
}

func (p *Parser) ExtractBuildSettings(output string) map[string]interface{} {
	settings := make(map[string]interface{})

	// Look for build settings in the output
	scanner := bufio.NewScanner(strings.NewReader(output))
	inBuildSettings := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "Build settings from command line:") {
			inBuildSettings = true
			continue
		}

		if inBuildSettings {
			// Stop parsing when we hit a new section
			if strings.HasPrefix(line, "===") || strings.HasPrefix(line, "***") {
				break
			}

			// Parse "KEY = value" format
			if strings.Contains(line, " = ") {
				parts := strings.SplitN(line, " = ", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					settings[key] = value
				}
			}
		}
	}

	return settings
}

func (p *Parser) IsSuccess(output string, commandType string) bool {
	switch commandType {
	case "build":
		return buildSuccessRegex.MatchString(output)
	case "test":
		return testSuccessRegex.MatchString(output)
	case "clean":
		return cleanSuccessRegex.MatchString(output)
	}
	return false
}

func (p *Parser) ExtractErrors(output string) []types.BuildError {
	var errors []types.BuildError

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := errorRegex.FindStringSubmatch(line); matches != nil {
			lineNum, _ := strconv.Atoi(matches[2])
			column, _ := strconv.Atoi(matches[3])

			if matches[4] == "error" {
				errors = append(errors, types.BuildError{
					File:     matches[1],
					Line:     lineNum,
					Column:   column,
					Message:  matches[5],
					Severity: matches[4],
				})
			}
		} else if matches := errorRegex2.FindStringSubmatch(line); matches != nil {
			if matches[2] == "error" {
				errors = append(errors, types.BuildError{
					File:     matches[1],
					Message:  matches[3],
					Severity: matches[2],
				})
			}
		}
	}

	return errors
}

func (p *Parser) ExtractWarnings(output string) []types.BuildWarning {
	var warnings []types.BuildWarning

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := errorRegex.FindStringSubmatch(line); matches != nil {
			if matches[4] == "warning" {
				lineNum, _ := strconv.Atoi(matches[2])
				column, _ := strconv.Atoi(matches[3])

				warnings = append(warnings, types.BuildWarning{
					File:    matches[1],
					Line:    lineNum,
					Column:  column,
					Message: matches[5],
				})
			}
		} else if matches := errorRegex2.FindStringSubmatch(line); matches != nil {
			if matches[2] == "warning" {
				warnings = append(warnings, types.BuildWarning{
					File:    matches[1],
					Message: matches[3],
				})
			}
		}
	}

	return warnings
}

func (p *Parser) ParseSchemes(output string) []string {
	var schemes []string

	scanner := bufio.NewScanner(strings.NewReader(output))
	inSchemesSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for the schemes section
		if strings.Contains(line, "Schemes:") {
			inSchemesSection = true
			continue
		}

		// Stop when we hit another section
		if inSchemesSection && (strings.Contains(line, "Targets:") || strings.Contains(line, "Build Configurations:") || line == "") {
			if strings.Contains(line, "Targets:") || strings.Contains(line, "Build Configurations:") {
				break
			}
			continue
		}

		// Extract scheme names (they're typically indented)
		if inSchemesSection && strings.HasPrefix(line, "    ") {
			scheme := strings.TrimSpace(line)
			if scheme != "" {
				schemes = append(schemes, scheme)
			}
		}
	}

	return schemes
}

func (p *Parser) ParseTargets(output string) []string {
	var targets []string

	scanner := bufio.NewScanner(strings.NewReader(output))
	inTargetsSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for the targets section
		if strings.Contains(line, "Targets:") {
			inTargetsSection = true
			continue
		}

		// Stop when we hit another section
		if inTargetsSection && (strings.Contains(line, "Build Configurations:") || strings.Contains(line, "Schemes:") || line == "") {
			if strings.Contains(line, "Build Configurations:") || strings.Contains(line, "Schemes:") {
				break
			}
			continue
		}

		// Extract target names (they're typically indented)
		if inTargetsSection && strings.HasPrefix(line, "    ") {
			target := strings.TrimSpace(line)
			if target != "" {
				targets = append(targets, target)
			}
		}
	}

	return targets
}

// DetectCrashIndicators scans output for known crash patterns
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

		// Swift runtime crash detection
		if swiftFatalErrorRegex.MatchString(line) {
			indicators.FatalErrorDetected = true
			indicators.SwiftRuntimeCrash = true
		}
		if swiftPreconditionRegex.MatchString(line) || swiftAssertionRegex.MatchString(line) {
			indicators.SwiftRuntimeCrash = true
		}
		if swiftForceUnwrapRegex.MatchString(line) || swiftIndexOutOfBoundsRegex.MatchString(line) {
			indicators.SwiftRuntimeCrash = true
		}
	}

	return indicators
}

// DetectSilentFailure detects cases where xcodebuild exits without proper output
func (p *Parser) DetectSilentFailure(output string, exitCode int) bool {
	// If exit code indicates failure but output is suspiciously small
	if exitCode != 0 && len(output) < 500 {
		return true
	}

	// If no success/failure markers found
	hasSuccessMarker := buildSuccessRegex.MatchString(output) ||
		testSuccessRegex.MatchString(output) ||
		cleanSuccessRegex.MatchString(output)
	hasFailureMarker := buildFailedRegex.MatchString(output) ||
		testFailedRegex.MatchString(output) ||
		cleanFailedRegex.MatchString(output)

	if !hasSuccessMarker && !hasFailureMarker && exitCode != 0 {
		return true
	}

	return false
}
