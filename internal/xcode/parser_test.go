package xcode

import (
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser returned nil")
	}
}

func TestParser_ParseBuildOutput(t *testing.T) {
	parser := NewParser()

	testOutput := `
Build succeeded
CompileC /path/to/build.o /path/to/source.m normal arm64
/path/to/source.m:10:5: error: undeclared identifier 'foo'
/path/to/source.m:15:3: warning: deprecated method
Archive path: /path/to/app.xcarchive
** BUILD FAILED **
`

	result := parser.ParseBuildOutput(testOutput)

	if result == nil {
		t.Fatal("ParseBuildOutput returned nil")
	}

	if result.Success {
		t.Error("Expected build to be marked as failed")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	} else {
		error := result.Errors[0]
		if error.File != "/path/to/source.m" {
			t.Errorf("Expected error file '/path/to/source.m', got '%s'", error.File)
		}
		if error.Line != 10 {
			t.Errorf("Expected error line 10, got %d", error.Line)
		}
		if error.Column != 5 {
			t.Errorf("Expected error column 5, got %d", error.Column)
		}
		if !strings.Contains(error.Message, "undeclared identifier") {
			t.Errorf("Expected error message to contain 'undeclared identifier', got '%s'", error.Message)
		}
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	} else {
		warning := result.Warnings[0]
		if warning.File != "/path/to/source.m" {
			t.Errorf("Expected warning file '/path/to/source.m', got '%s'", warning.File)
		}
		if warning.Line != 15 {
			t.Errorf("Expected warning line 15, got %d", warning.Line)
		}
		if !strings.Contains(warning.Message, "deprecated method") {
			t.Errorf("Expected warning message to contain 'deprecated method', got '%s'", warning.Message)
		}
	}

	if len(result.ArtifactPaths) != 1 {
		t.Errorf("Expected 1 artifact path, got %d", len(result.ArtifactPaths))
	} else if result.ArtifactPaths[0] != "/path/to/app.xcarchive" {
		t.Errorf("Expected artifact path '/path/to/app.xcarchive', got '%s'", result.ArtifactPaths[0])
	}
}

func TestParser_ParseTestOutput(t *testing.T) {
	parser := NewParser()

	testOutput := `
Test Suite 'All tests' started at 2023-10-01 10:00:00.000
Test Case 'MyTestClass.testSuccess' started.
Test Case 'MyTestClass.testSuccess' passed (0.001 seconds).
Test Case 'MyTestClass.testFailure' started.
Test Case 'MyTestClass.testFailure' failed (0.002 seconds).
Test Suite 'All tests' failed at 2023-10-01 10:00:01.000
** TEST FAILED **
`

	result := parser.ParseTestOutput(testOutput)

	if result == nil {
		t.Fatal("ParseTestOutput returned nil")
	}

	if result.Success {
		t.Error("Expected test to be marked as failed")
	}

	if result.TestSummary.TotalTests != 2 {
		t.Errorf("Expected 2 total tests, got %d", result.TestSummary.TotalTests)
	}

	if result.TestSummary.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", result.TestSummary.PassedTests)
	}

	if result.TestSummary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", result.TestSummary.FailedTests)
	}

	if len(result.TestSummary.TestResults) != 2 {
		t.Errorf("Expected 2 test results, got %d", len(result.TestSummary.TestResults))
	}

	if len(result.TestSummary.FailedTestsDetails) != 1 {
		t.Errorf("Expected 1 failed test detail, got %d", len(result.TestSummary.FailedTestsDetails))
	}
}

