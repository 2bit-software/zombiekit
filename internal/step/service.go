package step

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/zombiekit/brains/internal/initiative"
	"github.com/zombiekit/brains/internal/profile"
)

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
	"eat": {
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
	// Type is required for the "feature", "bug", and "refactor" steps.
	Type string
	// Name is required for the "feature", "bug", and "refactor" steps.
	Name string
	// Description is an optional description for the initiative.
	Description string
	// NewInitiative forces creation of a new initiative even if one is active.
	NewInitiative bool
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

// SetEmbeddedFS sets the embedded filesystem for default steps.
func (s *Service) SetEmbeddedFS(fsys fs.FS) {
	s.loader.SetEmbeddedFS(fsys)
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
func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
	// Load the step definition
	step, err := s.loader.Get(stepName)
	if err != nil {
		return nil, err
	}

	// Handle feature step specially - it creates initiatives with cycles
	if stepName == "feature" {
		return s.executeFeatureStep(step, opts)
	}

	// Handle bug step - creates bug-type initiative
	if stepName == "bug" {
		return s.executeBugStep(step, opts)
	}

	// Handle refactor step - creates refactor-type initiative
	if stepName == "refactor" {
		return s.executeRefactorStep(step, opts)
	}

	// Handle complete step specially - it marks initiative as complete
	if stepName == "complete" {
		return s.executeCompleteStep(step, opts)
	}

	// Get the initiative context
	var historyFolder string
	var cyclePath string

	// Check for initiative override
	if opts != nil && opts.Initiative != "" {
		historyFolder = filepath.Join(s.workDir, "history", opts.Initiative)
		if _, err := os.Stat(historyFolder); os.IsNotExist(err) {
			return nil, &StepError{
				Code:    "INITIATIVE_NOT_FOUND",
				Message: fmt.Sprintf("initiative '%s' not found in history/", opts.Initiative),
				Hint:    "Check the initiative path or use 'feature' to create a new one",
			}
		}
		cyclePath = historyFolder // Use initiative folder as cycle path for override
	} else {
		// Load active initiative from state
		state, err := s.stateManager.Load()
		if err != nil {
			return nil, fmt.Errorf("loading initiative state: %w", err)
		}

		if state.IsEmpty() {
			return nil, &StepError{
				Code:    "NO_ACTIVE_INITIATIVE",
				Message: "no active initiative",
				Hint:    "Run step='feature', 'bug', or 'refactor' with a name parameter to create a new initiative",
			}
		}

		historyFolder = filepath.Join(s.workDir, state.Initiative)
		// Use cycle path if available, otherwise fall back to initiative folder
		if state.Cycle != "" {
			cyclePath = filepath.Join(s.workDir, state.Cycle)
		} else {
			cyclePath = historyFolder
		}
	}

	// Check prerequisites for steps that require them
	if err := s.checkPrerequisite(stepName, cyclePath); err != nil {
		return nil, err
	}

	// Resolve file patterns to actual files
	filesToRead := s.resolveFiles(step.Files, cyclePath)

	// Compose profiles
	composedPrompt := ""
	if s.profileSvc != nil && len(step.Profiles) > 0 {
		result, err := s.profileSvc.Compose(step.Profiles)
		if err == nil {
			composedPrompt = result.Content
		}
		// If composition fails, continue with empty prompt
	}

	return &StepResponse{
		Directive:        step.Directive,
		HistoryFolder:    historyFolder,
		InitiativeFolder: historyFolder,
		CycleFolder:      cyclePath,
		FilesToRead:      filesToRead,
		ComposedPrompt:   composedPrompt,
	}, nil
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

// IsInitialized checks if the working directory has a .brains folder.
func (s *Service) IsInitialized() bool {
	brainsDir := filepath.Join(s.workDir, ".brains")
	_, err := os.Stat(brainsDir)
	return err == nil
}

// WorkDir returns the service's working directory.
func (s *Service) WorkDir() string {
	return s.workDir
}

// UpdateState updates the initiative state after step execution.
func (s *Service) UpdateState(stepName string, initiativeID string) error {
	state, err := s.stateManager.Load()
	if err != nil {
		return err
	}

	state.CurrentStep = stepName
	return s.stateManager.Save(state)
}

// executeBugStep handles the "bug" step that creates a bug-type initiative.
func (s *Service) executeBugStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
	if opts == nil {
		opts = &ExecuteOptions{}
	}
	opts.Type = "bug"
	return s.executeFeatureStep(step, opts)
}

// executeRefactorStep handles the "refactor" step that creates a refactor-type initiative.
func (s *Service) executeRefactorStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
	if opts == nil {
		opts = &ExecuteOptions{}
	}
	opts.Type = "refactor"
	return s.executeFeatureStep(step, opts)
}

// checkPrerequisite validates that a step's prerequisite is met.
// Returns nil if the prerequisite is met or the step has no prerequisite.
// Returns a StepError with code PREREQUISITE_NOT_MET if not met.
func (s *Service) checkPrerequisite(stepName string, cyclePath string) error {
	prereq, exists := stepPrerequisites[stepName]
	if !exists {
		return nil // No prerequisite for this step
	}

	artifactPath := filepath.Join(cyclePath, prereq.RequiredArtifact)

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

// executeCompleteStep handles the special "complete" step that marks an initiative as done.
func (s *Service) executeCompleteStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
	// Load active initiative
	state, err := s.stateManager.Load()
	if err != nil {
		return nil, fmt.Errorf("loading initiative state: %w", err)
	}
	if state.IsEmpty() {
		return nil, &StepError{
			Code:    "NO_ACTIVE_INITIATIVE",
			Message: "no active initiative to complete",
			Hint:    "There is no active initiative to mark as complete",
		}
	}

	historyFolder := filepath.Join(s.workDir, state.Initiative)

	// Create initiative service
	initSvc, err := initiative.NewService(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("creating initiative service: %w", err)
	}

	// Complete the initiative
	if err := initSvc.Complete(); err != nil {
		return nil, err
	}

	return &StepResponse{
		Directive:      step.Directive,
		HistoryFolder:  historyFolder,
		FilesToRead:    s.resolveFiles(step.Files, historyFolder),
		ComposedPrompt: "",
	}, nil
}
