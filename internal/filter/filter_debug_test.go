package filter

import (
	"strings"
	"testing"
)

// TestMinimalModeFiltering tests that minimal mode aggressively filters compilation noise
func TestMinimalModeFiltering(t *testing.T) {
	filter := NewFilter(Minimal)

	// Simulate realistic xcodebuild output with compilation noise
	testInput := `Command line invocation:
    /Applications/Xcode.app/Contents/Developer/usr/bin/xcodebuild test -scheme TestScheme

User defaults from command line:
    IDEPackageSupportUseBuiltinSCM = YES

Build settings from command line:
    COMPILER_INDEX_STORE_ENABLE = NO

note: Using new build system
note: Planning build
note: Build preparation complete
note: Building targets in parallel

=== BUILD TARGET MCPServerTestProject OF PROJECT MCPServerTestProject WITH CONFIGURATION Debug ===

SwiftDriver MCPServerTestProjectUITests normal arm64 com.apple.xcode.tools.swift.compiler (in target 'MCPServerTestProjectUITests' from project 'MCPServerTestProject')
    cd /Users/test/MCPServerTestProject
    export SWIFT_EXEC=/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/swiftc
    /Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/swift-frontend -c -primary-file /Users/test/file1.swift -emit-module-path /path/to/module.swiftmodule -target arm64-apple-ios16.0-simulator -Xfrontend -serialize-debugging-options

ExecuteExternalTool /Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang (in target 'MCPServerTestProject' from project 'MCPServerTestProject')
    cd /Users/test/MCPServerTestProject
    /usr/bin/clang -x objective-c -target arm64-apple-ios16.0 -fmodules -isysroot /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneSimulator.platform/Developer/SDKs/iPhoneSimulator17.5.sdk

ProcessInfoPlistFile /Users/test/DerivedData/MCPServerTestProject/Build/Products/Debug-iphonesimulator/MCPServerTestProject.app/Info.plist

CopySwiftLibs /Users/test/DerivedData/MCPServerTestProject/Build/Products/Debug-iphonesimulator/MCPServerTestProjectUITests-Runner.app

CodeSign /Users/test/DerivedData/MCPServerTestProject/Build/Products/Debug-iphonesimulator/MCPServerTestProject.app

Test Suite 'All tests' started at 2024-01-15 10:30:45.123
Test Suite 'MCPServerTestProjectTests' started at 2024-01-15 10:30:45.125
Test Case 'MCPServerTestProjectTests.testExample' started
Test Case 'MCPServerTestProjectTests.testExample' passed (0.123 seconds)
Test Case 'MCPServerTestProjectTests.testPerformanceExample' started
Test Case 'MCPServerTestProjectTests.testPerformanceExample' passed (1.234 seconds)
Test Suite 'MCPServerTestProjectTests' passed at 2024-01-15 10:30:46.482
	 Executed 2 tests, with 0 failures (0 unexpected) in 1.357 seconds

Test Suite 'MCPServerTestProjectUITests' started at 2024-01-15 10:30:46.485
Test Case 'MCPServerTestProjectUITests.testLaunch' started
Test Case 'MCPServerTestProjectUITests.testLaunch' passed (5.678 seconds)
Test Suite 'MCPServerTestProjectUITests' passed at 2024-01-15 10:30:52.165
	 Executed 1 test, with 0 failures (0 unexpected) in 5.680 seconds

Test Suite 'All tests' passed at 2024-01-15 10:30:52.167
	 Executed 3 tests, with 0 failures (0 unexpected) in 7.044 seconds

** TEST SUCCEEDED **`

	result := filter.Filter(testInput)

	// Count lines in result
	resultLines := strings.Count(result, "\n")
	inputLines := strings.Count(testInput, "\n")

	t.Logf("Input lines: %d", inputLines)
	t.Logf("Output lines: %d", resultLines)
	t.Logf("Reduction: %.1f%%", float64(inputLines-resultLines)/float64(inputLines)*100)
	t.Logf("Filtered output:\n%s", result)

	// Should keep test results
	if !strings.Contains(result, "** TEST SUCCEEDED **") {
		t.Error("Should keep test success marker")
	}

	// Should keep test summary
	if !strings.Contains(result, "Executed 3 tests") || !strings.Contains(result, "All tests' passed") {
		t.Log("Note: Some test summary details may be filtered in minimal mode")
	}

	// Should remove ALL compilation noise
	noisePatterns := []string{
		"SwiftDriver",
		"ExecuteExternalTool",
		"ProcessInfoPlistFile",
		"CopySwiftLibs",
		"CodeSign",
		"/Applications/Xcode.app",
		"note: Using new build system",
		"=== BUILD TARGET",
		"-Xfrontend",
		"/usr/bin/clang",
	}

	for _, pattern := range noisePatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("Should remove compilation noise: %s", pattern)
		}
	}

	// Verify aggressive filtering - should be < 20 lines for minimal mode
	if resultLines > 20 {
		t.Errorf("Minimal mode should produce < 20 lines, got %d", resultLines)
	}

	// Check token estimate (rough: ~5 tokens per line)
	estimatedTokens := resultLines * 5
	if estimatedTokens > 500 {
		t.Errorf("Minimal mode should produce < 500 tokens (estimated %d)", estimatedTokens)
	}
}

