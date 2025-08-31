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

type Screenshot struct{}

func (t *Screenshot) Name() string {
	return "screenshot"
}

func (t *Screenshot) Description() string {
	return "Capture screenshots from iOS/tvOS/watchOS simulators with automatic naming and format support"
}

func (t *Screenshot) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p types.ScreenshotParams
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
	}

	// Set default format
	if p.Format == "" {
		p.Format = "png"
	}

	// Generate output path if not specified
	if p.OutputPath == "" {
		p.OutputPath = t.generateScreenshotPath(p.Format)
	} else {
		// Ensure the output path has the correct extension
		if !strings.HasSuffix(strings.ToLower(p.OutputPath), "."+strings.ToLower(p.Format)) {
			p.OutputPath += "." + strings.ToLower(p.Format)
		}
	}

	result, err := t.captureScreenshot(ctx, &p)
	if err != nil {
		return &types.ScreenshotResult{
			Success:  false,
			Duration: time.Since(start),
		}, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (t *Screenshot) captureScreenshot(ctx context.Context, params *types.ScreenshotParams) (*types.ScreenshotResult, error) {
	if params.UDID == "" {
		return nil, fmt.Errorf("device UDID is required")
	}

	// Validate format
	supportedFormats := map[string]bool{
		"png":  true,
		"jpeg": true,
		"jpg":  true,
	}

	if !supportedFormats[strings.ToLower(params.Format)] {
		return nil, fmt.Errorf("unsupported format: %s (supported: png, jpeg, jpg)", params.Format)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(params.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build screenshot command
	args := []string{"simctl", "io", params.UDID, "screenshot"}
	
	// Add format specification for JPEG
	if strings.ToLower(params.Format) == "jpeg" || strings.ToLower(params.Format) == "jpg" {
		args = append(args, "--type", "jpeg")
	}
	
	args = append(args, params.OutputPath)

	// Execute screenshot command
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w\nOutput: %s", err, string(output))
	}

	// Get file info
	fileInfo, err := os.Stat(params.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Get dimensions (if possible) using sips command on macOS
	dimensions := t.getImageDimensions(params.OutputPath)

	return &types.ScreenshotResult{
		Success:    true,
		FilePath:   params.OutputPath,
		FileSize:   fileInfo.Size(),
		Dimensions: dimensions,
	}, nil
}

func (t *Screenshot) generateScreenshotPath(format string) string {
	// Create screenshots directory in current working directory
	screenshotsDir := "screenshots"
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("simulator_screenshot_%s.%s", timestamp, strings.ToLower(format))
	return filepath.Join(screenshotsDir, filename)
}

func (t *Screenshot) getImageDimensions(imagePath string) string {
	// Use sips command to get image dimensions on macOS
	cmd := exec.Command("sips", "-g", "pixelWidth", "-g", "pixelHeight", imagePath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse sips output
	lines := strings.Split(string(output), "\n")
	var width, height string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pixelWidth:") {
			width = strings.TrimSpace(strings.TrimPrefix(line, "pixelWidth:"))
		} else if strings.HasPrefix(line, "pixelHeight:") {
			height = strings.TrimSpace(strings.TrimPrefix(line, "pixelHeight:"))
		}
	}

	if width != "" && height != "" {
		return fmt.Sprintf("%sx%s", width, height)
	}

	return ""
}