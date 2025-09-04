# Token Overflow Fix Strategy

## Problem Analysis

Real-world usage shows that even with "minimal" filtering, xcode_build output exceeds token limits:
- Default mode: 708,095 tokens (28x over limit!)
- Minimal mode: 35,829 tokens (still 1.4x over limit!)
- Maximum allowed: 25,000 tokens

## Root Causes

1. **Line limits too high**: Current limits (20/200/800 lines) assume ~25 tokens per line, but actual output has much more
2. **Insufficient filtering**: Not aggressive enough in removing compilation details
3. **Missing character limit**: No total character count limit as safety net

## Solution Strategy

### 1. Drastically Reduce Line Limits
```go
// Current limits are way too high
case Minimal:
    return 10   // Was 20 - target ~250 tokens
case Standard:  
    return 50   // Was 200 - target ~1250 tokens  
case Verbose:
    return 200  // Was 800 - target ~5000 tokens
```

### 2. Add Character-Based Limits
```go
// Add total character limit as safety net
const (
    MinimalMaxChars  = 2000   // ~500 tokens
    StandardMaxChars = 10000  // ~2500 tokens
    VerboseMaxChars  = 40000  // ~10000 tokens
)
```

### 3. More Aggressive Filtering Rules

For minimal mode, ONLY keep:
- Build status (SUCCEEDED/FAILED)
- Error messages
- Critical warnings
- Final artifact path

Remove ALL:
- Compilation details
- Framework messages
- File processing logs
- Dependency resolution
- Progress indicators

### 4. Smart Summarization

Instead of showing all errors, summarize:
```
Build FAILED with 5 errors:
- ContentView.swift:16: Cannot find 'nonExistentFunction'
- AppDelegate.swift:42: Type mismatch
... (3 more errors)
```

### 5. Progressive Disclosure

Add a `truncation_info` field to response:
```json
{
  "output": "...",
  "truncation_info": {
    "total_lines": 5000,
    "shown_lines": 10,
    "total_chars": 250000,
    "shown_chars": 2000,
    "omitted_errors": 15
  }
}
```

## Implementation Steps

1. Update filter.go with new limits
2. Add character counting to filter
3. Enhance minimal mode filtering rules
4. Add truncation info to response
5. Test with large projects

## Expected Results

- Minimal: <1,000 tokens (currently 35,829)
- Standard: <5,000 tokens  
- Verbose: <20,000 tokens (currently 708,095)

This ensures all modes stay well under the 25,000 token limit.