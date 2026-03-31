package initiative

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Service provides initiative management functionality.
type Service struct {
	workDir      string
	stateManager *FileStateManager
}

// HistoryDir is the name of the history directory.
const HistoryDir = "history"

// InitiativeMDFile is the name of the initiative metadata file.
const InitiativeMDFile = "INITIATIVE.md"

// NewService creates a new initiative service for the given working directory.
func NewService(workDir string) (*Service, error) {
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	stateManager, err := NewStateManager(workDir)
	if err != nil {
		return nil, fmt.Errorf("creating state manager: %w", err)
	}

	return &Service{
		workDir:      workDir,
		stateManager: stateManager,
	}, nil
}

// Create creates a new initiative with the given type and name.
// Returns the created initiative and sets it as the active initiative.
// If steps is provided, the INITIATIVE.md will include a Steps section with a step table.
func (s *Service) Create(initType InitiativeType, name string, steps []WorkflowStep) (*Initiative, error) {
	// Validate type
	if !initType.IsValid() {
		return nil, &InitiativeError{
			Code:    "INVALID_TYPE",
			Message: fmt.Sprintf("invalid initiative type '%s'", initType),
			Hint:    "Type must be one of: feature, bug, refactor",
		}
	}

	// Normalize and validate name
	normalizedName := normalizeName(name)
	if err := validateName(normalizedName); err != nil {
		return nil, err
	}

	// Generate unique ID
	id := s.generateID(initType, normalizedName)

	// Create history directory if needed
	historyDir := filepath.Join(s.workDir, HistoryDir)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("creating history directory: %w", err)
	}

	// Create initiative folder
	initiativePath := filepath.Join(historyDir, id)
	if err := os.MkdirAll(initiativePath, 0755); err != nil {
		return nil, fmt.Errorf("creating initiative directory: %w", err)
	}

	now := time.Now()
	initiative := &Initiative{
		ID:        id,
		Type:      initType,
		Name:      normalizedName,
		Path:      initiativePath,
		Status:    StatusInProgress,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create INITIATIVE.md
	if err := s.createInitiativeMD(initiative, steps); err != nil {
		return nil, fmt.Errorf("creating INITIATIVE.md: %w", err)
	}

	// Set as active initiative (pointer only, no status)
	state := &InitiativeState{
		Initiative: filepath.Join(HistoryDir, id),
		Started:    now,
		Status:     StatusInProgress,
	}
	if err := s.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return initiative, nil
}

// List returns all initiatives from the history folder.
func (s *Service) List() ([]*Initiative, error) {
	historyDir := filepath.Join(s.workDir, HistoryDir)
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Initiative{}, nil
		}
		return nil, fmt.Errorf("reading history directory: %w", err)
	}

	var initiatives []*Initiative
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it has INITIATIVE.md
		initPath := filepath.Join(historyDir, entry.Name())
		mdPath := filepath.Join(initPath, InitiativeMDFile)
		if _, err := os.Stat(mdPath); os.IsNotExist(err) {
			continue
		}

		// Parse the folder name to extract type and name
		init := s.parseInitiativeFromFolder(entry.Name(), initPath)
		if init != nil {
			initiatives = append(initiatives, init)
		}
	}

	return initiatives, nil
}

// GetActive returns the currently active initiative, or nil if none.
func (s *Service) GetActive() (*Initiative, error) {
	state, err := s.stateManager.Load()
	if err != nil {
		return nil, err
	}

	if state.IsEmpty() {
		return nil, nil
	}

	// Build full path
	initPath := filepath.Join(s.workDir, state.Initiative)

	// Verify it exists
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return nil, nil
	}

	id := filepath.Base(state.Initiative)
	// Parse initiative info from folder name (type/name/status from folder + INITIATIVE.md)
	init := s.parseInitiativeFromFolder(id, initPath)
	if init == nil {
		return nil, nil
	}
	return init, nil
}

// FindActiveByNameAndType returns the active initiative if it matches the given name and type.
// Returns nil if no active initiative exists or if the active initiative doesn't match.
// This enables idempotent initiative creation - calling create with the same name+type
// returns the existing initiative instead of creating a duplicate.
func (s *Service) FindActiveByNameAndType(name string, initType InitiativeType) (*Initiative, error) {
	active, err := s.GetActive()
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, nil
	}

	// Use same normalization as Create() uses
	normalizedName := normalizeName(name)

	if active.Name == normalizedName && active.Type == initType {
		return active, nil
	}
	return nil, nil
}

// SetActive sets the specified initiative as active.
func (s *Service) SetActive(initiativeID string) error {
	// Check if initiative exists
	initPath := filepath.Join(s.workDir, HistoryDir, initiativeID)
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return &InitiativeError{
			Code:    "INITIATIVE_NOT_FOUND",
			Message: fmt.Sprintf("initiative '%s' not found in history/", initiativeID),
			Hint:    "Check the initiative path or use 'init' to create a new one",
		}
	}

	// Parse initiative info from folder name
	init := s.parseInitiativeFromFolder(initiativeID, initPath)
	if init == nil {
		return &InitiativeError{
			Code:    "INVALID_INITIATIVE",
			Message: fmt.Sprintf("could not parse initiative from folder '%s'", initiativeID),
		}
	}

	// Update state (pointer only, no status)
	state := &InitiativeState{
		Initiative: filepath.Join(HistoryDir, initiativeID),
		Started:    time.Now(),
		Status:     StatusInProgress,
	}

	return s.stateManager.Save(state)
}

