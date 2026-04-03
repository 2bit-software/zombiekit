# Technical Specification: Graphite Stack Branching

## Architecture

Three layers, each independently testable:

```
┌──────────────────────────────────────────────┐
│  Workflow Layer (markdown)                    │
│  new.md → keyword detection → USE_GRAPHITE   │
│  feature.md → reads USE_GRAPHITE → passes to │
│               initiative create              │
└──────────────┬───────────────────────────────┘
               │ use_graphite: true
┌──────────────▼───────────────────────────────┐
│  MCP Tool Layer (Go)                         │
│  initiative.Tool.createNewInitiative()       │
│  → reads use_graphite param                  │
│  → calls GitService.EnsureBranchGraphite()   │
│  → populates BranchingMethod in response     │
└──────────────┬───────────────────────────────┘
               │
┌──────────────▼───────────────────────────────┐
│  Service Layer (Go)                          │
│  step.GitService.EnsureBranchGraphite()      │
│  → gt create <branch> --no-interactive       │
│  → fallback: git checkout -b                 │
└──────────────────────────────────────────────┘

┌──────────────────────────────────────────────┐
│  Hook Layer (Go) — runs at conversation start│
│  hook.Handler.handleSessionStart()           │
│  → DetectGraphiteStatus(workDir)             │
│  → outputs status in system-reminder         │
└──────────────────────────────────────────────┘
```

## Data Flow

### Conversation Start

```
Claude Code SessionStart event
  → brains hook --event session-start (stdin: JSON with cwd)
    → handler.handleSessionStart()
      → resolves rules (existing)
      → DetectGraphiteStatus(event.CWD)
      → appends status to bodies
    → stdout: <system-reminder>...graphite: available, initialized, stacked...</system-reminder>
```

### New Work Flow

```
User: "/brains.new stack: add rate limiting"
  → new.md loads
    → detects "stack:" keyword → USE_GRAPHITE = true
    → checks startup hook output → graphite available + initialized? yes
    → appends "USE_GRAPHITE: true" to arguments
    → dispatches to feature.md workflow
  → feature.md step 1 (initiative check)
    → calls initiative create with use_graphite: true
      → createNewInitiative() reads use_graphite
      → calls EnsureBranchGraphite("feature", "add-rate-limiting")
        → gt create feat/add-rate-limiting --no-interactive
        → returns ("graphite", nil)
      → response: { branching_method: "graphite", ... }
```

### Implicit Stacking Flow

```
Startup hook output: "graphite: available, initialized, stacked"
User: "/brains.new add rate limiting" (no stack keyword)
  → new.md loads
    → no stacking keyword found
    → checks startup hook → sees "stacked" → USE_GRAPHITE = true
    → appends "USE_GRAPHITE: true" to arguments
    → (rest same as above)
```

## API Changes

### Initiative Tool — Input Schema Addition

```json
{
  "use_graphite": {
    "type": "boolean",
    "description": "Use graphite (gt) for branch creation to enable stacking"
  }
}
```

### Initiative Tool — Response Schema Addition

`CreateResponse` gains two fields:

```go
type CreateResponse struct {
    // ... existing fields ...
    BranchingMethod  string `json:"branching_method,omitempty"`
    BranchingWarning string `json:"branching_warning,omitempty"`
}
```

Values:
- `branching_method`: `"graphite"` | `"git"` | `""` (empty for idempotent/no-branch cases)
- `branching_warning`: populated only when graphite was requested but fell back to git

### Example Response — Graphite Success

```json
{
  "action": "create",
  "initiative_id": "abc123-feature-add-rate-limiting",
  "initiative_path": "/path/to/history/abc123-feature-add-rate-limiting",
  "branch": "abc123-feature-add-rate-limiting",
  "type": "feature",
  "name": "add-rate-limiting",
  "next_step": "feature",
  "already_existed": false,
  "copied_files": ["spec.md", "research.md"],
  "branching_method": "graphite"
}
```

### Example Response — Graphite Fallback

```json
{
  "action": "create",
  "branching_method": "git",
  "branching_warning": "graphite branch creation failed: exit status 1; fell back to git"
}
```

## Method Signatures

### `internal/hook/graphite.go` (new file)

```go
package hook

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// DetectGraphiteStatus checks graphite CLI availability, repo initialization,
// and current branch tracking status. Returns a human-readable status line.
func DetectGraphiteStatus(workDir string) string

// isGraphiteAvailable checks if the gt CLI is in PATH.
func isGraphiteAvailable() bool

// isGraphiteInitialized checks if the repo has a .graphite/ directory.
func isGraphiteInitialized(workDir string) bool

// isGraphiteTracked checks if the current branch is tracked by graphite.
// Returns false if gt info fails (untracked branch or graphite error).
func isGraphiteTracked(workDir string) bool
```

