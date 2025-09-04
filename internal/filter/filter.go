package filter

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type OutputMode string

const (
	Minimal  OutputMode = "minimal"  // Errors only
	Standard OutputMode = "standard" // Errors + warnings + summary
	Verbose  OutputMode = "verbose"  // Everything
)

type FilterAction string

const (
	Keep      FilterAction = "KEEP"
	Remove    FilterAction = "REMOVE"
	Summarize FilterAction = "SUMMARIZE"
)

type FilterRule struct {
	Pattern  *regexp.Regexp
	Action   FilterAction
	Priority int
	Name     string
}

type Filter struct {
	rules     []FilterRule
	mode      OutputMode
	stats     *FilterStats
	debugMode bool
	debugFile *os.File
}

type FilterStats struct {
	TotalLines         int
	FilteredLines      int
	KeptLines          int
	SummarizedSections int
	RulesApplied       map[string]int
}

func NewFilter(mode OutputMode) *Filter {
	f := &Filter{
		rules: getDefaultRules(),
		mode:  mode,
		stats: &FilterStats{
			RulesApplied: make(map[string]int),
		},
		debugMode: os.Getenv("MCP_FILTER_DEBUG") == "true",
	}
	
	// Enable debug logging if requested
	if f.debugMode {
		logDir := os.Getenv("MCP_FILTER_DEBUG_DIR")
		if logDir == "" {
			logDir = "/tmp"
		}
		timestamp := time.Now().Format("20060102_150405")
		logFile := filepath.Join(logDir, fmt.Sprintf("mcp_filter_%s_%s.log", mode, timestamp))
		
		if file, err := os.Create(logFile); err == nil {
			f.debugFile = file
			f.logDebug("=== Filter Debug Log Started ===")
			f.logDebug("Mode: %s", mode)
			f.logDebug("Time: %s", time.Now().Format(time.RFC3339))
			log.Printf("Filter debug logging to: %s", logFile)
		}
	}
	
	return f
}

