// Package rules provides session-aware rules loading, matching, and composition
// for injecting file-type-specific guidance into AI coding agent contexts.
package rules

import "time"

// RuleSource indicates where a rule was loaded from.
type RuleSource string

const (
	SourceProject RuleSource = "project"
	SourceParent  RuleSource = "parent"
	SourceGlobal  RuleSource = "global"
)

// Rule represents a single rules file loaded from disk.
type Rule struct {
	Name     string     // Filename without extension (e.g., "go")
	FileName string     // Full filename (e.g., "go.md")
	Source   RuleSource // Where this rule was loaded from
	FilePath string     // Absolute path to the file
	Paths    []string   // Glob patterns from frontmatter (nil = unconditional)
	Body     string     // Markdown content after frontmatter
}

// ID returns the deduplication key: "{source}:{filename}".
func (r *Rule) ID() string {
	return string(r.Source) + ":" + r.FileName
}

// IsUnconditional reports whether the rule has no path patterns
// and should be injected at session start.
func (r *Rule) IsUnconditional() bool {
	return len(r.Paths) == 0
}

// RuleFrontmatter is the YAML frontmatter parsed from a rules file.
// This is a superset of Claude Code's rules frontmatter (v1: paths only).
type RuleFrontmatter struct {
	Paths any `yaml:"paths"` // string, []string, or nil
}

// NormalizedPaths returns paths as a []string regardless of input format.
// Returns nil when no paths are set (unconditional rule).
// Does not split comma-separated strings — commas can appear in brace expansion.
func (f *RuleFrontmatter) NormalizedPaths() []string {
	if f.Paths == nil {
		return nil
	}

	switch v := f.Paths.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []any:
		paths := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				paths = append(paths, s)
			}
		}
		if len(paths) == 0 {
			return nil
		}
		return paths
	case []string:
		if len(v) == 0 {
			return nil
		}
		return v
	default:
		return nil
	}
}

// SessionState is persisted to /tmp/zk-session-{SESSION_ID}.json.
type SessionState struct {
	SessionID       string               `json:"session_id"`
	Agent           string               `json:"agent"`
	StartedAt       time.Time            `json:"started_at"`
	CompactionCount int                  `json:"compaction_count"`
	InjectedRules   map[string]time.Time `json:"injected_rules"`
}
