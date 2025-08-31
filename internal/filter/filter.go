package filter

import (
	"bufio"
	"regexp"
	"strings"
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
	rules []FilterRule
	mode  OutputMode
	stats *FilterStats
}

type FilterStats struct {
	TotalLines         int
	FilteredLines      int
	KeptLines          int
	SummarizedSections int
	RulesApplied       map[string]int
}

func NewFilter(mode OutputMode) *Filter {
	return &Filter{
		rules: getDefaultRules(),
		mode:  mode,
		stats: &FilterStats{
			RulesApplied: make(map[string]int),
		},
	}
}

func (f *Filter) Filter(output string) string {
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

	// Set line limits based on mode to prevent token overflow
	maxLines := f.getMaxLinesForMode()

	for scanner.Scan() {
		line := scanner.Text()
		f.stats.TotalLines++

		// Hard limit to prevent token overflow
		if f.stats.KeptLines >= maxLines {
			result.WriteString("\n... (output truncated at ")
			result.WriteString(string(f.mode))
			result.WriteString(" mode limit)\n")
			break
		}

		// Update context
		f.updateContext(line, context)

		// Apply filtering rules
		action := f.evaluateLine(line, context)

		switch action {
		case Keep:
			// Additional length check for very long lines
			if len(line) > 500 && f.mode != Verbose {
				line = line[:500] + "..."
			}
			result.WriteString(line)
			result.WriteString("\n")
			f.stats.KeptLines++
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

	return result.String()
}

func (f *Filter) getMaxLinesForMode() int {
	switch f.mode {
	case Minimal:
		return 20 // ~500 tokens max
	case Standard:
		return 200 // ~5000 tokens max
	case Verbose:
		return 800 // ~20000 tokens max
	default:
		return 200
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

	// Empty lines - remove in minimal mode, keep sparingly in other modes
	if cleanLine == "" {
		if f.mode == Minimal {
			return Remove
		}
		// In standard/verbose mode, keep some spacing but not all
		if context.LastLineWasEmpty {
			return Remove // Don't keep consecutive empty lines
		}
		context.LastLineWasEmpty = true
		return Keep
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
