package skill

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	claude "github.com/2bit-software/zombiekit/internal/recall/claude"
)

// DiscoverableItem represents a Claude Code skill or agent available for import.
type DiscoverableItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "skill" or "agent"
	Description string `json:"description"`
	SourcePath  string `json:"source_path"`
	IsShim      bool   `json:"-"`
}

// DiscoverAll combines skills and agents, sorted by name.
// Returns all non-shim items and a list of name collision warnings.
func DiscoverAll(workingDir string) ([]DiscoverableItem, []string, error) {
	claudeHome := claude.DefaultClaudePath()
	skillDirs, agentDirs := collectDirs(claudeHome, workingDir)
	return discoverAllInDirs(skillDirs, agentDirs)
}

// DiscoverSkills finds all Claude Code skills available for import.
// Excludes skills that are already zombiekit shims.
func DiscoverSkills(workingDir string) ([]DiscoverableItem, error) {
	claudeHome := claude.DefaultClaudePath()
	skillDirs, _ := collectDirs(claudeHome, workingDir)
	return discoverSkillsInDirs(skillDirs)
}

// DiscoverAgents finds all Claude Code agents available for import.
func DiscoverAgents(workingDir string) ([]DiscoverableItem, error) {
	claudeHome := claude.DefaultClaudePath()
	_, agentDirs := collectDirs(claudeHome, workingDir)
	return discoverAgentsInDirs(agentDirs)
}

func collectDirs(claudeHome, workingDir string) (skillDirs, agentDirs []string) {
	if claudeHome != "" {
		skillDirs = append(skillDirs, filepath.Join(claudeHome, "skills"))
		agentDirs = append(agentDirs, filepath.Join(claudeHome, "agents"))
	}
	if workingDir != "" {
		skillDirs = append(skillDirs, filepath.Join(workingDir, ".claude", "skills"))
		agentDirs = append(agentDirs, filepath.Join(workingDir, ".claude", "agents"))
	}
	return skillDirs, agentDirs
}

func discoverAllInDirs(skillDirs, agentDirs []string) ([]DiscoverableItem, []string, error) {
	skills, err := discoverSkillsInDirs(skillDirs)
	if err != nil {
		return nil, nil, err
	}

	agents, err := discoverAgentsInDirs(agentDirs)
	if err != nil {
		return nil, nil, err
	}

	nameSet := make(map[string]string, len(skills))
	for _, s := range skills {
		nameSet[s.Name] = "skill"
	}

	var warnings []string
	for _, a := range agents {
		if existing, ok := nameSet[a.Name]; ok {
			warnings = append(warnings, "name collision: "+a.Name+" exists as both "+existing+" and agent")
		}
	}

	all := append(skills, agents...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all, warnings, nil
}

func discoverSkillsInDirs(dirs []string) ([]DiscoverableItem, error) {
	var items []DiscoverableItem
	seen := make(map[string]bool)

	for _, dir := range dirs {
		resolved, err := resolveSymlink(dir)
		if err != nil {
			continue
		}

		entries, err := os.ReadDir(resolved)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			if seen[name] {
				continue
			}

			skillPath := filepath.Join(resolved, name, "SKILL.md")
			resolvedSkillPath, err := resolveSymlink(skillPath)
			if err != nil {
				continue
			}

			content, err := os.ReadFile(resolvedSkillPath)
			if err != nil {
				continue
			}

			rawFM, body, err := parseRawFrontmatter(content)
			if err != nil {
				continue
			}

			if IsShim(body) {
				continue
			}

			desc, _ := rawFM["description"].(string)

			seen[name] = true
			items = append(items, DiscoverableItem{
				Name:        name,
				Type:        "skill",
				Description: strings.TrimSpace(desc),
				SourcePath:  resolvedSkillPath,
			})
		}
	}

	return items, nil
}

func discoverAgentsInDirs(dirs []string) ([]DiscoverableItem, error) {
	var items []DiscoverableItem
	seen := make(map[string]bool)

	for _, dir := range dirs {
		resolved, err := resolveSymlink(dir)
		if err != nil {
			continue
		}

		entries, err := os.ReadDir(resolved)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".md")
			if seen[name] {
				continue
			}

			agentPath := filepath.Join(resolved, entry.Name())
			resolvedAgentPath, err := resolveSymlink(agentPath)
			if err != nil {
				continue
			}

			content, err := os.ReadFile(resolvedAgentPath)
			if err != nil {
				continue
			}

			rawFM, _, err := parseRawFrontmatter(content)
			if err != nil {
				continue
			}

			desc, _ := rawFM["description"].(string)

			seen[name] = true
			items = append(items, DiscoverableItem{
				Name:        name,
				Type:        "agent",
				Description: strings.TrimSpace(desc),
				SourcePath:  resolvedAgentPath,
			})
		}
	}

	return items, nil
}

func resolveSymlink(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}
