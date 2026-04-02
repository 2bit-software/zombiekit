# Implementation Plan: Session-Aware Rules Injection

## Overview

Add a `brains hook` CLI subcommand that reads hook events from stdin, resolves matching rules, tracks injection state per session via a JSON file, and outputs rules as plain text to stdout. Two new packages: `internal/rules` (rules loading, matching, composition) and `internal/hook` (hook event handling, session state, agent detection).

## Dependencies

New: `github.com/bmatcuk/doublestar/v4` — glob matching with `**` and brace expansion support.

Existing (reused): `github.com/adrg/frontmatter`, `github.com/urfave/cli/v2`, `encoding/json`.

## Implementation Steps

### Step 1: Rules Package — Types and Frontmatter

**Files**: `internal/rules/types.go`, `internal/rules/frontmatter.go`

Define core types and frontmatter parsing. Rules are markdown files with optional `paths` YAML frontmatter.

- `Rule` struct: Name, Source (project/global), Paths ([]string), Body (string), FilePath (string)
- `RuleFrontmatter` struct: Paths field (string or []string, matching Claude Code semantics)
- `ParseRule()` function using `adrg/frontmatter` (same pattern as `profile/frontmatter.go`)
- Handle paths normalization: array or single string, brace expansion passthrough to doublestar (no comma splitting — commas appear in brace expansion)

**Traces to**: FR-009 (frontmatter semantics), FR-005 (glob patterns)

### Step 2: Rules Package — Resolver

**Files**: `internal/rules/resolver.go`

Find and load rules from `.brains/rules/` directories. Follow the same ancestor-walk pattern as `profile/resolver.go`.

- `Resolver` struct: workingDir, homeDir
- `FindRulesDirs()` → ancestor walk from CWD up to git root, collecting `.brains/rules/` at each level (supports monorepos), then `~/.brains/rules/`
- `LoadRules(dirs)` → read all `.md` files from each directory, parse frontmatter, return map[string]*Rule
- Rule identity key: `{source}:{filename}` (e.g., `project:go.md`, `global:go.md`)
- Skip directories that don't exist (silently)

**Traces to**: FR-007 (resolve from project and global)

### Step 3: Rules Package — Matcher

**Files**: `internal/rules/matcher.go`

Match file paths against rule glob patterns using doublestar.

- `MatchRules(rules map[string]*Rule, filePath string) []*Rule` — return all rules whose paths patterns match
- `IsUnconditional(rule *Rule) bool` — true if rule has no paths (or paths resolves to catch-all)
- Use `doublestar.Match()` for each pattern against the file path
- Normalize file paths to forward slashes for cross-platform consistency

**Traces to**: FR-005 (glob matching), FR-001/FR-002 (path-based injection)

### Step 4: Rules Package — Service

**Files**: `internal/rules/service.go`

Top-level service combining resolver + matcher.

- `Service` struct: resolver, loaded rules cache
- `ResolveForFile(filePath string) []*Rule` — load rules, match against file path
- `ResolveUnconditional() []*Rule` — return all rules without paths
- `ResolveForFiles(filePaths []string) []*Rule` — batch version for MultiEdit (deduplicated)
- Rules are loaded from disk on each invocation (no persistent cache — hook binary is short-lived)

**Traces to**: FR-001, FR-002, FR-006, FR-007, FR-013

### Step 5: Hook Package — Types

**Files**: `internal/hook/types.go`

Define hook event structs for JSON deserialization.

- `HookEvent` struct: SessionID, HookEventName, CWD, Source (for SessionStart), ToolName, ToolInput, ToolResponse
- `ToolInput` struct: FilePath, Edits []EditEntry (for MultiEdit)
- `EditEntry` struct: FilePath, OldString, NewString
- `ToolResponse` struct: FilePath (camelCase via json tag)
- `SessionState` struct: SessionID, Agent, StartedAt, CompactionCount, InjectedRules map[string]time.Time

**Traces to**: Interface Contract in spec

### Step 6: Hook Package — Session State

**Files**: `internal/hook/session.go`

Manage the per-session JSON state file at `/tmp/zk-session-{SESSION_ID}.json`.

