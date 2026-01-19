package profile

import (
	"io/fs"
	"strings"
	"sync"
)

// Global registry for embedded filesystem
var (
	globalEmbeddedFS  fs.FS
	globalEmbeddedMu  sync.RWMutex
	embeddedDirPrefix = "profiles" // The directory name within embed.FS
)

// SetEmbeddedFS registers an embedded filesystem containing default profiles.
// This should be called during application initialization with the embed.FS
// from cmd/brains/embed.go.
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

// HasEmbeddedProfiles returns true if an embedded filesystem is registered
// and contains at least one profile.
func HasEmbeddedProfiles() bool {
	globalEmbeddedMu.RLock()
	defer globalEmbeddedMu.RUnlock()

	if globalEmbeddedFS == nil {
		return false
	}

	// Check if profiles directory exists and has .md files
	entries, err := fs.ReadDir(globalEmbeddedFS, embeddedDirPrefix)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true
		}
	}

	return false
}

// loadEmbeddedProfiles loads all profiles from the embedded filesystem.
// Returns an empty slice if no embedded filesystem is registered or if
// there are no valid profiles.
func loadEmbeddedProfiles() []*Profile {
	globalEmbeddedMu.RLock()
	fsys := globalEmbeddedFS
	globalEmbeddedMu.RUnlock()

	if fsys == nil {
		return nil
	}

	entries, err := fs.ReadDir(fsys, embeddedDirPrefix)
	if err != nil {
		return nil
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		profileName := strings.TrimSuffix(entry.Name(), ".md")
		filePath := embeddedDirPrefix + "/" + entry.Name()

		content, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		// Use virtual path format for embedded profiles
		virtualPath := "[embedded]/" + entry.Name()

		profile, err := ParseProfile(content, profileName, virtualPath, SourceEmbedded)
		if err != nil {
			// Skip profiles with parse errors - continue with valid profiles
			continue
		}

		profiles = append(profiles, profile)
	}

	return profiles
}

// loadProfilesFromEmbedded returns a map of embedded profiles keyed by name.
// This is a convenience wrapper around loadEmbeddedProfiles for use by
// BrainsSource and other components.
func loadProfilesFromEmbedded() map[string]*Profile {
	profiles := loadEmbeddedProfiles()
	result := make(map[string]*Profile, len(profiles))
	for _, p := range profiles {
		result[p.Name] = p
	}
	return result
}
