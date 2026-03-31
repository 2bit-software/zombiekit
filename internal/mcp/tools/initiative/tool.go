package initiative

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	internalInit "github.com/2bit-software/zombiekit/internal/initiative"
	"github.com/2bit-software/zombiekit/internal/step"
)

// Tool implements the MCP initiative tool for managing workflow initiatives.
type Tool struct {
	embeddedFS fs.FS
}

// NewTool creates a new initiative tool.
func NewTool() *Tool {
	return &Tool{}
}

// SetEmbeddedFS sets the embedded filesystem for templates.
func (t *Tool) SetEmbeddedFS(fsys fs.FS) {
	t.embeddedFS = fsys
}

// ToolDefinition represents an MCP tool definition.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "initiative",
		Description: "Manage workflow initiative lifecycle. Actions: create (start new initiative), status (check current initiative), complete (finish initiative), list (show all initiatives).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"create", "status", "complete", "list"},
					"description": "The lifecycle action to perform",
				},
				"dir": map[string]any{
					"type":        "string",
					"description": "Working directory containing the .brains folder",
				},
				"type": map[string]any{
					"type":        "string",
					"enum":        []string{"feature", "bug", "refactor"},
					"description": "Required for create: Type of initiative",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Required for create: Name/slug for the initiative (e.g., 'user-auth')",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Optional for create: Description of the initiative",
				},
			},
			"required": []string{"action", "dir"},
		},
	}
}

// Execute runs the initiative tool and returns the response as JSON.
func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action := getStringArg(args, "action")
	if action == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: action",
			Hint:    "Provide action (create|status|complete|list)",
		}
	}

	dir := getStringArg(args, "dir")
	if dir == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: dir",
			Hint:    "Provide dir (working directory with .brains folder)",
		}
	}

	// Check if directory is initialized
	brainsDir := filepath.Join(dir, ".brains")
	if _, err := os.Stat(brainsDir); os.IsNotExist(err) {
		return "", &ToolError{
			Code:    "NOT_INITIALIZED",
			Message: "directory does not contain a .brains folder",
			Hint:    "Run 'brains init' in the project directory first",
		}
	}

	switch action {
	case "create":
		return t.handleCreate(ctx, dir, args)
	case "status":
		return t.handleStatus(ctx, dir)
	case "complete":
		return t.handleComplete(ctx, dir)
	case "list":
		return t.handleList(ctx, dir)
	default:
		return "", &ToolError{
			Code:    "INVALID_ACTION",
			Message: fmt.Sprintf("invalid action: '%s'", action),
			Hint:    "Valid actions: create, status, complete, list",
		}
	}
}

// handleCreate handles the create action.
// This implementation is idempotent: calling create with the same name+type
// returns the existing initiative instead of creating a duplicate.
func (t *Tool) handleCreate(ctx context.Context, dir string, args map[string]any) (string, error) {
	initType := getStringArg(args, "type")
	if initType == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: type",
			Hint:    "Provide type (feature|bug|refactor) for create action",
		}
	}

	name := getStringArg(args, "name")
	if name == "" {
		return "", &ToolError{
			Code:    "MISSING_REQUIRED_PARAM",
			Message: "missing required parameter: name",
			Hint:    "Provide name (e.g., 'user-auth') for create action",
		}
	}

	// Create initiative service
	initSvc, err := internalInit.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating initiative service: %w", err)
	}

	// IDEMPOTENCY CHECK: See if active initiative matches the requested name+type
	existing, err := initSvc.FindActiveByNameAndType(name, internalInit.InitiativeType(initType))
	if err != nil {
		return "", fmt.Errorf("checking for existing initiative: %w", err)
	}

	if existing != nil {
		// Idempotent case: return existing initiative
		// Still run template copy (safe - skips existing files with content)
		skipped, copied, err := t.copyTemplatesToInitiative(dir, existing.Path)
		if err != nil {
			return "", fmt.Errorf("copying templates: %w", err)
		}

		resp := CreateResponse{
			Action:         "create",
			InitiativeID:   existing.ID,
			InitiativePath: existing.Path,
			Branch:         existing.ID,
			Type:           initType,
			Name:           name,
			NextStep:       initType,
			AlreadyExisted: true,
			SkippedFiles:   skipped,
			CopiedFiles:    copied,
		}
		return marshalResponse(resp)
	}

	return t.createNewInitiative(dir, initSvc, initType, name)
}

