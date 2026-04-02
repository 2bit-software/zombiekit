// Package skill provides utilities for installing Claude Code skills.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateName checks that name is safe to use as a skill directory name.
// Valid names are lowercase, alphanumeric with interior hyphens (e.g. "my-skill").
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid skill name %q. Use lowercase letters, digits, and hyphens (e.g. 'my-skill')", name)
	}
	return nil
}

// TargetDir returns the Claude skills directory for the given scope.
// When global is true, returns ~/.claude/skills/. Otherwise returns {workingDir}/.claude/skills/.
// If workingDir is empty, uses the process working directory.
func TargetDir(global bool, workingDir string) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		return filepath.Join(home, ".claude", "skills"), nil
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolving working dir: %w", err)
		}
	}
	return filepath.Join(workingDir, ".claude", "skills"), nil
}

// GenerateContent produces the SKILL.md file content for the given profile name and description.
// If description is empty, a generic fallback is used.
// The body delegates to mcp__zombiekit__profile-compose so the skill stays live without reinstalling.
func GenerateContent(name, description string) string {
	if description == "" {
		description = fmt.Sprintf("Delegates to the %s profile via profile-compose.", name)
	}
	// Preserve multi-line descriptions with proper YAML block scalar indentation.
	lines := strings.Split(description, "\n")
	indented := strings.Join(lines, "\n  ")
	return fmt.Sprintf(
		"---\nname: %s\ndescription: >\n  %s\nallowed-tools: mcp__zombiekit__profile-compose\n---\n\nCall `mcp__zombiekit__profile-compose` with `profiles: [\"%s\"]` and follow the returned instructions exactly.\n",
		name, indented, name,
	)
}

// WriteSkill creates {targetDir}/{name}/SKILL.md with the given content.
// Creates intermediate directories as needed. Idempotent — overwrites existing SKILL.md.
// Returns the full path to the written file on success.
func WriteSkill(targetDir, name, content string) (string, error) {
	skillDir := filepath.Join(targetDir, name)
	skillPath := filepath.Join(skillDir, "SKILL.md")

	if info, err := os.Stat(skillDir); err == nil && !info.IsDir() {
		return "", fmt.Errorf("%q exists as a file at %s. Remove it manually or choose a different name", name, skillDir)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("creating skill directory: %w", err)
	}

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing SKILL.md: %w", err)
	}

	return skillPath, nil
}
