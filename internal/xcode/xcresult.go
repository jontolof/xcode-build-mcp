package xcode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

// XCResultParser parses Xcode result bundles (.xcresult) for accurate test results
type XCResultParser struct {
	xcresulttoolPath string
}

// NewXCResultParser creates a new xcresult parser
func NewXCResultParser() *XCResultParser {
	return &XCResultParser{
		xcresulttoolPath: "xcrun", // Uses xcrun to find xcresulttool
	}
}

// XCResultSummary represents the parsed summary from an xcresult bundle
type XCResultSummary struct {
	TotalTests         int
	PassedTests        int
	FailedTestCount    int
	SkippedTests       int
	TestBundles        []types.TestBundle
	FailedTestDetails  []types.TestCase
	SkippedTestDetails []types.TestCase
	Duration           time.Duration
}

// ParseResultBundle parses an xcresult bundle and returns structured test results
func (p *XCResultParser) ParseResultBundle(bundlePath string) (*XCResultSummary, error) {
	// Verify the bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("xcresult bundle not found: %s", bundlePath)
	}

	// Get the test results using xcresulttool
	output, err := p.runXCResultTool(bundlePath, "get", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to read xcresult bundle: %w", err)
	}

	// Parse the JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse xcresult JSON: %w", err)
	}

	return p.extractTestSummary(result, bundlePath)
}

