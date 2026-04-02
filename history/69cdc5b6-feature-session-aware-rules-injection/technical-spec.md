# Technical Specification: Session-Aware Rules Injection

## Package Layout

```
internal/
  rules/
    types.go          # Rule, RuleFrontmatter, RuleSource
    frontmatter.go    # ParseRule(), paths normalization
    resolver.go       # FindRulesDirs(), LoadRules()
    matcher.go        # MatchRules(), IsUnconditional()
    service.go        # Service (top-level API)
    types_test.go
    frontmatter_test.go
    resolver_test.go
    matcher_test.go
    service_test.go
  hook/
    types.go          # HookEvent, SessionState, Agent
    session.go        # LoadState(), SaveState(), DeleteState()
    agent.go          # DetectAgent(), FormatOutput()
    handler.go        # Handler.Handle() dispatch
    handler_test.go
    session_test.go
    agent_test.go
  cli/
    hook.go           # newHookCommand() CLI wiring
```

## Type Definitions

### internal/rules/types.go

```go
package rules

import "time"

type RuleSource string

const (
    SourceProject RuleSource = "project"
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

// IsUnconditional returns true if the rule has no path patterns.
func (r *Rule) IsUnconditional() bool {
    return len(r.Paths) == 0
}

// RuleFrontmatter is the YAML frontmatter parsed from a rules file.
// Superset of Claude Code's rules frontmatter (v1: paths only).
type RuleFrontmatter struct {
    Paths interface{} `yaml:"paths"` // string, []string, or nil
}

// NormalizedPaths returns paths as a []string regardless of input format.
func (f *RuleFrontmatter) NormalizedPaths() []string {
    // Handles: nil, string, []string, []interface{}
    // Returns nil for no paths (unconditional)
    // Does NOT split comma-separated strings (commas can appear in brace expansion)
}
```

### internal/hook/types.go

```go
package hook

import "time"

type Agent string

const (
    AgentClaude Agent = "claude"
    AgentGemini Agent = "gemini"
)

// HookEvent is the JSON payload received from the agent's hook system via stdin.
type HookEvent struct {
    SessionID     string        `json:"session_id"`
    HookEventName string        `json:"hook_event_name"`
    CWD           string        `json:"cwd"`
    Source        string        `json:"source,omitempty"`         // SessionStart only
    ToolName      string        `json:"tool_name,omitempty"`      // PostToolUse only
    ToolInput     *ToolInput    `json:"tool_input,omitempty"`     // PostToolUse only
    ToolResponse  *ToolResponse `json:"tool_response,omitempty"`  // PostToolUse only
}

type ToolInput struct {
    FilePath string      `json:"file_path,omitempty"`
    Edits    []EditEntry `json:"edits,omitempty"` // MultiEdit only
}

type EditEntry struct {
    FilePath  string `json:"file_path"`
    OldString string `json:"old_string"`
    NewString string `json:"new_string"`
}

type ToolResponse struct {
    FilePath string `json:"filePath,omitempty"` // Note: camelCase from Claude Code
    Success  bool   `json:"success,omitempty"`
}

// SessionState is persisted to /tmp/zk-session-{SESSION_ID}.json.
type SessionState struct {
    SessionID       string               `json:"session_id"`
    Agent           Agent                `json:"agent"`
    StartedAt       time.Time            `json:"started_at"`
    CompactionCount int                  `json:"compaction_count"`
    InjectedRules   map[string]time.Time `json:"injected_rules"`
}
```

## Key Interfaces

### internal/rules/service.go

```go
package rules

type Service struct {
    workingDir string
    homeDir    string
}

func NewService(workingDir, homeDir string) *Service

// ResolveForFile returns all rules matching the given file path
// that have not been filtered by the caller.
func (s *Service) ResolveForFile(filePath string) ([]*Rule, error)

// ResolveUnconditional returns all rules with no paths field.
func (s *Service) ResolveUnconditional() ([]*Rule, error)

// ResolveForFiles returns deduplicated rules matching any of the given file paths.
func (s *Service) ResolveForFiles(filePaths []string) ([]*Rule, error)
```

### internal/hook/handler.go

