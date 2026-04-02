package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	skillNonAlphanumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
	skillMultiHyphen       = regexp.MustCompile(`-{2,}`)
)

// IsSkillDirectory returns true if dir contains a SKILL.md file.
func IsSkillDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// LoadSkillProfile loads a Profile from a skill directory (dir/SKILL.md).
// Sets IsSkill to true on the returned profile.
// Name comes from SKILL.md frontmatter; falls back to normalized directory name.
func LoadSkillProfile(dir string, source ProfileSource) (*Profile, error) {
	skillMD := filepath.Join(dir, "SKILL.md")
	content, err := os.ReadFile(skillMD)
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md in %s: %w", dir, err)
	}

	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing SKILL.md in %s: %w", dir, err)
	}

	name := fm.Name
	if name == "" {
		name = normalizeSkillDirName(filepath.Base(dir))
	}

	return &Profile{
		Name:        name,
		Path:        skillMD,
		Source:      source,
		Description: fm.Description,
		Includes:    fm.Includes,
		Inherits:    fm.GetInherits(),
		Type:        fm.Type,
		Body:        body,
		RawContent:  content,
		IsSkill:     true,
	}, nil
}

// normalizeSkillDirName converts a directory name to a valid profile name slug.
// e.g. "My Cool Skill" → "my-cool-skill", "Epic_Planner" → "epic-planner"
func normalizeSkillDirName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = skillNonAlphanumHyphen.ReplaceAllString(name, "")
	name = skillMultiHyphen.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}