- `LoadState(sessionID string) (*SessionState, error)` — read and unmarshal, return fresh state if missing/corrupt
- `SaveState(state *SessionState) error` — marshal and write atomically (temp file + rename)
- `DeleteState(sessionID string) error` — remove state file
- `IsRuleInjected(state *SessionState, ruleID string) bool`
- `MarkRuleInjected(state *SessionState, ruleID string)`
- `ResetInjectedRules(state *SessionState)` — clear the map, increment compaction count

**Traces to**: FR-003, FR-004, FR-011, FR-012

### Step 7: Hook Package — Agent Detection

**Files**: `internal/hook/agent.go`

Detect which agent is running and format output accordingly.

- `DetectAgent() Agent` — check `CLAUDE_SESSION_ID` first, then `GEMINI_SESSION_ID`
- `Agent` type: Claude, Gemini (enum/const)
- `FormatOutput(agent Agent, rulesContent string) string` — wrap in `<system-reminder>` for Claude, plain for Gemini
- `SessionID() string` — extract session ID from stdin JSON (authoritative source)

**Traces to**: FR-008

### Step 8: Hook Package — Event Handler

**Files**: `internal/hook/handler.go`

Core logic: receive event, resolve rules, check dedup, output.

- `Handler` struct: rulesService, workingDir
- `Handle(event *HookEvent) (string, error)` — main dispatch
- `handleSessionStart(event)` — reset tracking (all sources), inject unconditional rules, save state
- `handlePostToolUse(event)` — extract file paths, resolve matching rules, filter already-injected, output, save state
- `handleSessionEnd(event)` — delete state file

File path extraction logic per tool:
- Read: `event.ToolInput.FilePath`
- Write/Edit: `event.ToolInput.FilePath` (fallback: `event.ToolResponse.FilePath`)
- MultiEdit: iterate `event.ToolInput.Edits[].FilePath`

Skip rules with empty Body (no content to inject). Use single `<system-reminder>` wrapper for Claude (minimizes token overhead).

**Traces to**: FR-001, FR-002, FR-004, FR-006, FR-013, FR-014

### Step 9: CLI Integration

**Files**: `internal/cli/hook.go`, modify `internal/cli/root.go`

Wire up the `brains hook` CLI command.

- `newHookCommand()` → `*cli.Command` with `--event` flag (required, enum: session-start, post-tool-use, session-end)
- Action: read stdin JSON → parse HookEvent → create Handler → call Handle → print result to stdout
- Exit 0 on success, exit 1 on error (non-blocking)
- Add `newHookCommand()` to Commands slice in `NewApp()` in `root.go`

**Traces to**: FR-014

### Step 10: Tests

**Files**: `internal/rules/*_test.go`, `internal/hook/*_test.go`

- **Unit tests**: frontmatter parsing (various formats), glob matching (doublestar patterns), agent detection, file path extraction per tool type
- **Integration tests**: feed JSON stdin to handler, verify stdout output and state file mutations for each FR
- Key test scenarios from spec:
  - Read event → rules injected
  - Second read same type → no re-injection
  - SessionStart compact → state reset, unconditional rules in output
  - SessionStart resume → state reset
  - MultiEdit with two file types → both rule sets in output
  - Missing state file → fresh state created
  - SessionEnd → state file deleted
  - Project and global both have `go.md` → both injected

**Traces to**: All FRs via test mapping in spec

## Step Dependencies

```
Step 1 (types/frontmatter) ─┬─→ Step 2 (resolver) ─┐
                             └─→ Step 3 (matcher)  ─┴─→ Step 4 (service) ─┐
                                                                           │
Step 5 (hook types) ──→ Step 6 (session state) ─┐                         │
                                                 ├─→ Step 8 (handler) ─→ Step 9 (CLI) ─→ Step 10 (tests)
Step 7 (agent detection) ───────────────────────┘
```

Parallel groups:
- **Group A** (rules): Steps 1 → (2 + 3 parallel) → 4
- **Group B** (hook foundations): Steps 5 → 6, Step 7 (independent)
- Groups A and B are fully independent and can run in parallel
- Step 8 merges both groups

## Non-Goals

- No MCP tool for rules (CLI-only for now)
- No `.claude/rules/` reading (Claude handles its own)
- No custom API agent support
- No Bash tool file path extraction
- No rules caching between hook invocations (binary is short-lived)
- No file locking for session state (accept occasional double-injection)
