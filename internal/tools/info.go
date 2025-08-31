package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontolof/xcode-build-mcp/pkg/types"
)

type GetAppInfo struct{}

func (t *GetAppInfo) Name() string {
	return "get_app_info"
}

func (t *GetAppInfo) Description() string {
	return "Extract metadata from iOS/macOS app bundles including bundle ID, version, entitlements, and icon paths"
}

func (t *GetAppInfo) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p types.AppInfoParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	result, err := t.extractAppInfo(ctx, &p)
	if err != nil {
		return &types.AppInfoResult{
			Success:  false,
			Duration: time.Since(start),
		}, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (t *GetAppInfo) extractAppInfo(ctx context.Context, params *types.AppInfoParams) (*types.AppInfoResult, error) {
	// Validate parameters - need either app path or bundle ID + device
	if params.AppPath == "" && params.BundleID == "" {
		return nil, fmt.Errorf("either app_path or bundle_id must be specified")
	}

	if params.AppPath != "" {
		// Extract info from local app bundle
		return t.extractFromLocalBundle(ctx, params.AppPath)
	}

	// Extract info from installed app on device
	return t.extractFromInstalledApp(ctx, params)
}

func (t *GetAppInfo) extractFromLocalBundle(ctx context.Context, appPath string) (*types.AppInfoResult, error) {
	// Verify app bundle exists
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("app bundle not found: %s", appPath)
	}

	// Check if it's a valid app bundle
	if !strings.HasSuffix(appPath, ".app") && !strings.HasSuffix(appPath, ".ipa") {
		return nil, fmt.Errorf("invalid app bundle format (must be .app or .ipa): %s", appPath)
	}

	result := &types.AppInfoResult{Success: true}

	// Extract Info.plist path
	var infoPlistPath string
	if strings.HasSuffix(appPath, ".app") {
		infoPlistPath = filepath.Join(appPath, "Info.plist")
	} else {
		// For .ipa files, we need to extract the Info.plist
		return nil, fmt.Errorf(".ipa extraction not yet implemented - please extract the .app bundle first")
	}

	// Read Info.plist using plutil
	plistData, err := t.readInfoPlist(ctx, infoPlistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Info.plist: %w", err)
	}

	// Parse plist data
	if bundleID, ok := plistData["CFBundleIdentifier"].(string); ok {
		result.BundleID = bundleID
	}

	if version, ok := plistData["CFBundleShortVersionString"].(string); ok {
		result.Version = version
	}

	if buildNumber, ok := plistData["CFBundleVersion"].(string); ok {
		result.BuildNumber = buildNumber
	}

	if displayName, ok := plistData["CFBundleDisplayName"].(string); ok {
		result.DisplayName = displayName
	} else if displayName, ok := plistData["CFBundleName"].(string); ok {
		result.DisplayName = displayName
	}

	if minOSVersion, ok := plistData["MinimumOSVersion"].(string); ok {
		result.MinOSVersion = minOSVersion
	} else if minOSVersion, ok := plistData["LSMinimumSystemVersion"].(string); ok {
		result.MinOSVersion = minOSVersion
	}

	// Extract icon paths
	result.IconPaths = t.findIconFiles(appPath, plistData)

	// Extract entitlements if available
	entitlements, err := t.extractEntitlements(ctx, appPath)
	if err == nil {
		result.Entitlements = entitlements
	}

	return result, nil
}

func (t *GetAppInfo) extractFromInstalledApp(ctx context.Context, params *types.AppInfoParams) (*types.AppInfoResult, error) {
	// Auto-select device if not specified
	udid := params.UDID
	if udid == "" && params.DeviceType == "" {
		simulator, err := selectBestSimulator("")
		if err != nil {
			return nil, fmt.Errorf("failed to auto-select device: %w", err)
		}
		udid = simulator.UDID
	}

	if udid == "" {
		return nil, fmt.Errorf("device UDID is required for installed app info")
	}

	// Get app info from device using simctl
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "appinfo", udid, params.BundleID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get app info from device: %w\nOutput: %s", err, string(output))
	}

	// Parse simctl appinfo output (JSON format)
	var appInfo map[string]interface{}
	if err := json.Unmarshal(output, &appInfo); err != nil {
		// If JSON parsing fails, try to extract info from text output
		return t.parseAppInfoText(string(output), params.BundleID)
	}

	result := &types.AppInfoResult{
		Success:  true,
		BundleID: params.BundleID,
	}

	// Extract fields from JSON
	if bundleData, ok := appInfo[params.BundleID].(map[string]interface{}); ok {
		if version, ok := bundleData["CFBundleShortVersionString"].(string); ok {
			result.Version = version
		}
		if buildNumber, ok := bundleData["CFBundleVersion"].(string); ok {
			result.BuildNumber = buildNumber
		}
		if displayName, ok := bundleData["CFBundleDisplayName"].(string); ok {
			result.DisplayName = displayName
		}
		if minOSVersion, ok := bundleData["MinimumOSVersion"].(string); ok {
			result.MinOSVersion = minOSVersion
		}
	}

	return result, nil
}

