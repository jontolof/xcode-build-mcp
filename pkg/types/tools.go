package types

import (
	"time"
)

type BuildParams struct {
	ProjectPath  string            `json:"project_path,omitempty"`
	Workspace    string            `json:"workspace,omitempty"`
	Project      string            `json:"project,omitempty"`
	Scheme       string            `json:"scheme,omitempty"`
	Target       string            `json:"target,omitempty"`
	Configuration string           `json:"configuration,omitempty"`
	SDK          string            `json:"sdk,omitempty"`
	Destination  string            `json:"destination,omitempty"`
	Arch         string            `json:"arch,omitempty"`
	OutputMode   string            `json:"output_mode,omitempty"`
	Clean        bool              `json:"clean,omitempty"`
	Archive      bool              `json:"archive,omitempty"`
	DerivedData  string            `json:"derived_data,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	ExtraArgs    []string          `json:"extra_args,omitempty"`
}

type BuildResult struct {
	Success         bool                   `json:"success"`
	Duration        time.Duration          `json:"duration"`
	Output          string                 `json:"output"`
	FilteredOutput  string                 `json:"filtered_output"`
	Errors          []BuildError           `json:"errors,omitempty"`
	Warnings        []BuildWarning         `json:"warnings,omitempty"`
	ArtifactPaths   []string               `json:"artifact_paths,omitempty"`
	BuildSettings   map[string]interface{} `json:"build_settings,omitempty"`
	ExitCode        int                    `json:"exit_code"`
}

type TestParams struct {
	ProjectPath   string            `json:"project_path,omitempty"`
	Workspace     string            `json:"workspace,omitempty"`
	Project       string            `json:"project,omitempty"`
	Scheme        string            `json:"scheme,omitempty"`
	Target        string            `json:"target,omitempty"`
	TestPlan      string            `json:"test_plan,omitempty"`
	SDK           string            `json:"sdk,omitempty"`
	Destination   string            `json:"destination,omitempty"`
	OnlyTesting   []string          `json:"only_testing,omitempty"`
	SkipTesting   []string          `json:"skip_testing,omitempty"`
	OutputMode    string            `json:"output_mode,omitempty"`
	Parallel      bool              `json:"parallel,omitempty"`
	Coverage      bool              `json:"coverage,omitempty"`
	ResultBundle  string            `json:"result_bundle,omitempty"`
	DerivedData   string            `json:"derived_data,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	ExtraArgs     []string          `json:"extra_args,omitempty"`
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
	TotalTests     int           `json:"total_tests"`
	PassedTests    int           `json:"passed_tests"`
	FailedTests    int           `json:"failed_tests"`
	SkippedTests   int           `json:"skipped_tests"`
	TestResults    []TestCase    `json:"test_results"`
	FailedTestsDetails []TestCase `json:"failed_tests_details,omitempty"`
}

type TestCase struct {
	Name       string        `json:"name"`
	ClassName  string        `json:"class_name"`
	Status     string        `json:"status"`
	Duration   time.Duration `json:"duration"`
	Message    string        `json:"message,omitempty"`
	Location   string        `json:"location,omitempty"`
}

type Coverage struct {
	LineCoverage   float64            `json:"line_coverage"`
	BranchCoverage float64            `json:"branch_coverage"`
	FilesCoverage  []FileCoverage     `json:"files_coverage"`
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
	MaxDepth     int      `json:"max_depth,omitempty"`
	IncludeHidden bool    `json:"include_hidden,omitempty"`
	RootPath     string   `json:"root_path,omitempty"`
	Patterns     []string `json:"patterns,omitempty"`
}

type DiscoveryResult struct {
	Projects   []ProjectInfo `json:"projects"`
	Duration   time.Duration `json:"duration"`
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
	Platform     string `json:"platform,omitempty"`
	DeviceType   string `json:"device_type,omitempty"`
	Runtime      string `json:"runtime,omitempty"`
	Available    *bool  `json:"available,omitempty"`
	State        string `json:"state,omitempty"`
}

type SimulatorListResult struct {
	Simulators []SimulatorInfo `json:"simulators"`
	Duration   time.Duration   `json:"duration"`
}

type SimulatorInfo struct {
	UDID         string `json:"udid"`
	Name         string `json:"name"`
	DeviceType   string `json:"device_type"`
	Runtime      string `json:"runtime"`
	State        string `json:"state"`
	Available    bool   `json:"available"`
	Platform     string `json:"platform"`
}

type SimulatorControlParams struct {
	UDID      string `json:"udid"`
	Action    string `json:"action"`
	Timeout   int    `json:"timeout,omitempty"`
}

type SimulatorControlResult struct {
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
	Output         string        `json:"output"`
	PreviousState  string        `json:"previous_state"`
	CurrentState   string        `json:"current_state"`
}

type AppInstallParams struct {
	AppPath     string `json:"app_path"`
	UDID        string `json:"udid,omitempty"`
	DeviceType  string `json:"device_type,omitempty"`
	Replace     bool   `json:"replace,omitempty"`
}

type AppInstallResult struct {
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
	Output         string        `json:"output"`
	BundleID       string        `json:"bundle_id"`
	InstalledPath  string        `json:"installed_path,omitempty"`
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