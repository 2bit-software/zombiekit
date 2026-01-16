package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Service provides the core profile operations.
type Service struct {
	source ProfileSourceInterface
}

// NewService creates a new profile Service with the default brains source.
// If workingDir is empty, it uses the current working directory.
func NewService(workingDir string) (*Service, error) {
	return NewServiceWithSource(SourceTypeBrains, workingDir)
}

// NewServiceWithSource creates a new profile Service with the specified source.
// If workingDir is empty, it uses the current working directory.
func NewServiceWithSource(sourceType SourceType, workingDir string) (*Service, error) {
	source, err := NewSource(sourceType, workingDir)
	if err != nil {
		return nil, err
	}

	return &Service{
		source: source,
	}, nil
}

// NewServiceWithSourceInterface creates a new Service with a pre-configured source.
// This is useful for testing or when the source is already created.
func NewServiceWithSourceInterface(source ProfileSourceInterface) *Service {
	return &Service{
		source: source,
	}
}

// Compose composes the given profile names into a single result.
func (s *Service) Compose(profileNames []string) (*CompositionResult, error) {
	dirs, err := s.source.FindProfileDirs()
	if err != nil {
		return nil, fmt.Errorf("finding profile directories: %w", err)
	}

	profiles, err := s.source.LoadProfiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	composer := NewComposerWithSource(profiles, s.source)
	return composer.Compose(profileNames)
}

// List returns all available profiles from all sources.
func (s *Service) List() ([]ListEntry, error) {
	dirs, err := s.source.FindProfileDirs()
	if err != nil {
		return nil, fmt.Errorf("finding profile directories: %w", err)
	}

	allProfiles, err := s.source.LoadAllProfiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	var entries []ListEntry
	seen := make(map[string]bool)

	for name, versions := range allProfiles {
		for i, p := range versions {
			entry := ListEntry{
				Name:        name,
				Source:      p.Source,
				SourceStr:   p.Source.String(),
				Path:        p.Path,
				Description: p.Description,
				Includes:    p.Includes,
				Inherits:    p.Inherits,
				Shadowed:    i > 0, // First one wins, rest are shadowed
				Model:       p.Model,
				Color:       p.Color,
				Type:        p.Type,
			}
			if !seen[name] || entry.Shadowed {
				entries = append(entries, entry)
			}
			seen[name] = true
		}
	}

	return entries, nil
}

// Show returns the details of a specific profile.
func (s *Service) Show(name string, raw bool) (*ShowResult, error) {
	dirs, err := s.source.FindProfileDirs()
	if err != nil {
		return nil, fmt.Errorf("finding profile directories: %w", err)
	}

	profiles, err := s.source.LoadProfiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	profile, exists := profiles[name]
	if !exists {
		suggestions := s.findSimilar(profiles, name)
		return nil, &ProfileNotFoundError{
			Name:        name,
			Suggestions: suggestions,
		}
	}

	result := &ShowResult{
		Name:        profile.Name,
		Source:      profile.Source,
		SourceStr:   profile.Source.String(),
		Path:        profile.Path,
		Description: profile.Description,
		Includes:    profile.Includes,
		Inherits:    profile.Inherits,
		RawContent:  string(profile.RawContent),
		Model:       profile.Model,
		Color:       profile.Color,
		Type:        profile.Type,
	}

	if raw {
		result.Content = string(profile.RawContent)
	} else {
		// Resolve inheritance if applicable
		if profile.Inherits {
			chain, err := s.source.GetInheritanceChain(name)
			if err == nil && len(chain) > 1 {
				var parts []string
				var inherited []InheritedFrom
				for _, p := range chain {
					if p.Body != "" {
						parts = append(parts, p.Body)
					}
					if p.Path != profile.Path {
						inherited = append(inherited, InheritedFrom{
							Source: p.Source,
							Path:   p.Path,
						})
					}
				}
				result.Content = strings.Join(parts, "\n\n")
				result.InheritedFrom = inherited
			} else {
				result.Content = profile.Body
			}
		} else {
			result.Content = profile.Body
		}
	}

	return result, nil
}

// Create creates a new profile with the given name.
// If global is true, creates in the global directory instead of local.
func (s *Service) Create(name string, global bool) (string, error) {
	// Normalize name
	normalizedName := s.normalizeName(name)
	if err := s.validateName(normalizedName); err != nil {
		return "", err
	}

	return s.source.CreateProfile(normalizedName, global)
}

// GetInitDir returns the directory path for initialization.
// If global is true, returns the global directory.
func (s *Service) GetInitDir(global bool) (string, error) {
	return s.source.GetInitDir(global)
}

