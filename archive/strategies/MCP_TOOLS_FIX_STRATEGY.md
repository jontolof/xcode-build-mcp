# MCP Tools Fix Strategy

## Executive Summary
Three MCP tools are failing in production:
1. **launch_app** - Error -32603: Tool execution failed
2. **install_app** - Error -32603: Tool execution failed  
3. **capture_logs** - Tool stalls/hangs during execution

## Root Cause Analysis

### 1. launch_app Failure
**Error**: MCP error -32603 when launching with bundle_id and udid

**Likely Causes**:
- App not installed on the simulator (most likely)
- Bundle ID mismatch (com.centcode.LeMieLingueApp vs actual bundle ID)
- Simulator not booted
- Simulator in inconsistent state

**Code Location**: `internal/tools/launch.go:328-334`
```go
if strings.Contains(errorOutput, "App is not installed") {
    return result, fmt.Errorf("app with bundle ID %s is not installed on the device", params.BundleID)
}
```

### 2. install_app Failure  
**Error**: MCP error -32603 when app_path is "."

**Root Cause**: Invalid app path validation
- User passed "." as app_path (current directory)
- Tool expects a .app bundle path, not a directory
- Validation fails at `internal/tools/install.go:79-81`

**Code Issue**:
```go
if !isValidAppPath(params.AppPath) {
    return "", fmt.Errorf("app path does not exist: %s", params.AppPath)
}
```

### 3. capture_logs Stalling
**Symptom**: Tool hangs, doesn't return, shows "getpwuid_r did not find a match for uid 501"

**Root Causes**:
1. **Process name mismatch**: Looking for "LeMieLingueApp" but actual process name might be different
2. **log stream never ends**: The tool uses `log stream` which runs indefinitely
3. **No output buffering**: Scanner blocks waiting for lines that never come
4. **Context timeout not properly handled**: Tool may not respect timeout properly

**Code Location**: `internal/tools/logs.go:219-234`
```go
for scanner.Scan() && lineCount < params.MaxLines {
    // This loops forever if no matching logs come in
}
```

## Fix Strategy

### Phase 1: Immediate Fixes (High Priority)

#### Fix 1.1: Better Error Messages for launch_app
```go
// internal/tools/launch.go - Add pre-flight checks
func (t *LaunchAppTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // ... existing code ...
    
    // Add: Check if app is installed before attempting launch
    installedApps, err := t.getInstalledApps(ctx, targetUDID)
    if err == nil && !contains(installedApps, params.BundleID) {
        return "", fmt.Errorf("app '%s' is not installed on device %s. Install it first using install_app tool", params.BundleID, targetUDID)
    }
    
    // Add: Check if simulator is booted
    if !t.isSimulatorBooted(ctx, targetUDID) {
        return "", fmt.Errorf("simulator %s is not booted. Boot it first using simulator_control tool", targetUDID)
    }
}
```

#### Fix 1.2: Smart Path Resolution for install_app
```go
// internal/tools/install.go - Improve path handling
func (t *InstallAppTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // ... existing parsing ...
    
    // If path is ".", look for .app bundles in build directories
    if params.AppPath == "." || params.AppPath == "./" {
        appPath, err := t.findAppBundle(ctx, ".")
        if err != nil {
            return "", fmt.Errorf("no .app bundle found in current directory. Specify the exact path to the .app bundle")
        }
        params.AppPath = appPath
    }
    
    // Validate it's actually a .app bundle
    if !strings.HasSuffix(params.AppPath, ".app") {
        return "", fmt.Errorf("invalid app path: %s. Must be a .app bundle, not a directory", params.AppPath)
    }
}

// Helper to find .app bundles
func (t *InstallAppTool) findAppBundle(ctx context.Context, dir string) (string, error) {
    // Look in common build output directories
    searchPaths := []string{
        "build/Debug-iphonesimulator/*.app",
        "build/Release-iphonesimulator/*.app",
        "DerivedData/*/Build/Products/*/*.app",
        "*.app",
    }
    
    for _, pattern := range searchPaths {
        matches, _ := filepath.Glob(filepath.Join(dir, pattern))
        if len(matches) > 0 {
            return matches[0], nil
        }
    }
    
    return "", fmt.Errorf("no .app bundle found")
}
```

