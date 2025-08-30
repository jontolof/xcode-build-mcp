package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildParams_JSON(t *testing.T) {
	params := BuildParams{
		Project:       "MyApp.xcodeproj",
		Scheme:        "MyApp",
		Configuration: "Debug",
		SDK:           "iphoneos",
		Clean:         true,
		Environment: map[string]string{
			"ENV_VAR": "value",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal BuildParams: %v", err)
	}

	var decoded BuildParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BuildParams: %v", err)
	}

	if decoded.Project != params.Project {
		t.Errorf("Project mismatch: got %s, want %s", decoded.Project, params.Project)
	}
	if decoded.Scheme != params.Scheme {
		t.Errorf("Scheme mismatch: got %s, want %s", decoded.Scheme, params.Scheme)
	}
	if decoded.Clean != params.Clean {
		t.Errorf("Clean mismatch: got %v, want %v", decoded.Clean, params.Clean)
	}
}

func TestBuildResult_JSON(t *testing.T) {
	result := BuildResult{
		Success:        true,
		Duration:       5 * time.Second,
		Output:         "Build succeeded",
		FilteredOutput: "Build succeeded",
		ExitCode:       0,
		ArtifactPaths:  []string{"/path/to/app"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal BuildResult: %v", err)
	}

	var decoded BuildResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BuildResult: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, result.Success)
	}
	if decoded.ExitCode != result.ExitCode {
		t.Errorf("ExitCode mismatch: got %d, want %d", decoded.ExitCode, result.ExitCode)
	}
}

func TestTestParams_JSON(t *testing.T) {
	params := TestParams{
		Workspace:   "MyApp.xcworkspace",
		Scheme:      "MyAppTests",
		TestPlan:    "AllTests",
		Coverage:    true,
		Parallel:    true,
		OnlyTesting: []string{"MyAppTests/TestClass"},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal TestParams: %v", err)
	}

	var decoded TestParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TestParams: %v", err)
	}

	if decoded.Workspace != params.Workspace {
		t.Errorf("Workspace mismatch: got %s, want %s", decoded.Workspace, params.Workspace)
	}
	if decoded.Coverage != params.Coverage {
		t.Errorf("Coverage mismatch: got %v, want %v", decoded.Coverage, params.Coverage)
	}
	if len(decoded.OnlyTesting) != len(params.OnlyTesting) {
		t.Errorf("OnlyTesting length mismatch: got %d, want %d", 
			len(decoded.OnlyTesting), len(params.OnlyTesting))
	}
}

func TestTestResult_JSON(t *testing.T) {
	result := TestResult{
		Success:  true,
		Duration: 10 * time.Second,
		TestSummary: TestSummary{
			TotalTests:   100,
			PassedTests:  95,
			FailedTests:  5,
			SkippedTests: 0,
		},
		Coverage: &Coverage{
			LineCoverage:   85.5,
			BranchCoverage: 75.0,
		},
		ExitCode: 0,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal TestResult: %v", err)
	}

	var decoded TestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TestResult: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, result.Success)
	}
	if decoded.TestSummary.TotalTests != result.TestSummary.TotalTests {
		t.Errorf("TotalTests mismatch: got %d, want %d", 
			decoded.TestSummary.TotalTests, result.TestSummary.TotalTests)
	}
	if decoded.Coverage == nil {
		t.Fatal("Coverage should not be nil")
	}
	if decoded.Coverage.LineCoverage != result.Coverage.LineCoverage {
		t.Errorf("LineCoverage mismatch: got %f, want %f", 
			decoded.Coverage.LineCoverage, result.Coverage.LineCoverage)
	}
}

func TestSimulatorInfo_JSON(t *testing.T) {
	sim := SimulatorInfo{
		UDID:       "12345-67890",
		Name:       "iPhone 15 Pro",
		DeviceType: "iPhone 15 Pro",
		Runtime:    "iOS 17.0",
		State:      "Booted",
		Available:  true,
		Platform:   "iOS",
	}

	data, err := json.Marshal(sim)
	if err != nil {
		t.Fatalf("Failed to marshal SimulatorInfo: %v", err)
	}

	var decoded SimulatorInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SimulatorInfo: %v", err)
	}

	if decoded.UDID != sim.UDID {
		t.Errorf("UDID mismatch: got %s, want %s", decoded.UDID, sim.UDID)
	}
	if decoded.Name != sim.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, sim.Name)
	}
	if decoded.State != sim.State {
		t.Errorf("State mismatch: got %s, want %s", decoded.State, sim.State)
	}
	if decoded.Available != sim.Available {
		t.Errorf("Available mismatch: got %v, want %v", decoded.Available, sim.Available)
	}
}

func TestProjectInfo_JSON(t *testing.T) {
	project := ProjectInfo{
		Path:         "/path/to/MyApp.xcodeproj",
		Name:         "MyApp",
		Type:         "project",
		Schemes:      []string{"MyApp", "MyAppTests"},
		Targets:      []string{"MyApp", "MyAppTests", "MyAppUITests"},
		LastModified: time.Now(),
	}

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal ProjectInfo: %v", err)
	}

	var decoded ProjectInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ProjectInfo: %v", err)
	}

	if decoded.Path != project.Path {
		t.Errorf("Path mismatch: got %s, want %s", decoded.Path, project.Path)
	}
	if decoded.Name != project.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, project.Name)
	}
	if decoded.Type != project.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, project.Type)
	}
	if len(decoded.Schemes) != len(project.Schemes) {
		t.Errorf("Schemes length mismatch: got %d, want %d", 
			len(decoded.Schemes), len(project.Schemes))
	}
	if len(decoded.Targets) != len(project.Targets) {
		t.Errorf("Targets length mismatch: got %d, want %d", 
			len(decoded.Targets), len(project.Targets))
	}
}

func TestAppInstallResult_JSON(t *testing.T) {
	result := AppInstallResult{
		Success:       true,
		Duration:      3 * time.Second,
		Output:        "App installed successfully",
		BundleID:      "com.example.myapp",
		InstalledPath: "/path/to/installed/app",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal AppInstallResult: %v", err)
	}

	var decoded AppInstallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AppInstallResult: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, result.Success)
	}
	if decoded.BundleID != result.BundleID {
		t.Errorf("BundleID mismatch: got %s, want %s", decoded.BundleID, result.BundleID)
	}
	if decoded.InstalledPath != result.InstalledPath {
		t.Errorf("InstalledPath mismatch: got %s, want %s", 
			decoded.InstalledPath, result.InstalledPath)
	}
}

func TestAppLaunchResult_JSON(t *testing.T) {
	exitCode := 0
	result := AppLaunchResult{
		Success:   true,
		Duration:  2 * time.Second,
		Output:    "App launched",
		ProcessID: 12345,
		ExitCode:  &exitCode,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal AppLaunchResult: %v", err)
	}

	var decoded AppLaunchResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AppLaunchResult: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, result.Success)
	}
	if decoded.ProcessID != result.ProcessID {
		t.Errorf("ProcessID mismatch: got %d, want %d", decoded.ProcessID, result.ProcessID)
	}
	if decoded.ExitCode == nil || *decoded.ExitCode != *result.ExitCode {
		t.Errorf("ExitCode mismatch")
	}
}