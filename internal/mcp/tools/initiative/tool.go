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

	internalInit "github.com/zombiekit/brains/internal/initiative"
	"github.com/zombiekit/brains/internal/step"
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
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Definition returns the tool definition for MCP registration.
func (t *Tool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "initiative",
		Description: "Manage workflow initiative lifecycle. Actions: create (start new initiative), status (check current initiative), complete (finish initiative), list (show all initiatives).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"create", "status", "complete", "list"},
					"description": "The lifecycle action to perform",
				},
				"dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory containing the .brains folder",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"feature", "bug", "refactor"},
					"description": "Required for create: Type of initiative",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Required for create: Name/slug for the initiative (e.g., 'user-auth')",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Optional for create: Description of the initiative",
				},
			},
			"required": []string{"action", "dir"},
		},
	}
}

// Execute runs the initiative tool and returns the response as JSON.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
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
func (t *Tool) handleCreate(ctx context.Context, dir string, args map[string]interface{}) (string, error) {
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
		cyclePath, cycleID := t.findFirstCycle(existing.Path)

		// Still run template copy (safe - skips existing files with content)
		skipped, copied, err := t.copyTemplatesToCycle(dir, cyclePath)
		if err != nil {
			return "", fmt.Errorf("copying templates: %w", err)
		}

		resp := CreateResponse{
			Action:         "create",
			InitiativeID:   existing.ID,
			InitiativePath: existing.Path,
			CycleID:        cycleID,
			CyclePath:      cyclePath,
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

	// Check if a DIFFERENT initiative is active (error case)
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

	// Load workflow steps for the initiative type
	stepSvc, err := step.NewService(dir)
	if err != nil {
		return "", fmt.Errorf("creating step service: %w", err)
	}
	stepSvc.SetEmbeddedFS(t.embeddedFS)

	workflowSteps, err := stepSvc.GetWorkflowSteps(initType)
	if err != nil {
		// Non-fatal: proceed without steps if workflow not found
		workflowSteps = nil
	}

	// Convert step.WorkflowStep to initiative.WorkflowStep
	var initSteps []internalInit.WorkflowStep
	for _, ws := range workflowSteps {
		initSteps = append(initSteps, internalInit.WorkflowStep{
			Name:    ws.Name,
			Profile: ws.Profile,
		})
	}

	// Create the initiative
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

	// Create cycle within initiative
	cycleType := mapInitTypeToCycleType(internalInit.InitiativeType(initType))
	cycle, err := initSvc.CreateCycle(initiative.Path, cycleType, name)
	if err != nil {
		return "", fmt.Errorf("creating cycle: %w", err)
	}

	// Copy templates to cycle folder
	skipped, copied, err := t.copyTemplatesToCycle(dir, cycle.Path)
	if err != nil {
		return "", fmt.Errorf("copying templates: %w", err)
	}

	// Create git branch
	gitSvc := step.NewGitService(dir)
	branchName := initiative.ID
	_ = gitSvc.EnsureBranch(initType, name) // Ignore errors - git operations fail gracefully

	// Determine next step based on initiative type
	nextStep := initType
	if initType == "feature" || initType == "bug" || initType == "refactor" {
		nextStep = initType
	}

	resp := CreateResponse{
		Action:         "create",
		InitiativeID:   initiative.ID,
		InitiativePath: initiative.Path,
		CycleID:        cycle.ID,
		CyclePath:      cycle.Path,
		Branch:         branchName,
		Type:           initType,
		Name:           name,
		NextStep:       nextStep,
		AlreadyExisted: false,
		SkippedFiles:   skipped,
		CopiedFiles:    copied,
	}

	return marshalResponse(resp)
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
		CycleID:        status.CycleID,
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

// findFirstCycle finds the first cycle folder in an initiative directory.
// Returns the full path and cycle ID. If no cycle is found, returns the initiative path.
// Cycles are identified by the presence of spec.md or research.md files.
func (t *Tool) findFirstCycle(initiativePath string) (cyclePath, cycleID string) {
	entries, err := os.ReadDir(initiativePath)
	if err != nil {
		return initiativePath, "" // Fall back to initiative path
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			// Check if it looks like a cycle folder (contains spec.md or research.md)
			candidatePath := filepath.Join(initiativePath, entry.Name())
			if _, err := os.Stat(filepath.Join(candidatePath, "spec.md")); err == nil {
				return candidatePath, entry.Name()
			}
			if _, err := os.Stat(filepath.Join(candidatePath, "research.md")); err == nil {
				return candidatePath, entry.Name()
			}
		}
	}

	// No cycle found, use initiative path
	return initiativePath, ""
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

// copyTemplatesToCycle copies spec and research templates to the cycle folder.
// Returns lists of skipped and copied file names for idempotency reporting.
func (t *Tool) copyTemplatesToCycle(workDir, cyclePath string) (skipped, copied []string, err error) {
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

		destPath := filepath.Join(cyclePath, tmpl.dest)
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

// mapInitTypeToCycleType converts an initiative type to a cycle type.
func mapInitTypeToCycleType(t internalInit.InitiativeType) internalInit.CycleType {
	switch t {
	case internalInit.TypeFeature:
		return internalInit.CycleFeat
	case internalInit.TypeRefactor:
		return internalInit.CycleRef
	case internalInit.TypeBug:
		return internalInit.CycleFix
	default:
		return internalInit.CycleFeat
	}
}

// marshalResponse marshals a response to JSON.
func marshalResponse(resp interface{}) (string, error) {
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