// runXCResultTool executes xcresulttool with the given arguments
// Note: Modern Xcode requires --legacy flag for compatibility
func (p *XCResultParser) runXCResultTool(bundlePath string, args ...string) ([]byte, error) {
	fullArgs := append([]string{"xcresulttool"}, args...)
	// Add --legacy flag for modern Xcode compatibility (required as of Xcode 16+)
	fullArgs = append(fullArgs, "--legacy")
	fullArgs = append(fullArgs, "--path", bundlePath)

	cmd := exec.Command(p.xcresulttoolPath, fullArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("xcresulttool failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// extractTestSummary extracts test summary from the parsed xcresult JSON
func (p *XCResultParser) extractTestSummary(result map[string]interface{}, bundlePath string) (*XCResultSummary, error) {
	summary := &XCResultSummary{
		TestBundles:        []types.TestBundle{},
		FailedTestDetails:  []types.TestCase{},
		SkippedTestDetails: []types.TestCase{},
	}

	// Navigate the xcresult structure to find test results
	// The structure is: actions -> _values -> [] -> actionResult -> testsRef -> id
	actions, ok := result["actions"].(map[string]interface{})
	if !ok {
		return summary, nil
	}

	values, ok := actions["_values"].([]interface{})
	if !ok {
		return summary, nil
	}

	for _, action := range values {
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			continue
		}

		// Get the action result
		actionResult, ok := actionMap["actionResult"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a test action
		testsRef, ok := actionResult["testsRef"].(map[string]interface{})
		if !ok {
			continue
		}

		// Get the test results ID and fetch details
		testID, ok := testsRef["id"].(map[string]interface{})
		if !ok {
			continue
		}

		idValue, ok := testID["_value"].(string)
		if !ok {
			continue
		}

		// Fetch the detailed test results
		testDetails, err := p.fetchTestDetails(bundlePath, idValue)
		if err != nil {
			// Don't silently ignore - return error to caller
			return nil, fmt.Errorf("failed to fetch test details for ID %s: %w", idValue, err)
		}

		// Parse test details
		p.parseTestDetails(testDetails, summary)
	}

	return summary, nil
}

// fetchTestDetails fetches detailed test results by ID
func (p *XCResultParser) fetchTestDetails(bundlePath, refID string) (map[string]interface{}, error) {
	output, err := p.runXCResultTool(bundlePath, "get", "--format", "json", "--id", refID)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseTestDetails recursively parses test details and updates the summary
func (p *XCResultParser) parseTestDetails(details map[string]interface{}, summary *XCResultSummary) {
	// Get summaries array
	summaries, ok := details["summaries"].(map[string]interface{})
	if !ok {
		return
	}

	values, ok := summaries["_values"].([]interface{})
	if !ok {
		return
	}

	for _, summaryItem := range values {
		summaryMap, ok := summaryItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Get testable summaries
		testableSummaries, ok := summaryMap["testableSummaries"].(map[string]interface{})
		if !ok {
			continue
		}

		testableValues, ok := testableSummaries["_values"].([]interface{})
		if !ok {
			continue
		}

		for _, testable := range testableValues {
			p.parseTestableItem(testable, summary)
		}
	}
}

// parseTestableItem parses a single testable item (test bundle)
func (p *XCResultParser) parseTestableItem(testable interface{}, summary *XCResultSummary) {
	testableMap, ok := testable.(map[string]interface{})
	if !ok {
		return
	}

	// Get bundle name
	bundleName := ""
	if name, ok := testableMap["name"].(map[string]interface{}); ok {
		if value, ok := name["_value"].(string); ok {
			bundleName = value
		}
	}

	bundle := types.TestBundle{
		Name:     bundleName,
		Executed: true,
		Status:   "passed",
	}

	// Get tests
	if tests, ok := testableMap["tests"].(map[string]interface{}); ok {
		if testValues, ok := tests["_values"].([]interface{}); ok {
			p.parseTests(testValues, summary, &bundle)
		}
	}

	// Determine bundle type from name
	nameLower := strings.ToLower(bundleName)
	if strings.Contains(nameLower, "uitest") || strings.Contains(nameLower, "ui-test") {
		bundle.Type = "ui"
	} else if strings.Contains(nameLower, "performance") {
		bundle.Type = "performance"
	} else {
		bundle.Type = "unit"
	}

	summary.TestBundles = append(summary.TestBundles, bundle)
}

// parseTests recursively parses test groups and individual tests
func (p *XCResultParser) parseTests(tests []interface{}, summary *XCResultSummary, bundle *types.TestBundle) {
	for _, test := range tests {
		testMap, ok := test.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a test group (has subtests) or a leaf test
		if subtests, ok := testMap["subtests"].(map[string]interface{}); ok {
			if subtestValues, ok := subtests["_values"].([]interface{}); ok {
				p.parseTests(subtestValues, summary, bundle)
			}
			continue
		}

		// This is a leaf test - parse the result
		testCase := types.TestCase{}

		// Get test name
		if name, ok := testMap["name"].(map[string]interface{}); ok {
			if value, ok := name["_value"].(string); ok {
				testCase.Name = value
			}
		}

		// Get test identifier (includes class name)
		if identifier, ok := testMap["identifier"].(map[string]interface{}); ok {
			if value, ok := identifier["_value"].(string); ok {
				parts := strings.Split(value, "/")
				if len(parts) > 1 {
					// Normal test method: "ClassName/testMethodName"
					testCase.ClassName = parts[0]
				} else if len(parts) == 1 && parts[0] != "" {
					// Class-level skip: identifier is just "ClassName"
					// This happens when entire test class is skipped via @available
					testCase.ClassName = parts[0]
				}
			}
		}

		// Get test status
		if testStatus, ok := testMap["testStatus"].(map[string]interface{}); ok {
			if value, ok := testStatus["_value"].(string); ok {
				testCase.Status = strings.ToLower(value)
			}
		}

		// Get duration
		if duration, ok := testMap["duration"].(map[string]interface{}); ok {
			if value, ok := duration["_value"].(string); ok {
				if d, err := time.ParseDuration(value + "s"); err == nil {
					testCase.Duration = d
				}
			}
		}

		// Update counts
		summary.TotalTests++
		bundle.TestCount++

		switch testCase.Status {
		case "success":
			summary.PassedTests++
		case "":
			// Empty status indicates class-level skip (e.g., @available attribute)
			// The entire test class was skipped before any tests ran
			summary.SkippedTests++
			testCase.Status = "skipped (class)"
			if testCase.Message == "" {
				testCase.Message = "Test class skipped (likely via @available or conditional compilation)"
			}
			summary.SkippedTestDetails = append(summary.SkippedTestDetails, testCase)
		case "failure":
			summary.FailedTestCount++
			summary.FailedTestDetails = append(summary.FailedTestDetails, testCase)
			bundle.Status = "failed"

			// Try to get failure message
			if failureSummaries, ok := testMap["failureSummaries"].(map[string]interface{}); ok {
				if failureValues, ok := failureSummaries["_values"].([]interface{}); ok {
					for _, failure := range failureValues {
						if failureMap, ok := failure.(map[string]interface{}); ok {
							if message, ok := failureMap["message"].(map[string]interface{}); ok {
								if value, ok := message["_value"].(string); ok {
									testCase.Message = value
									break
								}
							}
						}
					}
				}
			}
		case "skipped":
			summary.SkippedTests++
			// Capture skip reason if available
			if summaryMessage, ok := testMap["summaryMessage"].(map[string]interface{}); ok {
				if value, ok := summaryMessage["_value"].(string); ok {
					testCase.Message = value
				}
			}
			summary.SkippedTestDetails = append(summary.SkippedTestDetails, testCase)
		case "expected failure", "expectedfailure":
			// Expected failures: tests that were expected to fail and did
			// Count as skipped since they don't affect pass/fail status
			summary.SkippedTests++
			// Capture reason for expected failure
			if summaryMessage, ok := testMap["summaryMessage"].(map[string]interface{}); ok {
				if value, ok := summaryMessage["_value"].(string); ok {
					testCase.Message = value
				}
			}
			summary.SkippedTestDetails = append(summary.SkippedTestDetails, testCase)
		default:
			// Unknown status - log for debugging and count as skipped
			// This prevents tests from disappearing from totals
			if os.Getenv("MCP_LOG_LEVEL") == "debug" {
				fmt.Fprintf(os.Stderr, "[xcresult] Warning: Unknown test status '%s' for test '%s', counting as skipped\n",
					testCase.Status, testCase.Name)
			}
			summary.SkippedTests++
			// Also capture the details for unknown/skipped statuses
			summary.SkippedTestDetails = append(summary.SkippedTestDetails, testCase)
		}
	}
}

// GenerateResultBundlePath creates a temporary path for storing xcresult bundle
func GenerateResultBundlePath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("xcode_test_%d.xcresult", time.Now().UnixNano()))
}

// CleanupResultBundle removes the temporary xcresult bundle
func CleanupResultBundle(path string) error {
	return os.RemoveAll(path)
}
