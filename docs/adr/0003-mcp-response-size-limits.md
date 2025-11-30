# ADR-0003: MCP Response Size Limits

**Status:** Accepted
**Date:** 2025-11-30
**Deciders:** Project Team
**Tags:** mcp, performance, limits

---

## Context

MCP protocol and LLM providers have practical limits on response sizes:
- **MCP specification**: 1MB maximum message size
- **LLM context windows**: Finite (varies by provider)
- **Token costs**: Larger responses = higher costs
- **Processing time**: Larger responses = slower LLM processing

### Initial Problem

Early implementation hit issues:
- 5,000 character limit in Standard mode caused truncation
- Test failures appeared after truncation point
- Exit code 65 indicated failures, but output showed "success"
- Unable to diagnose issues without full output

### The Dilemma

- Too small → Miss critical information (failures truncated)
- Too large → Context overflow, slow responses, high costs
- No limit → Risk 1MB MCP limit, context explosion

## Decision

Implement **dual limits (lines + characters)** with **conservative defaults**:

**Character Limits (Primary):**
- Minimal: 5,000 chars (~1,250 tokens)
- Standard: 40,000 chars (~10,000 tokens)
- Verbose: 200,000 chars (~50,000 tokens)

**Line Limits (Safety Net):**
- Minimal: 100 lines
- Standard: 800 lines
- Verbose: 4,000 lines

**MCP Hard Limit:** 1MB (never exceed)

## Alternatives Considered

### Option 1: 5K Character Limit (Original)

**Pros:**
- Minimal token usage
- Fast LLM processing
- Low cost

**Cons:**
- **Truncated failures** - Critical issue
- Unreliable for large test suites
- Users forced to use verbose mode

### Option 2: Unlimited (Return Everything)

**Pros:**
- Never truncates
- Complete information

**Cons:**
- Can hit 1MB MCP limit
- Context overflow
- Excessive token costs
- Slow processing

### Option 3: 40K Characters with Safety (Chosen)

**Pros:**
- **Accommodates most test runs** (171 tests fit comfortably)
- **Room for failures** (can show 50+ failures with details)
- **Well under MCP limit** (40K << 1MB)
- **Reasonable token cost** (~10K tokens)
- **Falls back gracefully** if limit hit

**Cons:**
- May still truncate truly massive test suites (>500 tests)
- Higher tokens than minimal 5K

### Option 4: Dynamic Limits Based on Content

**Pros:**
- Adaptive to content size
- Could optimize further

**Cons:**
- Complex logic
- Unpredictable output size
- Harder to reason about

## Consequences

### Positive

- **Reliable failure reporting** - No more truncated test failures
- **Predictable limits** - Users know what to expect per mode
- **MCP compliant** - Well under 1MB limit
- **Token budgets** - ~10K tokens for standard mode (reasonable)
- **Graceful truncation** - Shows what was kept/removed

### Negative

- **8x increase** from initial 5K limit (though still 90% reduction from raw output)
- **Large test suites** may still truncate (but with warnings)
- **Not dynamic** - Fixed limits regardless of content

### Neutral

- **Mode selection** - Users must choose appropriate mode
- **Dual limits** - Both lines and characters checked

## Implementation Notes

### Limit Enforcement

```go
if totalCharsWritten + len(lineToWrite) + 1 > maxChars {
    truncMsg := fmt.Sprintf("\n... (char limit reached: %d chars)\n", maxChars)
    result.WriteString(truncMsg)
    break
}
```

### Truncation Messages

- Always indicate when truncated
- Show how much was kept vs total
- Suggest verbose mode if needed

### XCResult Safety Net

Even if output is truncated, xcresult parsing ensures:
- Test counts are accurate
- All failures are known
- Critical failures appended to output

### Token Estimation

Conservative: `tokens ≈ chars / 4`
- 40,000 chars ≈ 10,000 tokens
- 200,000 chars ≈ 50,000 tokens

Actual varies by content (code vs prose).

### Line Length Limits

Individual lines capped at:
- Standard: 200 characters (prevent single-line overflow)
- Verbose: 500 characters

Very long lines get truncated with "...".

## Testing Strategy

- Test with 171-test suite (real-world case)
- Test with 389-test suite (larger case)
- Test with massive output (simulate 1000+ tests)
- Verify truncation messages appear
- Verify failures always visible

## References

- Related: ADR-0002 (Output Filtering Strategy)
- Original issue: 5K limit truncating test failures
- MCP Specification: https://spec.modelcontextprotocol.io/
- Implementation: `internal/filter/filter.go:276-288`
