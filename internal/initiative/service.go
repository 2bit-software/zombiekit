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
func (s *Service) Create(initType InitiativeType, name string) (*Initiative, error) {
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
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create INITIATIVE.md
	if err := s.createInitiativeMD(initiative); err != nil {
		return nil, fmt.Errorf("creating INITIATIVE.md: %w", err)
	}

	// Set as active initiative (pointer only, no status)
	state := &InitiativeState{
		Initiative:   filepath.Join(HistoryDir, id),
		Started:      now,
		LastActivity: now,
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
		Initiative:   filepath.Join(HistoryDir, initiativeID),
		Started:      time.Now(),
		LastActivity: time.Now(),
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

// generateID generates a unique initiative ID in format: {hex-timestamp}-{type}-{name}
func (s *Service) generateID(initType InitiativeType, name string) string {
	timestamp := fmt.Sprintf("%08x", time.Now().Unix())
	return fmt.Sprintf("%s-%s-%s", timestamp, initType, name)
}

// createInitiativeMD creates the INITIATIVE.md file for an initiative.
func (s *Service) createInitiativeMD(init *Initiative) error {
	content := fmt.Sprintf(`# Initiative: %s

**Type**: %s
**Status**: %s
**Created**: %s
**ID**: %s

## Description

<!-- Add a description of this initiative -->

## Goals

<!-- Define the goals for this initiative -->

## Progress

<!-- Track progress here -->
`, init.Name, init.Type, init.Status, init.CreatedAt.Format(time.RFC3339), init.ID)

	mdPath := filepath.Join(init.Path, InitiativeMDFile)
	return os.WriteFile(mdPath, []byte(content), 0644)
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
		Status:    StatusActive, // Default to active
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
