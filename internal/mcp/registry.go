package mcp

import (
	"context"
	"fmt"
	"sync"
)

type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tools, name)
}

func (r *Registry) GetTool(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[name]
}

func (r *Registry) ListTools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}

	return tools
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

type BaseTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

func NewBaseTool(name, description string, schema map[string]interface{}) BaseTool {
	return BaseTool{
		name:        name,
		description: description,
		schema:      schema,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) InputSchema() map[string]interface{} {
	return t.schema
}

func ParseStringParam(args map[string]interface{}, key string, required bool) (string, error) {
	value, exists := args[key]
	if !exists {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", key)
	}

	return str, nil
}

func ParseBoolParam(args map[string]interface{}, key string, defaultValue bool) bool {
	value, exists := args[key]
	if !exists {
		return defaultValue
	}

	boolVal, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return boolVal
}

func ParseArrayParam(args map[string]interface{}, key string) ([]interface{}, error) {
	value, exists := args[key]
	if !exists {
		return nil, nil
	}

	array, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an array", key)
	}

	return array, nil
}

func CreateJSONSchema(schemaType string, properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       schemaType,
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}