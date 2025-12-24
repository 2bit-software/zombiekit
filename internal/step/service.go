package step

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zombiekit/brains/internal/initiative"
	"github.com/zombiekit/brains/internal/profile"
)

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
	// Type is required for the "init" step.
	Type string
	// Name is required for the "init" step.
	Name string
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

	// Handle init step specially - it creates a new initiative
	if stepName == "init" {
		return s.executeInitStep(step, opts)
	}

	// Handle complete step specially - it marks initiative as complete
	if stepName == "complete" {
		return s.executeCompleteStep(step, opts)
	}

	// Get the initiative context
	var historyFolder string

	// Check for initiative override
	if opts != nil && opts.Initiative != "" {
		historyFolder = filepath.Join(s.workDir, "history", opts.Initiative)
		if _, err := os.Stat(historyFolder); os.IsNotExist(err) {
			return nil, &StepError{
				Code:    "INITIATIVE_NOT_FOUND",
				Message: fmt.Sprintf("initiative '%s' not found in history/", opts.Initiative),
				Hint:    "Check the initiative path or use 'init' to create a new one",
			}
		}
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
				Hint:    "Run step='init' with type and name parameters to create a new initiative",
			}
		}

		historyFolder = filepath.Join(s.workDir, state.Initiative)
	}

	// Resolve file patterns to actual files
	filesToRead := s.resolveFiles(step.Files, historyFolder)

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
		Directive:      step.Directive,
		HistoryFolder:  historyFolder,
		FilesToRead:    filesToRead,
		ComposedPrompt: composedPrompt,
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

// executeInitStep handles the special "init" step that creates a new initiative.
func (s *Service) executeInitStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
	// Validate required parameters
	if opts == nil || opts.Type == "" {
		return nil, &StepError{
			Code:    "MISSING_TYPE",
			Message: "type parameter is required for init step",
			Hint:    "Provide type: feature, bug, or refactor",
		}
	}
	if opts.Name == "" {
		return nil, &StepError{
			Code:    "MISSING_NAME",
			Message: "name parameter is required for init step",
			Hint:    "Provide a name for the initiative (e.g., 'user-auth')",
		}
	}

	// Validate type
	initType := initiative.InitiativeType(opts.Type)
	if !initType.IsValid() {
		return nil, &StepError{
			Code:    "INVALID_TYPE",
			Message: fmt.Sprintf("invalid initiative type '%s'", opts.Type),
			Hint:    "Type must be one of: feature, bug, refactor",
		}
	}

	// Create initiative service
	initSvc, err := initiative.NewService(s.workDir)
	if err != nil {
		return nil, fmt.Errorf("creating initiative service: %w", err)
	}

	// Create the initiative
	init, err := initSvc.Create(initType, opts.Name)
	if err != nil {
		return nil, err
	}

	// Return response with the new initiative's history folder
	return &StepResponse{
		Directive:      step.Directive,
		HistoryFolder:  init.Path,
		FilesToRead:    []string{},
		ComposedPrompt: "",
	}, nil
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
