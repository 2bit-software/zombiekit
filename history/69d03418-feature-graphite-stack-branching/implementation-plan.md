# Implementation Plan: Graphite Stack Branching

## Overview

Five implementation phases, ordered by dependency. Each phase is independently testable.

## Phase 1: Graphite Detection (internal/hook/)

**Goal**: Startup hook reports graphite status at conversation start.

**Depends on**: Nothing (foundation for everything else)

### 1.1 Create `internal/hook/graphite.go`

New file with graphite detection functions:

```go
func DetectGraphiteStatus(workDir string) string
func isGraphiteAvailable() bool           // exec.LookPath("gt")
func isGraphiteInitialized(workDir string) bool  // os.Stat(".graphite")
func isGraphiteTracked(workDir string) bool      // exec.Command("gt", "info", "--no-interactive") exit code
```

`DetectGraphiteStatus` returns one of:
- `"graphite: not available"`
- `"graphite: available, not initialized"`
- `"graphite: available, initialized"`
- `"graphite: available, initialized, stacked"`

### 1.2 Modify `internal/hook/handler.go`

In `handleSessionStart()`, after resolving rules:
- Call `DetectGraphiteStatus(event.CWD)` using the CWD from the hook event (already available as a parameter)
- Append the status string to `bodies` slice before `FormatOutput()`

No struct modification needed — `event.CWD` is the working directory, passed from `cli/hook.go:52`.

### 1.3 Create `internal/hook/graphite_test.go`

Tests for:
- `isGraphiteAvailable()` — true when `gt` is in PATH (environment-dependent, skip if not available)
- `isGraphiteInitialized()` — true when `.graphite/` dir exists in temp dir
- `isGraphiteTracked()` — exercise requires real graphite init, so test the false case (no `.graphite/` dir) and mock/skip the true case
- `DetectGraphiteStatus()` — integration test combining above

### Verification

Run `brains hook --event session-start` with a mock event payload, confirm output includes graphite status line.

---

## Phase 2: GitService Graphite Branch Creation (internal/step/)

**Goal**: `GitService` can create branches via graphite with fallback to git.

**Depends on**: Nothing (parallel with Phase 1)

### 2.1 Modify `internal/step/git.go`

Add new methods:

```go
// EnsureBranchGraphite creates a branch using graphite, falling back to git on failure.
// Returns ("graphite", nil) on success, ("git", nil) on fallback success,
// ("git", warning) on fallback with graphite error context.
func (g *GitService) EnsureBranchGraphite(initType, name string) (method string, err error)

func (g *GitService) isGraphiteAvailable() bool
func (g *GitService) createBranchGraphite(branchName string) error
```

`EnsureBranchGraphite` flow:
1. Check `isGitAvailable()` and `isGitRepository()` — if not, return `("", "", nil)` (graceful degradation, same as `EnsureBranch`)
2. Format branch name via existing `formatBranchName()`
3. If branch already exists:
   a. `switchToBranch()` to check it out
   b. If graphite available: try `gt track --parent <current-branch> --force --no-interactive` to add to stack (best-effort, ignore errors)
   c. Return `("git", "", nil)`
4. If graphite not available: create with `createBranch()` (git), return `("git", "", nil)`
5. Try `gt create <branchName> --no-interactive` — if success, return `("graphite", "", nil)`
6. On failure: fall back to `createBranch()` (git), return `("git", graphiteErr.Error(), nil)` — warning returned as second value, not as error

Return signature: `(method string, warning string, err error)` where:
- `method`: `"graphite"`, `"git"`, or `""` (graceful degradation)
- `warning`: populated only on graphite fallback (non-fatal)
- `err`: only for truly fatal errors (format name failure, both git and graphite fail)

### 2.2 Extend `internal/step/git_test.go`

New tests:
- `TestGitService_EnsureBranchGraphite_FallbackWhenNoGraphite` — graphite not in PATH, falls back to git
- `TestGitService_EnsureBranchGraphite_GracefulDegradation` — non-git dir, returns empty
- `TestGitService_IsGraphiteAvailable` — checks PATH lookup

