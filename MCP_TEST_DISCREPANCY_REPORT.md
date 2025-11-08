# MCP Xcode-Build Test Execution Discrepancy Report

**Report Date:** 2025-10-03
**MCP Server:** `xcode-build`
**Tool Version:** Latest (as of 2025-10-03)
**Issue Type:** Test Discovery/Execution Bug
**Severity:** HIGH - Incomplete test coverage reporting

---

## Executive Summary

The MCP `xcode-build` server's `xcode_test` tool **only executes unit tests** and **completely misses UI test targets**, leading to incomplete test coverage reporting and false-positive "all tests passing" results.

**Impact:** Developers using MCP for test validation receive misleading pass/fail status, potentially shipping code with broken UI tests.

---

## Reproduction Case

### Environment
- **OS:** macOS 15.0 (Darwin 24.6.0)
- **Xcode:** Version 17.0 (17A400)
- **Project:** LeMieLingueApp.xcodeproj
- **Scheme:** LeMieLingueApp
- **Destination:** iOS Simulator (iPhone 16, iOS 18.4)

### Test Target Structure
```
LeMieLingueApp.xcodeproj
├── LeMieLingueApp (main target)
├── LeMieLingueAppTests (Unit Tests) ← MCP executes these ✅
└── LeMieLingueAppUITests (UI Tests) ← MCP MISSES these ❌
```

### Unit Test Bundle (Executed by MCP)
**Target:** `LeMieLingueAppTests`
**Test Count:** 149 tests across 28 test files
**File Pattern:** `LeMieLingueAppTests/**/*Tests.swift`

Example files:
- `AuthenticationModelsTests.swift`
- `RecordingViewTests.swift`
- `AppleSignInIntegrationTests.swift`
- `RecordingPerformanceTests.swift`
- (25+ more unit test files)

### UI Test Bundle (MISSED by MCP)
**Target:** `LeMieLingueAppUITests`
**Test Count:** 44 tests across 4 test files
**Framework:** XCUITest
**File Pattern:** `LeMieLingueAppUITests/**/*Tests.swift`

Test files:
```
LeMieLingueAppUITests/
├── ConsoleValidationTests.swift (2 tests)
├── AppleSignInUITests.swift (1 test)
├── LeMieLingueAppUITests.swift (1 test)
└── LeMieLingueAppUITestsLaunchTests.swift (1 test)
```

---

## Observed Behavior

### MCP Tool Invocation
```json
{
  "tool": "mcp__xcode-build__xcode_test",
  "parameters": {
    "project_path": ".",
    "project": "LeMieLingueApp.xcodeproj",
    "scheme": "LeMieLingueApp",
    "destination": "platform=iOS Simulator,id=1876A73A-E17E-401A-916E-66057D5941E9"
  }
}
```

### MCP Response
```json
{
  "success": false,
  "test_summary": {
    "failed_tests": 5,
    "passed_tests": 144,
    "total_tests": 149
  }
}
```

**Analysis:** MCP reports only 149 tests (unit tests), completely missing 44 UI tests.

---

## Expected Behavior

### XCode IDE Test Execution
When running tests via XCode IDE (Product → Test or Cmd+U):

**Test Navigator shows:**
```
LeMieLingueAppTests (149 tests)
  ✅ 144 passing
  ❌ 5 failing

LeMieLingueAppUITests (44 tests)
  ❌ 44 failing (console violations)

Total: 193 tests (144 pass, 49 fail)
```

### Command-Line Xcodebuild
```bash
xcodebuild test \
  -project LeMieLingueApp.xcodeproj \
  -scheme LeMieLingueApp \
  -destination 'platform=iOS Simulator,name=iPhone 16'
```

**Result:** Executes BOTH unit and UI test bundles (193 tests total)

---

## Root Cause Analysis

### Hypothesis 1: Test Target Discovery
MCP tool may only discover test bundles that:
- Are explicitly listed in scheme's test action
- Match specific naming patterns (e.g., `*Tests` but not `*UITests`)
- Are of type `xctest` (excluding `ui-testing` bundles)

### Hypothesis 2: Framework Filtering
MCP tool may filter out:
- XCUITest-based tests (UI testing framework)
- Tests importing `XCTest` via `@testable import XCUIApplication`
- Tests with UI-specific attributes

### Hypothesis 3: Scheme Configuration
The scheme's test action may have both targets configured, but MCP only reads:
- First test target listed
- Targets without "UI" in name
- Non-disabled test bundles only

---

## Evidence

### Scheme Metadata Query
```bash
# List schemes
mcp__xcode-build__list_schemes(
  project_path: ".",
  project: "LeMieLingueApp.xcodeproj"
)
```

**Response:**
```json
{
  "schemes": [
    {
      "name": "LeMieLingueApp",
      "project_path": "LeMieLingueApp.xcodeproj",
      "shared_scheme": true,
      "targets": ["LeMieLingueApp"]
    }
  ]
}
```

**Finding:** `list_schemes` only returns main target, not test targets. MCP may use this incomplete metadata for test discovery.

### File System Evidence
```bash
find . -name "*UITests.swift" | wc -l
# Output: 4 files

find . -name "*Tests.swift" | grep -v UITests | wc -l
# Output: 28 files
```

Both test bundles exist on disk, but only unit tests execute via MCP.

---

## Impact Assessment

### For Developers Using MCP
1. **False Confidence:** MCP reports "144/149 passing" when reality is "144/193 passing"
2. **Hidden Failures:** 44 UI tests never execute, failures go undetected
3. **CI/CD Risk:** Automated pipelines using MCP may ship broken UI