func (f *Filter) Filter(output string) string {
	// Log input stats for debugging
	if f.debugMode {
		f.logDebug("=== Filter Input Stats ===")
		f.logDebug("Mode: %s", f.mode)
		f.logDebug("Total input length: %d chars", len(output))
		f.logDebug("Estimated input tokens: %d", len(output)/4)
		lines := strings.Count(output, "\n")
		f.logDebug("Total input lines: %d", lines)
		f.logDebug("First 1000 chars: %s", f.truncateString(output, 1000))
	}
	
	if f.mode == Verbose {
		// Even verbose mode needs limits to prevent token overflow
		return f.filterVerbose(output)
	}

	f.stats.TotalLines = 0
	f.stats.FilteredLines = 0
	f.stats.KeptLines = 0
	f.stats.SummarizedSections = 0

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Track context for better filtering decisions
	context := &FilterContext{
		InBuildPhase:     false,
		InErrorSection:   false,
		CurrentTarget:    "",
		BuildPhaseCount:  make(map[string]int),
		LastLineWasEmpty: false,
	}

	// Set limits based on mode to prevent token overflow
	maxLines := f.getMaxLinesForMode()
	maxChars := f.getMaxCharsForMode()
	totalCharsWritten := 0

	for scanner.Scan() {
		line := scanner.Text()
		f.stats.TotalLines++

		// Check both line and character limits
		if f.stats.KeptLines >= maxLines || totalCharsWritten >= maxChars {
			truncMsg := fmt.Sprintf("\n... (output truncated: %d/%d lines, %d chars max)\n",
				f.stats.KeptLines, f.stats.TotalLines, maxChars)
			result.WriteString(truncMsg)
			break
		}

		// Update context
		f.updateContext(line, context)

		// Apply filtering rules
		action := f.evaluateLine(line, context)

		switch action {
		case Keep:
			// Check char limit FIRST, before any writing
			lineToWrite := line
			cleanLine := strings.TrimSpace(line)
			
			// Handle empty lines
			if cleanLine == "" {
				// Check if even a newline would exceed limit
				if totalCharsWritten + 1 > maxChars {
					truncMsg := fmt.Sprintf("\n... (char limit reached: %d chars)\n", maxChars)
					result.WriteString(truncMsg)
					break
				}
				result.WriteString("\n")
				totalCharsWritten++
				continue // Don't count toward line limit, but we DID check char limit
			}
			
			// Strict length check for very long lines
			maxLineLength := 200
			if f.mode == Verbose {
				maxLineLength = 500
			}
			if len(lineToWrite) > maxLineLength {
				lineToWrite = lineToWrite[:maxLineLength] + "..."
			}
			
			// Check if adding this line would exceed char limit
			if totalCharsWritten + len(lineToWrite) + 1 > maxChars {
				truncMsg := fmt.Sprintf("\n... (char limit reached: %d chars)\n", maxChars)
				result.WriteString(truncMsg)
				break
			}
			
			result.WriteString(lineToWrite)
			result.WriteString("\n")
			f.stats.KeptLines++
			totalCharsWritten += len(lineToWrite) + 1
		case Remove:
			f.stats.FilteredLines++
		case Summarize:
			// For now, just keep summarized content
			result.WriteString(line)
			result.WriteString("\n")
			f.stats.KeptLines++
			f.stats.SummarizedSections++
		}
	}

	finalOutput := result.String()
	
	// Log final stats
	if f.debugMode {
		f.logDebug("=== Filter Output Stats ===")
		f.logDebug("Input lines: %d", f.stats.TotalLines)
		f.logDebug("Output lines: %d", f.stats.KeptLines)
		f.logDebug("Filtered lines: %d", f.stats.FilteredLines)
		f.logDebug("Output length: %d chars", len(finalOutput))
		f.logDebug("Estimated output tokens: %d", len(finalOutput)/4)
		if len(output) > 0 {
			reduction := (1.0 - float64(len(finalOutput))/float64(len(output)))*100
			f.logDebug("Reduction: %.1f%%", reduction)
		}
		f.logDebug("First 1000 chars of output: %s", f.truncateString(finalOutput, 1000))
		f.logDebug("=== End Filter ===")
	}
	
	return finalOutput
}

// logDebug writes to debug file if enabled
func (f *Filter) logDebug(format string, args ...interface{}) {
	if f.debugFile != nil {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(f.debugFile, "[%s] %s\n", time.Now().Format("15:04:05.000"), msg)
		f.debugFile.Sync() // Ensure it's written immediately
	}
}

// truncateString safely truncates a string for logging
func (f *Filter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Close cleans up the debug file if open
func (f *Filter) Close() {
	if f.debugFile != nil {
		f.debugFile.Close()
		f.debugFile = nil
	}
}

func (f *Filter) getMaxLinesForMode() int {
	switch f.mode {
	case Minimal:
		return 10 // Drastically reduced: ~250 tokens max
	case Standard:
		return 50 // Reduced from 200: ~1250 tokens max
	case Verbose:
		return 200 // Reduced from 800: ~5000 tokens max
	default:
		return 50
	}
}

// getMaxCharsForMode returns character limit to prevent token overflow
func (f *Filter) getMaxCharsForMode() int {
	switch f.mode {
	case Minimal:
		return 1000 // VERY strict: ~250 tokens max
	case Standard:
		return 5000 // Strict: ~1250 tokens max
	case Verbose:
		return 20000 // Limited: ~5000 tokens max
	default:
		return 5000
	}
}

func (f *Filter) filterVerbose(output string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))
	lineCount := 0
	maxLines := 800 // ~20000 tokens

	for scanner.Scan() {
		line := scanner.Text()

		// Check limit before adding line
		if lineCount >= maxLines {
			result.WriteString("\n... (output truncated at verbose mode limit)\n")
			break
		}

		// Skip only the most egregious noise even in verbose mode
		if strings.Contains(line, "-Xfrontend") ||
			strings.Contains(line, "-Xcc") ||
			strings.Contains(line, "-Xlinker") ||
			strings.Contains(line, "ClangStatCache") {
			continue
		}

		// Truncate very long lines
		if len(line) > 1000 {
			line = line[:1000] + "..."
		}

		result.WriteString(line)
		result.WriteString("\n")
		lineCount++
	}

	return result.String()
}

