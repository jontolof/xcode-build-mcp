package filter

import (
	"strings"
	"testing"
)

// TestFailureAwareFiltering verifies that test failures are always visible
// even when there are many passing tests that would normally exceed the char limit
func TestFailureAwareFiltering(t *testing.T) {
	// Simulate output with 377 passing tests + 8 failing tests
	// This replicates the real scenario from the iOS team
	var output strings.Builder

	// Add test suite header
	output.WriteString("Test Suite 'All tests' started at 2025-11-30 03:34:44.409\n")

	// Add 377 passing test cases (detailed output)
	for i := 1; i <= 377; i++ {
		output.WriteString("Test Suite 'PassingTestSuite' started at 2025-11-30 03:34:44.410\n")
		output.WriteString("Test Case '-[SnackaAppTests.PassingTest testPass]' started.\n")
		output.WriteString("Test Case '-[SnackaAppTests.PassingTest testPass]' passed (0.037 seconds).\n")
		output.WriteString("Test Suite 'PassingTestSuite' passed at 2025-11-30 03:34:44.498.\n")
		output.WriteString("\t Executed 1 tests, with 0 failures (0 unexpected) in 0.037 (0.089) seconds\n")
	}

	// Add 8 failing test cases - THESE MUST BE VISIBLE!
	failingTests := []string{
		"AuthenticationTests.testLoginWithInvalidCredentials",
		"RecordingTests.testAudioRecordingFailure",
		"NetworkTests.testTimeoutHandling",
		"DatabaseTests.testConcurrentWrites",
		"CacheTests.testEvictionPolicy",
		"ValidationTests.testEmailFormat",
		"ParserTests.testMalformedJSON",
		"SecurityTests.testXSSPrevention",
	}

	output.WriteString("Test Suite 'FailingTestSuite' started at 2025-11-30 03:36:40.000\n")
	for _, testName := range failingTests {
		output.WriteString("Test Case '-[SnackaAppTests." + testName + "]' started.\n")
		output.WriteString("Test Case '-[SnackaAppTests." + testName + "]' failed (0.123 seconds).\n")
	}
	output.WriteString("Test Suite 'FailingTestSuite' failed at 2025-11-30 03:36:41.000.\n")
	output.WriteString("\t Executed 8 tests, with 8 failures (0 unexpected) in 0.984 seconds\n")

	// Add final summary
	output.WriteString("Test Suite 'All tests' failed at 2025-11-30 03:36:41.005\n")
	output.WriteString("\t Executed 385 tests, with 8 failures (0 unexpected) in 120.000 seconds\n")
	output.WriteString("** TEST FAILED **\n")

	outputStr := output.String()

	// Test with Standard mode (the problematic case)
	filter := NewFilter(Standard)
	filtered := filter.Filter(outputStr)

	t.Logf("Original output: %d chars", len(outputStr))
	t.Logf("Filtered output: %d chars", len(filtered))
	t.Logf("Reduction: %.1f%%", (1.0-float64(len(filtered))/float64(len(outputStr)))*100)

	// Verify all 8 failures are visible in the filtered output
	for _, testName := range failingTests {
		if !strings.Contains(filtered, testName) {
			t.Errorf("CRITICAL: Failed test '%s' is missing from filtered output!", testName)
		}
	}

	// Verify final summary is included
	if !strings.Contains(filtered, "** TEST FAILED **") {
		t.Error("Final test result is missing")
	}

	if !strings.Contains(filtered, "Executed 385 tests, with 8 failures") {
		t.Error("Test summary is missing")
	}

	// Verify we're not showing all 377 passing test details
	passingCount := strings.Count(filtered, "testPass")
	if passingCount > 50 {
		t.Errorf("Too many passing test details shown: %d (should be minimal)", passingCount)
	}

	// Verify output is within reasonable limits (should be well under 40K)
	if len(filtered) > 40000 {
		t.Errorf("Filtered output exceeds 40K char limit: %d chars", len(filtered))
	}

	t.Logf("\n=== Filtered Output Sample (first 2000 chars) ===\n%s", filtered[:min(2000, len(filtered))])
	t.Logf("\n=== Filtered Output Sample (last 2000 chars) ===\n%s", filtered[max(0, len(filtered)-2000):])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestFailureVisibilityUnderLimit verifies failures are visible even at char limit
func TestFailureVisibilityUnderLimit(t *testing.T) {
	// Create output that's just under 40K chars with failures at the end
	var output strings.Builder

	output.WriteString("Test Suite 'All tests' started at 2025-11-30 03:34:44.409\n")

	// Add enough passing tests to approach the 40K limit
	for i := 1; i <= 500; i++ {
		output.WriteString("Test Suite 'PassingTestSuite' started at 2025-11-30 03:34:44.410\n")
		output.WriteString("Test Case '-[SnackaAppTests.PassingTest testPass]' started.\n")
		output.WriteString("Test Case '-[SnackaAppTests.PassingTest testPass]' passed (0.037 seconds).\n")
		output.WriteString("Test Suite 'PassingTestSuite' passed at 2025-11-30 03:34:44.498.\n")
		output.WriteString("\t Executed 1 tests, with 0 failures (0 unexpected) in 0.037 (0.089) seconds\n")
	}

	// Add critical failure information at the end
	output.WriteString("Test Case '-[SnackaAppTests.CriticalTest testImportantFeature]' started.\n")
	output.WriteString("Test Case '-[SnackaAppTests.CriticalTest testImportantFeature]' failed (0.123 seconds).\n")
	output.WriteString("Test Suite 'All tests' failed at 2025-11-30 03:36:41.005\n")
	output.WriteString("\t Executed 501 tests, with 1 failures (0 unexpected) in 120.000 seconds\n")
	output.WriteString("** TEST FAILED **\n")

	outputStr := output.String()
	filter := NewFilter(Standard)
	filtered := filter.Filter(outputStr)

	// The critical failure MUST be visible
	if !strings.Contains(filtered, "CriticalTest") {
		t.Error("CRITICAL: Failed test 'CriticalTest' is missing from filtered output!")
	}

	if !strings.Contains(filtered, "testImportantFeature") {
		t.Error("CRITICAL: Failed test method 'testImportantFeature' is missing!")
	}

	if !strings.Contains(filtered, "** TEST FAILED **") {
		t.Error("Final failure status is missing")
	}

	t.Logf("Output size: %d chars (original: %d)", len(filtered), len(outputStr))
	t.Logf("Critical failure visible: ✓")
}

// TestMinimalModeStillAggressive verifies minimal mode filters even more aggressively
func TestMinimalModeStillAggressive(t *testing.T) {
	var output strings.Builder

	output.WriteString("Test Suite 'All tests' started at 2025-11-30 03:34:44.409\n")

	// Add many test cases
	for i := 1; i <= 100; i++ {
		output.WriteString("Test Case '-[SnackaAppTests.Test testCase]' started.\n")
		output.WriteString("Test Case '-[SnackaAppTests.Test testCase]' passed (0.037 seconds).\n")
	}

	output.WriteString("Test Suite 'All tests' passed at 2025-11-30 03:36:41.005\n")
	output.WriteString("\t Executed 100 tests, with 0 failures (0 unexpected) in 10.000 seconds\n")
	output.WriteString("** TEST SUCCEEDED **\n")

	outputStr := output.String()
	filter := NewFilter(Minimal)
	filtered := filter.Filter(outputStr)

	// Minimal mode should be very small
	if len(filtered) > 5000 {
		t.Errorf("Minimal mode output too large: %d chars (should be under 5000)", len(filtered))
	}

	// But should still have the final result
	if !strings.Contains(filtered, "** TEST SUCCEEDED **") {
		t.Error("Final test result is missing from minimal mode")
	}

	t.Logf("Minimal mode output: %d chars", len(filtered))
}
