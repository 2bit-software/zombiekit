package step

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	zombiekit "github.com/2bit-software/zombiekit"
	"github.com/2bit-software/zombiekit/internal/initiative"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/adrg/frontmatter"
)

// Compile-time check that fs.FS is still used (for GetWorkflowSteps method).
var _ fs.FS = (fs.FS)(nil)

// stepPrerequisites defines the prerequisites for steps that require them.
var stepPrerequisites = map[string]StepPrerequisite{
	"plan": {
		RequiredArtifact: "spec.md",
		RequiredStatus:   "approved",
		Hint:             "Run feature, bug, or refactor first and approve the spec",
		BlockingStep:     "feature",
	},
	"tasks": {
		RequiredArtifact: "plan.md",
		RequiredStatus:   "approved",
		Hint:             "Run plan first and approve the plan",
		BlockingStep:     "plan",
	},
	"implement": {
		RequiredArtifact: "tasks.md",
		RequiredStatus:   "", // No status check, just existence
		Hint:             "Run tasks first to generate the task list",
		BlockingStep:     "tasks",
	},
}

// Service provides step execution and management functionality.
type Service struct {
	workDir      string
	loader       *Loader
	stateManager *initiative.FileStateManager
	profileSvc   *profile.Service
}

// ExecuteOptions provides optional parameters for step execution.
type ExecuteOptions struct {
	// Initiative overrides the active initiative (path relative to history/).
	Initiative string
}

// NewService creates a new step service for the given working directory.
func NewService(workDir string) (*Service, error) {
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	stateManager, err := initiative.NewStateManager(workDir)
	if err != nil {
		return nil, fmt.Errorf("creating state manager: %w", err)
	}

	profileSvc, err := profile.NewService(workDir)
	if err != nil {
		// Profile service may fail if not initialized, but we can still work
		profileSvc = nil
	}

	return &Service{
		workDir:      workDir,
		loader:       NewLoader(workDir),
		stateManager: stateManager,
		profileSvc:   profileSvc,
	}, nil
}

// SetGlobalDir overrides the global directory (useful for testing).
func (s *Service) SetGlobalDir(dir string) {
	s.loader.SetGlobalDir(dir)
}

// GetStep retrieves a step definition by name.
func (s *Service) GetStep(name string) (*Step, error) {
	return s.loader.Get(name)
}

// ListSteps returns all available step definitions.
func (s *Service) ListSteps() ([]*Step, error) {
	return s.loader.List()
}

// Execute runs a step and returns the structured response.
// The response includes the directive, history folder, files to read, and composed prompt.
// All steps now require an active initiative (created via the initiative tool).
func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
	step, err := s.loader.Get(stepName)
	if err != nil {
		return nil, err
	}

	historyFolder, err := s.resolveInitiativeFolder(opts)
	if err != nil {
		return nil, err
	}

	if err := s.checkPrerequisite(stepName, historyFolder); err != nil {
		return nil, err
	}

	return s.buildResponse(step, historyFolder, stepName), nil
}

// buildResponse assembles a StepResponse from the resolved step, initiative
// folder, and step name. It handles profile composition, workflow phases, and
// next-task detection for the implement step.
func (s *Service) buildResponse(step *Step, historyFolder, stepName string) *StepResponse {
	resp := &StepResponse{
		Directive:        step.Directive,
		HistoryFolder:    historyFolder,
		InitiativeFolder: historyFolder,
		FilesToRead:      s.resolveFiles(step.Files, historyFolder),
		ComposedPrompt:   s.composeProfiles(step.Profiles),
		Prerequisites:    PrerequisiteInfo{Met: true},
	}

	if stepName == "feature" || stepName == "bug" || stepName == "refactor" {
		resp.WorkflowPhases = buildWorkflowPhases()
	}

	if stepName == "implement" {
		resp.NextTask = s.findNextTask(historyFolder)
		if resp.NextTask == nil {
			resp.Directive = "All tasks complete! Run 'initiative complete' to finish."
		}
	}

	return resp
}

