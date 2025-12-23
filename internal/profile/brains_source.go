package profile

import (
	"fmt"
	"os"
	"path/filepath"
)

// BrainsSource implements ProfileSourceInterface for brains profiles.
// It wraps the existing Resolver to provide backward compatibility.
type BrainsSource struct {
	resolver   *Resolver
	workingDir string
	homeDir    string
}

// NewBrainsSource creates a new BrainsSource.
// If workingDir is empty, it uses the current working directory.
func NewBrainsSource(workingDir string) (*BrainsSource, error) {
	resolver, err := NewResolver(workingDir)
	if err != nil {
		return nil, err
	}

	if workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	homeDir, _ := os.UserHomeDir()

	return &BrainsSource{
		resolver:   resolver,
		workingDir: workingDir,
		homeDir:    homeDir,
	}, nil
}

// FindProfileDirs discovers .brains/profiles/ directories.
// Includes an embedded virtual directory as the lowest precedence fallback.
func (s *BrainsSource) FindProfileDirs() ([]ResolvedDirectory, error) {
	dirs, err := s.resolver.FindProfileDirs()
	if err != nil {
		return nil, err
	}

	// Append embedded as lowest precedence (after global)
	if HasEmbeddedProfiles() {
		dirs = append(dirs, ResolvedDirectory{
			Path:   "[embedded]",
			Source: SourceEmbedded,
		})
	}

	return dirs, nil
}

// LoadProfiles loads profiles from the given directories.
// Includes embedded profiles as lowest-precedence fallback.
func (s *BrainsSource) LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error) {
	// Separate embedded directory from filesystem directories
	var fsDirs []ResolvedDirectory
	var hasEmbedded bool
	for _, dir := range dirs {
		if dir.Source == SourceEmbedded {
			hasEmbedded = true
		} else {
			fsDirs = append(fsDirs, dir)
		}
	}

	// Load filesystem profiles first
	profiles, err := s.resolver.LoadProfiles(fsDirs)
	if err != nil {
		return nil, err
	}
	if profiles == nil {
		profiles = make(map[string]*Profile)
	}

	// Add embedded profiles that aren't already shadowed
	if hasEmbedded {
		for name, p := range loadProfilesFromEmbedded() {
			if _, exists := profiles[name]; !exists {
				profiles[name] = p
			}
		}
	}

	return profiles, nil
}

// LoadAllProfiles loads all profiles including shadowed ones.
// Includes embedded profiles for listing and shadowing display.
func (s *BrainsSource) LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error) {
	// Separate embedded directory from filesystem directories
	var fsDirs []ResolvedDirectory
	var hasEmbedded bool
	for _, dir := range dirs {
		if dir.Source == SourceEmbedded {
			hasEmbedded = true
		} else {
			fsDirs = append(fsDirs, dir)
		}
	}

	// Load filesystem profiles first
	profiles, err := s.resolver.LoadAllProfiles(fsDirs)
	if err != nil {
		return nil, err
	}
	if profiles == nil {
		profiles = make(map[string][]*Profile)
	}

	// Append embedded profiles (lowest precedence, shown as shadowed if others exist)
	if hasEmbedded {
		for name, p := range loadProfilesFromEmbedded() {
			profiles[name] = append(profiles[name], p)
		}
	}

	return profiles, nil
}

// GetInheritanceChain returns all versions of a profile for inheritance.
// Includes embedded profiles at the start of the chain (lowest precedence).
func (s *BrainsSource) GetInheritanceChain(name string) ([]*Profile, error) {
	chain, err := s.resolver.GetInheritanceChain(name)
	if err != nil {
		return nil, err
	}

	// Check for embedded version of this profile
	embedded := loadProfilesFromEmbedded()
	if embeddedProfile, ok := embedded[name]; ok {
		// Prepend embedded profile (it should be first = lowest precedence in chain)
		chain = append([]*Profile{embeddedProfile}, chain...)
	}

	return chain, nil
}

// CreateProfile creates a new brains profile.
func (s *BrainsSource) CreateProfile(name string, global bool) (string, error) {
	// Determine target directory
	var targetDir string
	if global {
		if s.homeDir == "" {
			return "", fmt.Errorf("home directory not available")
		}
		targetDir = filepath.Join(s.homeDir, ".brains", "profiles")
	} else {
		targetDir = filepath.Join(s.workingDir, ".brains", "profiles")
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", &NotInitializedError{Path: targetDir}
	}

	// Check if profile already exists
	filePath := filepath.Join(targetDir, name+".md")
	if _, err := os.Stat(filePath); err == nil {
		return "", &ProfileExistsError{Name: name, Path: filePath}
	}

	// Create profile with template content
	template := fmt.Sprintf(`---
name: %s
description:
includes: []
inherits: true
---

# %s

Add your profile content here.
`, name, toTitleCase(name))

	if err := os.WriteFile(filePath, []byte(template), 0o644); err != nil {
		return "", fmt.Errorf("writing profile: %w", err)
	}

	return filePath, nil
}

// GetInitDir returns the directory path for initialization.
func (s *BrainsSource) GetInitDir(global bool) (string, error) {
	if global {
		if s.homeDir == "" {
			return "", fmt.Errorf("home directory not available")
		}
		return filepath.Join(s.homeDir, ".brains", "profiles"), nil
	}
	return filepath.Join(s.workingDir, ".brains", "profiles"), nil
}

// DefaultInherits returns true - brains profiles inherit by default.
func (s *BrainsSource) DefaultInherits() bool {
	return true
}

// SourceName returns "brains" for error messages.
func (s *BrainsSource) SourceName() string {
	return "brains"
}

// GetResolver returns the underlying Resolver for use by Composer.
func (s *BrainsSource) GetResolver() *Resolver {
	return s.resolver
}
