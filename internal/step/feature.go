package step

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/zombiekit/brains/internal/initiative"
)

// FeatureExecuteOptions extends ExecuteOptions with feature-specific parameters.
type FeatureExecuteOptions struct {
	// Name is the feature name (required).
	Name string
	// Type is the initiative type (defaults to "feature").
	Type string
	// Description is an optional description for the initiative.
	Description string
	// NewInitiative forces creation of a new initiative even if one is active.
	NewInitiative bool
}

// executeFeatureStep handles the special "feature" step that creates initiatives with cycles.
func (s *Service) executeFeatureStep(step *Step, opts *ExecuteOptions) (*StepResponse, error) {
	// Validate required parameters
	if opts == nil || opts.Name == "" {
		return nil, &StepError{
			Code:    "MISSING_NAME",
			Message: "name parameter is required for feature step",
			Hint:    "Provide a name for the feature (e.g., 'user-auth')",
		}
	}

	// Determine initiative type (default to feature)
	initType := initiative.InitiativeType(opts.Type)
	if opts.Type == "" {
		initType = initiative.TypeFeature
	}
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

	var init *initiative.Initiative
	var cycle *initiative.Cycle
	var initiativeFolder string
	var createGitBranch bool

	// Load current state to check for active initiative
	state, err := s.stateManager.Load()
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	// Check if we should create a new initiative or add a cycle to existing
	if state.IsEmpty() || opts.NewInitiative {
		// Create new initiative
		init, err = initSvc.Create(initType, opts.Name)
		if err != nil {
			return nil, err
		}
		initiativeFolder = init.Path
		createGitBranch = true

		// Write initiative metadata with YAML frontmatter
		if err := s.writeInitiativeMetadata(init); err != nil {
			return nil, fmt.Errorf("writing initiative metadata: %w", err)
		}
	} else {
		// Use existing initiative
		initiativeFolder = filepath.Join(s.workDir, state.Initiative)
		createGitBranch = false
	}

	// Map initiative type to cycle type
	cycleType := mapInitTypeToCycleType(initType)

	// Create cycle within initiative
	cycle, err = initSvc.CreateCycle(initiativeFolder, cycleType, opts.Name)
	if err != nil {
		return nil, err
	}

	// Copy templates to cycle folder
	if err := s.copyTemplatesToCycle(cycle.Path); err != nil {
		return nil, fmt.Errorf("copying templates: %w", err)
	}

	// Create git branch if this is a new initiative
	if createGitBranch {
		gitSvc := NewGitService(s.workDir)
		// Ignore errors - git operations should fail gracefully
		_ = gitSvc.EnsureBranch(string(initType), opts.Name)
	}

	// Update state with cycle path
	newState := &initiative.InitiativeState{
		Initiative:   filepath.Join(initiative.HistoryDir, filepath.Base(initiativeFolder)),
		Cycle:        filepath.Join(initiative.HistoryDir, filepath.Base(initiativeFolder), cycle.ID),
		Started:      state.Started,
		LastActivity: time.Now(),
		CurrentStep:  "feature",
	}
	if newState.Started.IsZero() {
		newState.Started = time.Now()
	}
	if err := s.stateManager.Save(newState); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	// Resolve file patterns to actual files
	filesToRead := s.resolveFiles(step.Files, cycle.Path)

	// Add previous cycle artifacts if not the first cycle
	if cycle.Number > 1 {
		prevArtifacts := s.getPreviousCycleArtifacts(initiativeFolder, cycle.ID)
		filesToRead = append(filesToRead, prevArtifacts...)
	}

	// Compose profiles
	composedPrompt := ""
	if s.profileSvc != nil && len(step.Profiles) > 0 {
		result, err := s.profileSvc.Compose(step.Profiles)
		if err == nil {
			composedPrompt = result.Content
		}
	}

	return &StepResponse{
		Directive:        step.Directive,
		HistoryFolder:    cycle.Path, // Backward compatibility
		InitiativeFolder: initiativeFolder,
		CycleFolder:      cycle.Path,
		FilesToRead:      filesToRead,
		ComposedPrompt:   composedPrompt,
		WorkflowPhases:   buildWorkflowPhases(),
	}, nil
}

