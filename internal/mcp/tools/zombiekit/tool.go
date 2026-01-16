// Package zombiekit provides the MCP feature tool implementation.
package zombiekit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// TemplateFilePath is the relative path from user home directory to the template file.
const TemplateFilePath = ".brains/templates/step.feature.md"

// Tool implements the ZombieKit feature MCP tool.
type Tool struct{}

// NewTool creates a new ZombieKit feature tool.
func NewTool() *Tool {
	return &Tool{}
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
		Name:        "feature",
		Description: "Returns the contents of the step feature template file (~/.brains/templates/step.feature.md)",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// Execute runs the tool and returns the template file contents.
// It takes no parameters and returns the contents of ~/.brains/templates/step.feature.md.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// Resolve home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}

	// Build full path
	filePath := filepath.Join(homeDir, TemplateFilePath)

	// Read file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("file not found (path: %s)", filePath)
		}
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("permission denied (path: %s)", filePath)
		}
		return "", fmt.Errorf("failed to read file (path: %s): %w", filePath, err)
	}

	return string(content), nil
}