```go
package hook

import "github.com/2bit-software/zombiekit/internal/rules"

type Handler struct {
    rules *rules.Service
    agent Agent
}

func NewHandler(workingDir, homeDir string, agent Agent) (*Handler, error)

// Handle dispatches the hook event and returns text to write to stdout.
// Returns empty string when no rules need injection.
func (h *Handler) Handle(event *HookEvent) (string, error)
```

## Event Flow

### SessionStart (startup | resume | compact)

```
stdin JSON → parse HookEvent
  → LoadState (or create fresh)
  → ResetInjectedRules (clear map)
  → if source == "compact": increment CompactionCount
  → ResolveUnconditional() → get all unconditional rules
  → filter out already-injected (should be empty after reset)
  → mark each as injected in state
  → SaveState
  → FormatOutput(agent, concatenated rule bodies)
  → print to stdout
```

### PostToolUse (Read | Write | Edit | MultiEdit)

```
stdin JSON → parse HookEvent
  → extract file paths (per tool type)
  → LoadState (or create fresh if missing)
  → for each file path:
      → ResolveForFile(path) → matching rules
      → filter out rules already in state.InjectedRules
  → deduplicate across all paths
  → mark each new rule as injected in state
  → SaveState
  → FormatOutput(agent, concatenated new rule bodies)
  → print to stdout
```

### SessionEnd

```
stdin JSON → parse HookEvent
  → DeleteState(sessionID)
  → exit 0 (no output)
```

## Output Format

### Claude Code (CLAUDE_SESSION_ID set)

```
<system-reminder>
# Go Standards

- Use `any` instead of `interface{}`
- Always check errors with context

# General Coding Standards

- Compare alternative approaches with pros and cons
</system-reminder>
```

Single `<system-reminder>` wrapper around all rules, concatenated with a blank line separator. This minimizes token overhead.

### Gemini CLI (GEMINI_SESSION_ID set)

```
# Go Standards

- Use `any` instead of `interface{}`
- Always check errors with context

# General Coding Standards

- Compare alternative approaches with pros and cons
```

Plain markdown, rules concatenated with blank line separator.

## Glob Matching Details

Using `doublestar.Match()` with forward-slash normalized paths.

```go
import "github.com/bmatcuk/doublestar/v4"

func matchRule(rule *Rule, filePath string) bool {
    normalized := filepath.ToSlash(filePath)
    for _, pattern := range rule.Paths {
        matched, err := doublestar.Match(pattern, normalized)
        if err == nil && matched {
            return true
        }
    }
    return false
}
```

Supported patterns:
- `**/*.go` — all Go files in any directory
- `src/**/*.{ts,tsx}` — TypeScript files under src/
- `*.md` — markdown files in project root
- `src/api/**` — all files under src/api/

## Session State File Lifecycle

| Event | Action |
|-------|--------|
| SessionStart startup | Create new state file, inject unconditional rules |
| SessionStart resume | Reset injected rules, inject unconditional rules |
| SessionStart compact | Reset injected rules, increment compaction_count, inject unconditional rules |
| PostToolUse | Load state, inject new matching rules, update state |
| SessionEnd | Delete state file |
| State missing mid-session | Create fresh state, proceed as new session |
| State corrupted | Delete and create fresh state |

## CLI Command Structure

```
brains hook --event session-start    # SessionStart handler
brains hook --event post-tool-use    # PostToolUse handler
brains hook --event session-end      # SessionEnd handler
```

No subcommands — single `hook` command with `--event` flag. The flag value maps directly to the handler method.

## Error Handling

- All errors exit with code 1 (non-blocking per hook protocol)
- Errors are written to stderr, never stdout (stdout is the injection channel)
- State file errors (missing, corrupt) are recovered silently — never fail the hook
- Rules directory missing → skip silently
- Invalid JSON on stdin → exit 1, stderr message
- No matching rules → exit 0, empty stdout

## Performance Budget

Target: < 100ms p99 per invocation.

| Operation | Budget |
|-----------|--------|
| JSON stdin parse | ~0.1ms |
| State file read | ~0.2ms |
| Rules directory scan + file reads | ~2-5ms |
| Glob matching | ~0.1ms per rule |
| State file write | ~0.2ms |
| **Total** | **~3-6ms typical** |

Well within budget. The Go binary startup time (~5-10ms) is the dominant cost.