Skip graphite-specific tests when `gt` is not available (same pattern as existing git tests).

### Verification

`go test ./internal/step/ -run TestGitService_EnsureBranchGraphite`

---

## Phase 3: Initiative Tool — `use_graphite` Parameter (internal/mcp/tools/initiative/)

**Goal**: MCP tool accepts `use_graphite` and uses graphite branching when requested.

**Depends on**: Phase 2

### 3.1 Modify `internal/mcp/tools/initiative/types.go`

Add to `CreateResponse`:
```go
BranchingMethod  string `json:"branching_method,omitempty"`
BranchingWarning string `json:"branching_warning,omitempty"`
```

### 3.2 Modify `internal/mcp/tools/initiative/tool.go`

**Tool definition** — add `use_graphite` to `InputSchema.properties`:
```go
"use_graphite": map[string]any{
    "type":        "boolean",
    "description": "Use graphite (gt) for branch creation to enable stacking",
},
```

**`createNewInitiative()`** — change branching logic:
```go
// Replace:
gitSvc := step.NewGitService(dir)
_ = gitSvc.EnsureBranch(initType, name)

// With:
gitSvc := step.NewGitService(dir)
var branchingMethod, branchingWarning string
if useGraphite {
    method, warning, _ := gitSvc.EnsureBranchGraphite(initType, name)
    branchingMethod = method
    branchingWarning = warning
} else {
    _ = gitSvc.EnsureBranch(initType, name)
    branchingMethod = "git"
}
```

Need to pass `args` to `createNewInitiative()` or extract `useGraphite` in `handleCreate()` and pass as param.

**Add `getBoolArg` helper**:
```go
func getBoolArg(args map[string]any, key string) bool {
    if val, ok := args[key]; ok {
        if b, ok := val.(bool); ok {
            return b
        }
    }
    return false
}
```

**Idempotent path** (`handleCreate` existing-initiative case): Set `BranchingMethod: ""` (omitempty handles this).

### 3.3 Update `internal/mcp/tools/initiative/tool_test.go`

Add test for `use_graphite=true` parameter handling and response fields.

### Verification

Call `initiative create` with `use_graphite: true` via MCP, verify response contains `branching_method`.

---

## Phase 4: Workflow Markdown Changes (embed/)

**Goal**: `new.md` detects stacking intent, `feature.md` passes it through.

**Depends on**: Phases 1-3 (needs hook output and tool parameter)

### 4.1 Modify `embed/commands/new.md`

Add a new section **"Graphite Stacking Detection"** between "Pre-Classification: Branch Check" and "Classification Task".

Content:
```markdown
## Graphite Stacking Detection

Check the startup hook output for graphite status and the user input for stacking keywords.

### Detection Logic

1. **Anti-stacking check**: If user input contains "no stack", "no graphite", or "git branch" (case-insensitive), set `USE_GRAPHITE = false` and skip all graphite logic.

2. **Explicit stacking**: If user input starts with "stack:" or contains "use graphite", "gt stack", or "graphite stack" (case-insensitive), set `USE_GRAPHITE = true`.

3. **Implicit stacking**: If the startup hook output contains "graphite: available, initialized, stacked", set `USE_GRAPHITE = true`.

4. **Otherwise**: `USE_GRAPHITE = false`.

### Uninitialized Repo Handling

If `USE_GRAPHITE = true` (from step 2, explicit keyword) but the startup hook reported "graphite: available, not initialized":
- Ask the user via `AskUserQuestion`: "Graphite is installed but this repo isn't initialized. Run `gt init`?"
- If yes: Run `gt init --no-interactive` via Bash, then proceed with `USE_GRAPHITE = true`
- If no: Set `USE_GRAPHITE = false`, proceed normally

### Metadata Append

If `USE_GRAPHITE = true`, append to the user input before dispatching to the workflow:
```
---
USE_GRAPHITE: true
```
```

### 4.2 Modify all workflow files — Step 1 initiative create call

