# Technical Spec: Bash Hook Command Warnings

## Data Types

### `internal/rules/types.go`

```go
type Rule struct {
    Name                string
    FileName            string
    Source              RuleSource
    FilePath            string
    Paths               []string // existing
    Commands            []string // NEW — ordered, author-declared
    RequiresFiles       []string // NEW — walk cwd up to repo root
    RequiresFilesAbsent []string // NEW
    Body                string
}

func (r *Rule) IsUnconditional() bool {
    return len(r.Paths) == 0 && len(r.Commands) == 0
}

type RuleFrontmatter struct {
    Paths               any `yaml:"paths"`
    Commands            any `yaml:"commands"`
    RequiresFiles       any `yaml:"requires_files"`
    RequiresFilesAbsent any `yaml:"requires_files_absent"`
}
// Add NormalizedCommands, NormalizedRequiresFiles,
// NormalizedRequiresFilesAbsent — each mirrors NormalizedPaths.
```

### `internal/rules/match.go` (shared by command + file paths)

```go
type RuleMatch struct {
    Rule    *Rule
    Trigger string // "" for file-rule matches, the command prefix for bash matches
}
```

### `internal/hook/types.go`

```go
type ToolInput struct {
    FilePath string      `json:"file_path,omitempty"`
    Edits    []EditEntry `json:"edits,omitempty"`
    Command  string      `json:"command,omitempty"` // NEW
}

type MatchedRule struct {
    ID      string
    Trigger string
}

type HandleResult struct {
    Output        string
    MatchedRules  []MatchedRule // RENAMED from MatchedRuleIDs
    SkippedRules  []MatchedRule // RENAMED from SkippedRuleIDs
}
```

### `internal/hook/state.go` (dedup)

`SessionState.InjectedRules` key format changes from `"{ruleID}"` to
`"{ruleID}|{trigger}"`. Migration on load:

```go
func migrateInjectedKeys(state *SessionState) {
    for k, v := range state.InjectedRules {
        if !strings.Contains(k, "|") {
            state.InjectedRules[k+"|"] = v
            delete(state.InjectedRules, k)
        }
    }
}
```

Called once at the end of `LoadState`.

## Algorithms

### Command tokenization

```go
func SplitSegments(cmd string) []string {
    // Replace &&, ||, ; , | with a sentinel, then split.
    // Order matters: && before &.
    sep := "\x00"
    for _, op := range []string{"&&", "||", ";", "|"} {
        cmd = strings.ReplaceAll(cmd, op, sep)
    }
    raw := strings.Split(cmd, sep)
    out := make([]string, 0, len(raw))
    for _, s := range raw {
        s = strings.TrimSpace(s)
        if s != "" {
            out = append(out, s)
        }
    }
    return out
}

func StripEnvPrefix(segment string) string {
    fields := strings.Fields(segment)
    i := 0
    for i < len(fields) {
        if !isEnvAssignment(fields[i]) {
            break
        }
        i++
    }
    return strings.Join(fields[i:], " ")
}

func isEnvAssignment(s string) bool {
    eq := strings.Index(s, "=")
    if eq <= 0 {
        return false
    }
    name := s[:eq]
    for _, r := range name {
        if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
            return false
        }
    }
    return true
}

func MatchCommandPrefix(segment, matcher string) bool {
    return segment == matcher || strings.HasPrefix(segment, matcher+" ")
}
```

### Rule matching

```go
func MatchRulesByCommand(rules []*Rule, cmd string) []RuleMatch {
    segments := make([]string, 0)
    for _, s := range SplitSegments(cmd) {
        segments = append(segments, StripEnvPrefix(s))
    }
    var out []RuleMatch
    for _, r := range rules {
        if len(r.Commands) == 0 {
            continue
        }
        trigger := firstMatchingCommand(r.Commands, segments)
        if trigger != "" {
            out = append(out, RuleMatch{Rule: r, Trigger: trigger})
        }
    }
    return out
}
```

Iteration order: rules in resolver order (project → parent → global),
commands in author-declared order within each rule. Stable and
deterministic.

### Gate evaluation

```go
func (g *GateResolver) Passes(rule *Rule) bool {
    for _, rel := range rule.RequiresFiles {
        if g.resolve(rel) == "" {
            return false
        }
    }
    for _, rel := range rule.RequiresFilesAbsent {
        if g.resolve(rel) != "" {
            return false
        }
    }
    return true
}

// resolve walks g.cwd upward to g.repoRoot, returning the first hit
// or "" if none.
func (g *GateResolver) resolve(rel string) string
```

`repoRoot` is computed once at `NewGateResolver` time by walking up
for a `.git` directory. Falls back to `cwd` if none found. Single
walk per hook event.

## Hook Flow

```
PreToolUse
├── tool_name == "Bash" → handlePreBash
│   ├── load state (with migration)
│   ├── rules.ResolveForCommand(cmd, cwd)
│   │   ├── loadAll rules
│   │   ├── filter rules with Commands set
│   │   ├── gate.Passes filter
│   │   └── MatchRulesByCommand → []RuleMatch
│   ├── for each match: dedup check (ruleID|trigger)
│   ├── mark injected, collect bodies
│   ├── SaveState
│   └── FormatPreToolOutput(bodies)
└── file-path tool → existing path, updated to use MatchedRule shape
```

## Back-compat

- Existing state files loaded via `LoadState` are migrated in-place;
  no format change on disk is required beyond the first re-save.
- Existing file-glob rules are untouched. A rule with only `paths:`
  behaves identically.
- `HandleResult` field rename is internal; no JSON protocol exposure.
- `hook_log.go` (uncommitted) needs to consume `[]MatchedRule` instead
  of `[]string`. Trivial one-line change at each call site.

## Test Matrix

| Case | Expectation |
|---|---|
| `go test ./...` with rule `commands: [go test]` | fires, trigger = "go test" |
| `go test ./...` then `go test -count=1` | fires once, then deduped |
| `go test` then `go run main.go`, same rule | both fire, triggers "go test" and "go run" |
| `gopher test-helper` | no match |
| `CGO_ENABLED=0 go test` | fires |
| `cd x && go test` | fires |
| rule with `requires_files: [Taskfile.yml]`, no Taskfile | does not fire |
| rule with `requires_files_absent: [Taskfile.yml]`, no Taskfile | fires |
| both required and absent set, both satisfied | fires |
| legacy state with bare `ruleID` key | loads, migrates, subsequent dedup works |
| same rule has both `paths:` and `commands:` | fires on file path, fires on bash — independent |
