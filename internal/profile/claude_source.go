package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClaudeSource implements ProfileSourceInterface for Claude agents.
// It reads from .claude/agents/ directories (local and global only).
type ClaudeSource struct {
	workingDir string
	homeDir    string
}

// NewClaudeSource creates a new ClaudeSource.
// If workingDir is empty, it uses the current working directory.
func NewClaudeSource(workingDir string) (*ClaudeSource, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	homeDir, _ := os.UserHomeDir()

	return &ClaudeSource{
		workingDir: absWorkingDir,
		homeDir:    homeDir,
	}, nil
}

// FindProfileDirs discovers .claude/agents/ directories.
// Claude only uses local and global (no parent traversal).
func (s *ClaudeSource) FindProfileDirs() ([]ResolvedDirectory, error) {
	var dirs []ResolvedDirectory

	// Local: {CWD}/.claude/agents/
	localPath := filepath.Join(s.workingDir, ".claude", "agents")
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		dirs = append(dirs, ResolvedDirectory{
			Path:   localPath,
			Source: SourceLocal,
		})
	}

	// Global: ~/.claude/agents/
	if s.homeDir != "" {
		globalPath := filepath.Join(s.homeDir, ".claude", "agents")
		if info, err := os.Stat(globalPath); err == nil && info.IsDir() {
			dirs = append(dirs, ResolvedDirectory{
				Path:   globalPath,
				Source: SourceGlobal,
			})
		}
	}

	return dirs, nil
}

// LoadProfiles loads Claude agents from the given directories.
func (s *ClaudeSource) LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	for _, dir := range dirs {
		dirProfiles, err := s.loadProfilesFromDir(dir)
		if err != nil {
			continue
		}

		for name, profile := range dirProfiles {
			if _, exists := profiles[name]; !exists {
				profiles[name] = profile
			}
		}
	}

	return profiles, nil
}

// LoadAllProfiles loads all Claude agents including shadowed ones.
func (s *ClaudeSource) LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error) {
	profiles := make(map[string][]*Profile)

	for _, dir := range dirs {
		dirProfiles, err := s.loadProfilesFromDir(dir)
		if err != nil {
			continue
		}

		for name, profile := range dirProfiles {
			profiles[name] = append(profiles[name], profile)
		}
	}

	return profiles, nil
}

// loadProfilesFromDir loads all .md files from a single agents directory.
func (s *ClaudeSource) loadProfilesFromDir(dir ResolvedDirectory) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	entries, err := os.ReadDir(dir.Path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir.Path, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Extract agent name from filename (without .md extension)
		agentName := strings.TrimSuffix(name, ".md")
		filePath := filepath.Join(dir.Path, name)

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		profile, err := ParseClaudeProfile(content, agentName, filePath, dir.Source)
		if err != nil {
			continue
		}

		profiles[agentName] = profile
	}

	return profiles, nil
}

// GetInheritanceChain returns all versions of an agent for inheritance.
// Claude agents have simpler inheritance: just local > global.
func (s *ClaudeSource) GetInheritanceChain(name string) ([]*Profile, error) {
	dirs, err := s.FindProfileDirs()
	if err != nil {
		return nil, err
	}

	allProfiles, err := s.LoadAllProfiles(dirs)
	if err != nil {
		return nil, err
	}

	versions := allProfiles[name]
	if len(versions) == 0 {
		return nil, nil
	}

	// Reverse: we collected in precedence order (local first),
	// but for inheritance we need global first
	chain := make([]*Profile, len(versions))
	for i, p := range versions {
		chain[len(versions)-1-i] = p
	}

	return chain, nil
}

// CreateProfile creates a new Claude agent.
func (s *ClaudeSource) CreateProfile(name string, global bool) (string, error) {
	// Determine target directory
	var targetDir string
	if global {
		if s.homeDir == "" {
			return "", fmt.Errorf("home directory not available")
		}
		targetDir = filepath.Join(s.homeDir, ".claude", "agents")
	} else {
		targetDir = filepath.Join(s.workingDir, ".claude", "agents")
	}

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", &NotInitializedError{Path: targetDir}
	}

	// Check if agent already exists
	filePath := filepath.Join(targetDir, name+".md")
	if _, err := os.Stat(filePath); err == nil {
		return "", &ProfileExistsError{Name: name, Path: filePath}
	}

	// Create agent with Claude template
	template := fmt.Sprintf(`---
name: %s
description:
model:
color:
includes: []
inherits: false
---

Add your agent instructions here.
`, name)

	if err := os.WriteFile(filePath, []byte(template), 0o644); err != nil {
		return "", fmt.Errorf("writing agent: %w", err)
	}

	return filePath, nil
}

// GetInitDir returns the directory path for initialization.
func (s *ClaudeSource) GetInitDir(global bool) (string, error) {
	if global {
		if s.homeDir == "" {
			return "", fmt.Errorf("home directory not available")
		}
		return filepath.Join(s.homeDir, ".claude", "agents"), nil
	}
	return filepath.Join(s.workingDir, ".claude", "agents"), nil
}

// DefaultInherits returns false - Claude agents don't inherit by default.
func (s *ClaudeSource) DefaultInherits() bool {
	return false
}

// SourceName returns "claude" for error messages.
func (s *ClaudeSource) SourceName() string {
	return "claude"
}

// findSimilar finds agent names similar to the given name.
func (s *ClaudeSource) findSimilar(profiles map[string]*Profile, name string) []string {
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