The initiative tool's `handleCreate` already does branch creation via `EnsureBranch()` (tool.go:216). Workflow Step 3 "Create Branch" is a redundant safeguard that becomes a no-op when the initiative tool already created the branch.

**Strategy**: Pass `use_graphite: true` to the `initiative create` call in **Step 1** of each workflow. Step 3 stays as-is (it'll see the branch already exists and skip).

**Files to update** — add `USE_GRAPHITE` metadata parsing to Step 1:
- `embed/workflows/feature.md` — Step 1 "Initiative Check"
- `embed/workflows/feature-light.md` — Step 1 "Initiative Check"
- `embed/workflows/bug.md` — Step 1 "Initiative Check"
- `embed/workflows/refactor.md` — Step 1 "Initiative Check"
- `embed/workflows/unmanaged.md` — Step 1 "Initiative Check"

Each workflow's Step 1 should:
1. Check if arguments contain `USE_GRAPHITE: true` metadata block
2. If present, pass `use_graphite: true` to the `initiative create` MCP tool call

Template text to add to each workflow's Step 1:
```markdown
   - Check if user input contains `USE_GRAPHITE: true` metadata block
   - If present, pass `use_graphite: true` when calling `mcp__zombiekit__initiative` create
```

### Verification

Manual E2E test: start a conversation, run `/brains.new stack: test feature`, verify graphite branch creation.

---

## Phase 5: Tests & Integration Verification

**Goal**: All tests pass, E2E flow verified.

**Depends on**: Phases 1-4

### 5.1 Run existing tests

```bash
task dev -- test
```

Ensure no regressions.

### 5.2 Integration test plan

1. Create a temp git repo
2. Initialize graphite: `gt init`
3. Call `initiative create` with `use_graphite: true`
4. Verify branch exists and is graphite-tracked (`gt info` succeeds)
5. Verify `CreateResponse` has `branching_method: "graphite"`

### 5.3 Fallback test plan

1. Create a temp git repo (no graphite init)
2. Call `EnsureBranchGraphite()` when `gt` is available but repo not initialized
3. Verify fallback to `git checkout -b`
4. Verify returned method is `"git"` with warning

---

## Dependency Graph

```
Phase 1 (hook/graphite detection) ─┐
                                    ├─── Phase 4 (workflow markdown)
Phase 2 (GitService graphite)  ────┤
         │                         │
         └── Phase 3 (initiative   ┘
              tool parameter)

Phase 5 (tests) depends on all above
```

Phases 1 and 2 can be implemented in parallel.
Phase 3 depends on Phase 2.
Phase 4 depends on Phases 1 and 3.
Phase 5 runs last.

## Files Changed Summary

| File | Change Type | Phase |
|------|------------|-------|
| `internal/hook/graphite.go` | **New** | 1 |
| `internal/hook/graphite_test.go` | **New** | 1 |
| `internal/hook/handler.go` | Modify (add graphite detection call using event.CWD) | 1 |
| `internal/step/git.go` | Modify (add EnsureBranchGraphite + helpers) | 2 |
| `internal/step/git_test.go` | Modify (add graphite tests) | 2 |
| `internal/mcp/tools/initiative/types.go` | Modify (add response fields) | 3 |
| `internal/mcp/tools/initiative/tool.go` | Modify (add parameter, getBoolArg, branching logic) | 3 |
| `internal/mcp/tools/initiative/tool_test.go` | Modify (add graphite tests) | 3 |
| `embed/commands/new.md` | Modify (add stacking detection section) | 4 |
| `embed/workflows/feature.md` | Modify (Step 1: pass use_graphite to initiative create) | 4 |
| `embed/workflows/feature-light.md` | Modify (Step 1: pass use_graphite to initiative create) | 4 |
| `embed/workflows/bug.md` | Modify (Step 1: pass use_graphite to initiative create) | 4 |
| `embed/workflows/refactor.md` | Modify (Step 1: pass use_graphite to initiative create) | 4 |
| `embed/workflows/unmanaged.md` | Modify (Step 1: pass use_graphite to initiative create) | 4 |