func (f *Filter) evaluateLine(line string, context *FilterContext) FilterAction {
	cleanLine := strings.TrimSpace(line)

	// Empty lines - always remove, we'll handle spacing in output
	if cleanLine == "" {
		return Remove
	}
	context.LastLineWasEmpty = false

	// Always keep critical information based on mode
	switch f.mode {
	case Minimal:
		return f.evaluateMinimalMode(line, context)
	case Standard:
		return f.evaluateStandardMode(line, context)
	default:
		return Keep
	}
}

func (f *Filter) evaluateMinimalMode(line string, context *FilterContext) FilterAction {
	// AGGRESSIVE FILTERING: Remove compilation noise first
	if f.isCompilationNoise(line) {
		f.recordRuleUsage("compilation-noise-removed")
		return Remove
	}

	// STRICT WHITELIST for minimal mode - only keep absolutely essential info
	keepPatterns := []string{
		"** TEST SUCCEEDED **",
		"** TEST FAILED **",
		"** BUILD SUCCEEDED **",
		"** BUILD FAILED **",
		"** CLEAN SUCCEEDED **",
		"** CLEAN FAILED **",
		": error:",
		": fatal error:",
		"Test Suite 'All tests' passed",
		"Test Suite 'All tests' failed",
		" tests passed,",
		" tests failed,",
		"Executed ", // Test summary line
	}

	for _, pattern := range keepPatterns {
		if strings.Contains(line, pattern) {
			f.recordRuleUsage("minimal-whitelist-keep")
			return Keep
		}
	}

	// Remove everything else in minimal mode
	f.recordRuleUsage("minimal-filter")
	return Remove
}

func (f *Filter) evaluateStandardMode(line string, context *FilterContext) FilterAction {
	// First remove compilation noise
	if f.isCompilationNoise(line) {
		f.recordRuleUsage("compilation-noise-removed")
		return Remove
	}

	// Remove verbose compilation details
	if f.isVerboseCompilation(line) {
		f.recordRuleUsage("compilation-filter")
		return Remove
	}

	// Remove framework noise
	if f.isFrameworkNoise(line) {
		f.recordRuleUsage("framework-filter")
		return Remove
	}

	// Now check what to keep - expanded whitelist for standard mode

	// Keep errors always
	if f.isError(line) {
		f.recordRuleUsage("error-keep")
		return Keep
	}

	// Keep warnings
	if f.isWarning(line) {
		f.recordRuleUsage("warning-keep")
		return Keep
	}

	// Keep final build/test/clean results
	if f.isBuildResult(line) || f.isTestResult(line) || f.isCleanResult(line) {
		f.recordRuleUsage("result-keep")
		return Keep
	}

	// Keep test case details in standard mode
	if strings.Contains(line, "Test Case '") || strings.Contains(line, "Test Suite '") {
		f.recordRuleUsage("test-detail-keep")
		return Keep
	}

	// Keep simplified progress indicators
	if strings.Contains(line, "Testing target") || strings.Contains(line, "Running tests") {
		f.recordRuleUsage("progress-keep")
		return Keep
	}

	// Keep important configuration
	if strings.Contains(line, "scheme:") || strings.Contains(line, "destination:") {
		f.recordRuleUsage("config-keep")
		return Keep
	}
	
	// Keep package resolution info (important for debugging)
	if strings.Contains(line, "Resolve Package") || strings.Contains(line, "Resolved source packages") {
		f.recordRuleUsage("package-keep")
		return Keep
	}
	
	// Keep command invocation
	if strings.Contains(line, "Command line invocation") || strings.Contains(line, "/xcodebuild") {
		f.recordRuleUsage("command-keep")
		return Keep
	}
	
	// Keep important metadata  
	if strings.Contains(line, "appintentsmetadataprocessor") && strings.Contains(line, "warning") {
		f.recordRuleUsage("metadata-warning-keep")
		return Keep
	}

	// Default to remove if not explicitly kept
	f.recordRuleUsage("standard-filter")
	return Remove
}