// createNewInitiative handles the second half of create: validates no conflicting
// active initiative exists, loads workflow steps, creates the initiative record,
// copies templates, and creates the git branch.
func (t *Tool) createNewInitiative(dir string, initSvc *internalInit.Service, initType, name string) (string, error) {
	active, err := initSvc.GetActive()
	if err != nil {
		return "", fmt.Errorf("checking active initiative: %w", err)
	}
	if active != nil {
		return "", &ToolError{
			Code:    "INITIATIVE_ALREADY_ACTIVE",
			Message: fmt.Sprintf("a different initiative is already active: %s", active.ID),
			Hint:    "Complete or abandon the current initiative first with 'initiative complete'",
		}
	}

	initSteps, err := loadWorkflowSteps(dir, initType)
	if err != nil {
		return "", err
	}

	initiative, err := initSvc.Create(internalInit.InitiativeType(initType), name, initSteps)
	if err != nil {
		if initErr, ok := err.(*internalInit.InitiativeError); ok {
			return "", &ToolError{
				Code:    initErr.Code,
				Message: initErr.Message,
				Hint:    initErr.Hint,
			}
		}
		return "", err
	}

	skipped, copied, err := t.copyTemplatesToInitiative(dir, initiative.Path)
	if err != nil {
		return "", fmt.Errorf("copying templates: %w", err)
	}

	// Best-effort git branch creation
	gitSvc := step.NewGitService(dir)
	_ = gitSvc.EnsureBranch(initType, name)

	resp := CreateResponse{
		Action:         "create",
		InitiativeID:   initiative.ID,
		InitiativePath: initiative.Path,
		Branch:         initiative.ID,
		Type:           initType,
		Name:           name,
		NextStep:       initType,
		AlreadyExisted: false,
		SkippedFiles:   skipped,
		CopiedFiles:    copied,
	}

	return marshalResponse(resp)
}

// loadWorkflowSteps loads and converts workflow steps for an initiative type.
// Returns nil steps (not an error) when no workflow is defined.
func loadWorkflowSteps(dir, initType string) ([]internalInit.WorkflowStep, error) {
	stepSvc, err := step.NewService(dir)
	if err != nil {
		return nil, fmt.Errorf("creating step service: %w", err)
	}

	workflowSteps, err := stepSvc.GetWorkflowSteps(initType)
	if err != nil {
		return nil, nil
	}

	initSteps := make([]internalInit.WorkflowStep, len(workflowSteps))
	for i, ws := range workflowSteps {
		initSteps[i] = internalInit.WorkflowStep{
			Name:    ws.Name,
			Profile: ws.Profile,
		}
	}
	return initSteps, nil
}

// handleStatus handles the status action.
func (t *Tool) handleStatus(ctx context.Context, dir string) (string, error) {
	initSvc, err := internalInit.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating initiative service: %w", err)
	}

	status, err := initSvc.Status()
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	resp := StatusResponse{
		Action:         "status",
		Active:         status.Active,
		InitiativeID:   status.InitiativeID,
		InitiativeType: status.InitiativeType,
		CurrentStep:    status.CurrentStep,
		AvailableDocs:  status.AvailableDocs,
		SuggestedNext:  status.SuggestedNext,
		HistoryPath:    status.HistoryPath,
		InitiativeFile: status.InitiativeFile,
		Files:          status.Files,
	}

	return marshalResponse(resp)
}

// handleComplete handles the complete action.
func (t *Tool) handleComplete(ctx context.Context, dir string) (string, error) {
	initSvc, err := internalInit.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating initiative service: %w", err)
	}

	// Get active initiative first for the response
	active, err := initSvc.GetActive()
	if err != nil {
		return "", fmt.Errorf("getting active initiative: %w", err)
	}
	if active == nil {
		return "", &ToolError{
			Code:    "NO_ACTIVE_INITIATIVE",
			Message: "no active initiative to complete",
			Hint:    "Use 'initiative create' to start a new initiative first",
		}
	}

	initiativeID := active.ID

	// Complete the initiative
	if err := initSvc.Complete(); err != nil {
		if initErr, ok := err.(*internalInit.InitiativeError); ok {
			return "", &ToolError{
				Code:    initErr.Code,
				Message: initErr.Message,
				Hint:    initErr.Hint,
			}
		}
		return "", err
	}

	resp := CompleteResponse{
		Action:       "complete",
		InitiativeID: initiativeID,
		CompletedAt:  time.Now(),
	}

	return marshalResponse(resp)
}