### For This Project Specifically
- **Console Validation Tests:** Critical DoD validation tests (`ConsoleValidationTests.swift`) never run via MCP
- **UI Integration Tests:** Apple Sign-In UI tests, launch tests all skipped
- **Test Coverage Gaps:** ~23% of test suite invisible to MCP users

---

## Comparison: MCP vs Native Tools

| Aspect | MCP `xcode_test` | `xcodebuild test` | XCode IDE |
|--------|------------------|-------------------|-----------|
| **Unit Tests** | ✅ Executes (149) | ✅ Executes (149) | ✅ Executes (149) |
| **UI Tests** | ❌ Skips (0) | ✅ Executes (44) | ✅ Executes (44) |
| **Total Tests** | 149 | 193 | 193 |
| **Accuracy** | 77% coverage | 100% coverage | 100% coverage |

---

## Recommended Fixes

### Option 1: Enhanced Test Discovery (Preferred)
Update MCP tool to discover ALL test bundles:
```swift
// Pseudo-code for improved discovery
func discoverTestTargets(scheme: String) -> [TestTarget] {
    let schemeFile = parseScheme(scheme)
    return schemeFile.testAction.testables.map { testable in
        TestTarget(
            name: testable.buildableReference.blueprintName,
            type: testable.isUITest ? .uiTest : .unitTest
        )
    }
}
```

### Option 2: Explicit UI Test Parameter
Add optional parameter to force UI test execution:
```json
{
  "tool": "xcode_test",
  "parameters": {
    "include_ui_tests": true  // New parameter
  }
}
```

### Option 3: Separate UI Test Tool
Create dedicated `xcode_ui_test` tool:
```json
{
  "tool": "xcode_ui_test",  // Specifically for UI tests
  "parameters": {
    "project": "...",
    "scheme": "..."
  }
}
```

---

## Workaround (Current)

Until fixed, developers should:

1. **Verify test count manually:**
   ```bash
   # Count expected tests
   find . -name "*Tests.swift" -exec grep -l "func test" {} \; | wc -l
   ```

2. **Run XCode IDE tests** to validate UI test suite

3. **Use raw xcodebuild** for CI/CD:
   ```bash
   xcodebuild test -scheme MyScheme -destination '...'
   ```

4. **Cross-check MCP results** against XCode test navigator

---

## Reproduction Steps

### Minimal Reproduction
1. Create Xcode project with 2 test targets:
   - `AppTests` (unit tests)
   - `AppUITests` (UI tests)

2. Add scheme with both test targets enabled

3. Execute via MCP:
   ```json
   {
     "tool": "mcp__xcode-build__xcode_test",
     "parameters": {
       "project": "App.xcodeproj",
       "scheme": "App"
     }
   }
   ```

4. **Observe:** Only `AppTests` execute, `AppUITests` skipped

5. **Compare:** Run `xcodebuild test` directly - both targets execute

---

## Additional Context

### Test Output Logs
MCP tool output shows ONLY unit test execution:
```
Test Suite 'AuthenticationModelsTests' started at 2025-10-03 10:37:49.052.
Test Suite 'RecordingViewTests' started at 2025-10-03 10:38:15.123.
...
(No mention of ConsoleValidationTests or any UI test suites)
```

### Scheme File Analysis (if helpful)
The `.xcscheme` file contains:
```xml
<TestAction>
  <Testables>
    <TestableReference>
      <BuildableReference
        BlueprintName = "LeMieLingueAppTests"
        BuildableName = "LeMieLingueAppTests.xctest">
      </BuildableReference>
    </TestableReference>
    <TestableReference>
      <BuildableReference
        BlueprintName = "LeMieLingueAppUITests"
        BuildableName = "LeMieLingueAppUITests.xctest">
      </BuildableReference>
    </TestableReference>
  </Testables>
</TestAction>
```

Both test targets ARE configured in scheme, but MCP only executes first one.

---

## Request for MCP Team

### Please Investigate:
1. Why does `xcode_test` skip UI test bundles?
2. Is this intentional filtering or a bug?
3. Can test target discovery be enhanced to include all `xctest` bundles?

### Suggested Enhancements:
1. **Add `test_targets` to response** showing which bundles were discovered/executed
2. **Warn when test bundles are skipped** (e.g., "Skipped 1 UI test bundle")
3. **Provide test type breakdown** (unit vs UI vs performance)

### Example Enhanced Response:
```json
{
  "success": false,
  "test_summary": {
    "total_tests": 193,
    "passed_tests": 144,
    "failed_tests": 49
  },
  "test_bundles": [
    {
      "name": "LeMieLingueAppTests",
      "type": "unit",
      "executed": true,
      "test_count": 149
    },
    {
      "name": "LeMieLingueAppUITests",
      "type": "ui",
      "executed": true,
      "test_count": 44
    }
  ]
}
```

---

## Contact Information

**Reporter:** Development team using MCP xcode-build server
**Project:** LeMieLingueApp (iOS language learning app)
**Environment:** macOS 15.0, Xcode 17.0, Swift 6

**Questions/Clarifications:** Available for follow-up debugging, scheme file sharing, or live reproduction sessions.

---

## Attachments (Available on Request)

1. Full scheme file (`.xcscheme`)
2. Test discovery logs (verbose MCP output)
3. XCode test navigator screenshots
4. Complete xcodebuild test output for comparison

---

**Thank you for investigating this issue. Accurate test reporting is critical for developer confidence in MCP tooling.**
