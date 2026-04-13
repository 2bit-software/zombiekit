# Implementation Plan: Bash Hook Command Warnings

Traces to `spec.md` (FR-001..FR-008) and `research.md`.

## Ordered Steps

### 1. Extend `RuleFrontmatter` and `Rule` (internal/rules/types.go)
Add fields + normalization helpers. No behavior change yet.

- `RuleFrontmatter.Commands any` (string | []string | nil) + `NormalizedCommands() []string`
- `RuleFrontmatter.RequiresFiles any` + `NormalizedRequiresFiles() []string`
- `RuleFrontmatter.RequiresFilesAbsent any` + `NormalizedRequiresFilesAbsent() []string`
- `Rule.Commands []string`
- `Rule.RequiresFiles []string`
- `Rule.RequiresFilesAbsent []string`
- `Rule.IsUnconditional()` update: return `len(r.Paths) == 0 && len(r.Commands) == 0`. A rule with only `commands:` must NOT fire at SessionStart. (FR-005)

Resolver (`internal/rules/resolver.go`, already loads frontmatter — not yet read, confirm during implementation) must populate the new fields from normalized frontmatter.

**Depends on**: nothing.
**Traces to**: FR-002, FR-004, FR-004b, FR-005.

### 2. Command tokenizer + matcher (internal/rules/command_matcher.go, new file)

Pure functions, no I/O. Easy to unit test.

```go
// SplitSegments splits a raw command on top-level &&, ||, ;, | separators.
// Does not parse quoting or subshells — naive split, documented as such.
func SplitSegments(cmd string) []string

// StripEnvPrefix removes leading VAR=value assignments from a segment.
// "CGO_ENABLED=0 go test ./..." -> "go test ./..."
func StripEnvPrefix(segment string) string

// MatchCommandPrefix reports whether `segment` matches `matcher` as a
// whole-token prefix: segment == matcher || strings.HasPrefix(segment, matcher+" ").
func MatchCommandPrefix(segment, matcher string) bool

// MatchRulesByCommand returns (rule, matchedTrigger) pairs for every rule
// whose Commands contain a prefix that matches any segment of cmd.
// Only the FIRST matching trigger per rule is returned (stable on author order).
func MatchRulesByCommand(rules []*Rule, cmd string) []RuleMatch
```

`RuleMatch` struct: `{ Rule *Rule; Trigger string }`. Used everywhere a rule+trigger pair flows.

**Depends on**: step 1 (Rule.Commands).
**Traces to**: FR-003.

### 3. File-existence gate (internal/rules/gate.go, new file)

```go
// GateResolver checks requires_files / requires_files_absent against disk.
type GateResolver struct {
    cwd      string
    repoRoot string // walked from cwd to nearest .git, or cwd if none
    stat     func(string) (os.FileInfo, error) // injectable for tests
}

func NewGateResolver(cwd string) *GateResolver
func (g *GateResolver) Passes(rule *Rule) bool
```

`Passes` returns false if any `RequiresFiles` entry is missing (when walked from cwd up to repoRoot), or any `RequiresFilesAbsent` entry is present. Both nil → passes.

Filesystem access is behind the injected `stat` func so tests don't need real files. The default uses `os.Stat`.

**Depends on**: step 1.
**Traces to**: FR-004, FR-004b, FR-004c.

### 4. Service resolution path (internal/rules/service.go)

Add one new method:

```go
// ResolveForCommand returns (rule, trigger) pairs for rules whose
// commands match any segment of `cmd` AND whose file-existence gates pass.
func (s *Service) ResolveForCommand(cmd, cwd string) ([]RuleMatch, error)
```

Internally: `loadAll` → filter to rules with non-empty `Commands` → run `GateResolver.Passes` → `MatchRulesByCommand` → drop empty-body rules. Keep `ResolveForFiles` untouched.