// handleList handles the list action.
func (t *Tool) handleList(ctx context.Context, dir string) (string, error) {
	initSvc, err := internalInit.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating initiative service: %w", err)
	}

	initiatives, err := initSvc.List()
	if err != nil {
		return "", fmt.Errorf("listing initiatives: %w", err)
	}

	summaries := make([]InitiativeSummary, len(initiatives))
	for i, init := range initiatives {
		summaries[i] = InitiativeSummary{
			ID:     init.ID,
			Type:   string(init.Type),
			Name:   init.Name,
			Status: string(init.Status),
			Path:   init.Path,
		}
	}

	resp := ListResponse{
		Action:      "list",
		Initiatives: summaries,
	}

	return marshalResponse(resp)
}

// copyTemplateIfNotExists copies template content to destination if destination doesn't exist
// or is empty/whitespace-only. Returns whether the file was copied.
// This enables idempotent template copying - existing content is preserved.
func copyTemplateIfNotExists(templateContent []byte, destPath string) (copied bool, err error) {
	// Check if destination exists
	if _, err := os.Stat(destPath); err == nil {
		// File exists - check if it has non-whitespace content
		content, err := os.ReadFile(destPath)
		if err != nil {
			return false, fmt.Errorf("reading existing file %s: %w", destPath, err)
		}
		if len(bytes.TrimSpace(content)) > 0 {
			return false, nil // Skip - file has content
		}
		// File is empty or whitespace-only, fall through to overwrite
	} else if !os.IsNotExist(err) {
		// Unexpected error (not just "file doesn't exist")
		return false, fmt.Errorf("checking file %s: %w", destPath, err)
	}

	// Copy template (file doesn't exist OR is empty/whitespace)
	if err := os.WriteFile(destPath, templateContent, 0644); err != nil {
		return false, fmt.Errorf("writing template to %s: %w", destPath, err)
	}
	return true, nil
}

// copyTemplatesToInitiative copies spec and research templates to the initiative folder.
// Returns lists of skipped and copied file names for idempotency reporting.
func (t *Tool) copyTemplatesToInitiative(workDir, initiativePath string) (skipped, copied []string, err error) {
	embFS := t.embeddedFS
	if embFS == nil {
		embFS = step.GetTemplateFS()
	}
	if embFS == nil {
		return nil, nil, fmt.Errorf("no embedded template filesystem available")
	}

	templates := []struct {
		src  string
		dest string
	}{
		{"templates/spec-template.md", "spec.md"},
		{"templates/research-template.md", "research.md"},
	}

	for _, tmpl := range templates {
		// First check if local override exists
		localPath := filepath.Join(workDir, ".brains", "templates", filepath.Base(tmpl.src))
		var content []byte

		if _, statErr := os.Stat(localPath); statErr == nil {
			content, err = os.ReadFile(localPath)
		} else {
			content, err = fs.ReadFile(embFS, tmpl.src)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("reading template %s: %w", tmpl.src, err)
		}

		destPath := filepath.Join(initiativePath, tmpl.dest)
		wasCopied, copyErr := copyTemplateIfNotExists(content, destPath)
		if copyErr != nil {
			return nil, nil, fmt.Errorf("copying template %s: %w", tmpl.dest, copyErr)
		}

		if wasCopied {
			copied = append(copied, tmpl.dest)
		} else {
			skipped = append(skipped, tmpl.dest)
		}
	}

	return skipped, copied, nil
}

// marshalResponse marshals a response to JSON.
func marshalResponse(resp any) (string, error) {
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding response: %w", err)
	}
	return string(jsonData), nil
}

// getStringArg extracts a string argument from the args map.
func getStringArg(args map[string]any, key string) string {
	if val, ok := args[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// ToolError represents an error in the initiative tool with an error code.
type ToolError struct {
	Code    string
	Message string
	Hint    string
}

func (e *ToolError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
