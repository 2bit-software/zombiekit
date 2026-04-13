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
	Name                string     // Filename without extension (e.g., "go")
	FileName            string     // Full filename (e.g., "go.md")
	Source              RuleSource // Where this rule was loaded from
	FilePath            string     // Absolute path to the file
	Paths               []string   // Glob patterns from frontmatter (nil = unconditional)
	Commands            []string   // Command prefix triggers (nil = not a command rule)
	RequiresFiles       []string   // Files that must exist for the rule to fire
	RequiresFilesAbsent []string   // Files that must be missing for the rule to fire
	Body                string     // Markdown content after frontmatter
}

// ID returns the deduplication key: "{source}:{filename}".
func (r *Rule) ID() string {
	return string(r.Source) + ":" + r.FileName
}

// IsUnconditional reports whether the rule should be injected at session
// start. A rule is unconditional only when it has neither path nor command
// triggers — command rules always fire through the Bash hook path instead.
func (r *Rule) IsUnconditional() bool {
	return len(r.Paths) == 0 && len(r.Commands) == 0
}

// RuleFrontmatter is the YAML frontmatter parsed from a rules file.
// This is a superset of Claude Code's rules frontmatter.
type RuleFrontmatter struct {
	Paths               any `yaml:"paths"`                 // string, []string, or nil
	Commands            any `yaml:"commands"`              // string, []string, or nil
	RequiresFiles       any `yaml:"requires_files"`        // string, []string, or nil
	RequiresFilesAbsent any `yaml:"requires_files_absent"` // string, []string, or nil
}

// NormalizedPaths returns paths as a []string regardless of input format.
// Returns nil when no paths are set (unconditional rule).
// Does not split comma-separated strings — commas can appear in brace expansion.
func (f *RuleFrontmatter) NormalizedPaths() []string {
	return normalizeStringList(f.Paths)
}

// NormalizedCommands returns the command triggers as a []string, or nil
// when the rule declares no command triggers.
func (f *RuleFrontmatter) NormalizedCommands() []string {
	return normalizeStringList(f.Commands)
}

// NormalizedRequiresFiles returns the required-present file list, or nil
// when the rule has no presence gate.
func (f *RuleFrontmatter) NormalizedRequiresFiles() []string {
	return normalizeStringList(f.RequiresFiles)
}

// NormalizedRequiresFilesAbsent returns the required-absent file list, or
// nil when the rule has no absence gate.
func (f *RuleFrontmatter) NormalizedRequiresFilesAbsent() []string {
	return normalizeStringList(f.RequiresFilesAbsent)
}

// normalizeStringList coerces a YAML-decoded "string or list of strings"
// value into a []string, dropping empty entries. Returns nil for empty input.
func normalizeStringList(v any) []string {
	if v == nil {
		return nil
	}
	switch typed := v.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []string:
		if len(typed) == 0 {
			return nil
		}
		return typed
	default:
		return nil
	}
}

// SessionState is persisted to /tmp/zk-session-{SESSION_ID}.json.
// InjectedRules keys are composite "ruleID|trigger" strings; file-glob rules
// use an empty trigger, command rules use the matched command prefix.
type SessionState struct {
	SessionID       string               `json:"session_id"`
	Agent           string               `json:"agent"`
	StartedAt       time.Time            `json:"started_at"`
	CompactionCount int                  `json:"compaction_count"`
	InjectedRules   map[string]time.Time `json:"injected_rules"`
}
