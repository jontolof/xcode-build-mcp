package types

import (
	"time"
)

type BuildParams struct {
	ProjectPath   string            `json:"project_path,omitempty"`
	Workspace     string            `json:"workspace,omitempty"`
	Project       string            `json:"project,omitempty"`
	Scheme        string            `json:"scheme,omitempty"`
	Target        string            `json:"target,omitempty"`
	Configuration string            `json:"configuration,omitempty"`
	SDK           string            `json:"sdk,omitempty"`
	Destination   string            `json:"destination,omitempty"`
	Arch          string            `json:"arch,omitempty"`
	OutputMode    string            `json:"output_mode,omitempty"`
	Clean         bool              `json:"clean,omitempty"`
	Archive       bool              `json:"archive,omitempty"`
	DerivedData   string            `json:"derived_data,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	ExtraArgs     []string          `json:"extra_args,omitempty"`
}

type BuildResult struct {
	Success        bool                   `json:"success"`
	Duration       time.Duration          `json:"duration"`
	Output         string                 `json:"output"`
	FilteredOutput string                 `json:"filtered_output"`
	Errors         []BuildError           `json:"errors,omitempty"`
	Warnings       []BuildWarning         `json:"warnings,omitempty"`
	ArtifactPaths  []string               `json:"artifact_paths,omitempty"`
	BuildSettings  map[string]interface{} `json:"build_settings,omitempty"`
	ExitCode       int                    `json:"exit_code"`
}

type TestParams struct {
	ProjectPath  string            `json:"project_path,omitempty"`
	Workspace    string            `json:"workspace,omitempty"`
	Project      string            `json:"project,omitempty"`
	Scheme       string            `json:"scheme,omitempty"`
	Target       string            `json:"target,omitempty"`
	TestPlan     string            `json:"test_plan,omitempty"`
	SDK          string            `json:"sdk,omitempty"`
	Destination  string            `json:"destination,omitempty"`
	OnlyTesting  []string          `json:"only_testing,omitempty"`
	SkipTesting  []string          `json:"skip_testing,omitempty"`
	OutputMode   string            `json:"output_mode,omitempty"`
	Parallel     bool              `json:"parallel,omitempty"`
	Coverage     bool              `json:"coverage,omitempty"`
	ResultBundle string            `json:"result_bundle,omitempty"`
	DerivedData  string            `json:"derived_data,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	ExtraArgs    []string          `json:"extra_args,omitempty"`
}

type TestResult struct {
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
	Output         string        `json:"output"`
	FilteredOutput string        `json:"filtered_output"`
	TestSummary    TestSummary   `json:"test_summary"`
	Coverage       *Coverage     `json:"coverage,omitempty"`
	ExitCode       int           `json:"exit_code"`
}

type TestSummary struct {
	TotalTests         int          `json:"total_tests"`
	PassedTests        int          `json:"passed_tests"`
	FailedTests        int          `json:"failed_tests"`
	SkippedTests       int          `json:"skipped_tests"`
	TestResults        []TestCase   `json:"test_results"`
	FailedTestsDetails []TestCase   `json:"failed_tests_details,omitempty"`
	TestBundles        []TestBundle `json:"test_bundles,omitempty"`
}

type TestCase struct {
	Name      string        `json:"name"`
	ClassName string        `json:"class_name"`
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration"`
	Message   string        `json:"message,omitempty"`
	Location  string        `json:"location,omitempty"`
}

type TestBundle struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Executed  bool          `json:"executed"`
	Status    string        `json:"status"`
	TestCount int           `json:"test_count"`
	Duration  time.Duration `json:"duration,omitempty"`
}

type Coverage struct {
	LineCoverage   float64        `json:"line_coverage"`
	BranchCoverage float64        `json:"branch_coverage"`
	FilesCoverage  []FileCoverage `json:"files_coverage"`
}

type FileCoverage struct {
	Path           string  `json:"path"`
	LineCoverage   float64 `json:"line_coverage"`
	BranchCoverage float64 `json:"branch_coverage"`
	CoveredLines   int     `json:"covered_lines"`
	TotalLines     int     `json:"total_lines"`
}

type CleanParams struct {
	ProjectPath string `json:"project_path,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	Project     string `json:"project,omitempty"`
	Target      string `json:"target,omitempty"`
	DerivedData string `json:"derived_data,omitempty"`
	CleanBuild  bool   `json:"clean_build,omitempty"`
}

type CleanResult struct {
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
	Output         string        `json:"output"`
	FilteredOutput string        `json:"filtered_output"`
	CleanedPaths   []string      `json:"cleaned_paths"`
	ExitCode       int           `json:"exit_code"`
}

type ProjectDiscovery struct {
	MaxDepth      int      `json:"max_depth,omitempty"`
	IncludeHidden bool     `json:"include_hidden,omitempty"`
	RootPath      string   `json:"root_path,omitempty"`
	Patterns      []string `json:"patterns,omitempty"`
}

type DiscoveryResult struct {
	Projects []ProjectInfo `json:"projects"`
	Duration time.Duration `json:"duration"`
}

type ProjectInfo struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Schemes      []string  `json:"schemes"`
	Targets      []string  `json:"targets"`
	LastModified time.Time `json:"last_modified"`
}

type SimulatorListParams struct {
	Platform   string `json:"platform,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	Runtime    string `json:"runtime,omitempty"`
	Available  *bool  `json:"available,omitempty"`
	State      string `json:"state,omitempty"`
}

