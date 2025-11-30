# ADR-0002: Output Filtering Strategy

**Status:** Accepted
**Date:** 2025-11-30
**Deciders:** Project Team
**Tags:** performance, token-optimization, mcp

---

## Context

Xcodebuild produces extremely verbose output (100K-200K characters for a typical test run), which when returned through MCP:
- Consumes massive amounts of LLM context (~50K tokens)
- Causes context overflow in extended sessions
- Includes 90%+ noise (compilation details, framework messages)
- Slows down LLM response times
- Makes it hard to identify actual failures

### The Challenge

Balance between:
- **Information completeness** - Don't hide failures
- **Token efficiency** - Stay within reasonable context budgets
- **Response quality** - Provide actionable information

## Decision

Implement **two-pass failure-aware filtering** with configurable output modes:

1. **Pass 1**: Identify critical lines (failures, errors, summaries)
2. **Pass 2**: Output only critical content + context
3. **XCResult parsing**: Parse structured .xcresult bundle as authoritative source
4. **Silent failure handling**: Append xcresult failures if missing from filtered text

### Output Modes

- **Minimal** (~1,250 tokens): Errors + final result only
- **Standard** (~10,000 tokens): Errors + warnings + test summaries
- **Verbose** (~50,000 tokens): Everything except worst noise

## Alternatives Considered

### Option 1: No Filtering (Return Everything)

**Pros:**
- Complete information
- No risk of hiding failures
- Simple implementation

**Cons:**
- 50K+ tokens per test run
- Context overflow in multi-operation sessions
- LLMs struggle with noise
- Slow processing

### Option 2: Fixed Line Limit (e.g., first 1000 lines)

**Pros:**
- Simple to implement
- Predictable output size
- Fast

**Cons:**
- May truncate critical failures
- No semantic understanding
- Failures often at end of output

### Option 3: Keyword-Based Filtering

**Pros:**
- Moderately effective
- Fast pattern matching

**Cons:**
- Misses context around failures
- Can't detect silent failures
- Brittle patterns

### Option 4: Two-Pass Failure-Aware Filtering (Chosen)

**Pros:**
- **90%+ reduction** while preserving failures
- **Guarantees failure visibility** (critical lines always kept)
- **Configurable** (minimal/standard/verbose)
- **XCResult fallback** (catches silent failures)

**Cons:**
- More complex implementation
- Two passes over output
- Pattern matching maintenance

## Consequences

### Positive

- **90-99% output reduction** achieved
- **Zero missed failures** (xcresult + text parsing)
- **Fast LLM responses** (10K vs 50K+ tokens)
- **Extended sessions possible** (context doesn't overflow)
- **Clear failure presentation** (noise removed)

### Negative

- **Pattern matching brittleness** - Xcode output format changes require updates
- **XCResult dependency** - Requires temporary .xcresult bundle generation
- **Maintenance overhead** - Filter patterns need updates for new Xcode versions

### Neutral

- **Character + line limits** - Dual limits prevent overflow
- **Mode selection required** - Users must choose appropriate verbosity

## Implementation Notes

### Filtering Pipeline

```
Raw xcodebuild output (100K-200K chars)
    ↓
Pass 1: Identify critical lines
    ↓
Pass 2: Output only critical + context
    ↓
Check for silent failures
    ↓
Append xcresult failures if missing
    ↓
Final filtered output (5K-40K chars)
```

### Character Limits

| Mode | Line Limit | Char Limit | Est. Tokens |
|------|-----------|------------|-------------|
| Minimal | 100 | 5,000 | ~1,250 |
| Standard | 800 | 40,000 | ~10,000 |
| Verbose | 4,000 | 200,000 | ~50,000 |

### Critical Line Patterns

Always kept:
- `** TEST FAILED **`
- `** BUILD FAILED **`
- `: error:`
- `: fatal error:`
- Test failure lines
- Final test summaries

Always removed:
- Compilation commands
- Framework loading
- SwiftDriver noise
- Code signing details

### Silent Failure Handling

When xcresult shows failures but filtered output doesn't:
1. Replace misleading "All tests passed" with accurate counts
2. Append "=== Failed Tests ===" section
3. Include test names from xcresult

## References

- Related: ADR-0003 (MCP Response Size Limits)
- Related: ADR-0001 (Crash Detection)
- Implementation: `internal/filter/filter.go`
- Test output parsing: `internal/xcode/parser.go`
- XCResult parsing: `internal/xcode/xcresult.go`