// Complete marks the active initiative as completed and clears the active state.
func (s *Service) Complete() error {
	state, err := s.stateManager.Load()
	if err != nil {
		return err
	}

	if state.IsEmpty() {
		return &InitiativeError{
			Code:    "NO_ACTIVE_INITIATIVE",
			Message: "no active initiative to complete",
			Hint:    "There is no active initiative to mark as complete",
		}
	}

	// Clear the state (marks as completed by removing active)
	return s.stateManager.Clear()
}

// StatusResult contains the status of the active initiative.
type StatusResult struct {
	Active         bool     `json:"active"`
	InitiativeID   string   `json:"initiative_id,omitempty"`
	InitiativeType string   `json:"initiative_type,omitempty"`
	CurrentStep    string   `json:"current_step,omitempty"`
	StepStatus     string   `json:"step_status,omitempty"`
	StepsCompleted int      `json:"steps_completed,omitempty"`
	StepsTotal     int      `json:"steps_total,omitempty"`
	AvailableDocs  []string `json:"available_docs,omitempty"`
	SuggestedNext  string   `json:"suggested_next,omitempty"`
	HistoryPath    string   `json:"history_path,omitempty"`
	InitiativeFile string   `json:"initiative_file,omitempty"`
	Files          []string `json:"files,omitempty"`
}

// Status returns the status of the active initiative.
// Returns active=false if no initiative is active.
func (s *Service) Status() (*StatusResult, error) {
	state, err := s.stateManager.Load()
	if err != nil {
		return nil, err
	}

	if state.IsEmpty() {
		return &StatusResult{
			Active:        false,
			SuggestedNext: "initiative create",
		}, nil
	}

	// Get active initiative info
	init, err := s.GetActive()
	if err != nil || init == nil {
		return &StatusResult{
			Active:        false,
			SuggestedNext: "initiative create",
		}, nil
	}

	// Parse INITIATIVE.md for step state
	mdPath := filepath.Join(init.Path, InitiativeMDFile)
	parsed, err := ParseInitiativeMD(mdPath)

	var currentStep, stepStatus string
	var stepsCompleted, stepsTotal int

	if err == nil && parsed != nil {
		currentStep, stepStatus, stepsCompleted, stepsTotal = analyzeSteps(parsed)
	}

	// Find available docs in initiative folder
	availableDocs := s.findAvailableDocs(init.Path)

	// Determine suggested next step based on current step or available artifacts
	suggestedNext := s.determineSuggestedNext(availableDocs, currentStep)

	// Build relative paths for client use
	historyPath := state.Initiative // Already relative (e.g., "history/695c116e-feature-go-project-setup")
	initiativeFile := filepath.Join(state.Initiative, InitiativeMDFile)

	// Build list of relative file paths to read
	var files []string
	for _, doc := range availableDocs {
		if strings.HasSuffix(doc, "/") {
			// Directory, skip
			continue
		}
		files = append(files, filepath.Join(state.Initiative, doc))
	}

	return &StatusResult{
		Active:         true,
		InitiativeID:   init.ID,
		InitiativeType: string(init.Type),
		CurrentStep:    currentStep,
		StepStatus:     stepStatus,
		StepsCompleted: stepsCompleted,
		StepsTotal:     stepsTotal,
		AvailableDocs:  availableDocs,
		SuggestedNext:  suggestedNext,
		HistoryPath:    historyPath,
		InitiativeFile: initiativeFile,
		Files:          files,
	}, nil
}

// analyzeSteps counts completed steps and identifies the current step from a parsed initiative.
func analyzeSteps(parsed *ParsedInitiative) (currentStep, stepStatus string, stepsCompleted, stepsTotal int) {
	stepsTotal = len(parsed.Steps)

	for _, step := range parsed.Steps {
		if step.Status == StepCompleted || step.Status == StepSkipped {
			stepsCompleted++
		}
		if step.Status == StepInProgress {
			currentStep = step.Name
			stepStatus = string(step.Status)
		}
	}

	if currentStep == "" {
		if next := parsed.NextStep(); next != nil {
			currentStep = next.Name
			stepStatus = string(next.Status)
		}
	}

	return currentStep, stepStatus, stepsCompleted, stepsTotal
}

// findAvailableDocs scans the initiative folder for known artifact files.
func (s *Service) findAvailableDocs(initiativePath string) []string {
	knownDocs := []string{"spec.md", "research.md", "plan.md", "tasks.md", "data-model.md", "quickstart.md"}
	var available []string

	for _, doc := range knownDocs {
		docPath := filepath.Join(initiativePath, doc)
		if _, err := os.Stat(docPath); err == nil {
			available = append(available, doc)
		}
	}

	// Check for contracts directory
	contractsDir := filepath.Join(initiativePath, "contracts")
	if info, err := os.Stat(contractsDir); err == nil && info.IsDir() {
		available = append(available, "contracts/")
	}

	return available
}

