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

type Screenshot struct {
	name        string
	description string
	schema      map[string]interface{}
}

func NewScreenshot() *Screenshot {
	schema := createJSONSchema("object", map[string]interface{}{
		"udid": map[string]interface{}{
			"type":        "string",
			"description": "UDID of the target simulator or device (optional for auto-detection)",
		},
		"device_type": map[string]interface{}{
			"type":        "string",
			"description": "Device type filter for auto-selection if UDID not provided",
		},
		"output_path": map[string]interface{}{
			"type":        "string",
			"description": "Output path for the screenshot file",
		},
		"format": map[string]interface{}{
			"type":        "string",
			"description": "Image format (png, jpg) - default: png",
		},
	}, []string{})

	return &Screenshot{
		name:        "screenshot",
		description: "Capture screenshots from iOS/tvOS/watchOS simulators with automatic naming and format support",
		schema:      schema,
	}
}

func (t *Screenshot) Name() string {
	return t.name
}

func (t *Screenshot) Description() string {
	return t.description
}

func (t *Screenshot) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *Screenshot) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// Validate that parameters are provided
	if len(args) == 0 {
		return "", fmt.Errorf("parameters cannot be empty")
	}

	var p types.ScreenshotParams

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
	if outputPath, exists := args["output_path"]; exists {
		if str, ok := outputPath.(string); ok {
			p.OutputPath = str
		}
	}
	if format, exists := args["format"]; exists {
		if str, ok := format.(string); ok {
			p.Format = str
		}
	}

	start := time.Now()

	// Auto-select device if not specified
	if p.UDID == "" && p.DeviceType == "" {
		simulator, err := selectBestSimulator("")
		if err != nil {
			errorResult := &types.ScreenshotResult{
				Success:  false,
				Duration: time.Since(start),
			}
			resultJSON, _ := json.Marshal(errorResult)
			return string(resultJSON), fmt.Errorf("failed to auto-select device: %w", err)
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
		errorResult := &types.ScreenshotResult{
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

func (t *Screenshot) captureScreenshot(ctx context.Context, params *types.ScreenshotParams) (*types.ScreenshotResult, error) {
	if params.UDID == "" {
		return &types.ScreenshotResult{Success: false}, fmt.Errorf("device UDID is required")
	}

	// Validate format
	supportedFormats := map[string]bool{
		"png":  true,
		"jpeg": true,
		"jpg":  true,
	}

	if !supportedFormats[strings.ToLower(params.Format)] {
		return &types.ScreenshotResult{Success: false}, fmt.Errorf("unsupported format: %s (supported: png, jpeg, jpg)", params.Format)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(params.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return &types.ScreenshotResult{Success: false}, fmt.Errorf("failed to create output directory: %w", err)
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
		return &types.ScreenshotResult{Success: false}, fmt.Errorf("screenshot failed: %w\nOutput: %s", err, string(output))
	}

	// Get file info
	fileInfo, err := os.Stat(params.OutputPath)
	if err != nil {
		return &types.ScreenshotResult{Success: false}, fmt.Errorf("failed to get file info: %w", err)
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