// mapInitTypeToCycleType converts an initiative type to a cycle type.
func mapInitTypeToCycleType(t initiative.InitiativeType) initiative.CycleType {
	switch t {
	case initiative.TypeFeature:
		return initiative.CycleFeat
	case initiative.TypeRefactor:
		return initiative.CycleRef
	case initiative.TypeBug:
		return initiative.CycleFix
	default:
		return initiative.CycleFeat
	}
}

// writeInitiativeMetadata creates INITIATIVE.md with YAML frontmatter.
func (s *Service) writeInitiativeMetadata(init *initiative.Initiative) error {
	now := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(`---
status: active
type: %s
created: %s
updated: %s
---

# Initiative: %s

**ID**: %s

## Description

<!-- Add a description of this initiative -->

## Goals

<!-- Define the goals for this initiative -->

## Cycles

| # | Type | Status | Created |
|---|------|--------|---------|

## Progress

<!-- Track progress here -->
`, init.Type, now, now, init.Name, init.ID)

	mdPath := filepath.Join(init.Path, "INITIATIVE.md")
	return os.WriteFile(mdPath, []byte(content), 0644)
}

// copyTemplatesToCycle copies spec and research templates to the cycle folder.
func (s *Service) copyTemplatesToCycle(cyclePath string) error {
	embFS := s.getEmbeddedFS()
	if embFS == nil {
		return fmt.Errorf("no embedded filesystem available")
	}

	// Templates to copy
	templates := []struct {
		src  string
		dest string
	}{
		{"templates/spec-template.md", "spec.md"},
		{"templates/research-template.md", "research.md"},
	}

	for _, tmpl := range templates {
		// First check if local override exists
		localPath := filepath.Join(s.workDir, ".brains", "templates", filepath.Base(tmpl.src))
		var content []byte
		var err error

		if _, statErr := os.Stat(localPath); statErr == nil {
			// Local override exists
			content, err = os.ReadFile(localPath)
		} else {
			// Use embedded
			content, err = fs.ReadFile(embFS, tmpl.src)
		}

		if err != nil {
			return fmt.Errorf("reading template %s: %w", tmpl.src, err)
		}

		destPath := filepath.Join(cyclePath, tmpl.dest)
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", tmpl.dest, err)
		}
	}

	return nil
}

// getEmbeddedFS returns the embedded filesystem for templates.
func (s *Service) getEmbeddedFS() fs.FS {
	if s.loader != nil && s.loader.embeddedFS != nil {
		return s.loader.embeddedFS
	}
	return GetEmbeddedFS()
}

// getPreviousCycleArtifacts returns paths to artifacts from previous cycles.
func (s *Service) getPreviousCycleArtifacts(initPath, currentCycleID string) []string {
	var artifacts []string

	entries, err := os.ReadDir(initPath)
	if err != nil {
		return artifacts
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == currentCycleID {
			continue
		}

		// Skip non-cycle directories
		if entry.Name() == "audit" || entry.Name() == ".git" {
			continue
		}

		cyclePath := filepath.Join(initPath, entry.Name())

		// Add research.md if exists
		researchPath := filepath.Join(cyclePath, "research.md")
		if _, err := os.Stat(researchPath); err == nil {
			artifacts = append(artifacts, researchPath)
		}

		// Add spec.md if exists
		specPath := filepath.Join(cyclePath, "spec.md")
		if _, err := os.Stat(specPath); err == nil {
			artifacts = append(artifacts, specPath)
		}
	}

	return artifacts
}

// buildWorkflowPhases returns the workflow phase definitions for the feature step.
func buildWorkflowPhases() []Phase {
	return []Phase{
		{
			Name:        "research",
			Description: "Gather context and domain knowledge through parallel research agents",
			Agents:      []string{"research-codebase", "research-domain"},
			Outputs:     []string{"research.md"},
			Parallel:    true,
		},
		{
			Name:        "create",
			Description: "Synthesize specification from research findings",
			Agents:      []string{"spec-writer"},
			Outputs:     []string{"spec.md"},
			Parallel:    false,
		},
		{
			Name:        "audit",
			Description: "Check specification quality and completeness with severity classification",
			Agents:      []string{"audit-completeness", "audit-ai-readiness"},
			Outputs:     []string{"audit/{date}.md"},
			Parallel:    true,
		},
		{
			Name:        "highlight",
			Description: "Present key decisions for user approval before proceeding",
			Agents:      []string{"highlighter"},
			Outputs:     []string{},
			Parallel:    false,
		},
	}
}