func TestParser_ParseTestOutputWithBundles(t *testing.T) {
	parser := NewParser()

	testOutput := `
Test Suite 'All tests' started at 2024-01-15 10:30:45.123
Test Suite 'LeMieLingueAppTests' started at 2024-01-15 10:30:45.125
Test Case 'LeMieLingueAppTests.testExample' started
Test Case 'LeMieLingueAppTests.testExample' passed (0.123 seconds)
Test Case 'LeMieLingueAppTests.testPerformance' started
Test Case 'LeMieLingueAppTests.testPerformance' passed (1.234 seconds)
Test Suite 'LeMieLingueAppTests' passed at 2024-01-15 10:30:46.482
	 Executed 2 tests, with 0 failures (0 unexpected) in 1.357 seconds

Test Suite 'LeMieLingueAppUITests' started at 2024-01-15 10:30:46.485
Test Case 'LeMieLingueAppUITests.testLaunch' started
Test Case 'LeMieLingueAppUITests.testLaunch' passed (5.678 seconds)
Test Suite 'LeMieLingueAppUITests' passed at 2024-01-15 10:30:52.165
	 Executed 1 test, with 0 failures (0 unexpected) in 5.680 seconds

Test Suite 'All tests' passed at 2024-01-15 10:30:52.167
	 Executed 3 tests, with 0 failures (0 unexpected) in 7.044 seconds

** TEST SUCCEEDED **
`

	result := parser.ParseTestOutput(testOutput)

	if result == nil {
		t.Fatal("ParseTestOutput returned nil")
	}

	if !result.Success {
		t.Error("Expected test to be marked as successful")
	}

	// Check test counts
	if result.TestSummary.TotalTests != 3 {
		t.Errorf("Expected 3 total tests, got %d", result.TestSummary.TotalTests)
	}

	if result.TestSummary.PassedTests != 3 {
		t.Errorf("Expected 3 passed tests, got %d", result.TestSummary.PassedTests)
	}

	// Check test bundles
	if len(result.TestSummary.TestBundles) != 2 {
		t.Fatalf("Expected 2 test bundles, got %d", len(result.TestSummary.TestBundles))
	}

	// Find unit test bundle
	var unitBundle, uiBundle *struct {
		Name      string
		Type      string
		Executed  bool
		Status    string
		TestCount int
	}

	for i := range result.TestSummary.TestBundles {
		bundle := &result.TestSummary.TestBundles[i]
		if bundle.Name == "LeMieLingueAppTests" {
			unitBundle = &struct {
				Name      string
				Type      string
				Executed  bool
				Status    string
				TestCount int
			}{bundle.Name, bundle.Type, bundle.Executed, bundle.Status, bundle.TestCount}
		} else if bundle.Name == "LeMieLingueAppUITests" {
			uiBundle = &struct {
				Name      string
				Type      string
				Executed  bool
				Status    string
				TestCount int
			}{bundle.Name, bundle.Type, bundle.Executed, bundle.Status, bundle.TestCount}
		}
	}

	// Verify unit test bundle
	if unitBundle == nil {
		t.Fatal("Unit test bundle not found")
	}
	if unitBundle.Type != "unit" {
		t.Errorf("Expected unit test bundle type 'unit', got '%s'", unitBundle.Type)
	}
	if !unitBundle.Executed {
		t.Error("Expected unit test bundle to be marked as executed")
	}
	if unitBundle.Status != "passed" {
		t.Errorf("Expected unit test bundle status 'passed', got '%s'", unitBundle.Status)
	}
	if unitBundle.TestCount != 2 {
		t.Errorf("Expected unit test bundle to have 2 tests, got %d", unitBundle.TestCount)
	}

	// Verify UI test bundle
	if uiBundle == nil {
		t.Fatal("UI test bundle not found")
	}
	if uiBundle.Type != "ui" {
		t.Errorf("Expected UI test bundle type 'ui', got '%s'", uiBundle.Type)
	}
	if !uiBundle.Executed {
		t.Error("Expected UI test bundle to be marked as executed")
	}
	if uiBundle.Status != "passed" {
		t.Errorf("Expected UI test bundle status 'passed', got '%s'", uiBundle.Status)
	}
	if uiBundle.TestCount != 1 {
		t.Errorf("Expected UI test bundle to have 1 test, got %d", uiBundle.TestCount)
	}
}

func TestParser_DetectBundleType(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name         string
		suiteName    string
		expectedType string
	}{
		{"UI tests with UITest suffix", "MyAppUITests", "ui"},
		{"UI tests with UITest in name", "MyAppUITestSuite", "ui"},
		{"UI tests with UI-Test", "MyApp-UI-Tests", "ui"},
		{"Performance tests", "MyAppPerformanceTests", "performance"},
		{"Performance tests abbreviated", "MyAppPerfTests", "performance"},
		{"Integration tests", "MyAppIntegrationTests", "integration"},
		{"Unit tests", "MyAppTests", "unit"},
		{"Unit tests explicit", "MyAppUnitTests", "unit"},
		{"Default to unit", "SomeRandomName", "unit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.detectBundleType(tt.suiteName)
			if result != tt.expectedType {
				t.Errorf("detectBundleType(%q) = %q, expected %q", tt.suiteName, result, tt.expectedType)
			}
		})
	}
}

