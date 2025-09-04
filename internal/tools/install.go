package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/internal/common"
	"github.com/jontolof/xcode-build-mcp/internal/xcode"
	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type InstallAppTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewInstallAppTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *InstallAppTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"app_path": map[string]interface{}{
			"type":        "string",
			"description": "Path to the .app bundle or .ipa file to install",
		},
		"udid": map[string]interface{}{
			"type":        "string",
			"description": "UDID of the target simulator or device (optional for auto-detection)",
		},
		"device_type": map[string]interface{}{
			"type":        "string",
			"description": "Device type filter for auto-selection if UDID not provided",
		},
		"replace": map[string]interface{}{
			"type":        "boolean",
			"description": "Replace the app if it's already installed (default: true)",
		},
	}, []string{"app_path"})

	return &InstallAppTool{
		name:        "install_app",
		description: "Install iOS/tvOS/watchOS apps on simulators or devices",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *InstallAppTool) Name() string {
	return t.name
}

func (t *InstallAppTool) Description() string {
	return t.description
}

func (t *InstallAppTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *InstallAppTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	params, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	// Smart path resolution for "." or directory paths
	if params.AppPath == "." || params.AppPath == "./" {
		appPath, err := t.findAppBundle(".")
		if err != nil {
			return "", fmt.Errorf("no .app bundle found in current directory. Build your project first with xcode_build, then specify the exact path to the .app bundle")
		}
		params.AppPath = appPath
		t.logger.Printf("Auto-detected app bundle: %s", appPath)
	}

	// Validate it's actually a .app bundle
	if !strings.HasSuffix(params.AppPath, ".app") {
		return "", fmt.Errorf("invalid app path: %s. Must be a .app bundle, not a directory", params.AppPath)
	}

	// Debug logging if enabled
	if os.Getenv("MCP_DEBUG") == "true" {
		fmt.Printf("DEBUG: install_app called with params: %+v\n", params)
	}

	t.logger.Printf("Installing app %s", params.AppPath)

	// Validate app path exists
	if _, err := os.Stat(params.AppPath); os.IsNotExist(err) {
		return "", fmt.Errorf("app path does not exist: %s", params.AppPath)
	}

	// Resolve target device if not provided
	targetUDID := params.UDID
	if targetUDID == "" {
		detectedUDID, err := t.selectBestDevice(ctx, params.DeviceType)
		if err != nil {
			return "", fmt.Errorf("failed to auto-detect device: %w", err)
		}
		targetUDID = detectedUDID
		t.logger.Printf("Auto-selected device: %s", targetUDID)
	}

	// Extract bundle ID from app
	bundleID, err := t.extractBundleID(ctx, params.AppPath)
	if err != nil {
		t.logger.Printf("Warning: Could not extract bundle ID: %v", err)
		bundleID = "Unknown"
	}

	// Perform the installation
	output, err := t.installApp(ctx, params.AppPath, targetUDID, params.Replace)
	if err != nil {
		return "", fmt.Errorf("failed to install app: %w", err)
	}

	duration := time.Since(start)
	t.logger.Printf("Successfully installed app %s (bundle: %s) in %v", params.AppPath, bundleID, duration)

	result := &types.AppInstallResult{
		Success:  true,
		Duration: duration,
		Output:   output,
		BundleID: bundleID,
	}

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

func (t *InstallAppTool) parseParams(args map[string]interface{}) (*types.AppInstallParams, error) {
	params := &types.AppInstallParams{
		Replace: true, // Default to true for convenience
	}

	// Parse app_path (required)
	appPath, err := parseStringParam(args, "app_path", true)
	if err != nil {
		return nil, err
	}

	// Expand path if needed
	if strings.HasPrefix(appPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			appPath = strings.Replace(appPath, "~", homeDir, 1)
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(appPath)
	if err == nil {
		params.AppPath = absPath
	} else {
		params.AppPath = appPath
	}

	// Parse udid (optional)
	if udid, err := parseStringParam(args, "udid", false); err != nil {
		return nil, err
	} else if udid != "" {
		params.UDID = udid
	}

	// Parse device_type (optional)
	if deviceType, err := parseStringParam(args, "device_type", false); err != nil {
		return nil, err
	} else if deviceType != "" {
		params.DeviceType = deviceType
	}

	// Parse replace (optional)
	params.Replace = parseBoolParam(args, "replace", true)

	return params, nil
}

// findAppBundle searches for .app bundles in common build directories
func (t *InstallAppTool) findAppBundle(dir string) (string, error) {
	// Look in common build output directories
	searchPaths := []string{
		"build/Debug-iphonesimulator/*.app",
		"build/Release-iphonesimulator/*.app",
		"DerivedData/*/Build/Products/Debug-iphonesimulator/*.app",
		"DerivedData/*/Build/Products/Release-iphonesimulator/*.app",
		"*.app",
	}

	for _, pattern := range searchPaths {
		fullPattern := filepath.Join(dir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err == nil && len(matches) > 0 {
			// Return the first match, preferring Debug over Release
			for _, match := range matches {
				if strings.Contains(match, "Debug") {
					return match, nil
				}
			}
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no .app bundle found in build directories")
}

func (t *InstallAppTool) selectBestDevice(ctx context.Context, deviceTypeFilter string) (string, error) {
	// Get list of available booted simulators
	args := []string{"xcrun", "simctl", "list", "devices", "--json"}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return "", err
	}

	if !result.Success() {
		return "", fmt.Errorf("failed to list devices: %s", result.StderrOutput)
	}

	// Parse JSON output
	var simctlOutput struct {
		Devices map[string][]struct {
			UDID         string `json:"udid"`
			Name         string `json:"name"`
			State        string `json:"state"`
			IsAvailable  bool   `json:"isAvailable"`
			DeviceTypeId string `json:"deviceTypeIdentifier"`
		} `json:"devices"`
	}

	if err := json.Unmarshal([]byte(result.Output), &simctlOutput); err != nil {
		return "", fmt.Errorf("failed to parse device list: %w", err)
	}

	// Find the best device (prefer booted iOS simulators)
	var candidates []struct {
		UDID       string
		Name       string
		DeviceType string
		State      string
		Score      int // Higher is better
	}

	for runtime, devices := range simctlOutput.Devices {
		platform := "iOS"
		if strings.Contains(strings.ToLower(runtime), "watchos") {
			platform = "watchOS"
		} else if strings.Contains(strings.ToLower(runtime), "tvos") {
			platform = "tvOS"
		}

		for _, device := range devices {
			if !device.IsAvailable {
				continue
			}

			deviceType := extractDeviceTypeFromId(device.DeviceTypeId)

			// Apply device type filter if provided
			if deviceTypeFilter != "" && !strings.Contains(strings.ToLower(deviceType), strings.ToLower(deviceTypeFilter)) {
				continue
			}

			score := 0

			// Prefer iOS devices
			if platform == "iOS" {
				score += 100
			} else if platform == "tvOS" {
				score += 50
			}

			// Prefer booted devices
			if device.State == "Booted" {
				score += 200
			} else if device.State == "Shutdown" {
				score += 10
			}

			// Prefer iPhone over iPad
			if strings.Contains(deviceType, "iPhone") {
				score += 30
			} else if strings.Contains(deviceType, "iPad") {
				score += 20
			}

			candidates = append(candidates, struct {
				UDID       string
				Name       string
				DeviceType string
				State      string
				Score      int
			}{
				UDID:       device.UDID,
				Name:       device.Name,
				DeviceType: deviceType,
				State:      device.State,
				Score:      score,
			})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no suitable devices found")
	}

	// Sort by score (highest first)
	bestCandidate := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Score > bestCandidate.Score {
			bestCandidate = candidate
		}
	}

	t.logger.Printf("Selected device: %s (%s) - State: %s", bestCandidate.Name, bestCandidate.DeviceType, bestCandidate.State)
	return bestCandidate.UDID, nil
}

func extractDeviceTypeFromId(deviceTypeId string) string {
	// Extract device type from identifier like "com.apple.CoreSimulator.SimDeviceType.iPhone-15-Pro"
	parts := strings.Split(deviceTypeId, ".")
	if len(parts) > 0 {
		deviceType := parts[len(parts)-1]
		// Convert from "iPhone-15-Pro" to "iPhone 15 Pro"
		deviceType = strings.ReplaceAll(deviceType, "-", " ")
		return deviceType
	}
	return deviceTypeId
}

func (t *InstallAppTool) installApp(ctx context.Context, appPath, udid string, replace bool) (string, error) {
	args := []string{"xcrun", "simctl", "install", udid, appPath}

	// If replace is false, we might need to uninstall first
	// However, simctl install typically replaces by default, so we'll proceed

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return "", err
	}

	if !result.Success() {
		errorOutput := result.StderrOutput
		if errorOutput == "" {
			errorOutput = result.Output
		}

		// Check for common error cases and provide better error messages
		if strings.Contains(errorOutput, "Unable to install") {
			return "", fmt.Errorf("unable to install app: %s", strings.TrimSpace(errorOutput))
		} else if strings.Contains(errorOutput, "device is not booted") {
			return "", fmt.Errorf("target device is not booted, please boot the simulator first")
		} else if strings.Contains(errorOutput, "No such file or directory") {
			return "", fmt.Errorf("app bundle not found at path: %s", appPath)
		} else if strings.Contains(errorOutput, "Invalid app") {
			return "", fmt.Errorf("invalid app bundle: %s", strings.TrimSpace(errorOutput))
		}

		return "", fmt.Errorf("installation failed (exit code %d): %s", result.ExitCode, strings.TrimSpace(errorOutput))
	}

	return result.Output, nil
}

func (t *InstallAppTool) extractBundleID(ctx context.Context, appPath string) (string, error) {
	// Check if it's a .app bundle or .ipa
	ext := filepath.Ext(appPath)

	var infoPlistPath string
	if ext == ".app" {
		// For .app bundles, Info.plist is directly in the bundle
		infoPlistPath = filepath.Join(appPath, "Info.plist")
	} else if ext == ".ipa" {
		// For .ipa files, we would need to extract the Info.plist
		// This is more complex, so we'll skip it for now
		return "", fmt.Errorf(".ipa bundle ID extraction not implemented")
	} else {
		return "", fmt.Errorf("unsupported app format: %s", ext)
	}

	// Use plutil to read the bundle identifier
	args := []string{"plutil", "-p", infoPlistPath}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return "", err
	}

	if !result.Success() {
		return "", fmt.Errorf("failed to read Info.plist: %s", result.StderrOutput)
	}

	// Parse the plist output to find CFBundleIdentifier
	lines := strings.Split(result.Output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "CFBundleIdentifier") {
			// Look for the next line which should contain the value
			parts := strings.SplitN(line, "=>", 2)
			if len(parts) == 2 {
				bundleID := strings.TrimSpace(parts[1])
				bundleID = strings.Trim(bundleID, `"`)
				return bundleID, nil
			}
		}
	}

	return "", fmt.Errorf("CFBundleIdentifier not found in Info.plist")
}
