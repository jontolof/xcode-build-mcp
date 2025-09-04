# Xcode Build MCP Testing Guide

This guide provides comprehensive testing instructions for the xcode-build-mcp server. Follow these steps to test all 14 tools across different scenarios and output modes.

## Prerequisites

1. **Build the server**:
   ```bash
   go build -o xcode-build-mcp cmd/server/main.go
   ```

2. **Enable debug logging** (recommended):
   ```bash
   export MCP_LOG_LEVEL=debug
   export MCP_FILTER_DEBUG=true
   export MCP_FILTER_DEBUG_DIR=/tmp/mcp_debug
   mkdir -p /tmp/mcp_debug
   ```

3. **Have test projects ready**:
   - iOS project (.xcodeproj)
   - Workspace (.xcworkspace) 
   - Running iOS Simulator
   - Sample .app bundle for installation

## Testing Structure

Each test includes:
- **Purpose**: What the test validates
- **Command**: Exact MCP tool call
- **Expected Result**: What should happen
- **Debug Files**: Where to find logs
- **Failure Analysis**: Common issues to check

---

## 1. Core Build Tools

### 1.1 Universal Build Tool (`xcode_build`)

#### Test 1.1.1: Minimal Output Mode - Success Case
**Purpose**: Verify minimal filtering shows only essential information

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "output_mode": "minimal"
    }
  }
}
```

**Expected Result**:
- Success status clearly indicated
- Build summary present
- Filtering stats show high reduction (>85%)
- Critical build status preserved

**Debug Files**: `/tmp/mcp_debug/mcp_filter_minimal_*.log`

#### Test 1.1.2: Standard Output Mode - Success Case
**Purpose**: Verify standard filtering includes warnings and progress

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "output_mode": "standard"
    }
  }
}
```

**Expected Result**:
- Success status and duration
- Warnings included if any
- Build progress indicators
- Moderate filtering reduction (70-85%)

#### Test 1.1.3: Verbose Output Mode - Success Case  
**Purpose**: Verify verbose mode preserves more details

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "output_mode": "verbose"
    }
  }
}
```

**Expected Result**:
- Detailed output up to 80,000 characters
- Lower filtering reduction (50-70%)
- More compilation details preserved

#### Test 1.1.4: Build Failure Case
**Purpose**: Verify failed builds provide clear error information

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "NonExistentScheme",
      "output_mode": "standard"
    }
  }
}
```

**Expected Result**:
- `"success": false`
- Clear failure summary with exit code
- Debug hint present if output is minimal
- Error details preserved

#### Test 1.1.5: Clean Build
**Purpose**: Test clean build functionality

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "clean": true
    }
  }
}
```

#### Test 1.1.6: Archive Build
**Purpose**: Test archive creation

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "archive": true,
      "configuration": "Release"
    }
  }
}
```

### 1.2 Universal Test Tool (`xcode_test`)

#### Test 1.2.1: Run Tests - Standard Mode
**Purpose**: Verify test execution and result parsing

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_test",
    "arguments": {
      "project_path": "/path/to/your/project",
      "scheme": "YourScheme",
      "output_mode": "standard"
    }
  }
}
```

**Expected Result**:
- Test results clearly shown
- Pass/fail counts
- Individual test case results in standard mode

#### Test 1.2.2: Failed Tests - Minimal Mode
**Purpose**: Verify failed test reporting in minimal mode

**Expected Result**:
- Test failures clearly indicated
- Critical test information preserved
- Summary shows failure count

### 1.3 Clean Tool (`xcode_clean`)

#### Test 1.3.1: Clean Project
**Purpose**: Test project cleaning

**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_clean",
    "arguments": {
      "project_path": "/path/to/your/project"
    }
  }
}
```

---

## 2. Discovery Tools

### 2.1 Discover Projects (`discover_projects`)

#### Test 2.1.1: Find Projects in Directory
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "discover_projects",
    "arguments": {
      "root_path": "/path/to/search/directory",
      "max_depth": 3
    }
  }
}
```

**Expected Result**:
- Lists all .xcodeproj and .xcworkspace files
- Includes metadata for each project

### 2.2 List Schemes (`list_schemes`)

#### Test 2.2.1: List Schemes for Project
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_schemes",
    "arguments": {
      "project_path": "/path/to/your/project"
    }
  }
}
```