#### Fix 1.3: Prevent capture_logs from Hanging
```go
// internal/tools/logs.go - Add proper timeout and buffering
func (t *CaptureLogs) captureLogs(ctx context.Context, params *types.LogCaptureParams) (*types.LogCaptureResult, error) {
    // ... existing setup ...
    
    // Use 'log show' with --last parameter instead of 'log stream'
    args := []string{"simctl", "spawn", params.UDID, "log", "show", "--last", "2m"}
    
    // Alternative: If we must use stream, add --timeout
    // args = append(args, "--timeout", fmt.Sprintf("%ds", params.TimeoutSecs))
    
    // Add buffered reading with timeout
    done := make(chan bool)
    go func() {
        for scanner.Scan() && lineCount < params.MaxLines {
            select {
            case <-cmdCtx.Done():
                done <- true
                return
            default:
                line := scanner.Text()
                // Process line...
            }
        }
        done <- true
    }()
    
    // Wait for completion or timeout
    select {
    case <-done:
        // Finished normally
    case <-cmdCtx.Done():
        // Timeout or cancelled
        cmd.Process.Kill()
        return &types.LogCaptureResult{
            Success: false,
            Message: "Log capture timed out",
        }, nil
    }
}
```

### Phase 2: Robustness Improvements

#### Fix 2.1: Add Retry Logic
```go
// internal/xcode/executor.go - Add retry for transient failures
func (e *Executor) ExecuteWithRetry(ctx context.Context, args []string, maxRetries int) (*CommandResult, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        result, err := e.ExecuteCommand(ctx, args)
        if err == nil && result.Success() {
            return result, nil
        }
        lastErr = err
        
        // Check if retryable
        if !isRetryableError(err) {
            break
        }
        
        time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
    }
    return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
```

#### Fix 2.2: Add Debug Logging
```go
// Add MCP_DEBUG environment variable support
func (t *LaunchAppTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    if os.Getenv("MCP_DEBUG") == "true" {
        t.logger.Printf("DEBUG: launch_app called with args: %+v", args)
    }
    // ... rest of execution
    if err != nil && os.Getenv("MCP_DEBUG") == "true" {
        t.logger.Printf("DEBUG: launch_app error: %v", err)
    }
}
```

### Phase 3: User Experience Improvements

#### Fix 3.1: Better Error Messages
```go
// Provide actionable error messages
errors.New("App not installed. Run: install_app with app_path='/path/to/YourApp.app'")
errors.New("Simulator not booted. Run: simulator_control with action='boot'")
errors.New("No .app bundle found. Build your project first with xcode_build")
```

#### Fix 3.2: Auto-Recovery
```go
// Auto-boot simulator if not booted
if !isBooted {
    t.logger.Printf("Auto-booting simulator %s", udid)
    if err := t.bootSimulator(ctx, udid); err != nil {
        return "", fmt.Errorf("failed to auto-boot simulator: %w", err)
    }
}
```

## Implementation Priority

1. **CRITICAL - Fix capture_logs hanging** (Fix 1.3)
   - Switch from `log stream` to `log show`
   - Add proper timeout handling
   - Prevents tool from hanging indefinitely

2. **HIGH - Fix install_app path handling** (Fix 1.2)
   - Add .app bundle discovery
   - Better error messages
   - Handle "." path correctly

3. **HIGH - Fix launch_app error handling** (Fix 1.1)
   - Pre-flight checks
   - Clear error messages
   - Guide user to solution

4. **MEDIUM - Add debug logging** (Fix 2.2)
   - Help diagnose future issues
   - MCP_DEBUG environment variable

5. **LOW - Add retry logic** (Fix 2.1)
   - Handle transient failures
   - Improve reliability

## Testing Plan

### Test Cases
1. **launch_app**:
   - Test with uninstalled app → Should give clear error
   - Test with wrong bundle ID → Should give clear error
   - Test with unbooted simulator → Should auto-boot or give clear error

2. **install_app**:
   - Test with "." path → Should find .app bundle
   - Test with directory path → Should give clear error
   - Test with valid .app path → Should succeed

3. **capture_logs**:
   - Test with timeout → Should return within timeout
   - Test with no matching logs → Should return empty, not hang
   - Test with bundle filter → Should only return matching logs

## Rollback Plan
If fixes cause new issues:
1. Revert to previous version
2. Add feature flags for new behavior
3. Test thoroughly in staging environment

## Success Metrics
- Zero hanging tools
- Clear, actionable error messages
- 90% reduction in "Tool execution failed" errors
- All tools complete within timeout period