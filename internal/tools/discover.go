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

type DiscoverProjectsTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executor    *xcode.Executor
	parser      *xcode.Parser
	logger      common.Logger
}

func NewDiscoverProjectsTool(executor *xcode.Executor, parser *xcode.Parser, logger common.Logger) *DiscoverProjectsTool {
	schema := createJSONSchema("object", map[string]interface{}{
		"root_path": map[string]interface{}{
			"type":        "string",
			"description": "Root directory to search for Xcode projects (defaults to current directory)",
		},
		"max_depth": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum directory depth to search (default: 3)",
			"minimum":     1,
			"maximum":     10,
		},
		"include_hidden": map[string]interface{}{
			"type":        "boolean",
			"description": "Include hidden directories in search (default: false)",
		},
		"patterns": map[string]interface{}{
			"type":        "array",
			"description": "Custom file patterns to match (default: [\"*.xcodeproj\", \"*.xcworkspace\"])",
			"items": map[string]interface{}{
				"type": "string",
			},
		},
	}, []string{})

	return &DiscoverProjectsTool{
		name:        "discover_projects",
		description: "Discover Xcode projects and workspaces in a directory tree with metadata extraction",
		schema:      schema,
		executor:    executor,
		parser:      parser,
		logger:      logger,
	}
}

func (t *DiscoverProjectsTool) Name() string {
	return t.name
}

func (t *DiscoverProjectsTool) Description() string {
	return t.description
}

func (t *DiscoverProjectsTool) InputSchema() map[string]interface{} {
	return t.schema
}

func (t *DiscoverProjectsTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	discoveryParams, err := t.parseParams(args)
	if err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	start := time.Now()

	// Set root path default if not provided
	if discoveryParams.RootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		discoveryParams.RootPath = cwd
	}

	t.logger.Printf("Discovering projects in %s (max depth: %d)", discoveryParams.RootPath, discoveryParams.MaxDepth)

	projects, err := t.discoverProjects(ctx, *discoveryParams)
	if err != nil {
		return "", fmt.Errorf("project discovery failed: %w", err)
	}

	duration := time.Since(start)
	t.logger.Printf("Discovery completed in %v, found %d projects", duration, len(projects))

	result := &types.DiscoveryResult{
		Projects: projects,
		Duration: duration,
	}
	
	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	
	return string(resultJSON), nil
}

func (t *DiscoverProjectsTool) discoverProjects(ctx context.Context, params types.ProjectDiscovery) ([]types.ProjectInfo, error) {
	var projects []types.ProjectInfo
	seen := make(map[string]bool)

	err := t.walkDirectory(ctx, params.RootPath, 0, params, &projects, seen)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (t *DiscoverProjectsTool) walkDirectory(ctx context.Context, path string, depth int, params types.ProjectDiscovery, projects *[]types.ProjectInfo, seen map[string]bool) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check depth limit
	if depth > params.MaxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		// Log error but don't fail the entire discovery
		t.logger.Printf("Cannot read directory %s: %v", path, err)
		return nil
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		// Skip if already processed (handle symlinks)
		absPath, err := filepath.Abs(fullPath)
		if err == nil && seen[absPath] {
			continue
		}
		seen[absPath] = true

		// Skip hidden files/directories unless explicitly requested
		if !params.IncludeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check if this matches any of our patterns
		if t.matchesPatterns(entry.Name(), params.Patterns) {
			projectInfo, err := t.extractProjectInfo(ctx, fullPath)
			if err != nil {
				t.logger.Printf("Failed to extract info for %s: %v", fullPath, err)
				continue
			}
			*projects = append(*projects, projectInfo)
		}

		// Recurse into directories (but skip bundle directories like .xcodeproj)
		if entry.IsDir() && !t.isBundleDirectory(entry.Name()) {
			err := t.walkDirectory(ctx, fullPath, depth+1, params, projects, seen)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *DiscoverProjectsTool) matchesPatterns(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			t.logger.Printf("Invalid pattern %s: %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (t *DiscoverProjectsTool) isBundleDirectory(name string) bool {
	bundleExtensions := []string{".xcodeproj", ".xcworkspace", ".app", ".framework", ".bundle"}
	for _, ext := range bundleExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func (t *DiscoverProjectsTool) extractProjectInfo(ctx context.Context, projectPath string) (types.ProjectInfo, error) {
	info := types.ProjectInfo{
		Path: projectPath,
		Name: filepath.Base(projectPath),
	}

	// Determine project type
	if strings.HasSuffix(projectPath, ".xcworkspace") {
		info.Type = "workspace"
	} else if strings.HasSuffix(projectPath, ".xcodeproj") {
		info.Type = "project"
	} else {
		info.Type = "unknown"
	}

	// Get file modification time
	fileInfo, err := os.Stat(projectPath)
	if err == nil {
		info.LastModified = fileInfo.ModTime()
	}

	// Extract schemes and targets
	schemes, err := t.extractSchemes(ctx, projectPath, info.Type)
	if err != nil {
		t.logger.Printf("Failed to extract schemes for %s: %v", projectPath, err)
	} else {
		info.Schemes = schemes
	}

	targets, err := t.extractTargets(ctx, projectPath, info.Type)
	if err != nil {
		t.logger.Printf("Failed to extract targets for %s: %v", projectPath, err)
	} else {
		info.Targets = targets
	}

	return info, nil
}

func (t *DiscoverProjectsTool) extractSchemes(ctx context.Context, projectPath, projectType string) ([]string, error) {
	// Build xcodebuild command to list schemes
	args := []string{"xcodebuild", "-list"}
	
	if projectType == "workspace" {
		args = append(args, "-workspace", projectPath)
	} else {
		args = append(args, "-project", projectPath)
	}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	return t.parser.ParseSchemes(result.Output), nil
}

func (t *DiscoverProjectsTool) extractTargets(ctx context.Context, projectPath, projectType string) ([]string, error) {
	// Build xcodebuild command to list targets
	args := []string{"xcodebuild", "-list"}
	
	if projectType == "workspace" {
		args = append(args, "-workspace", projectPath)
	} else {
		args = append(args, "-project", projectPath)
	}

	result, err := t.executor.ExecuteCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	return t.parser.ParseTargets(result.Output), nil
}

func (t *DiscoverProjectsTool) parseParams(args map[string]interface{}) (*types.ProjectDiscovery, error) {
	params := &types.ProjectDiscovery{
		MaxDepth:      3,
		IncludeHidden: false,
		Patterns:      []string{"*.xcodeproj", "*.xcworkspace"},
	}

	// Parse root_path
	if rootPath, err := parseStringParam(args, "root_path", false); err != nil {
		return nil, err
	} else if rootPath != "" {
		params.RootPath = rootPath
	}

	// Parse max_depth
	if maxDepth, exists := args["max_depth"]; exists {
		if depth, ok := maxDepth.(float64); ok {
			params.MaxDepth = int(depth)
		} else if depth, ok := maxDepth.(int); ok {
			params.MaxDepth = depth
		} else {
			return nil, fmt.Errorf("max_depth must be a number")
		}
	}

	// Parse include_hidden
	params.IncludeHidden = parseBoolParam(args, "include_hidden", false)

	// Parse patterns
	if patterns, err := parseArrayParam(args, "patterns"); err != nil {
		return nil, err
	} else if patterns != nil {
		params.Patterns = []string{}
		for _, pattern := range patterns {
			if strPattern, ok := pattern.(string); ok {
				params.Patterns = append(params.Patterns, strPattern)
			}
		}
	}

	return params, nil
}