// determineSuggestedNext suggests the next step based on available artifacts.
func (s *Service) determineSuggestedNext(availableDocs []string, currentStep string) string {
	hasDoc := func(name string) bool {
		for _, d := range availableDocs {
			if d == name {
				return true
			}
		}
		return false
	}

	// If no spec, suggest starting with feature/bug/refactor step
	if !hasDoc("spec.md") {
		return "feature"
	}

	// If spec exists but no plan, suggest plan
	if !hasDoc("plan.md") {
		return "plan"
	}

	// If plan exists but no tasks, suggest tasks
	if !hasDoc("tasks.md") {
		return "tasks"
	}

	// If tasks exist, suggest implement
	return "implement"
}

// generateID generates a unique initiative ID in format: {hex-timestamp}-{type}-{name}
func (s *Service) generateID(initType InitiativeType, name string) string {
	timestamp := fmt.Sprintf("%08x", time.Now().Unix())
	return fmt.Sprintf("%s-%s-%s", timestamp, initType, name)
}

// WorkflowStep represents a step in a workflow (used for initiative creation).
type WorkflowStep struct {
	Name    string
	Profile string
}

// createInitiativeMD creates the INITIATIVE.md file for an initiative.
// If steps is provided, a Steps section with a step table is included.
func (s *Service) createInitiativeMD(init *Initiative, steps []WorkflowStep) error {
	var builder strings.Builder

	// Header section
	builder.WriteString(fmt.Sprintf("# Initiative: %s\n\n", init.Name))
	builder.WriteString(fmt.Sprintf("**Type**: %s\n", init.Type))
	builder.WriteString(fmt.Sprintf("**Status**: %s\n", init.Status))
	builder.WriteString(fmt.Sprintf("**Created**: %s\n", init.CreatedAt.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("**ID**: %s\n\n", init.ID))

	// Steps section (if steps provided)
	if len(steps) > 0 {
		builder.WriteString("## Steps\n\n")

		// Create step table
		builder.WriteString("| Step | Status | Updated |\n")
		builder.WriteString("|------|--------|--------|\n")
		for i, step := range steps {
			status := "pending"
			updated := "-"
			if i == 0 {
				status = "in_progress"
				updated = time.Now().Format("2006-01-02 15:04")
			}
			builder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", step.Name, status, updated))
		}
		builder.WriteString("\n")
	}

	// Description section
	builder.WriteString("## Description\n\n")
	builder.WriteString("<!-- Add a description of this initiative -->\n\n")

	// Goals section
	builder.WriteString("## Goals\n\n")
	builder.WriteString("<!-- Define the goals for this initiative -->\n\n")

	// Progress section
	builder.WriteString("## Progress\n\n")
	builder.WriteString("<!-- Track progress here -->\n")

	mdPath := filepath.Join(init.Path, InitiativeMDFile)
	return os.WriteFile(mdPath, []byte(builder.String()), 0644)
}

// parseInitiativeFromFolder parses an Initiative from a folder name.
// Expected format: {timestamp}-{type}-{name}
func (s *Service) parseInitiativeFromFolder(folderName, path string) *Initiative {
	// Split by first two dashes
	parts := strings.SplitN(folderName, "-", 3)
	if len(parts) < 3 {
		return nil
	}

	initType := InitiativeType(parts[1])
	if !initType.IsValid() {
		return nil
	}

	// Try to read INITIATIVE.md for more details
	mdPath := filepath.Join(path, InitiativeMDFile)
	info, err := os.Stat(mdPath)

	var createdAt, updatedAt time.Time
	if err == nil {
		createdAt = info.ModTime()
		updatedAt = info.ModTime()
	}

	return &Initiative{
		ID:        folderName,
		Type:      initType,
		Name:      parts[2],
		Path:      path,
		Status:    StatusInProgress, // Default to active
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// normalizeName normalizes an initiative name to slug format.
func normalizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove invalid characters
	re := regexp.MustCompile(`[^a-z0-9-]`)
	name = re.ReplaceAllString(name, "")
	// Collapse multiple hyphens
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")
	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")
	return name
}

// validateName validates a normalized initiative name.
func validateName(name string) error {
	if name == "" {
		return &InitiativeError{
			Code:    "INVALID_NAME",
			Message: "initiative name cannot be empty",
		}
	}

	// Must be lowercase alphanumeric with hyphens
	re := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
	if !re.MatchString(name) {
		return &InitiativeError{
			Code:    "INVALID_NAME",
			Message: fmt.Sprintf("invalid initiative name '%s'", name),
			Hint:    "Name must be lowercase alphanumeric with hyphens (e.g., 'user-auth')",
		}
	}

	return nil
}

// InitiativeError represents an error in initiative operations.
type InitiativeError struct {
	Code    string
	Message string
	Hint    string
}

func (e *InitiativeError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// MarshalJSON implements json.Marshaler for InitiativeError.
func (e *InitiativeError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"code":    e.Code,
		"message": e.Message,
		"hint":    e.Hint,
	})
}