// Write writes a profile with the given name and content to disk.
// Location must be "local" or "global".
// If overwrite is false and the profile exists, returns ProfileExistsError.
// Creates the target directory if it doesn't exist.
// Uses atomic write (temp file + rename) for safety.
func (s *Service) Write(name, content, location string, overwrite bool) (string, error) {
	// Normalize name
	normalizedName := s.normalizeName(name)
	if err := s.validateName(normalizedName); err != nil {
		return "", err
	}

	// Validate location
	if location != "local" && location != "global" {
		return "", fmt.Errorf("invalid location %q: must be 'local' or 'global'", location)
	}

	// Get target directory
	targetDir, err := s.source.GetInitDir(location == "global")
	if err != nil {
		return "", fmt.Errorf("getting target directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", targetDir, err)
	}

	// Build file path
	filePath := filepath.Join(targetDir, normalizedName+".md")

	// Check if file exists
	if !overwrite {
		if _, err := os.Stat(filePath); err == nil {
			return "", &ProfileExistsError{Name: normalizedName, Path: filePath}
		}
	}

	// Write atomically: temp file + rename
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		// Clean up temp file on rename failure
		os.Remove(tempFile)
		return "", fmt.Errorf("renaming temp file: %w", err)
	}

	return filePath, nil
}

// SourceName returns the human-readable name of the source.
func (s *Service) SourceName() string {
	return s.source.SourceName()
}

// toTitleCase converts hyphenated names to title case.
func toTitleCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// Validate checks all profiles for errors.
func (s *Service) Validate() (*ValidationResult, error) {
	dirs, err := s.source.FindProfileDirs()
	if err != nil {
		return nil, fmt.Errorf("finding profile directories: %w", err)
	}

	profiles, err := s.source.LoadProfiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	result := &ValidationResult{
		Valid:           true,
		ProfilesChecked: len(profiles),
		Errors:          []ValidationError{},
	}

	// Check for missing includes
	for name, p := range profiles {
		for _, includeName := range p.Includes {
			if _, exists := profiles[includeName]; !exists {
				suggestions := s.findSimilar(profiles, includeName)
				result.Errors = append(result.Errors, ValidationError{
					Profile:     name,
					Code:        "MISSING_INCLUDE",
					Message:     fmt.Sprintf("includes non-existent profile %q", includeName),
					Suggestions: suggestions,
				})
				result.Valid = false
			}
		}
	}

	// Check for cycles
	visited := make(map[string]bool)
	pathSet := make(map[string]bool)
	for name := range profiles {
		if !visited[name] {
			if cycleErr := s.detectCycle(name, profiles, visited, pathSet, nil); cycleErr != nil {
				result.Errors = append(result.Errors, ValidationError{
					Profile: name,
					Code:    "CIRCULAR_DEPENDENCY",
					Message: "circular dependency detected",
					Cycle:   cycleErr.Cycle,
				})
				result.Valid = false
			}
		}
	}

	return result, nil
}

// detectCycle detects cycles in the profile include graph.
func (s *Service) detectCycle(name string, profiles map[string]*Profile, visited, pathSet map[string]bool, path []string) *CycleError {
	pathSet[name] = true
	path = append(path, name)

	profile, exists := profiles[name]
	if !exists {
		delete(pathSet, name)
		return nil
	}

	for _, included := range profile.Includes {
		if pathSet[included] {
			cyclePath := append(path, included)
			return &CycleError{Cycle: cyclePath}
		}
		if visited[included] {
			continue
		}
		if cycleErr := s.detectCycle(included, profiles, visited, pathSet, path); cycleErr != nil {
			return cycleErr
		}
	}

	delete(pathSet, name)
	visited[name] = true
	return nil
}

// normalizeName normalizes a profile name (lowercase, hyphens).
func (s *Service) normalizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	re := regexp.MustCompile(`[^a-z0-9-]`)
	name = re.ReplaceAllString(name, "")
	// Collapse multiple hyphens
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")
	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")
	return name
}

// validateName validates a normalized profile name.
func (s *Service) validateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("profile name too long (max 64 characters)")
	}
	return nil
}

// findSimilar finds profile names similar to the given name.
func (s *Service) findSimilar(profiles map[string]*Profile, name string) []string {
	var suggestions []string
	nameLower := strings.ToLower(name)

	for profileName := range profiles {
		profileLower := strings.ToLower(profileName)
		if strings.Contains(profileLower, nameLower) ||
			strings.Contains(nameLower, profileLower) ||
			(len(nameLower) >= 3 && strings.HasPrefix(profileLower, nameLower[:3])) {
			suggestions = append(suggestions, profileName)
		}
	}

	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions
}

// NotInitializedError is returned when the profile directory doesn't exist.
type NotInitializedError struct {
	Path string
}

func (e *NotInitializedError) Error() string {
	return fmt.Sprintf("profile directory not found at %s (run 'brains init' first)", e.Path)
}

// ProfileExistsError is returned when a profile already exists.
type ProfileExistsError struct {
	Name string
	Path string
}

func (e *ProfileExistsError) Error() string {
	return fmt.Sprintf("profile %q already exists at %s", e.Name, e.Path)
}
