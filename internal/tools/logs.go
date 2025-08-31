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

type CaptureLogs struct{}

func (t *CaptureLogs) Name() string {
	return "capture_logs"
}

func (t *CaptureLogs) Description() string {
	return "Capture and stream device/simulator logs with filtering and real-time monitoring capabilities"
}

func (t *CaptureLogs) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p types.LogCaptureParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	// Auto-select device if not specified
	if p.UDID == "" && p.DeviceType == "" {
		simulator, err := selectBestSimulator("")
		if err != nil {
			return nil, fmt.Errorf("failed to auto-select device: %w", err)
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
		return &types.LogCaptureResult{
			Success:  false,
			Duration: time.Since(start),
		}, err
	}

	result.Duration = time.Since(start)
	return result, nil
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

