// Package workflow provides the MCP workflow-compose tool.
package workflow

import (
	"context"
	"fmt"

	"github.com/2bit-software/zombiekit/internal/workflow"
)

// Tool implements the MCP workflow-compose tool.
type Tool struct{}

// NewTool creates a new workflow Tool.
func NewTool() *Tool {
	return &Tool{}
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "workflow-compose",
		Description: "Load a workflow by name. Workflows are entry points for starting work.",
	}
}

// ToolDefinition describes an MCP tool.
type ToolDefinition struct {
	Name        string
	Description string
}

// HandleCompose loads and returns a workflow by name.
func (t *Tool) HandleCompose(ctx context.Context, args map[string]any) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	workingDir := getWorkingDir(args)
	svc, err := workflow.NewService(workingDir)
	if err != nil {
		return "", fmt.Errorf("initializing workflow service: %w", err)
	}

	wf, err := svc.Load(name)
	if err != nil {
		return "", formatError(err)
	}

	return wf.Content, nil
}

// getWorkingDir extracts the working_directory parameter from args.
func getWorkingDir(args map[string]any) string {
	if wd, ok := args["working_directory"].(string); ok && wd != "" {
		return wd
	}
	return ""
}

// formatError formats workflow errors.
func formatError(err error) error {
	switch e := err.(type) {
	case *workflow.WorkflowNotFoundError:
		return fmt.Errorf("workflow %q not found", e.Name)
	default:
		return err
	}
}
