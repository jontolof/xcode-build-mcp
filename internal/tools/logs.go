package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type CaptureLogs struct {
	name        string
	description string
	schema      map[string]interface{}
}

func NewCaptureLogs() *CaptureLogs {
	schema := createJSONSchema("object", map[string]interface{}{
		"udid": map[string]interface{}{
			"type":        "string",
			"description": "UDID of the target simulator or device (optional for auto-detection)",
		},
		"device_type": map[string]interface{}{
			"type":        "string",
			"description": "Device type filter for auto-selection if UDID not provided",
		},
		"bundle_id": map[string]interface{}{
			"type":        "string",
			"description": "Bundle identifier to filter logs for specific app",
		},
		"log_level": map[string]interface{}{
			"type":        "string",
			"description": "Log level filter (error, fault, info, debug)",
		},
		"filter_text": map[string]interface{}{
			"type":        "string",
			"description": "Text filter for log messages",
		},
		"max_lines": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum number of log lines to capture (default: 100)",
		},
		"timeout_secs": map[string]interface{}{
			"type":        "integer",
			"description": "Timeout in seconds for log capture (default: 30)",
		},
	}, []string{})

	return &CaptureLogs{
		name:        "capture_logs",
		description: "Capture and stream device/simulator logs with filtering and real-time monitoring capabilities",
		schema:      schema,
	}
}

func (t *CaptureLogs) Name() string {
	return t.name
}

func (t *CaptureLogs) Description() string {
	return t.description
}

func (t *CaptureLogs) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *CaptureLogs) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	var p types.LogCaptureParams
	
	// Parse parameters from args
	if udid, exists := args["udid"]; exists {
		if str, ok := udid.(string); ok {
			p.UDID = str
		}
	}
	if deviceType, exists := args["device_type"]; exists {
		if str, ok := deviceType.(string); ok {
			p.DeviceType = str
		}
	}
	if bundleID, exists := args["bundle_id"]; exists {
		if str, ok := bundleID.(string); ok {
			p.BundleID = str
		}
	}
	if logLevel, exists := args["log_level"]; exists {
		if str, ok := logLevel.(string); ok {
			p.LogLevel = str
		}
	}
	if filterText, exists := args["filter_text"]; exists {
		if str, ok := filterText.(string); ok {
			p.FilterText = str
		}
	}
	if maxLines, exists := args["max_lines"]; exists {
		if num, ok := maxLines.(float64); ok {
			p.MaxLines = int(num)
		}
	}
	if timeoutSecs, exists := args["timeout_secs"]; exists {
		if num, ok := timeoutSecs.(float64); ok {
			p.TimeoutSecs = int(num)
		}
	}

	start := time.Now()

	// Auto-select device if not specified
	if p.UDID == "" && p.DeviceType == "" {
		simulator, err := selectBestSimulator("")
		if err != nil {
			result := &types.LogCaptureResult{
				Success:  false,
				Duration: time.Since(start),
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), fmt.Errorf("failed to auto-select device: %w", err)
		}
		p.UDID = simulator.UDID
		p.DeviceType = simulator.DeviceType
	}

	// Set defaults
	if p.MaxLines == 0 {
		p.MaxLines = 100
	}
	if p.TimeoutSecs == 0 {
		p.TimeoutSecs = 30
	}

	result, err := t.captureLogs(ctx, &p)
	if err != nil {
		errorResult := &types.LogCaptureResult{
			Success:  false,
			Duration: time.Since(start),
		}
		resultJSON, _ := json.Marshal(errorResult)
		return string(resultJSON), err
	}

	result.Duration = time.Since(start)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

func (t *CaptureLogs) captureLogs(ctx context.Context, params *types.LogCaptureParams) (*types.LogCaptureResult, error) {
	// Build log command arguments
	args := []string{"simctl", "spawn"}
	
	if params.UDID != "" {
		args = append(args, params.UDID)
	} else {
		return nil, fmt.Errorf("device UDID is required")
	}

	args = append(args, "log", "stream")

	// Add filtering options
	if params.BundleID != "" {
		args = append(args, "--predicate", fmt.Sprintf("process == '%s'", params.BundleID))
	}

	if params.LogLevel != "" {
		switch strings.ToLower(params.LogLevel) {
		case "error":
			args = append(args, "--level", "error")
		case "fault":
			args = append(args, "--level", "fault")
		case "info":
			args = append(args, "--level", "info")
		case "debug":
			args = append(args, "--level", "debug")
		default:
			args = append(args, "--level", "default")
		}
	}

	// Add text filtering if specified
	if params.FilterText != "" {
		// Use grep-style filtering for text content
		args = append(args, "--predicate", fmt.Sprintf("eventMessage CONTAINS '%s'", params.FilterText))
	}

	// Set timeout context
	timeout := time.Duration(params.TimeoutSecs) * time.Second
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute log command
	cmd := exec.CommandContext(cmdCtx, "xcrun", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log command: %w", err)
	}

	// Parse log output
	var logEntries []types.LogEntry
	scanner := bufio.NewScanner(stdout)
	lineCount := 0
	truncated := false

	// Regex patterns for parsing log lines
	logPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}\.\d+[+-]\d{4})\s+(\w+)\s+(\w+)\s+(\[.*?\])?\s*(.*)$`)

	for scanner.Scan() && lineCount < params.MaxLines {
		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := t.parseLogLine(line, logPattern)
		if entry != nil {
			logEntries = append(logEntries, *entry)
			lineCount++
		}

		// Check for cancellation
		select {
		case <-cmdCtx.Done():
			break
		default:
		}
	}

	// Check if we hit the line limit
	if lineCount >= params.MaxLines && scanner.Scan() {
		truncated = true
	}

	// Stop the command
	if err := cmd.Process.Kill(); err != nil && !strings.Contains(err.Error(), "process already finished") {
		// Log warning but don't fail
	}

	// Wait for command to exit
	cmd.Wait()

	return &types.LogCaptureResult{
		Success:   true,
		LogLines:  logEntries,
		Truncated: truncated,
	}, nil
}

func (t *CaptureLogs) parseLogLine(line string, pattern *regexp.Regexp) *types.LogEntry {
	matches := pattern.FindStringSubmatch(line)
	if len(matches) < 6 {
		// Fallback for lines that don't match the expected format
		return &types.LogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "info",
			Message:   line,
		}
	}

	timestamp := matches[1]
	level := matches[2]
	category := matches[3]
	process := matches[4]
	message := matches[5]

	// Clean up process field
	if process != "" {
		process = strings.Trim(process, "[]")
	}

	// Parse timestamp to RFC3339 format
	parsedTime, err := time.Parse("2006-01-02 15:04:05.000000-0700", timestamp)
	if err == nil {
		timestamp = parsedTime.Format(time.RFC3339)
	}

	return &types.LogEntry{
		Timestamp: timestamp,
		Level:     strings.ToLower(level),
		Category:  category,
		Process:   process,
		Message:   strings.TrimSpace(message),
	}
}


