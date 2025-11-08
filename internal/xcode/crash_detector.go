package xcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

// SimulatorCrashDetector monitors DiagnosticReports for simulator crashes
type SimulatorCrashDetector struct {
	diagnosticPath string
	startTime      time.Time
}

// NewSimulatorCrashDetector creates a new crash detector
func NewSimulatorCrashDetector() *SimulatorCrashDetector {
	homeDir, _ := os.UserHomeDir()
	return &SimulatorCrashDetector{
		diagnosticPath: filepath.Join(homeDir, "Library/Logs/DiagnosticReports"),
		startTime:      time.Now(),
	}
}

// CheckForCrashes searches for crash reports created after the detector was initialized
func (d *SimulatorCrashDetector) CheckForCrashes(processName string) ([]types.CrashReport, error) {
	var crashes []types.CrashReport

	// Check if diagnostic reports directory exists
	if _, err := os.Stat(d.diagnosticPath); os.IsNotExist(err) {
		return crashes, nil
	}

	// List crash reports (IPS format used by macOS 12+)
	files, err := filepath.Glob(filepath.Join(d.diagnosticPath, "*.ips"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Only check crashes that occurred during our execution
		if info.ModTime().Before(d.startTime) {
			continue
		}

		// Check if related to simulator or test processes
		baseName := filepath.Base(file)
		if strings.Contains(baseName, "Simulator") ||
			strings.Contains(baseName, "testmanagerd") ||
			strings.Contains(baseName, "xctest") ||
			strings.Contains(baseName, "simctl") ||
			(processName != "" && strings.Contains(baseName, processName)) {

			crash, err := d.parseCrashReport(file)
			if err == nil {
				crashes = append(crashes, crash)
			}
		}
	}

	return crashes, nil
}

// parseCrashReport parses a macOS crash report in IPS format
func (d *SimulatorCrashDetector) parseCrashReport(path string) (types.CrashReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return types.CrashReport{}, err
	}

	var report map[string]interface{}
	if err := json.Unmarshal(data, &report); err != nil {
		return types.CrashReport{}, err
	}

	crash := types.CrashReport{
		FilePath:  path,
		Timestamp: time.Now(),
	}

	// Extract process name
	if procName, ok := report["procName"].(string); ok {
		crash.ProcessName = procName
	}

	// Extract process path
	if procPath, ok := report["procPath"].(string); ok {
		crash.ProcessPath = procPath
	}

	// Extract exception type
	if exception, ok := report["exception"].(map[string]interface{}); ok {
		if exType, ok := exception["type"].(string); ok {
			crash.ExceptionType = exType
		}
	}

	// Extract termination signal
	if termination, ok := report["termination"].(map[string]interface{}); ok {
		if code, ok := termination["code"].(float64); ok {
			crash.Signal = int(code)
		}
	}

	// Try to get timestamp from crash report
	if timestamp, ok := report["captureTime"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			crash.Timestamp = t
		}
	}

	return crash, nil
}
