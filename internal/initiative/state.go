package initiative

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// StateManager manages the persistent state of the active initiative.
type StateManager interface {
	// Load reads the current state from storage.
	Load() (*InitiativeState, error)
	// Save writes the state to storage.
	Save(state *InitiativeState) error
	// Lock acquires an exclusive lock on the state file.
	Lock() (unlock func(), err error)
	// Path returns the path to the state file.
	Path() string
}

// FileStateManager implements StateManager using file-based storage with flock locking.
type FileStateManager struct {
	// path is the absolute path to the state file.
	path string
	// lockFile is the file handle used for locking.
	lockFile *os.File
}

// StateFileName is the name of the state file.
const StateFileName = "active.json"

// BrainsDir is the name of the brains configuration directory.
const BrainsDir = ".brains"

// NewStateManager creates a new FileStateManager for the given working directory.
// The state file will be at {workDir}/.brains/active.json.
func NewStateManager(workDir string) (*FileStateManager, error) {
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
	}

	statePath := filepath.Join(workDir, BrainsDir, StateFileName)
	return &FileStateManager{
		path: statePath,
	}, nil
}

// Path returns the path to the state file.
func (m *FileStateManager) Path() string {
	return m.path
}

// Load reads the current state from the state file.
// If the file doesn't exist, returns an empty state (no error).
func (m *FileStateManager) Load() (*InitiativeState, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No state file means no active initiative
			return &InitiativeState{}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state InitiativeState
	if len(data) == 0 {
		// Empty file means no active initiative
		return &InitiativeState{}, nil
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &state, nil
}

// Save writes the state to the state file.
// Creates the directory if it doesn't exist.
func (m *FileStateManager) Save(state *InitiativeState) error {
	// Ensure parent directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Write atomically using temp file + rename
	tempPath := m.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tempPath, m.path); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// Lock acquires an exclusive lock on the state file.
// Returns an unlock function that must be called to release the lock.
// The lock file is separate from the state file to avoid issues with
// reading/writing while locked.
func (m *FileStateManager) Lock() (unlock func(), err error) {
	// Ensure parent directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}

	lockPath := m.path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	// Acquire exclusive lock using flock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	m.lockFile = f

	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		m.lockFile = nil
	}, nil
}

// Clear removes the state file, effectively clearing the active initiative.
func (m *FileStateManager) Clear() error {
	err := os.Remove(m.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing state file: %w", err)
	}
	return nil
}
