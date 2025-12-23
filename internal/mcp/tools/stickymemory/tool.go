// Package stickymemory provides the MCP stickymemory tool implementation.
package stickymemory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zombiekit/brains/internal/memory"
)

// Tool implements the stickymemory MCP tool.
type Tool struct {
	storage memory.Storage
}

// NewTool creates a new stickymemory tool with the given storage backend.
func NewTool(storage memory.Storage) *Tool {
	return &Tool{storage: storage}
}

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "stickymemory",
		Description: "Persistent memory storage for saving and retrieving information between sessions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "The operation to perform",
					"enum":        []string{"get", "set", "list", "delete", "search", "clear"},
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name/key of the memory item",
					"pattern":     "^[a-zA-Z0-9._-]+$",
					"maxLength":   255,
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to store (required for 'set' operation)",
					"maxLength":   1048576,
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of items to return (for 'list' and 'search' operations)",
					"minimum":     1,
					"maximum":     100,
					"default":     10,
				},
			},
			"required": []string{"operation"},
		},
	}
}

// Execute runs the tool with the given arguments.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return "", fmt.Errorf("operation is required")
	}

	switch operation {
	case "get":
		return t.handleGet(ctx, args)
	case "set":
		return t.handleSet(ctx, args)
	case "list":
		return t.handleList(ctx, args)
	case "delete":
		return t.handleDelete(ctx, args)
	case "search":
		return t.handleSearch(ctx, args)
	case "clear":
		return t.handleClear(ctx)
	default:
		return "", fmt.Errorf("invalid operation: %s", operation)
	}
}

func (t *Tool) handleGet(ctx context.Context, args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required for get operation")
	}

	result, err := t.storage.Get(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to get memory: %w", err)
	}

	if !result.HasValue() {
		return "", fmt.Errorf("memory not found: %s", name)
	}

	item := result.Value()
	response := map[string]interface{}{
		"name":       item.Name,
		"content":    item.Content,
		"version":    item.Version,
		"created_at": item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at": item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return toJSON(response)
}

func (t *Tool) handleSet(ctx context.Context, args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required for set operation")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required for set operation")
	}

	// Validate content size
	if len(content) > memory.MaxContentSize {
		return "", fmt.Errorf("content too large: maximum size is 1MB")
	}

	if err := t.storage.Set(ctx, name, content); err != nil {
		return "", fmt.Errorf("failed to set memory: %w", err)
	}

	// Get the version of the newly set item
	result, err := t.storage.Get(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	version := 1
	if result.HasValue() {
		version = result.Value().Version
	}

	response := map[string]interface{}{
		"success": true,
		"name":    memory.SanitizeName(name),
		"version": version,
	}

	return toJSON(response)
}

func (t *Tool) handleList(ctx context.Context, args map[string]interface{}) (string, error) {
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	items, err := t.storage.List(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to list memories: %w", err)
	}

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	response := make([]map[string]interface{}, len(items))
	for i, item := range items {
		response[i] = map[string]interface{}{
			"name":       item.Name,
			"size":       item.Size,
			"version":    item.Version,
			"updated_at": item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return toJSON(response)
}

func (t *Tool) handleDelete(ctx context.Context, args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required for delete operation")
	}

	if err := t.storage.Delete(ctx, name); err != nil {
		return "", fmt.Errorf("failed to delete memory: %w", err)
	}

	response := map[string]interface{}{
		"success": true,
		"name":    memory.SanitizeName(name),
	}

	return toJSON(response)
}

func (t *Tool) handleSearch(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["name"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("name (search query) is required for search operation")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	items, err := t.storage.List(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to search memories: %w", err)
	}

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	response := make([]map[string]interface{}, len(items))
	for i, item := range items {
		response[i] = map[string]interface{}{
			"name":       item.Name,
			"size":       item.Size,
			"version":    item.Version,
			"updated_at": item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return toJSON(response)
}

func (t *Tool) handleClear(ctx context.Context) (string, error) {
	count, err := t.storage.Clear(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to clear memories: %w", err)
	}

	response := map[string]interface{}{
		"success": true,
		"count":   count,
	}

	return toJSON(response)
}

func toJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(data), nil
}
