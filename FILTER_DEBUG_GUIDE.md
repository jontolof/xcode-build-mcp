# Filter Debug Guide

## Problem
The MCP server is producing outputs that exceed token limits (25,000 tokens max), even with filtering enabled:
- Standard mode: 34,815 tokens (1.4x over limit!)
- Default mode: 708,095 tokens (28x over limit!)

## Debug Mode Setup

Enable filter debug logging to capture raw xcodebuild output and analyze why filtering isn't working:

### 1. Enable Debug Mode

Set environment variables before running your MCP client:

```bash
# Enable filter debug logging
export MCP_FILTER_DEBUG=true

# Optional: Set custom log directory (defaults to /tmp)
export MCP_FILTER_DEBUG_DIR=/path/to/logs

# Run your MCP client as usual
```

### 2. Run Your Build

Execute the xcode_build command that's causing token overflow:

```json
{
  "tool": "xcode_build",
  "parameters": {
    "project_path": ".",
    "project": "YourProject.xcodeproj",
    "scheme": "YourScheme",
    "output_mode": "minimal"  // or "standard"
  }
}
```

### 3. Check Debug Logs

Debug logs will be created in the specified directory with timestamps:
- `/tmp/mcp_filter_minimal_20240115_143022.log`
- `/tmp/mcp_filter_standard_20240115_143025.log`

Each log contains:
- **Input Stats**: Original size, line count, estimated tokens
- **Filter Process**: Which lines were kept/removed and why
- **Output Stats**: Final size, reduction percentage
- **First 1000 chars**: Sample of input/output for verification

### 4. Analyze the Logs

Look for:
1. **Input size**: Is the raw xcodebuild output massive?
2. **Reduction percentage**: Is the filter actually reducing output?
3. **Kept lines**: Are too many lines being kept?
4. **Character count**: Are individual lines too long?

## Example Debug Output

```
[14:30:22.123] === Filter Input Stats ===
[14:30:22.123] Mode: standard
[14:30:22.123] Total input length: 2835260 chars
[14:30:22.123] Estimated input tokens: 708815
[14:30:22.123] Total input lines: 15234
[14:30:22.124] First 1000 chars: CompileSwift normal arm64...

[14:30:22.456] === Filter Output Stats ===
[14:30:22.456] Input lines: 15234
[14:30:22.456] Output lines: 50
[14:30:22.456] Filtered lines: 15184
[14:30:22.456] Output length: 139260 chars
[14:30:22.456] Estimated output tokens: 34815
[14:30:22.456] Reduction: 95.1%
```

## Troubleshooting

### If reduction is low (<90%):
- Filter rules may not be aggressive enough
- Check which patterns are being kept unnecessarily
- Look for compilation noise that's not being filtered

### If output is still too large after filtering:
- Character limits may be too high
- Individual lines may be extremely long
- Consider using "minimal" mode instead of "standard"

### If filter crashes or hangs:
- Check for infinite loops in the log
- Look for memory issues with very large inputs
- Verify regex patterns aren't catastrophically backtracking

## Sharing Debug Logs

When reporting issues, please share:
1. The debug log file
2. Your MCP client version
3. The xcode_build parameters used
4. The actual token count error message

This helps identify why the filter isn't reducing output sufficiently.