Also refactor `MatchRules` (file-glob path) to skip rules that have `commands:` set — those belong only on the command path, even if `paths:` is also set. (Actually, spec allows both — re-read: "A rule may declare both `paths:` and `commands:` — they're independent dispatch lanes and never fire in the same hook event." Keep the rule eligible on both paths; the hook just won't call both paths for one event.)

**Depends on**: steps 1-3.
**Traces to**: FR-002, FR-004c.

### 5. Dedup key refactor (internal/hook/state.go or wherever IsRuleInjected lives)

Change `SessionState.InjectedRules` key from bare `ruleID` to composite `ruleID|trigger`. Back-compat: file rules use empty trigger → key is `ruleID|`, loaded JSON with bare `ruleID` still works if we migrate on load (one-time rewrite in `LoadState`).

New helpers:

```go
func IsRuleInjectedFor(state *SessionState, ruleID, trigger string) bool
func MarkRuleInjectedFor(state *SessionState, ruleID, trigger string)
```

Keep old `IsRuleInjected(state, id)` as a thin wrapper passing `trigger=""` so existing callers compile during refactor, then remove once all callers updated.

Migration on load: for each old key that has no `|`, rewrite to `key + "|"`.

**Depends on**: step 2 (RuleMatch concept).
**Traces to**: FR-004d.

### 6. HandleResult carries triggers (internal/hook/handler.go)

Change `HandleResult.MatchedRuleIDs []string` to `MatchedRules []MatchedRule`, where `MatchedRule { ID, Trigger string }`. Same for `SkippedRuleIDs`. Update all call sites in `internal/cli/hook.go` and any audit code.

File-rule path populates entries with `Trigger: ""`.

**Depends on**: step 5.
**Traces to**: FR-004e, FR-008.

### 7. Extend `ToolInput.Command` (internal/hook/types.go)

Add `Command string `json:"command,omitempty"`` to the existing struct. One line.

**Depends on**: nothing.
**Traces to**: FR-001.

### 8. PreToolUse Bash branch (internal/hook/handler.go)

Extend `handlePreToolUse`:

```go
func (h *Handler) handlePreToolUse(event *HookEvent) (HandleResult, error) {
    if event.ToolName == "Bash" {
        return h.handlePreBash(event)
    }
    // existing file-path path, unchanged except HandleResult shape
}

func (h *Handler) handlePreBash(event *HookEvent) (HandleResult, error) {
    if event.ToolInput == nil || event.ToolInput.Command == "" {
        return HandleResult{}, nil
    }
    state := LoadState(event.SessionID, h.agent)
    matches, err := h.rules.ResolveForCommand(event.ToolInput.Command, event.CWD)
    if err != nil {
        return HandleResult{}, err
    }
    var bodies []string
    var matched, skipped []MatchedRule
    for _, m := range matches {
        if IsRuleInjectedFor(state, m.Rule.ID(), m.Trigger) {
            skipped = append(skipped, MatchedRule{m.Rule.ID(), m.Trigger})
            continue
        }
        MarkRuleInjectedFor(state, m.Rule.ID(), m.Trigger)
        matched = append(matched, MatchedRule{m.Rule.ID(), m.Trigger})
        bodies = append(bodies, m.Rule.Body)
    }
    if err := SaveState(state); err != nil {
        return HandleResult{}, err
    }
    return HandleResult{
        Output:       FormatPreToolOutput(h.agent, bodies),
        MatchedRules: matched,
        SkippedRules: skipped,
    }, nil
}
```

**Depends on**: steps 4-7.
**Traces to**: FR-001, FR-006, FR-007.

### 9. Tests

New:
- `internal/rules/command_matcher_test.go` — unit tests for `SplitSegments`, `StripEnvPrefix`, `MatchCommandPrefix`, `MatchRulesByCommand`. Cover: simple prefix, chained (`cd x && go test`), env prefix (`CGO_ENABLED=0 go test`), non-match (`gopher test`), multi-segment multiple matches.
- `internal/rules/gate_test.go` — inject fake `stat`, test `requires_files` pass/fail, `requires_files_absent` pass/fail, both set, neither set, walk-up behavior.
- `internal/hook/handler_test.go` — add Bash-path tests: single rule fires once, same trigger deduped, different triggers against same rule both fire, Taskfile-gated rule fires when file present, alternate rule fires when absent.

Existing `handler_test.go` updates: field names on `HandleResult` (`MatchedRuleIDs` → `MatchedRules`).

**Depends on**: steps 1-8.
**Traces to**: SC-001..SC-005.

### 10. Documentation (INFRASTRUCTURE.md)

Add a "Bash command rules" subsection under the rules docs. Include the two-rule Taskfile example from the spec discussion (present-body, absent-body). Document the matcher limits (no shell parsing, prefix-only, segment split on &&/||/;/|).

**Depends on**: nothing.
**Traces to**: user-facing correctness; no FR.

## Spike Phase

**No spikes needed.** All pieces use patterns already present in the codebase:
- Frontmatter normalization → copy `NormalizedPaths` shape.
- Gate evaluation → `os.Stat` + injectable func, standard Go.
- Command tokenization → `strings` only, no libraries.
- Dedup refactor → existing map, composite key.

## Risks & Weakest Links

- **Dedup migration bugs**: one-time rewrite of loaded state. Add a test that loads a legacy state file (bare `ruleID` keys) and confirms the rewrite.
- **Gate walk-up cost**: `os.Stat` per rule per Bash event. With ~5 rules and ~2 required files each, that's ~10 stats per invocation. Acceptable; can cache per-event via the `GateResolver`.
- **Ordering stability**: `MatchRulesByCommand` must iterate rules and commands in declared order so `Trigger` is deterministic. Test this.
- **Empty trigger collisions**: if a file rule and a command rule share a ruleID (they won't — different files — but defensively), the dedup keys wouldn't collide because the trigger suffix differs.

## Open Items Carried from Spec

- Trailing `*` wildcard on `commands:` — not in v1.
- `requires_files` as globs — not in v1.
- Walk-up resolution stops at first `.git` or filesystem root.