### `internal/step/git.go` (additions)

```go
// EnsureBranchGraphite creates a branch using graphite stacking, with fallback to git.
// Returns:
//   - method: "graphite", "git", or "" (graceful degradation)
//   - warning: non-empty when graphite failed but git succeeded (non-fatal)
//   - err: only for truly fatal errors (both graphite and git failed)
func (g *GitService) EnsureBranchGraphite(initType, name string) (method, warning string, err error)

// isGraphiteAvailable checks if gt CLI is in PATH.
func (g *GitService) isGraphiteAvailable() bool

// createBranchGraphite creates a branch using gt create.
func (g *GitService) createBranchGraphite(branchName string) error
```

### `internal/hook/handler.go` (modifications)

No struct changes needed. Use `event.CWD` directly in `handleSessionStart`:

```go
func (h *Handler) handleSessionStart(event *HookEvent) (string, error) {
    // ... existing rules resolution ...

    // NEW: append graphite status
    graphiteStatus := DetectGraphiteStatus(event.CWD)
    if graphiteStatus != "" {
        bodies = append(bodies, graphiteStatus)
    }

    // ... existing FormatOutput call ...
}
```

### `internal/mcp/tools/initiative/tool.go` (modifications)

```go
// createNewInitiative signature change — needs useGraphite param
func (t *Tool) createNewInitiative(
    dir string,
    initSvc *internalInit.Service,
    initType, name string,
    useGraphite bool,  // NEW
) (string, error)
```

Helper addition:
```go
func getBoolArg(args map[string]any, key string) bool
```

## Startup Hook Output Format

The graphite status line is appended as a body to the existing system-reminder output.
No change to the hook configuration in `settings.json` — the existing `brains hook --event session-start` handler produces the output.

Output example (within `<system-reminder>` tags alongside existing rules):
```
# CLI Contract
...existing rules content...

graphite: available, initialized, stacked
```

## Keyword Detection Patterns

Detected in `new.md` via prompt analysis (Claude reads and interprets these):

**Stacking keywords** (case-insensitive):
- `stack:` — prefix pattern (e.g., "stack: add feature")
- `use graphite` — explicit request
- `gt stack` — graphite command reference
- `graphite stack` — explicit tool + action

**Anti-stacking keywords** (case-insensitive, override everything):
- `no stack`
- `no graphite`
- `git branch`

**Implicit signal** (from startup hook output):
- `stacked` in graphite status line → auto-enable unless anti-keyword present

## Error Handling

| Scenario | Behavior | Response Field |
|----------|----------|----------------|
| `gt create` succeeds | Use graphite branch | `branching_method: "graphite"` |
| `gt create` fails | Fallback to `git checkout -b` | `branching_method: "git"`, `branching_warning: "<error>"` |
| `gt` not in PATH | Fallback to `git checkout -b` | `branching_method: "git"` |
| Not a git repo | Graceful degradation (no branch) | `branching_method: ""` |
| Branch already exists | Switch to it via git | `branching_method: "git"` |
| `use_graphite` not set | Use `git checkout -b` (existing path) | `branching_method: "git"` |
| Idempotent create | No branching occurs | `branching_method: ""` |

## Testing Strategy

### Unit Tests (no external dependencies)

| Test | File | What |
|------|------|------|
| `isGraphiteInitialized` | `hook/graphite_test.go` | Create/don't create `.graphite/` dir |
| `DetectGraphiteStatus` partial | `hook/graphite_test.go` | Test with controlled filesystem state |
| `formatBranchName` | `step/git_test.go` | Existing tests, no changes |
| `getBoolArg` | `initiative/tool_test.go` | Bool extraction from map |
| Response fields | `initiative/tool_test.go` | Verify JSON marshaling of new fields |

### Integration Tests (require git, skip if unavailable)

| Test | File | What |
|------|------|------|
| `EnsureBranchGraphite` fallback | `step/git_test.go` | No graphite → falls back to git |
| `EnsureBranchGraphite` graceful | `step/git_test.go` | Non-git dir → returns empty |
| Initiative create with `use_graphite` | `initiative/tool_test.go` | End-to-end parameter passthrough |

### Manual E2E Tests

1. Start conversation in graphite-initialized repo → verify startup hook output
2. `/brains.new stack: test feature` → verify graphite branch created
3. From graphite-tracked branch, `/brains.new add feature` → verify implicit stacking
4. `/brains.new no stack add feature` → verify regular git branching