// composeProfiles runs profile composition and returns the composed content,
// or an empty string if composition is unavailable or fails.
func (s *Service) composeProfiles(profiles []string) string {
	if s.profileSvc == nil || len(profiles) == 0 {
		return ""
	}
	result, err := s.profileSvc.Compose(profiles)
	if err != nil {
		return ""
	}
	return result.Content
}

// resolveInitiativeFolder determines the initiative folder path from an explicit
// override or the active initiative state. Returns the absolute path to the
// history folder for the resolved initiative.
func (s *Service) resolveInitiativeFolder(opts *ExecuteOptions) (string, error) {
	if opts != nil && opts.Initiative != "" {
		folder := filepath.Join(s.workDir, "history", opts.Initiative)
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			return "", &StepError{
				Code:    "INITIATIVE_NOT_FOUND",
				Message: fmt.Sprintf("initiative '%s' not found in history/", opts.Initiative),
				Hint:    "Check the initiative path or use 'initiative create' to create a new one",
			}
		}
		return folder, nil
	}

	state, err := s.stateManager.Load()
	if err != nil {
		return "", fmt.Errorf("loading initiative state: %w", err)
	}

	if state.IsEmpty() {
		return "", &StepError{
			Code:    "NO_ACTIVE_INITIATIVE",
			Message: "no active initiative",
			Hint:    "Use 'initiative create' to start a new initiative first",
		}
	}

	return filepath.Join(s.workDir, state.Initiative), nil
}

// resolveFiles expands glob patterns relative to the history folder.
// Returns absolute paths to all matching files.
func (s *Service) resolveFiles(patterns []string, historyFolder string) []string {
	if historyFolder == "" || len(patterns) == 0 {
		return []string{}
	}

	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Make pattern absolute relative to history folder
		absPattern := filepath.Join(historyFolder, pattern)

		matches, err := filepath.Glob(absPattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}

		for _, match := range matches {
			// Skip directories
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}

			// Deduplicate
			if !seen[match] {
				seen[match] = true
				files = append(files, match)
			}
		}
	}

	return files
}

// WorkDir returns the service's working directory.
func (s *Service) WorkDir() string {
	return s.workDir
}

// GetWorkflowSteps returns the default steps for a workflow type.
// It parses the workflow profile frontmatter to extract the step sequence.
func (s *Service) GetWorkflowSteps(workflowType string) ([]WorkflowStep, error) {
	// Map workflow types to profile names
	profileName := workflowType
	switch workflowType {
	case "feature", "bug", "refactor":
		// These are the valid workflow types
	default:
		return nil, fmt.Errorf("unknown workflow type: %s", workflowType)
	}

	// Try to load the profile content
	var content []byte
	var err error

	// Check local profiles first
	localPath := filepath.Join(s.workDir, ".brains", "profiles", profileName+".md")
	content, err = os.ReadFile(localPath)
	if err != nil {
		// Try global profiles
		homeDir, _ := os.UserHomeDir()
		globalPath := filepath.Join(homeDir, ".brains", "profiles", profileName+".md")
		content, err = os.ReadFile(globalPath)
		if err != nil {
			// Fall back to embedded profiles
			content, err = fs.ReadFile(zombiekit.EmbeddedProfiles, "profiles/"+profileName+".md")
			if err != nil {
				return nil, fmt.Errorf("profile not found: %s", profileName)
			}
		}
	}

	// Parse the frontmatter
	var meta WorkflowMeta
	_, err = frontmatter.Parse(strings.NewReader(string(content)), &meta)
	if err != nil {
		return nil, fmt.Errorf("parsing profile frontmatter: %w", err)
	}

	return meta.Steps, nil
}