**Expected Result**:
- All available schemes listed
- Shared/user scheme distinction
- Target information included

---

## 3. Simulator Management Tools

### 3.1 List Simulators (`list_simulators`)

#### Test 3.1.1: List All Simulators
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_simulators",
    "arguments": {}
  }
}
```

#### Test 3.1.2: Filter by Platform
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_simulators",
    "arguments": {
      "platform": "iOS",
      "state": "Booted"
    }
  }
}
```

### 3.2 Simulator Control (`simulator_control`)

#### Test 3.2.1: Boot Simulator
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "simulator_control",
    "arguments": {
      "udid": "SIMULATOR_UDID_HERE",
      "action": "boot"
    }
  }
}
```

#### Test 3.2.2: Shutdown Simulator
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "simulator_control",
    "arguments": {
      "udid": "SIMULATOR_UDID_HERE", 
      "action": "shutdown"
    }
  }
}
```

---

## 4. App Management Tools

### 4.1 Install App (`install_app`)

#### Test 4.1.1: Install to Simulator
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "install_app",
    "arguments": {
      "app_path": "/path/to/YourApp.app",
      "udid": "SIMULATOR_UDID_HERE"
    }
  }
}
```

#### Test 4.1.2: Auto-detect Simulator
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "install_app",
    "arguments": {
      "app_path": "/path/to/YourApp.app",
      "device_type": "iPhone"
    }
  }
}
```

### 4.2 Launch App (`launch_app`)

#### Test 4.2.1: Launch with Bundle ID
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "launch_app",
    "arguments": {
      "bundle_id": "com.example.YourApp",
      "udid": "SIMULATOR_UDID_HERE"
    }
  }
}
```

#### Test 4.2.2: Launch with Environment Variables
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "launch_app",
    "arguments": {
      "bundle_id": "com.example.YourApp",
      "environment": {
        "TEST_MODE": "1",
        "DEBUG": "true"
      }
    }
  }
}
```

### 4.3 Get App Info (`get_app_info`)

#### Test 4.3.1: Extract from .app Bundle
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "get_app_info",
    "arguments": {
      "app_path": "/path/to/YourApp.app",
      "include_entitlements": true,
      "include_icon_paths": true
    }
  }
}
```

---

## 5. Debugging and Monitoring Tools

### 5.1 Capture Logs (`capture_logs`)

#### Test 5.1.1: Capture App Logs
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "capture_logs",
    "arguments": {
      "bundle_id": "com.example.YourApp",
      "max_lines": 50,
      "timeout_secs": 10
    }
  }
}
```

#### Test 5.1.2: Filter by Log Level
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "capture_logs",
    "arguments": {
      "log_level": "error",
      "filter_text": "crash",
      "max_lines": 100
    }
  }
}
```

### 5.2 Screenshot (`screenshot`)

#### Test 5.2.1: Take PNG Screenshot
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "screenshot",
    "arguments": {
      "output_path": "/tmp/test_screenshot.png",
      "format": "png"
    }
  }
}
```

### 5.3 Describe UI (`describe_ui`)

#### Test 5.3.1: Get UI Hierarchy - Tree Format
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "describe_ui",
    "arguments": {
      "output_format": "tree"
    }
  }
}
```

#### Test 5.3.2: Get UI Hierarchy - JSON Format
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "describe_ui",
    "arguments": {
      "output_format": "json",
      "filter_type": "button"
    }
  }
}
```

### 5.4 UI Interact (`ui_interact`)

#### Test 5.4.1: Tap Coordinates
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "ui_interact",
    "arguments": {
      "action": "tap",
      "x": 200,
      "y": 300
    }
  }
}
```

#### Test 5.4.2: Type Text
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "ui_interact",
    "arguments": {
      "action": "type",
      "text": "Hello World"
    }
  }
}
```

---

## 6. Edge Cases and Error Scenarios

