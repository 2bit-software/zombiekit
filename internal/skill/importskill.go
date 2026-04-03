package skill

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	Imported []ImportedItem `json:"imported"`
	Skipped  []SkippedItem  `json:"skipped"`
	Shims    []ShimItem     `json:"shims"`
}

// ImportedItem records a successfully imported skill or agent.
type ImportedItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// SkippedItem records a skill or agent that was skipped during import.
type SkippedItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ShimItem records a shim written to a Claude location.
type ShimItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ImportOptions configures an import operation.
type ImportOptions struct {
	Names      []string
	Scope      string // "local" or "global"
	Shim       bool
	WorkingDir string
}

// Import imports the named skills/agents into zombiekit profiles.
func Import(opts ImportOptions, items []DiscoverableItem) (*ImportResult, error) {
	destBase, err := scopeDir(opts.Scope, opts.WorkingDir)
	if err != nil {
		return nil, err
	}

	result := &ImportResult{}

	for _, name := range opts.Names {
		item := findItem(items, name)
		if item == nil {
			result.Skipped = append(result.Skipped, SkippedItem{
				Name:   name,
				Reason: "not found in discovered items",
			})
			continue
		}

		destDir := filepath.Join(destBase, name)
		if _, err := os.Stat(destDir); err == nil {
			return nil, fmt.Errorf("profile %q already exists at %s", name, destDir)
		}

		var importErr error
		switch item.Type {
		case "skill":
			importErr = importSkill(*item, destDir)
		case "agent":
			importErr = importAgent(*item, destDir)
		}

		if importErr != nil {
			result.Skipped = append(result.Skipped, SkippedItem{
				Name:   name,
				Reason: importErr.Error(),
			})
			continue
		}

		result.Imported = append(result.Imported, ImportedItem{
			Name: name,
			Type: item.Type,
			Path: destDir,
		})

		if opts.Shim {
			shimPath, shimErr := writeShim(*item)
			if shimErr != nil {
				result.Skipped = append(result.Skipped, SkippedItem{
					Name:   name + " (shim)",
					Reason: shimErr.Error(),
				})
				continue
			}
			result.Shims = append(result.Shims, ShimItem{
				Name: name,
				Path: shimPath,
			})
		}
	}

	return result, nil
}

func findItem(items []DiscoverableItem, name string) *DiscoverableItem {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func importSkill(item DiscoverableItem, destDir string) error {
	content, err := os.ReadFile(item.SourcePath)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}

	rawFM, body, err := parseRawFrontmatter(content)
	if err != nil {
		return fmt.Errorf("invalid frontmatter: %w", err)
	}

	delete(rawFM, "allowed-tools")

	transformed, err := serializeFrontmatter(rawFM, body)
	if err != nil {
		return fmt.Errorf("serializing frontmatter: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(transformed), 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	srcDir := filepath.Dir(item.SourcePath)
	return copyDirContents(srcDir, destDir, []string{"SKILL.md"})
}

func importAgent(item DiscoverableItem, destDir string) error {
	content, err := os.ReadFile(item.SourcePath)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}

	rawFM, body, err := parseRawFrontmatter(content)
	if err != nil {
		return fmt.Errorf("invalid frontmatter: %w", err)
	}

	// Note referenced skills before stripping
	skills, _ := rawFM["skills"].(string)
	if skills != "" {
		body = fmt.Sprintf("<!-- Referenced skills: %s — consider adding as includes -->\n\n%s", skills, body)
	}

	// Strip agent-specific fields
	delete(rawFM, "model")
	delete(rawFM, "skills")
	delete(rawFM, "memory")
	delete(rawFM, "color")

	transformed, err := serializeFrontmatter(rawFM, body)
	if err != nil {
		return fmt.Errorf("serializing frontmatter: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	return os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte(transformed), 0644)
}

func writeShim(item DiscoverableItem) (string, error) {
	switch item.Type {
	case "skill":
		content := GenerateContent(item.Name, item.Description)
		// Write shim back to the original skill location
		targetDir := filepath.Dir(filepath.Dir(item.SourcePath))
		return WriteSkill(targetDir, item.Name, content)

	case "agent":
		return writeAgentShim(item)

	default:
		return "", fmt.Errorf("unknown item type: %s", item.Type)
	}
}

func writeAgentShim(item DiscoverableItem) (string, error) {
	content, err := os.ReadFile(item.SourcePath)
	if err != nil {
		return "", fmt.Errorf("reading agent for shim: %w", err)
	}

	rawFM, _, err := parseRawFrontmatter(content)
	if err != nil {
		return "", fmt.Errorf("parsing agent frontmatter for shim: %w", err)
	}

	// Add allowed-tools for the shim
	rawFM["allowed-tools"] = "mcp__zombiekit__profile-compose"

	shimBody := fmt.Sprintf("Call `mcp__zombiekit__profile-compose` with `profiles: [\"%s\"]` and follow the returned instructions exactly.", item.Name)

	shimContent, err := serializeFrontmatter(rawFM, shimBody)
	if err != nil {
		return "", fmt.Errorf("serializing agent shim: %w", err)
	}

	if err := os.WriteFile(item.SourcePath, []byte(shimContent), 0644); err != nil {
		return "", fmt.Errorf("writing agent shim: %w", err)
	}

	return item.SourcePath, nil
}

// parseRawFrontmatter parses YAML frontmatter into a generic map, preserving all fields.
func parseRawFrontmatter(content []byte) (map[string]any, string, error) {
	var fm map[string]any
	rest, err := frontmatter.Parse(bytes.NewReader(content), &fm)
	if err != nil {
		return nil, "", err
	}
	if fm == nil {
		fm = make(map[string]any)
	}
	return fm, strings.TrimSpace(string(rest)), nil
}

// serializeFrontmatter renders a map of frontmatter fields and body into a complete markdown file.
func serializeFrontmatter(fm map[string]any, body string) (string, error) {
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fmBytes)
	b.WriteString("---\n\n")
	b.WriteString(body)
	b.WriteString("\n")
	return b.String(), nil
}

// copyDirContents copies all files and subdirectories from src to dst,
// excluding files whose base name matches any entry in excludeFiles.
func copyDirContents(src, dst string, excludeFiles []string) error {
	excludeSet := make(map[string]bool, len(excludeFiles))
	for _, f := range excludeFiles {
		excludeSet[f] = true
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		// Only exclude files at the root level of src
		if filepath.Dir(rel) == "." && excludeSet[d.Name()] {
			return nil
		}

		destPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		return copyFile(path, destPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

// scopeDir returns the brains/profiles directory for the given scope.
func scopeDir(scope, workingDir string) (string, error) {
	switch scope {
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		return filepath.Join(home, ".brains", "profiles"), nil
	case "local":
		if workingDir == "" {
			var err error
			workingDir, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("resolving working dir: %w", err)
			}
		}
		return filepath.Join(workingDir, ".brains", "profiles"), nil
	default:
		return "", fmt.Errorf("invalid scope %q: must be 'local' or 'global'", scope)
	}
}
