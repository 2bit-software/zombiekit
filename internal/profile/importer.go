package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Importer handles importing profiles from external sources to brains format.
type Importer struct {
	claudeSource *ClaudeSource
	workingDir   string
	homeDir      string
}

// NewImporter creates an Importer for the given working directory.
func NewImporter(workingDir string) (*Importer, error) {
	return NewImporterWithHomeDir(workingDir, "")
}

// NewImporterWithHomeDir creates an Importer with custom home directory (for testing).
func NewImporterWithHomeDir(workingDir, homeDir string) (*Importer, error) {
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

	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}

	claudeSource, err := NewClaudeSource(absWorkingDir)
	if err != nil {
		return nil, fmt.Errorf("creating claude source: %w", err)
	}
	// Override home directory for custom home (used in tests)
	claudeSource.homeDir = homeDir

	return &Importer{
		claudeSource: claudeSource,
		workingDir:   absWorkingDir,
		homeDir:      homeDir,
	}, nil
}

// Import imports all Claude agents to brains profiles.
// If dryRun is true, no files are written.
func (i *Importer) Import(dryRun bool) (*ImportResult, error) {
	result := &ImportResult{
		DryRun:           dryRun,
		CreatedPaths:     []string{},
		OverwrittenPaths: []string{},
		FailedAgents:     []ImportFailure{},
	}

	// Find Claude agent directories
	dirs, err := i.claudeSource.FindProfileDirs()
	if err != nil {
		return nil, fmt.Errorf("finding claude directories: %w", err)
	}

	// Handle case when no Claude directories exist
	if len(dirs) == 0 {
		return result, nil
	}

	// Load all Claude agents (including shadowed ones)
	allProfiles, err := i.claudeSource.LoadAllProfiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("loading claude agents: %w", err)
	}

	// Process each agent
	for _, profiles := range allProfiles {
		for _, profile := range profiles {
			if err := i.importProfile(profile, dryRun, result); err != nil {
				result.Failed++
				result.FailedAgents = append(result.FailedAgents, ImportFailure{
					AgentName: profile.Name,
					AgentPath: profile.Path,
					Error:     err.Error(),
				})
			}
		}
	}

	return result, nil
}

// importProfile imports a single Claude agent to a brains profile.
func (i *Importer) importProfile(agent *Profile, dryRun bool, result *ImportResult) error {
	// Determine target directory based on source scope
	var targetDir string
	switch agent.Source {
	case SourceLocal:
		targetDir = filepath.Join(i.workingDir, ".brains", "profiles")
	case SourceGlobal:
		targetDir = filepath.Join(i.homeDir, ".brains", "profiles")
	default:
		return fmt.Errorf("unsupported source type: %s", agent.Source.String())
	}

	targetPath := filepath.Join(targetDir, agent.Name+".md")

	// Check if target already exists
	exists := false
	if _, err := os.Stat(targetPath); err == nil {
		exists = true
	}

	// Convert Claude frontmatter to brains format
	content, err := i.convertClaudeToBrains(agent)
	if err != nil {
		return fmt.Errorf("converting frontmatter: %w", err)
	}

	if !dryRun {
		// Ensure target directory exists
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Write the profile
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing profile: %w", err)
		}
	}

	// Track result
	if exists {
		result.Overwritten++
		result.OverwrittenPaths = append(result.OverwrittenPaths, targetPath)
	} else {
		result.Created++
		result.CreatedPaths = append(result.CreatedPaths, targetPath)
	}

	return nil
}

// BrainsFrontmatter represents the YAML frontmatter for brains profiles.
type BrainsFrontmatter struct {
	Name        string   `yaml:"name,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Includes    []string `yaml:"includes,omitempty"`
	Inherits    bool     `yaml:"inherits"`
}

// convertClaudeToBrains converts a Claude agent to brains profile format.
// It preserves name, description, includes; discards model, color; forces inherits to false.
func (i *Importer) convertClaudeToBrains(agent *Profile) (string, error) {
	// Build brains frontmatter
	fm := BrainsFrontmatter{
		Name:        agent.Name,
		Description: agent.Description,
		Inherits:    false, // Always false for imported profiles
	}

	// Only include non-empty includes
	if len(agent.Includes) > 0 {
		fm.Includes = agent.Includes
	}

	// Marshal frontmatter to YAML
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	// Build final content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n")

	// Add body if present
	if agent.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(agent.Body)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