type SimulatorListResult struct {
	Simulators []SimulatorInfo `json:"simulators"`
	Duration   time.Duration   `json:"duration"`
}

type SimulatorInfo struct {
	UDID       string `json:"udid"`
	Name       string `json:"name"`
	DeviceType string `json:"device_type"`
	Runtime    string `json:"runtime"`
	State      string `json:"state"`
	Available  bool   `json:"available"`
	Platform   string `json:"platform"`
}

type SimulatorControlParams struct {
	UDID    string `json:"udid"`
	Action  string `json:"action"`
	Timeout int    `json:"timeout,omitempty"`
}

type SimulatorControlResult struct {
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	Output        string        `json:"output"`
	PreviousState string        `json:"previous_state"`
	CurrentState  string        `json:"current_state"`
}

type AppInstallParams struct {
	AppPath    string `json:"app_path"`
	UDID       string `json:"udid,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	Replace    bool   `json:"replace,omitempty"`
}

type AppInstallResult struct {
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	Output        string        `json:"output"`
	BundleID      string        `json:"bundle_id"`
	InstalledPath string        `json:"installed_path,omitempty"`
}

type AppLaunchParams struct {
	BundleID    string            `json:"bundle_id"`
	UDID        string            `json:"udid,omitempty"`
	DeviceType  string            `json:"device_type,omitempty"`
	Arguments   []string          `json:"arguments,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WaitForExit bool              `json:"wait_for_exit,omitempty"`
}

type AppLaunchResult struct {
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Output    string        `json:"output"`
	ProcessID int           `json:"process_id,omitempty"`
	ExitCode  *int          `json:"exit_code,omitempty"`
}

type SchemesListParams struct {
	ProjectPath string `json:"project_path,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	Project     string `json:"project,omitempty"`
}

type SchemesListResult struct {
	Schemes  []SchemeInfo  `json:"schemes"`
	Duration time.Duration `json:"duration"`
}

type SchemeInfo struct {
	Name         string   `json:"name"`
	ProjectPath  string   `json:"project_path"`
	SharedScheme bool     `json:"shared_scheme"`
	Targets      []string `json:"targets,omitempty"`
}

type LogCaptureParams struct {
	UDID        string `json:"udid,omitempty"`
	DeviceType  string `json:"device_type,omitempty"`
	BundleID    string `json:"bundle_id,omitempty"`
	LogLevel    string `json:"log_level,omitempty"`
	FilterText  string `json:"filter_text,omitempty"`
	FollowMode  bool   `json:"follow_mode,omitempty"`
	MaxLines    int    `json:"max_lines,omitempty"`
	TimeoutSecs int    `json:"timeout_secs,omitempty"`
}

type LogCaptureResult struct {
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	LogLines  []LogEntry    `json:"log_lines"`
	Truncated bool          `json:"truncated,omitempty"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Category  string `json:"category,omitempty"`
	Message   string `json:"message"`
	Process   string `json:"process,omitempty"`
}

type ScreenshotParams struct {
	UDID       string `json:"udid,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	OutputPath string `json:"output_path,omitempty"`
	Format     string `json:"format,omitempty"`
}

type ScreenshotResult struct {
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	FilePath   string        `json:"file_path"`
	FileSize   int64         `json:"file_size,omitempty"`
	Dimensions string        `json:"dimensions,omitempty"`
}

type UIDescribeParams struct {
	UDID        string `json:"udid,omitempty"`
	DeviceType  string `json:"device_type,omitempty"`
	Format      string `json:"format,omitempty"`
	MaxDepth    int    `json:"max_depth,omitempty"`
	IncludeText bool   `json:"include_text,omitempty"`
}

type UIDescribeResult struct {
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	UIHierarchy  interface{}   `json:"ui_hierarchy"`
	ElementCount int           `json:"element_count"`
}

type UIInteractParams struct {
	UDID        string                 `json:"udid,omitempty"`
	DeviceType  string                 `json:"device_type,omitempty"`
	Action      string                 `json:"action"`
	Target      string                 `json:"target,omitempty"`
	Coordinates []float64              `json:"coordinates,omitempty"`
	Text        string                 `json:"text,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type UIInteractResult struct {
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output"`
	Found    bool          `json:"found,omitempty"`
}

type AppInfoParams struct {
	AppPath    string `json:"app_path,omitempty"`
	BundleID   string `json:"bundle_id,omitempty"`
	UDID       string `json:"udid,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
}

type AppInfoResult struct {
	Success      bool                   `json:"success"`
	Duration     time.Duration          `json:"duration"`
	BundleID     string                 `json:"bundle_id,omitempty"`
	Version      string                 `json:"version,omitempty"`
	BuildNumber  string                 `json:"build_number,omitempty"`
	DisplayName  string                 `json:"display_name,omitempty"`
	MinOSVersion string                 `json:"min_os_version,omitempty"`
	Entitlements map[string]interface{} `json:"entitlements,omitempty"`
	IconPaths    []string               `json:"icon_paths,omitempty"`
}
