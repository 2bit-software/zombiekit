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
func (s *BrainsSource) FindProfileDirs() ([]ResolvedDirectory, error) {
	return s.resolver.FindProfileDirs()
}

// LoadProfiles loads profiles from the given directories.
func (s *BrainsSource) LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error) {
	return s.resolver.LoadProfiles(dirs)
}

// LoadAllProfiles loads all profiles including shadowed ones.
func (s *BrainsSource) LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error) {
	return s.resolver.LoadAllProfiles(dirs)
}

// GetInheritanceChain returns all versions of a profile for inheritance.
func (s *BrainsSource) GetInheritanceChain(name string) ([]*Profile, error) {
	return s.resolver.GetInheritanceChain(name)
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
