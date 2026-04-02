package rules

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvedDir represents a .brains/rules/ directory that was found.
type ResolvedDir struct {
	Path   string     // Absolute path to the rules directory
	Source RuleSource // The source type (project, parent, global)
}

// Resolver finds and loads rules from .brains/rules/ directories.
type Resolver struct {
	workingDir string
	homeDir    string
}

// NewResolver creates a new Resolver starting from the given working directory.
func NewResolver(workingDir, homeDir string) (*Resolver, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, err
	}

	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}

	return &Resolver{
		workingDir: absWorkingDir,
		homeDir:    homeDir,
	}, nil
}

// FindRulesDirs finds all .brains/rules/ directories from the working directory
// up to the git root, plus the global directory. Returns directories in
// precedence order: local first, parent directories, then global last.
func (r *Resolver) FindRulesDirs() []ResolvedDir {
	dirs := r.walkAncestorRulesDirs()

	if globalDir, ok := r.globalRulesDir(); ok {
		dirs = append(dirs, globalDir)
	}

	return dirs
}

// LoadRules loads all rule files from the given directories.
// Rules from all directories are collected independently (no shadowing).
func (r *Resolver) LoadRules(dirs []ResolvedDir) []*Rule {
	var rules []*Rule

	for _, dir := range dirs {
		dirRules := r.loadRulesFromDir(dir)
		rules = append(rules, dirRules...)
	}

	return rules
}

func (r *Resolver) walkAncestorRulesDirs() []ResolvedDir {
	var dirs []ResolvedDir
	gitRoot := r.findGitRoot()
	current := r.workingDir
	isFirst := true

	for {
		rulesPath := filepath.Join(current, ".brains", "rules")
		if info, err := os.Stat(rulesPath); err == nil && info.IsDir() {
			source := SourceParent
			if isFirst {
				source = SourceProject
			}
			dirs = append(dirs, ResolvedDir{
				Path:   rulesPath,
				Source: source,
			})
		}
		isFirst = false

		if gitRoot != "" && current == gitRoot {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return dirs
}

func (r *Resolver) globalRulesDir() (ResolvedDir, bool) {
	if r.homeDir == "" {
		return ResolvedDir{}, false
	}
	globalPath := filepath.Join(r.homeDir, ".brains", "rules")
	info, err := os.Stat(globalPath)
	if err != nil || !info.IsDir() {
		return ResolvedDir{}, false
	}
	return ResolvedDir{Path: globalPath, Source: SourceGlobal}, true
}

func (r *Resolver) findGitRoot() string {
	current := r.workingDir
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func (r *Resolver) loadRulesFromDir(dir ResolvedDir) []*Rule {
	entries, err := os.ReadDir(dir.Path)
	if err != nil {
		return nil
	}

	var rules []*Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir.Path, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		ruleName := strings.TrimSuffix(entry.Name(), ".md")
		rule, err := ParseRule(content, ruleName, filePath, dir.Source)
		if err != nil {
			continue
		}

		rules = append(rules, rule)
	}

	return rules
}