func (t *GetAppInfo) readInfoPlist(ctx context.Context, plistPath string) (map[string]interface{}, error) {
	// Use plutil to convert plist to JSON
	cmd := exec.CommandContext(ctx, "plutil", "-convert", "json", "-o", "-", plistPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read plist: %w", err)
	}

	var plistData map[string]interface{}
	if err := json.Unmarshal(output, &plistData); err != nil {
		return nil, fmt.Errorf("failed to parse plist JSON: %w", err)
	}

	return plistData, nil
}

func (t *GetAppInfo) findIconFiles(appPath string, plistData map[string]interface{}) []string {
	var iconPaths []string

	// Look for icon entries in Info.plist
	iconKeys := []string{
		"CFBundleIcons",
		"CFBundleIconFiles",
		"CFBundleIconFile",
	}

	for _, key := range iconKeys {
		if iconData, exists := plistData[key]; exists {
			iconPaths = append(iconPaths, t.extractIconPathsFromData(iconData, appPath)...)
		}
	}

	// If no icons found in plist, look for common icon files
	if len(iconPaths) == 0 {
		commonIcons := []string{
			"AppIcon60x60@2x.png",
			"AppIcon60x60@3x.png",
			"Icon-60@2x.png",
			"Icon-60@3x.png",
			"Icon.png",
			"Icon@2x.png",
			"icon.png",
		}

		for _, iconName := range commonIcons {
			iconPath := filepath.Join(appPath, iconName)
			if _, err := os.Stat(iconPath); err == nil {
				iconPaths = append(iconPaths, iconPath)
			}
		}
	}

	return iconPaths
}

func (t *GetAppInfo) extractIconPathsFromData(iconData interface{}, appPath string) []string {
	var paths []string

	switch data := iconData.(type) {
	case string:
		// Single icon file
		iconPath := filepath.Join(appPath, data)
		if _, err := os.Stat(iconPath); err == nil {
			paths = append(paths, iconPath)
		}
	case []interface{}:
		// Array of icon files
		for _, item := range data {
			if iconName, ok := item.(string); ok {
				iconPath := filepath.Join(appPath, iconName)
				if _, err := os.Stat(iconPath); err == nil {
					paths = append(paths, iconPath)
				}
			}
		}
	case map[string]interface{}:
		// Icon dictionary (iOS style)
		if primaryIcons, ok := data["CFBundlePrimaryIcon"].(map[string]interface{}); ok {
			if iconFiles, ok := primaryIcons["CFBundleIconFiles"].([]interface{}); ok {
				for _, item := range iconFiles {
					if iconName, ok := item.(string); ok {
						iconPath := filepath.Join(appPath, iconName)
						if _, err := os.Stat(iconPath); err == nil {
							paths = append(paths, iconPath)
						}
					}
				}
			}
		}
	}

	return paths
}

func (t *GetAppInfo) extractEntitlements(ctx context.Context, appPath string) (map[string]interface{}, error) {
	// Try to extract entitlements using codesign
	entitlementsPath := filepath.Join(appPath, "embedded.mobileprovision")
	if _, err := os.Stat(entitlementsPath); err == nil {
		return t.extractEntitlementsFromMobileProvision(ctx, entitlementsPath)
	}

	// Alternative: try to get entitlements from the binary
	executableName := t.findExecutable(appPath)
	if executableName != "" {
		executablePath := filepath.Join(appPath, executableName)
		return t.extractEntitlementsFromBinary(ctx, executablePath)
	}

	return nil, fmt.Errorf("no entitlements found")
}

func (t *GetAppInfo) extractEntitlementsFromBinary(ctx context.Context, binaryPath string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "codesign", "-d", "--entitlements", ":-", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract entitlements: %w", err)
	}

	// Parse entitlements plist
	var entitlements map[string]interface{}
	if err := json.Unmarshal(output, &entitlements); err != nil {
		// Try parsing as plist
		return t.parsePlistString(string(output))
	}

	return entitlements, nil
}

func (t *GetAppInfo) extractEntitlementsFromMobileProvision(ctx context.Context, provisionPath string) (map[string]interface{}, error) {
	// Mobile provision files are complex - this is a simplified implementation
	return nil, fmt.Errorf("mobile provision parsing not yet implemented")
}

func (t *GetAppInfo) findExecutable(appPath string) string {
	// Try to find the main executable from Info.plist
	infoPlistPath := filepath.Join(appPath, "Info.plist")
	plistData, err := t.readInfoPlist(context.Background(), infoPlistPath)
	if err != nil {
		return ""
	}

	if executableName, ok := plistData["CFBundleExecutable"].(string); ok {
		return executableName
	}

	return ""
}

func (t *GetAppInfo) parsePlistString(plistString string) (map[string]interface{}, error) {
	// Simple plist string parsing - in a real implementation, you'd use a proper plist parser
	return nil, fmt.Errorf("plist string parsing not implemented")
}

func (t *GetAppInfo) parseAppInfoText(output string, bundleID string) (*types.AppInfoResult, error) {
	// Parse text output from simctl appinfo when JSON parsing fails
	result := &types.AppInfoResult{
		Success:  true,
		BundleID: bundleID,
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "CFBundleShortVersionString") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				result.Version = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		} else if strings.Contains(line, "CFBundleVersion") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				result.BuildNumber = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		} else if strings.Contains(line, "CFBundleDisplayName") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				result.DisplayName = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}
	}

	return result, nil
}