### 6.1 Invalid Parameters

#### Test 6.1.1: Missing Required Parameters
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {}
  }
}
```
**Expected Result**: Clear error message about missing parameters

#### Test 6.1.2: Invalid File Paths
**Command**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "xcode_build",
    "arguments": {
      "project_path": "/nonexistent/path",
      "scheme": "Test"
    }
  }
}
```

### 6.2 System State Issues

#### Test 6.2.1: No Simulators Available
**Purpose**: Test behavior when no simulators are booted

#### Test 6.2.2: Xcode Not Found
**Purpose**: Test error handling when Xcode tools unavailable

---

## 7. Debug Log Analysis

After running tests, analyze debug logs:

### 7.1 Filter Debug Logs
**Location**: `/tmp/mcp_debug/mcp_filter_*_*.log`

**Check for**:
- Input/output character counts
- Token reduction percentages
- Rule application statistics
- Truncation events

**Example Analysis**:
```bash
# View latest filter log
ls -la /tmp/mcp_debug/mcp_filter_*.log | tail -1 | xargs cat

# Search for truncation events
grep -i "truncated" /tmp/mcp_debug/mcp_filter_*.log

# Check reduction percentages
grep "Reduction:" /tmp/mcp_debug/mcp_filter_*.log
```

### 7.2 Server Logs
**Check for**:
- Command execution times
- Error patterns
- Memory usage
- Tool registration issues

### 7.3 Common Issues and Solutions

#### Issue: Empty Output Despite Successful Build
**Debug Steps**:
1. Check filter debug log for input stats
2. Verify critical patterns are being matched
3. Check if output mode is appropriate
4. Look for truncation messages

#### Issue: High Memory Usage
**Debug Steps**:
1. Check for very large inputs in filter logs
2. Monitor character limits being enforced
3. Verify filtering rules are working

#### Issue: Slow Performance
**Debug Steps**:
1. Check command execution times in server logs
2. Monitor filter processing time
3. Look for redundant operations

---

## 8. Automated Test Script

Create this test script for batch testing:

```bash
#!/bin/bash
# save as test_all_tools.sh

export MCP_LOG_LEVEL=debug
export MCP_FILTER_DEBUG=true
export MCP_FILTER_DEBUG_DIR=/tmp/mcp_debug
mkdir -p /tmp/mcp_debug

# Start server in background
./xcode-build-mcp stdio &
SERVER_PID=$!

# Give server time to start
sleep 2

# Test each tool (examples - customize for your environment)
echo "Testing tool discovery..."
echo '{"method":"tools/list","params":{}}' | nc localhost 3000

echo "Testing xcode_build minimal mode..."
# Add your test calls here

# Cleanup
kill $SERVER_PID
```

---

## 9. Success Criteria

For each test, verify:

### ✅ Functional Requirements
- [ ] Tool executes without errors
- [ ] Response format is valid JSON
- [ ] Expected fields are present
- [ ] File operations work correctly

### ✅ Filtering Requirements  
- [ ] Output reduction >70% in standard mode
- [ ] Output reduction >85% in minimal mode
- [ ] Critical information preserved
- [ ] Build status always clear

### ✅ Performance Requirements
- [ ] Commands complete within reasonable time
- [ ] Memory usage stays reasonable
- [ ] No memory leaks over extended use

### ✅ Error Handling Requirements
- [ ] Invalid parameters return helpful errors
- [ ] System failures are handled gracefully
- [ ] Debug information is available

---

## 10. Troubleshooting Guide

### Common Problems:

**Problem**: Filter logs show no reduction
**Solution**: Check if input has filterable content, verify rules are loading

**Problem**: Build succeeds but shows as failed
**Solution**: Check exit code parsing logic, verify success patterns

**Problem**: Simulator operations fail
**Solution**: Verify simulators are available, check device selection logic

**Problem**: Very slow filtering
**Solution**: Check input size, verify character limits are enforced

---

This guide should help you comprehensively test the xcode-build-mcp server and identify any issues through debug log analysis. Adjust the file paths and scheme names to match your specific test environment.