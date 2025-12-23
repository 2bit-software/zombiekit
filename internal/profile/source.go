package profile

import (
	"fmt"
)

// SourceType represents the type of profile source.
type SourceType string

const (
	// SourceTypeBrains is the default source using .brains/profiles/ directories.
	SourceTypeBrains SourceType = "brains"
	// SourceTypeClaude uses .claude/agents/ directories.
	SourceTypeClaude SourceType = "claude"
)

// ValidSourceTypes returns all valid source type values.
func ValidSourceTypes() []SourceType {
	return []SourceType{SourceTypeBrains, SourceTypeClaude}
}

// String returns the string representation of the source type.
func (s SourceType) String() string {
	return string(s)
}

// ParseSourceType parses a string into a SourceType.
// Returns an error if the string is not a valid source type.
func ParseSourceType(s string) (SourceType, error) {
	switch s {
	case "brains", "":
		return SourceTypeBrains, nil
	case "claude":
		return SourceTypeClaude, nil
	default:
		return "", fmt.Errorf("unknown source type %q (valid: brains, claude)", s)
	}
}

// ProfileSourceInterface abstracts profile operations across different backends.
// This interface enables the CLI to work with different profile sources
// (brains profiles, Claude agents) using a common API.
type ProfileSourceInterface interface {
	// FindProfileDirs discovers available profile directories.
	// Returns directories in precedence order (highest precedence first).
	FindProfileDirs() ([]ResolvedDirectory, error)

	// LoadProfiles loads profiles from the given directories.
	// Profiles are keyed by name, with earlier directories taking precedence.
	LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error)

	// LoadAllProfiles loads all profiles including shadowed ones.
	// Returns profiles grouped by name, with all versions from different sources.
	LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error)

	// GetInheritanceChain returns all versions of a profile for inheritance resolution.
	// Returns profiles in order from global to local.
	GetInheritanceChain(name string) ([]*Profile, error)

	// CreateProfile creates a new profile with the given name.
	// If global is true, creates in the global directory.
	// Returns the path to the created file.
	CreateProfile(name string, global bool) (string, error)

	// GetInitDir returns the directory path for initialization.
	// If global is true, returns the global directory.
	GetInitDir(global bool) (string, error)

	// DefaultInherits returns the default value for the inherits field.
	// Brains defaults to true, Claude defaults to false.
	DefaultInherits() bool

	// SourceName returns a human-readable name for error messages.
	SourceName() string
}

// NewSource creates a ProfileSourceInterface for the given source type.
// If workingDir is empty, it uses the current working directory.
func NewSource(sourceType SourceType, workingDir string) (ProfileSourceInterface, error) {
	switch sourceType {
	case SourceTypeBrains:
		return NewBrainsSource(workingDir)
	case SourceTypeClaude:
		return NewClaudeSource(workingDir)
	default:
		return nil, fmt.Errorf("unknown source type %q", sourceType)
	}
}
