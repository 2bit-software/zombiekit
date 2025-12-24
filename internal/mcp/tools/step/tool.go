// Package step provides the MCP step tool implementation.
package step

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	internalStep "github.com/zombiekit/brains/internal/step"
)

// Tool implements the MCP step tool for executing workflow steps.
type Tool struct {
	embeddedFS fs.FS
}

// NewTool creates a new step tool.
func NewTool() *Tool {
	return &Tool{}
}

// SetEmbeddedFS sets the embedded filesystem for default steps.
func (t *Tool) SetEmbeddedFS(fsys fs.FS) {
	t.embeddedFS = fsys
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
		Name:        "step",
		Description: "Execute a workflow step within an initiative. Returns directive text, history folder path, files to read, and composed profile prompt.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"step": map[string]interface{}{
					"type":        "string",
					"description": "Step name to execute. Built-in steps: init, feature, specify, plan, tasks, implement, audit, clarify, complete",
				},
				"dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory containing the .brains folder. Used for profile resolution and initiative state.",
				},
				"initiative": map[string]interface{}{
					"type":        "string",
					"description": "Optional: Override the current active initiative. Path relative to history/ folder (e.g., '675d8a3f-feature-user-auth')",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"feature", "bug", "refactor"},
					"description": "Initiative type. Required for 'init' and 'feature' steps when creating new initiative.",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name/slug for the new initiative or cycle (e.g., 'user-auth'). Required for 'init' and 'feature' steps.",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Optional: Description of the feature or initiative.",
				},
				"new_initiative": map[string]interface{}{
					"type":        "boolean",
					"description": "Optional: Force creation of a new initiative even if one is active. Default false.",
				},
			},
			"required": []string{"step", "dir"},
		},
	}
}

// Execute runs the step tool and returns the step response as JSON.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract required parameters
	stepName := getStringArg(args, "step")
	if stepName == "" {
		return "", fmt.Errorf("missing required parameter: step")
	}

	dir := getStringArg(args, "dir")
	if dir == "" {
		return "", fmt.Errorf("missing required parameter: dir")
	}

	// Check if directory is initialized
	brainsDir := filepath.Join(dir, ".brains")
	if _, err := os.Stat(brainsDir); os.IsNotExist(err) {
		return "", &StepToolError{
			Code:       "NOT_INITIALIZED",
			Message:    "directory does not contain a .brains folder",
			Suggestion: "Run 'brains init' in the project directory first",
		}
	}

	// Create step service
	svc, err := internalStep.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating step service: %w", err)
	}

	// Set embedded filesystem if available
	if t.embeddedFS != nil {
		svc.SetEmbeddedFS(t.embeddedFS)
	}

	// Build execution options
	opts := &internalStep.ExecuteOptions{
		Initiative:    getStringArg(args, "initiative"),
		Type:          getStringArg(args, "type"),
		Name:          getStringArg(args, "name"),
		Description:   getStringArg(args, "description"),
		NewInitiative: getBoolArg(args, "new_initiative"),
	}

	// Execute the step
	resp, err := svc.Execute(stepName, opts)
	if err != nil {
		// Check if it's a StepError
		if stepErr, ok := err.(*internalStep.StepError); ok {
			return "", &StepToolError{
				Code:       stepErr.Code,
				Message:    stepErr.Message,
				Suggestion: stepErr.Hint,
			}
		}
		return "", err
	}

	// Update state with current step (ignore errors)
	_ = svc.UpdateState(stepName, opts.Initiative)

	// Convert response to JSON
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding response: %w", err)
	}

	return string(jsonData), nil
}

// getStringArg extracts a string argument from the args map.
func getStringArg(args map[string]interface{}, key string) string {
	if val, ok := args[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolArg extracts a boolean argument from the args map.
func getBoolArg(args map[string]interface{}, key string) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// StepToolError represents an error in the step tool with an error code.
type StepToolError struct {
	Code       string
	Message    string
	Suggestion string
}

func (e *StepToolError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
