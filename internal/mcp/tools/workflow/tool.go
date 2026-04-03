// Package workflow provides the MCP workflow-load tool.
package workflow

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/2bit-software/zombiekit/internal/workflow"
)

// Tool implements the MCP workflow-load tool.
type Tool struct {
	commandsFS  fs.FS
	workflowsFS fs.FS
}

// NewTool creates a new workflow Tool.
// commandsFS is the embedded FS for commands (embed/commands/).
// workflowsFS is the embedded FS for workflows (embed/workflows/).
func NewTool(commandsFS, workflowsFS fs.FS) *Tool {
	return &Tool{
		commandsFS:  commandsFS,
		workflowsFS: workflowsFS,
	}
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "workflow-load",
		Description: "Load a command or workflow by name. Commands are slash-command entry points (new, next, complete, help). Workflows are multi-step orchestrations (feature, bug, refactor, feature-light, unmanaged).",
	}
}

// ToolDefinition describes an MCP tool.
type ToolDefinition struct {
	Name        string
	Description string
}

// HandleLoad loads and returns a command or workflow by name.
func (t *Tool) HandleLoad(ctx context.Context, args map[string]any) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	contentType, _ := args["type"].(string)
	if contentType == "" {
		contentType = "workflow"
	}

	workingDir := getWorkingDir(args)

	var subdir string
	var embeddedFS fs.FS

	switch contentType {
	case "command":
		subdir = "commands"
		embeddedFS = t.commandsFS
	case "workflow":
		subdir = "workflows"
		embeddedFS = t.workflowsFS
	default:
		return "", fmt.Errorf("type must be %q or %q, got %q", "command", "workflow", contentType)
	}

	svc, err := workflow.NewServiceForSubdir(workingDir, subdir, embeddedFS)
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
