package step

import (
	"io/fs"
	"sync"
)

// Global registry for embedded step filesystem
var (
	globalEmbeddedFS fs.FS
	globalEmbeddedMu sync.RWMutex
)

// SetEmbeddedFS registers an embedded filesystem containing default step definitions.
// This should be called during application initialization with the embed.FS
// from the zombiekit package.
func SetEmbeddedFS(fsys fs.FS) {
	globalEmbeddedMu.Lock()
	defer globalEmbeddedMu.Unlock()
	globalEmbeddedFS = fsys
}

// GetEmbeddedFS returns the registered embedded filesystem.
// Returns nil if no embedded filesystem has been registered.
func GetEmbeddedFS() fs.FS {
	globalEmbeddedMu.RLock()
	defer globalEmbeddedMu.RUnlock()
	return globalEmbeddedFS
}

// HasEmbeddedSteps returns true if an embedded filesystem is registered
// and contains at least one step definition.
func HasEmbeddedSteps() bool {
	globalEmbeddedMu.RLock()
	defer globalEmbeddedMu.RUnlock()

	if globalEmbeddedFS == nil {
		return false
	}

	// Check if steps directory exists and has .md files
	entries, err := fs.ReadDir(globalEmbeddedFS, embeddedStepsPrefix)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 3 && entry.Name()[len(entry.Name())-3:] == ".md" {
			return true
		}
	}

	return false
}
