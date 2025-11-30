# Token Optimization Analysis - Test Output

## Current Status (MCP_USE_LOGS_004.md)

### What's Working ✅
- **Accurate test detection**: 8 failures detected (matches Xcode!)
- **XCResult parsing**: Now working with `--legacy` flag
- **Character limits**: Increased to 40,000 chars
- **Structured data**: Complete failure details in JSON
- **90% reduction**: From ~100K tokens to ~10.4K tokens

### The Warning
```
⚠ Large MCP response (~10.4k tokens), this can fill up context quickly
```

## Token Breakdown

### Current Implementation
```
Test Run: 389 tests, 8 failures
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Structured Data:        ~400 tokens
  - test_summary
  - test_bundles
  - crash_indicators
  - failed_tests_details (8 failures)

Filtered Output:        ~10,000 tokens
  - All 377 passing tests shown
  - All 8 failing tests shown
  - Truncated at 40K chars
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total:                  ~10,400 tokens
```

### MCP Limits
- **Hard limit**: 1 MB (~250,000 tokens)
- **Current usage**: 10,400 tokens (4.2% of limit)
- **Status**: ✅ Well within limits

## The Problem

**Common Case (97% pass rate):**
- User doesn't need 377 passing test details
- Only needs: summary + 8 failure details
- Current token cost: 10,400 tokens
- **Optimal token cost: ~2,000 tokens**

**Edge Case (all tests fail):**
- User needs all failure details
- Current approach is correct
- Token cost: appropriate for the situation

## Proposed Optimization Strategy

### Smart Filtering Based on Test Results

```go
func (f *Filter) smartFilterTestOutput(output string, testSummary TestSummary) string {
    if testSummary.FailedTests == 0 {
        // All tests passed - minimal output
        return f.filterPassingTestsOnly(output)
    } else {
        // Tests failed - focus on failures
        return f.filterWithFailureFocus(output, testSummary)
    }
}
```

### Strategy 1: Failure-Focused Filtering (Recommended)

**When tests fail:**
1. Show summary (test counts, duration)
2. Show **only failed tests** in detail
3. Show passing test count, but not individual passing tests
4. Show error messages and stack traces for failures

**Token savings:**
- Before: 10,400 tokens (all 389 tests)
- After: ~2,500 tokens (summary + 8 failures)
- **Reduction: 76%** while keeping critical info

**Example output:**
```
Test Suite 'All tests' started
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Summary:
  Total: 389 tests
  Passed: 377 tests ✅ (details omitted, use --verbose to see all)
  Failed: 8 tests ❌

Failed Tests:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. RecordingViewTests.recordingViewButtonHasGestureRecognizer
   Error: view(RecordingView.self).vStack().group(5) found VStack<...> instead of Group
   File: InspectableView.swift:53
   Duration: 0.001s

2. RecordingViewTests.recordingViewDisplaysModeSelectionCorrectly
   Error: view(RecordingView.self).vStack().vStack(4).picker(0) found HStack<...> instead of Picker
   File: InspectableView.swift:53
   Duration: 0.001s

... (6 more failures)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Duration: 2m 4.7s
Exit Code: 65
```

### Strategy 2: Progressive Disclosure (Alternative)

**Initial response:**
- Summary only (~500 tokens)
- Structured data with all details
- User can request verbose output if needed

**Follow-up (if user asks):**
- Full detailed output

**Pros:**
- Minimal initial tokens
- Details available on demand

**Cons:**
- Requires additional interaction
- More complex implementation
- User might not know details are available

### Strategy 3: Mode-Based Optimization (Hybrid)

**Minimal mode** (~500 tokens):
- Summary only
- Failed test count and names
- No passing test details

**Standard mode** (~2,500 tokens):
- Summary
- Failed test details (full)
- Passing test count only

**Verbose mode** (~10,000-40,000 tokens):
- Everything (current behavior)

## Recommendation: Implement Strategy 1

### Why Failure-Focused Filtering?

1. **Matches user intent**: When tests fail, users care about WHY, not the 377 that passed
2. **Optimal tokens**: ~2,500 tokens for typical failure case vs 10,400 current
3. **No information loss**: All critical data still present
4. **Simple to implement**: Single filter change, no API changes
5. **Works with current modes**: Enhance standard/minimal, keep verbose as-is

### Implementation Plan

