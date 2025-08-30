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
	errorRegex   = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(error|warning):\s*(.+)$`)
	errorRegex2  = regexp.MustCompile(`^(.+?):\s*(error|warning):\s*(.+)$`)
	
	// Build success/failure
	buildSuccessRegex = regexp.MustCompile(`\*\* BUILD SUCCEEDED \*\*`)
	buildFailedRegex  = regexp.MustCompile(`\*\* BUILD FAILED \*\*`)
	
	// Test results
	testSuccessRegex = regexp.MustCompile(`\*\* TEST SUCCEEDED \*\*`)
	testFailedRegex  = regexp.MustCompile(`\*\* TEST FAILED \*\*`)
	testCaseRegex    = regexp.MustCompile(`Test Case '(.+?)' (passed|failed|started) \((\d+\.\d+) seconds\)`)
	
	// Archive/export paths
	archiveRegex = regexp.MustCompile(`Archive path: (.+\.xcarchive)`)
	exportRegex  = regexp.MustCompile(`Export path: (.+)`)
	
	// Clean results
	cleanSuccessRegex = regexp.MustCompile(`\*\* CLEAN SUCCEEDED \*\*`)
	cleanFailedRegex  = regexp.MustCompile(`\*\* CLEAN FAILED \*\*`)
)

func (p *Parser) ParseBuildOutput(output string) *types.BuildResult {
	result := &types.BuildResult{
		Output:    output,
		Errors:    []types.BuildError{},
		Warnings:  []types.BuildWarning{},
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
			TestResults: []types.TestCase{},
			FailedTestsDetails: []types.TestCase{},
		},
	}
	
	var currentTest *types.TestCase
	
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
	
	return result
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