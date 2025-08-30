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
	TotalLines    int
	FilteredLines int
	KeptLines     int
	SummarizedSections int
	RulesApplied  map[string]int
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
		return output
	}

	f.stats.TotalLines = 0
	f.stats.FilteredLines = 0
	f.stats.KeptLines = 0
	f.stats.SummarizedSections = 0

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))
	
	// Track context for better filtering decisions
	context := &FilterContext{
		InBuildPhase:    false,
		InErrorSection:  false,
		CurrentTarget:   "",
		BuildPhaseCount: make(map[string]int),
	}

	for scanner.Scan() {
		line := scanner.Text()
		f.stats.TotalLines++
		
		// Update context
		f.updateContext(line, context)
		
		// Apply filtering rules
		action := f.evaluateLine(line, context)
		
		switch action {
		case Keep:
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

func (f *Filter) evaluateLine(line string, context *FilterContext) FilterAction {
	cleanLine := strings.TrimSpace(line)
	
	// Empty lines - keep in minimal spacing
	if cleanLine == "" {
		return Keep
	}
	
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
	
	// Keep errors always
	if f.isError(line) {
		f.recordRuleUsage("error-keep")
		return Keep
	}
	
	// Keep build results
	if f.isBuildResult(line) {
		f.recordRuleUsage("build-result-keep")
		return Keep
	}
	
	// Keep test results
	if f.isTestResult(line) {
		f.recordRuleUsage("test-result-keep")
		return Keep
	}
	
	// Keep clean results
	if f.isCleanResult(line) {
		f.recordRuleUsage("clean-result-keep")
		return Keep
	}
	
	// Keep artifact paths
	if f.isArtifactPath(line) {
		f.recordRuleUsage("artifact-keep")
		return Keep
	}
	
	// Remove everything else in minimal mode
	f.recordRuleUsage("minimal-filter")
	return Remove
}

func (f *Filter) evaluateStandardMode(line string, context *FilterContext) FilterAction {
	
	// Keep all minimal mode content
	if action := f.evaluateMinimalMode(line, context); action == Keep {
		return Keep
	}
	
	// Additionally keep warnings
	if f.isWarning(line) {
		f.recordRuleUsage("warning-keep")
		return Keep
	}
	
	// Keep progress indicators
	if f.isProgressIndicator(line) {
		f.recordRuleUsage("progress-keep")
		return Keep
	}
	
	// Keep configuration info
	if f.isConfigurationInfo(line) {
		f.recordRuleUsage("config-keep")
		return Keep
	}
	
	// Filter out verbose compilation details
	if f.isVerboseCompilation(line) {
		f.recordRuleUsage("compilation-filter")
		return Remove
	}
	
	// Filter out framework noise
	if f.isFrameworkNoise(line) {
		f.recordRuleUsage("framework-filter")
		return Remove
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
	resultPatterns := []string{
		"** BUILD SUCCEEDED **",
		"** BUILD FAILED **",
		"BUILD TARGET",
		"=== BUILD",
	}
	
	for _, pattern := range resultPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (f *Filter) isTestResult(line string) bool {
	testPatterns := []string{
		"** TEST SUCCEEDED **",
		"** TEST FAILED **",
		"Test Case ",
		"Test Suite ",
		" passed (",
		" failed (",
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
	artifactPatterns := []string{
		"Archive path:",
		"Export path:",
		".xcarchive",
		".app",
		".ipa",
	}
	
	for _, pattern := range artifactPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
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
	}
	
	lowerLine := strings.ToLower(line)
	verboseCount := 0
	for _, pattern := range verbosePatterns {
		if strings.Contains(lowerLine, pattern) {
			verboseCount++
		}
	}
	
	// If line contains multiple verbose compilation flags, filter it
	return verboseCount >= 2
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
	InBuildPhase    bool
	InErrorSection  bool
	CurrentTarget   string
	BuildPhaseCount map[string]int
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