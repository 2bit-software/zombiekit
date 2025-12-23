package profile

import (
	"bytes"
	"strings"

	"github.com/adrg/frontmatter"
)

// ClaudeFrontmatter represents the YAML frontmatter in a Claude agent file.
// Claude agents have additional fields like model and color.
type ClaudeFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Model       string   `yaml:"model"`   // Claude model (e.g., "opus", "sonnet")
	Color       string   `yaml:"color"`   // UI color for Claude Code display
	Includes    []string `yaml:"includes"`
	Inherits    *bool    `yaml:"inherits"` // Pointer to detect unset vs explicit false
}

// GetInherits returns the inherits value, defaulting to false for Claude agents.
func (f ClaudeFrontmatter) GetInherits() bool {
	if f.Inherits == nil {
		return false // Claude agents default to not inheriting
	}
	return *f.Inherits
}

// ParseClaudeFrontmatter parses a Claude agent file's content.
func ParseClaudeFrontmatter(content []byte) (ClaudeFrontmatter, string, error) {
	var fm ClaudeFrontmatter
	rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
	if err != nil {
		return ClaudeFrontmatter{}, "", err
	}

	return fm, strings.TrimSpace(string(rest)), nil
}

// ParseClaudeProfile parses a Claude agent file and returns a Profile struct.
func ParseClaudeProfile(content []byte, name, path string, source ProfileSource) (*Profile, error) {
	fm, body, err := ParseClaudeFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Use frontmatter name if provided, otherwise use the passed name
	profileName := fm.Name
	if profileName == "" {
		profileName = name
	}

	return &Profile{
		Name:        profileName,
		Path:        path,
		Source:      source,
		Description: fm.Description,
		Includes:    fm.Includes,
		Inherits:    fm.GetInherits(),
		Body:        body,
		RawContent:  content,
		Model:       fm.Model,
		Color:       fm.Color,
	}, nil
}
