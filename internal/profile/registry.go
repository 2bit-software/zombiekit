package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// Registry stores known .brains/ directories across projects.
type Registry struct {
	Directories []RegistryEntry `json:"directories"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// RegistryEntry represents a single known .brains/ directory.
type RegistryEntry struct {
	Path       string    `json:"path"`         // Absolute path to .brains/ directory
	AddedAt    time.Time `json:"added_at"`     // When this entry was first added
	LastSeenAt time.Time `json:"last_seen_at"` // When this entry was last verified
}

// RegistryManager handles reading and writing the registry file with file locking.
type RegistryManager struct {
	registryPath string
	lockPath     string
}

// NewRegistryManager creates a new RegistryManager.
// The registry is stored at ~/.brains/registry.json with a .lock file for concurrency.
func NewRegistryManager() (*RegistryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	brainsDir := filepath.Join(homeDir, ".brains")
	registryPath := filepath.Join(brainsDir, "registry.json")
	lockPath := filepath.Join(brainsDir, "registry.json.lock")

	return &RegistryManager{
		registryPath: registryPath,
		lockPath:     lockPath,
	}, nil
}

// loadUnlocked reads the registry without acquiring a lock (caller must hold lock).
func (rm *RegistryManager) loadUnlocked() (*Registry, error) {
	data, err := os.ReadFile(rm.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{
				Directories: []RegistryEntry{},
				UpdatedAt:   time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	return &registry, nil
}

// saveUnlocked writes the registry without acquiring a lock (caller must hold lock).
func (rm *RegistryManager) saveUnlocked(registry *Registry) error {
	registry.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	// Ensure the parent directory exists
	dir := filepath.Dir(rm.registryPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	// Write to temp file then rename for atomicity
	tmpPath := rm.registryPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}

	if err := os.Rename(tmpPath, rm.registryPath); err != nil {
		os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("renaming registry: %w", err)
	}

	return nil
}

// Register adds or updates a .brains/ directory in the registry.
func (rm *RegistryManager) Register(brainsPath string) error {
	// Ensure we have an absolute path
	absPath, err := filepath.Abs(brainsPath)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	fileLock := flock.New(rm.lockPath)
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer fileLock.Unlock()

	registry, err := rm.loadUnlocked()
	if err != nil {
		return err
	}

	now := time.Now()
	found := false

	for i := range registry.Directories {
		if registry.Directories[i].Path == absPath {
			registry.Directories[i].LastSeenAt = now
			found = true
			break
		}
	}

	if !found {
		registry.Directories = append(registry.Directories, RegistryEntry{
			Path:       absPath,
			AddedAt:    now,
			LastSeenAt: now,
		})
	}

	return rm.saveUnlocked(registry)
}