func TestParser_ParseCleanOutput(t *testing.T) {
	parser := NewParser()

	testOutput := `
Cleaning build products and build folder
Removed /path/to/derived/data
Cleaning /path/to/build/folder
** CLEAN SUCCEEDED **
`

	result := parser.ParseCleanOutput(testOutput)

	if result == nil {
		t.Fatal("ParseCleanOutput returned nil")
	}

	if !result.Success {
		t.Error("Expected clean to be marked as successful")
	}

	if len(result.CleanedPaths) == 0 {
		t.Error("Expected some cleaned paths to be detected")
	}
}

func TestParser_ExtractBuildSettings(t *testing.T) {
	parser := NewParser()

	testOutput := `
Build settings from command line:
    ARCHS = arm64
    CONFIGURATION_BUILD_DIR = /path/to/build
    PLATFORM_NAME = iphoneos
    
=== BUILD TARGET MyApp OF PROJECT MyProject WITH CONFIGURATION Debug ===
`

	settings := parser.ExtractBuildSettings(testOutput)

	if len(settings) == 0 {
		t.Error("Expected build settings to be extracted")
	}

	if settings["ARCHS"] != "arm64" {
		t.Errorf("Expected ARCHS = arm64, got %v", settings["ARCHS"])
	}

	if settings["PLATFORM_NAME"] != "iphoneos" {
		t.Errorf("Expected PLATFORM_NAME = iphoneos, got %v", settings["PLATFORM_NAME"])
	}
}

func TestParser_IsSuccess(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		output      string
		commandType string
		expected    bool
	}{
		{"build success", "** BUILD SUCCEEDED **", "build", true},
		{"build failed", "** BUILD FAILED **", "build", false},
		{"test success", "** TEST SUCCEEDED **", "test", true},
		{"test failed", "** TEST FAILED **", "test", false},
		{"clean success", "** CLEAN SUCCEEDED **", "clean", true},
		{"clean failed", "** CLEAN FAILED **", "clean", false},
		{"unknown command", "** BUILD SUCCEEDED **", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsSuccess(tt.output, tt.commandType)
			if result != tt.expected {
				t.Errorf("IsSuccess(%q, %q) = %v, expected %v", tt.output, tt.commandType, result, tt.expected)
			}
		})
	}
}

func TestParser_ExtractErrors(t *testing.T) {
	parser := NewParser()

	testOutput := `
/path/to/file1.m:10:5: error: undeclared identifier 'foo'
/path/to/file2.m:20:3: warning: deprecated method
/path/to/file3.m: error: compilation failed
Some normal output
`

	errors := parser.ExtractErrors(testOutput)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Check first error (with line and column)
	if errors[0].File != "/path/to/file1.m" {
		t.Errorf("Expected first error file '/path/to/file1.m', got '%s'", errors[0].File)
	}
	if errors[0].Line != 10 {
		t.Errorf("Expected first error line 10, got %d", errors[0].Line)
	}

	// Check second error (without line and column)
	if errors[1].File != "/path/to/file3.m" {
		t.Errorf("Expected second error file '/path/to/file3.m', got '%s'", errors[1].File)
	}
	if errors[1].Line != 0 {
		t.Errorf("Expected second error line 0, got %d", errors[1].Line)
	}
}

func TestParser_ExtractWarnings(t *testing.T) {
	parser := NewParser()

	testOutput := `
/path/to/file1.m:10:5: error: undeclared identifier 'foo'
/path/to/file2.m:20:3: warning: deprecated method
/path/to/file3.m: warning: unused variable
Some normal output
`

	warnings := parser.ExtractWarnings(testOutput)

	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}

	// Check first warning (with line and column)
	if warnings[0].File != "/path/to/file2.m" {
		t.Errorf("Expected first warning file '/path/to/file2.m', got '%s'", warnings[0].File)
	}
	if warnings[0].Line != 20 {
		t.Errorf("Expected first warning line 20, got %d", warnings[0].Line)
	}

	// Check second warning (without line and column)
	if warnings[1].File != "/path/to/file3.m" {
		t.Errorf("Expected second warning file '/path/to/file3.m', got '%s'", warnings[1].File)
	}
	if warnings[1].Line != 0 {
		t.Errorf("Expected second warning line 0, got %d", warnings[1].Line)
	}
}