func (f *Filter) updateContext(line string, context *FilterContext) {
	cleanLine := strings.TrimSpace(line)

	// Detect build phases
	if strings.Contains(cleanLine, "=== BUILD TARGET") {
		context.InBuildPhase = true
		// Extract target name
		parts := strings.Fields(cleanLine)
		for i, part := range parts {
			if part == "TARGET" && i+1 < len(parts) {
				context.CurrentTarget = parts[i+1]
				break
			}
		}
	}

	if strings.Contains(cleanLine, "** BUILD") && (strings.Contains(cleanLine, "SUCCEEDED") || strings.Contains(cleanLine, "FAILED")) {
		context.InBuildPhase = false
	}

	// Detect error sections
	if f.isError(line) {
		context.InErrorSection = true
	}
}

// Content type detection methods
func (f *Filter) isError(line string) bool {
	errorPatterns := []string{
		": error:",
		"** BUILD FAILED **",
		"** TEST FAILED **",
		"** CLEAN FAILED **",
		"fatal error:",
		"compilation error:",
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range errorPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isWarning(line string) bool {
	warningPatterns := []string{
		": warning:",
		"warning: ",
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range warningPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isBuildResult(line string) bool {
	// Only match final build results, not intermediate build phases
	resultPatterns := []string{
		"** BUILD SUCCEEDED **",
		"** BUILD FAILED **",
	}

	for _, pattern := range resultPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isTestResult(line string) bool {
	// Be more specific to avoid matching compilation output
	testPatterns := []string{
		"** TEST SUCCEEDED **",
		"** TEST FAILED **",
		"Test Case '",    // Note the quote - actual test results have quotes
		"Test Suite '",   // Note the quote
		" passed (",      // Keep for test case results
		" failed (",      // Keep for test case results
		" tests passed,", // Summary line
		" tests failed,", // Summary line
		"Executed ",      // Test execution summary
	}

	for _, pattern := range testPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isCleanResult(line string) bool {
	cleanPatterns := []string{
		"** CLEAN SUCCEEDED **",
		"** CLEAN FAILED **",
		"Removed ",
		"Cleaning ",
	}

	for _, pattern := range cleanPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isArtifactPath(line string) bool {
	// Be more specific - only match actual artifact outputs, not compilation paths
	artifactPatterns := []string{
		"Archive path:",
		"Export path:",
		"Product Path:",
		"Exported to:",
	}

	for _, pattern := range artifactPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	// Only match .xcarchive/.ipa at end of lines (actual outputs)
	if strings.HasSuffix(strings.TrimSpace(line), ".xcarchive") ||
		strings.HasSuffix(strings.TrimSpace(line), ".ipa") {
		return true
	}

	return false
}

func (f *Filter) isProgressIndicator(line string) bool {
	progressPatterns := []string{
		"Phase ",
		"Target ",
		"Compiling ",
		"Linking ",
		"Copying ",
		"Processing ",
	}

	trimmed := strings.TrimSpace(line)
	for _, pattern := range progressPatterns {
		if strings.HasPrefix(trimmed, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isConfigurationInfo(line string) bool {
	configPatterns := []string{
		"Build settings from command line:",
		"=== CONFIGURATION:",
		"SDK:",
		"PLATFORM:",
	}

	for _, pattern := range configPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isVerboseCompilation(line string) bool {
	verbosePatterns := []string{
		"/usr/bin/clang",
		"-x objective-c",
		"-fmodules",
		"-fdiagnostics-color",
		"-target arm64",
		"-isysroot",
		"-iframework",
		"CompileC",
		"CompileSwift",
		"CompileSwiftSources",
		"Ld /",
		"GenerateDSYMFile",
		"ProcessInfoPlistFile",
		"CopySwiftLibs",
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range verbosePatterns {
		if strings.Contains(lowerLine, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

func (f *Filter) isFrameworkNoise(line string) bool {
	frameworkPatterns := []string{
		"note: Using new build system",
		"note: Planning build",
		"note: Build preparation complete",
		"note: Constructing build description",
		"note: Building targets in parallel",
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range frameworkPatterns {
		if strings.Contains(lowerLine, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (f *Filter) isCompilationNoise(line string) bool {
	// Aggressive filtering of compilation noise for minimal mode
	noisePatterns := []string{
		"SwiftDriver",
		"SwiftCompile",
		"ExecuteExternalTool",
		"ProcessInfoPlistFile",
		"ClangStatCache",
		"CopySwiftLibs",
		"CodeSign",
		"builtin-",
		"Ld /Users",
		"Ld /private",
		"/Applications/Xcode.app/Contents/Developer",
		"-Xlinker",
		"-Xfrontend",
		"-Xcc",
		"-module-name",
		"-target arm64",
		"-target x86_64",
		"CompileC",
		"CompileSwift",
		"CompileSwiftSources",
		"GenerateDSYMFile",
		"CreateBuildDirectory",
		"CreateUniversalBinary",
		"PhaseScriptExecution",
		"Touch /",
		"CpResource",
		"CopyPlistFile",
		"ProcessProductPackaging",
		"RegisterExecutionPolicyException",
		"Validate /",
		"=== BUILD TARGET",
		"=== BUILD",
		"Build settings from command line:",
		"Command line invocation:",
		"/usr/bin/xcodebuild",
		"User defaults from command line:",
		"Build description signature:",
		"Build description path:",
		"note:",
		"-I/",
		"-F/",
		"-L/",
		".swiftmodule",
		".xctest",
		".app/",
		"-emit-module",
		"-emit-dependencies",
		"-emit-objc-header",
		"-incremental",
		"-serialize-diagnostics",
		"-parseable-output",
		"cd /",
		"export ",
		"/usr/bin/clang",
		"/usr/bin/swiftc",
		"/usr/bin/swift",
		"DerivedData/",
	}

	for _, pattern := range noisePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) recordRuleUsage(ruleName string) {
	f.stats.RulesApplied[ruleName]++
}

func (f *Filter) GetStats() *FilterStats {
	return f.stats
}

func (f *Filter) ReductionPercentage() float64 {
	if f.stats.TotalLines == 0 {
		return 0
	}
	return float64(f.stats.FilteredLines) / float64(f.stats.TotalLines) * 100
}

type FilterContext struct {
	InBuildPhase     bool
	InErrorSection   bool
	CurrentTarget    string
	BuildPhaseCount  map[string]int
	LastLineWasEmpty bool
}

func getDefaultRules() []FilterRule {
	return []FilterRule{
		{
			Pattern:  regexp.MustCompile(`\*\* .+ (SUCCEEDED|FAILED) \*\*`),
			Action:   Keep,
			Priority: 100,
			Name:     "build-results",
		},
		{
			Pattern:  regexp.MustCompile(`: (error|warning):`),
			Action:   Keep,
			Priority: 95,
			Name:     "errors-warnings",
		},
		{
			Pattern:  regexp.MustCompile(`Test Case .+ (passed|failed)`),
			Action:   Keep,
			Priority: 90,
			Name:     "test-results",
		},
		{
			Pattern:  regexp.MustCompile(`/usr/bin/clang.*-x objective-c`),
			Action:   Remove,
			Priority: 80,
			Name:     "verbose-clang",
		},
		{
			Pattern:  regexp.MustCompile(`note: (Using|Planning|Building|Constructing)`),
			Action:   Remove,
			Priority: 75,
			Name:     "build-notes",
		},
	}
}