// UpdateState updates the step status in INITIATIVE.md after step execution.
// It marks the previous in-progress step as completed and the new step as in-progress.
func (s *Service) UpdateState(stepName string, initiativeID string) error {
	// Load active initiative state
	state, err := s.stateManager.Load()
	if err != nil || state.IsEmpty() {
		return nil // No active initiative, nothing to update
	}

	// Parse INITIATIVE.md
	initiativePath := filepath.Join(s.workDir, state.Initiative)
	mdPath := filepath.Join(initiativePath, "INITIATIVE.md")

	parsed, err := initiative.ParseInitiativeMD(mdPath)
	if err != nil {
		return nil // Can't parse, skip update
	}

	// Update step status: mark current in-progress as completed, new step as in-progress
	now := time.Now().Format("2006-01-02 15:04")

	// First, complete any in-progress step
	for i := range parsed.Steps {
		if parsed.Steps[i].Status == initiative.StepInProgress {
			parsed.Steps[i].Status = initiative.StepCompleted
			parsed.Steps[i].Updated = now
		}
	}

	// Then mark the new step as in-progress
	for i := range parsed.Steps {
		if parsed.Steps[i].Name == stepName {
			parsed.Steps[i].Status = initiative.StepInProgress
			parsed.Steps[i].Updated = now
			break
		}
	}

	// Write back to INITIATIVE.md
	return parsed.WriteTo(mdPath)
}

// findNextTask parses tasks.md and finds the first unchecked task.
func (s *Service) findNextTask(initiativePath string) *TaskInfo {
	tasksPath := filepath.Join(initiativePath, "tasks.md")
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	currentPhase := ""

	for _, line := range lines {
		// Track current phase (## Phase X: ...)
		if strings.HasPrefix(line, "## Phase") || strings.HasPrefix(line, "## ") {
			currentPhase = strings.TrimPrefix(line, "## ")
			currentPhase = strings.TrimSpace(currentPhase)
		}

		// Find unchecked task: - [ ] TXXX ...
		if strings.Contains(line, "- [ ]") {
			// Extract task ID and description
			// Format: - [ ] T001 Description here
			trimmed := strings.TrimSpace(line)
			trimmed = strings.TrimPrefix(trimmed, "- [ ]")
			trimmed = strings.TrimSpace(trimmed)

			// Split into ID and description
			parts := strings.SplitN(trimmed, " ", 2)
			taskID := ""
			description := trimmed
			if len(parts) >= 1 {
				taskID = parts[0]
			}
			if len(parts) >= 2 {
				description = parts[1]
			}

			return &TaskInfo{
				ID:          taskID,
				Description: description,
				Phase:       currentPhase,
			}
		}
	}

	return nil // All tasks complete
}

// checkPrerequisite validates that a step's prerequisite is met.
// Returns nil if the prerequisite is met or the step has no prerequisite.
// Returns a StepError with code PREREQUISITE_NOT_MET if not met.
func (s *Service) checkPrerequisite(stepName string, initiativePath string) error {
	prereq, exists := stepPrerequisites[stepName]
	if !exists {
		return nil // No prerequisite for this step
	}

	artifactPath := filepath.Join(initiativePath, prereq.RequiredArtifact)

	// Check if artifact exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return &StepError{
			Code:    "PREREQUISITE_NOT_MET",
			Message: fmt.Sprintf("required artifact '%s' not found", prereq.RequiredArtifact),
			Hint:    prereq.Hint,
		}
	}

	// If no status check required, we're done
	if prereq.RequiredStatus == "" {
		return nil
	}

	// Check frontmatter status
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		return &StepError{
			Code:    "PREREQUISITE_NOT_MET",
			Message: fmt.Sprintf("cannot read artifact '%s': %v", prereq.RequiredArtifact, err),
			Hint:    prereq.Hint,
		}
	}

	var meta struct {
		Status string `yaml:"status"`
	}
	_, err = frontmatter.Parse(strings.NewReader(string(content)), &meta)
	if err != nil {
		// If frontmatter parsing fails, check if status is not in frontmatter format
		return &StepError{
			Code:    "PREREQUISITE_NOT_MET",
			Message: fmt.Sprintf("artifact '%s' has no valid frontmatter", prereq.RequiredArtifact),
			Hint:    prereq.Hint,
		}
	}

	if meta.Status != prereq.RequiredStatus {
		return &StepError{
			Code:    "PREREQUISITE_NOT_MET",
			Message: fmt.Sprintf("artifact '%s' has status '%s', requires '%s'", prereq.RequiredArtifact, meta.Status, prereq.RequiredStatus),
			Hint:    prereq.Hint,
		}
	}

	return nil
}