```go
// internal/filter/filter.go

func (f *Filter) Filter(output string, testSummary *types.TestSummary) string {
    // Existing logic...

    // If we have test summary metadata, use smart filtering
    if testSummary != nil {
        return f.smartFilterForTests(output, testSummary)
    }

    // Fall back to existing behavior for builds, etc.
    return f.filterStandard(output)
}

func (f *Filter) smartFilterForTests(output string, summary *types.TestSummary) string {
    if f.mode == Verbose {
        return f.filterVerbose(output) // Existing behavior
    }

    if summary.FailedTests == 0 {
        // All passed - super concise
        return f.filterAllPassed(output, summary)
    } else {
        // Some failed - focus on failures
        return f.filterWithFailures(output, summary)
    }
}

func (f *Filter) filterWithFailures(output string, summary *types.TestSummary) string {
    var result strings.Builder
    scanner := newSafeScanner(strings.NewReader(output))

    inFailedTest := false
    currentTestName := ""

    for scanner.Scan() {
        line := scanner.Text()

        // Always keep: summary lines, failures, errors
        if f.isSummaryLine(line) || f.isFailureLine(line) {
            result.WriteString(line + "\n")
            continue
        }

        // Track if we're in a failed test block
        if strings.Contains(line, "Test Case '") {
            testName := f.extractTestName(line)
            // Check if this test is in failed list
            inFailedTest = f.isFailedTest(testName, summary.FailedTestsDetails)
            currentTestName = testName
        }

        // Keep lines from failed tests
        if inFailedTest {
            result.WriteString(line + "\n")
        }

        // Skip passing test details (unless in failed test context)
        if strings.Contains(line, "passed (") && !inFailedTest {
            // Passing test - skip unless it's context for a failure
            continue
        }
    }

    return result.String()
}
```

### Token Comparison

**Test Run: 389 tests, 8 failures**

| Approach | Tokens | Reduction | Information Loss |
|----------|--------|-----------|------------------|
| **Current (all tests)** | 10,400 | 90% vs raw | None |
| **Failure-focused** | 2,500 | 97.5% vs raw | Passing test details (not needed) |
| **Minimal (summary only)** | 500 | 99.5% vs raw | Failure details (needed!) |
| **Raw xcodebuild** | 100,000+ | - | - |

### Benefits

1. **76% token reduction** for common failure case
2. **No information loss** - all failure details preserved
3. **Better UX** - users see what matters
4. **Scalable** - works for 8 failures or 80 failures
5. **Maintains 90%+ reduction** vs raw output

## Alternative: Accept Current State

### Pros of Current Approach
- Simple, already implemented
- All information visible
- 90% reduction already excellent
- Well under MCP limits (4% usage)

### Cons of Current Approach
- Wastes tokens on passing test details
- Claude Code warning confuses users
- Repeated test runs fill context faster
- Not optimized for common case

## Decision Matrix

| Criteria | Current | Failure-Focused | Minimal |
|----------|---------|-----------------|---------|
| **Captures failures** | ✅ Yes | ✅ Yes | ❌ Names only |
| **Token efficient** | ⚠️ 10.4k | ✅ 2.5k | ✅ 0.5k |
| **User-friendly** | ⚠️ Verbose | ✅ Focused | ❌ Too terse |
| **Implementation** | ✅ Done | ⚠️ Medium | ✅ Easy |
| **Scalability** | ❌ Linear growth | ✅ Scales with failures | ✅ Constant |

## Recommendation

**Implement Failure-Focused Filtering (Strategy 1)**

### Why?
1. **Best balance** of information density and token efficiency
2. **User intent alignment** - show what failed, not what passed
3. **76% further reduction** while keeping critical info
4. **No breaking changes** - enhance existing modes
5. **Addresses Claude Code warning** - 2.5k tokens is "small"

### When to implement?
- **Phase 1 (current)**: Ship current fixes - they're working! ✅
- **Phase 2 (next)**: Add failure-focused filtering
- **Optional**: Make it configurable via environment variable

### Environment variable
```bash
export MCP_TEST_OUTPUT_FOCUS=failures  # New default
export MCP_TEST_OUTPUT_FOCUS=all       # Current behavior
export MCP_TEST_OUTPUT_FOCUS=summary   # Minimal
```

## Conclusion

**Current state: Success! ✅**
- All 8 failures correctly detected
- 90% token reduction achieved
- Well within MCP limits

**Future optimization: Failure-focused filtering**
- Would reduce to 2.5k tokens (76% further reduction)
- Better UX - show what matters
- No information loss
- Simple to implement

**Immediate action: Ship current fixes**
- They're working correctly
- Address real user pain (missed failures)
- Token warning is conservative, not blocking