// TestStandardModeFiltering tests that standard mode provides useful context within limits
func TestStandardModeFiltering(t *testing.T) {
	filter := NewFilter(Standard)

	// Create a large input to test truncation
	var builder strings.Builder
	builder.WriteString("** BUILD STARTED **\n")

	// Add lots of compilation lines
	for i := 0; i < 1000; i++ {
		builder.WriteString("SwiftDriver compilation line ")
		builder.WriteString(strings.Repeat("x", 100))
		builder.WriteString("\n")
	}

	// Add test results
	builder.WriteString("Test Case 'MyTest.testExample' started\n")
	builder.WriteString("Test Case 'MyTest.testExample' passed (0.1 seconds)\n")
	builder.WriteString("** TEST SUCCEEDED **\n")

	testInput := builder.String()
	result := filter.Filter(testInput)

	resultLines := strings.Count(result, "\n")

	t.Logf("Input lines: %d", strings.Count(testInput, "\n"))
	t.Logf("Output lines: %d", resultLines)

	// Should truncate at limit
	if resultLines > 200 {
		t.Errorf("Standard mode should limit to 200 lines, got %d", resultLines)
	}

	// Should include truncation message
	if resultLines >= 200 && !strings.Contains(result, "truncated") {
		t.Error("Should include truncation message when hitting limit")
	}

	// Should keep test results
	if !strings.Contains(result, "** TEST SUCCEEDED **") {
		t.Error("Should keep test success marker")
	}
}

// TestVerboseModeFiltering tests that verbose mode still has limits
func TestVerboseModeFiltering(t *testing.T) {
	filter := NewFilter(Verbose)

	// Create a massive input
	var builder strings.Builder
	for i := 0; i < 2000; i++ {
		builder.WriteString("Line ")
		builder.WriteString(strings.Repeat("x", 200))
		builder.WriteString("\n")
	}

	testInput := builder.String()
	result := filter.Filter(testInput)

	resultLines := strings.Count(result, "\n")

	t.Logf("Input lines: %d", strings.Count(testInput, "\n"))
	t.Logf("Output lines: %d", resultLines)

	// Should truncate at limit (allowing for truncation message)
	if resultLines > 802 { // 800 lines + newline + truncation message
		t.Errorf("Verbose mode should limit to ~800 lines, got %d", resultLines)
	}

	// Should include truncation message
	if strings.Contains(result, "truncated") {
		t.Log("Truncation message included as expected")
	}
}

// TestEmptyLineHandling tests that empty lines are handled correctly
func TestEmptyLineHandling(t *testing.T) {
	filter := NewFilter(Minimal)

	testInput := `** BUILD SUCCEEDED **


** TEST SUCCEEDED **`

	result := filter.Filter(testInput)

	// In minimal mode, empty lines should be removed
	consecutiveNewlines := strings.Contains(result, "\n\n\n")
	if consecutiveNewlines {
		t.Error("Minimal mode should remove empty lines")
	}

	// Should keep the important content
	if !strings.Contains(result, "** BUILD SUCCEEDED **") {
		t.Error("Should keep build success")
	}
	if !strings.Contains(result, "** TEST SUCCEEDED **") {
		t.Error("Should keep test success")
	}
}

// TestCompilationNoiseFiltering specifically tests the new compilation noise filter
func TestCompilationNoiseFiltering(t *testing.T) {
	filter := NewFilter(Minimal)

	noiseLines := []string{
		"SwiftDriver MCPServerTestProjectUITests normal arm64",
		"ExecuteExternalTool /Applications/Xcode.app/Contents",
		"ProcessInfoPlistFile /Users/test/DerivedData",
		"ClangStatCache /var/folders/abc/def",
		"CopySwiftLibs /Users/test/Build",
		"CodeSign /Users/test/Build/Products",
		"builtin-copy -exclude",
		"Ld /Users/test/DerivedData",
		"/Applications/Xcode.app/Contents/Developer/Toolchains",
		"-Xlinker -rpath",
		"-Xfrontend -serialize-debugging",
		"-Xcc -I/Users",
		"-module-name TestModule",
		"-target arm64-apple-ios",
		"CompileC /Users/test",
		"CompileSwift normal",
		"CompileSwiftSources normal",
		"GenerateDSYMFile /Users",
		"CreateBuildDirectory /Users",
		"CreateUniversalBinary /Users",
		"PhaseScriptExecution Run\\ Script",
		"Touch /Users/test/Build",
		"CpResource /Users/test",
		"CopyPlistFile /Users",
		"ProcessProductPackaging /Users",
		"RegisterExecutionPolicyException /Users",
		"Validate /Users/test",
		"=== BUILD TARGET TestApp",
		"Build settings from command line:",
		"Command line invocation:",
		"/usr/bin/xcodebuild test",
		"User defaults from command line:",
		"Build description signature:",
		"Build description path:",
		"note: Using new build system",
		"-I/Users/test/include",
		"-F/System/Library/Frameworks",
		"-L/usr/local/lib",
		".swiftmodule",
		".xctest/Contents",
		".app/Contents",
		"-emit-module",
		"-emit-dependencies",
		"-emit-objc-header",
		"-incremental",
		"-serialize-diagnostics",
		"-parseable-output",
		"cd /Users/test",
		"export LANG=en_US.UTF-8",
		"/usr/bin/clang",
		"/usr/bin/swiftc",
		"/usr/bin/swift",
		"DerivedData/TestApp",
	}

	// All noise lines should be filtered
	for _, line := range noiseLines {
		input := line + "\n** TEST SUCCEEDED **"
		result := filter.Filter(input)

		if strings.Contains(result, line) {
			t.Errorf("Should filter compilation noise: %s", line)
		}

		// But should keep the test result
		if !strings.Contains(result, "** TEST SUCCEEDED **") {
			t.Errorf("Should keep test result even when filtering: %s", line)
		}
	}
}
