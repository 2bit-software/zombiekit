# Reuse Audit: Graphite Stack Branching

Audit of the implementation plan against existing codebase to identify duplicates, overlaps, and reuse opportunities.

---

## Phase 1: Graphite Detection (internal/hook/)

### 1.1 `isGraphiteAvailable()` — exec.LookPath("gt")
**Classification**: RELATED
**Existing code**:
- `internal/step/git.go:40-43` — `isGitAvailable()` uses `exec.LookPath("git")`, identical pattern
- `internal/sandbox/sandbox.go:100-103` — `Available()` uses `exec.LookPath("sbx")`, identical pattern
- `internal/cmux/manager.go:13` — `exec.LookPath("cmux")` in constructor
- `internal/mcp/tools/ghpr/tool.go:27` — `exec.LookPath("gh")` in constructor

**Recommendation**: No reusable function exists (each package has its own inline check). Follow the `sandbox.Available()` one-liner pattern for a standalone boolean function. The `step/git.go:isGitAvailable()` is the closest structural match since it's also a method returning bool.

### 1.2 `isGraphiteInitialized()` — os.Stat(".graphite")
**Classification**: NONE
**Existing code**: No existing code checks for a `.graphite` directory anywhere in the codebase. All `.graphite` references are in the feature's own spec/plan docs.
**Recommendation**: New code needed. Simple `os.Stat(filepath.Join(workDir, ".graphite"))` check.

### 1.3 `isGraphiteTracked()` — gt info exit code
**Classification**: NONE
**Existing code**: No `gt` command execution exists anywhere in the Go codebase (`grep "gt"` in .go files returned zero relevant results).
**Recommendation**: New code needed. Follow the `exec.Command` + `cmd.Dir` + `cmd.Run()` pattern from `step/git.go:46-49` (`isGitRepository`).

### 1.4 `DetectGraphiteStatus()` — composite detection
**Classification**: NONE
**Existing code**: No equivalent composite status detection exists.
**Recommendation**: New code needed. Pure function composing the three checks above.

### 1.5 New file `internal/hook/graphite.go`
**Classification**: NONE
**Existing code**: No graphite-related files exist in `internal/hook/`.
**Recommendation**: New file needed. Straightforward addition.

### 1.6 Modify `handler.go` — add workDir field to Handler
**Classification**: OVERLAP
**Existing code**: `internal/hook/handler.go:16` — `NewHandler(workingDir, homeDir string, agent Agent)` already receives `workingDir` but only passes it to `rules.NewService()`. The `workingDir` is not stored on the Handler struct.

Additionally, `handleSessionStart` receives the `HookEvent` which contains `event.CWD` (see `internal/hook/types.go:9`). This is the actual CWD from the hook invocation.

**Recommendation**: The plan proposes adding `workDir string` to the Handler struct (Option A). However, `event.CWD` is already available in `handleSessionStart` via the event parameter. Use `event.CWD` directly instead of storing a redundant field. This is simpler and avoids the question of which CWD to use (the one from Handler construction vs. the one from the event). The CLI entrypoint at `internal/cli/hook.go:52` passes `event.CWD` as the `workingDir` to `NewHandler`, so they should be the same, but using `event.CWD` directly is cleaner.

### 1.7 Modify `handler.go` — append graphite status to bodies
**Classification**: RELATED
**Existing code**: `internal/hook/handler.go:58-63` — the bodies slice is assembled from unconditional rules, then passed to `FormatOutput()`. The pattern is: build a `[]string`, join in `FormatOutput`. Also, `internal/hook/agent.go:25-37` — `FormatOutput` joins bodies with `\n\n` and wraps in `<system-reminder>` for Claude.
**Recommendation**: Follow the existing pattern. Append the graphite status string to the `bodies` slice before `FormatOutput()` is called. No structural changes needed beyond adding the call.

---

## Phase 2: GitService Graphite Branch Creation (internal/step/)

### 2.1 `EnsureBranchGraphite()` — new method on GitService
**Classification**: RELATED
**Existing code**: `internal/step/git.go:23-37` — `EnsureBranch()` is the exact method being extended. It follows: check availability -> format name -> check exists -> switch or create. The graphite variant follows the same structure with an additional graphite path.
**Recommendation**: New method needed. Reuse `formatBranchName()`, `branchExists()`, `switchToBranch()`, `createBranch()` from the existing `EnsureBranch` flow. The new method adds graphite-specific logic on top.

### 2.2 `isGraphiteAvailable()` on GitService
**Classification**: OVERLAP
**Existing code**: Phase 1 plans `isGraphiteAvailable()` in `internal/hook/graphite.go`. Phase 2 plans the same function on `GitService` in `internal/step/git.go`.
**Recommendation**: This is two copies of the same `exec.LookPath("gt")` check. Both are needed because they serve different contexts (hook detection vs. step execution), and they're in different packages. Since it's a one-liner, duplication is acceptable and preferable to creating a shared utility package for a single boolean check. However, be aware this is intentional duplication.

### 2.3 `createBranchGraphite()` — gt create execution
**Classification**: NONE
**Existing code**: No `gt` command execution exists in the codebase.
**Recommendation**: New code needed. Follow the `createBranch()` pattern at `internal/step/git.go:71-79` (exec.Command + cmd.Dir + CombinedOutput).

---

## Phase 3: Initiative Tool — use_graphite Parameter

### 3.1 `getBoolArg()` helper in initiative package
**Classification**: DUPLICATE
**Existing code**:
- `internal/mcp/tools/git/types.go:94-102` — **identical** `getBoolArg(args map[string]any, key string) bool` implementation
- `internal/mcp/tools/ghpr/types.go:75-83` — **identical** copy

The initiative package at `internal/mcp/tools/initiative/tool.go:475-483` already has `getStringArg` but **does not** have `getBoolArg`.

**Recommendation**: The plan proposes adding `getBoolArg` to the initiative package. This is the correct approach given the existing pattern: each MCP tool package has its own copy of these helpers (git, ghpr, initiative each have their own `getStringArg`; git and ghpr each have their own `getBoolArg`). Add an identical copy to the initiative package. **Note**: There is a latent codebase smell here (3+ copies of the same helpers), but consolidation is out of scope.

### 3.2 `use_graphite` parameter in tool Definition
**Classification**: NONE
**Existing code**: The initiative tool definition at `internal/mcp/tools/initiative/tool.go:41-74` has no `use_graphite` property.
**Recommendation**: New parameter addition to InputSchema.properties. Follow the existing property pattern.

### 3.3 `BranchingMethod` and `BranchingWarning` fields on CreateResponse
**Classification**: NONE
**Existing code**: `internal/mcp/tools/initiative/types.go:16-28` — `CreateResponse` has no branching-related fields beyond `Branch string`.
**Recommendation**: New fields needed. Use `omitempty` JSON tags as planned.

### 3.4 Modify `createNewInitiative()` — branching logic
**Classification**: OVERLAP
**Existing code**: `internal/mcp/tools/initiative/tool.go:215-216`:
```go
gitSvc := step.NewGitService(dir)
_ = gitSvc.EnsureBranch(initType, name)
```
This is the exact code being modified.
**Recommendation**: Extend with conditional logic. The `args` map is not currently passed to `createNewInitiative()` — the function signature is `createNewInitiative(dir string, initSvc *internalInit.Service, initType, name string)`. Either add `useGraphite bool` as a parameter or pass the full `args map[string]any`. Passing `useGraphite bool` is cleaner.

### 3.5 Idempotent path (handleCreate existing-initiative case)
**Classification**: RELATED
**Existing code**: `internal/mcp/tools/initiative/tool.go:150-171` — the idempotent case constructs a `CreateResponse` but does no branching. The plan says to leave `BranchingMethod` empty here, which `omitempty` handles automatically.
**Recommendation**: No change needed for the idempotent path. The new `omitempty` fields will naturally be omitted.

---

## Phase 4: Workflow Markdown Changes

### 4.1 Graphite Stacking Detection in `new.md`
**Classification**: RELATED
**Existing code**: `embed/commands/new.md:96-103` — **AutoMode Detection** is the exact pattern to follow:
```markdown
### AutoMode Detection
Before classification, check if the user input contains the keyword **automode** (case-insensitive).
- If detected: Strip "automode" from the input text and set `AUTOMODE = true` for this session.
- If not detected: Set `AUTOMODE = false`.
```
Also, `embed/commands/new.md:123-158` — **Linear Ticket Detection** follows the metadata append pattern:
```markdown
LINEAR_TICKET: DEV-101
LINEAR_URL: https://linear.app/...
LINEAR_TITLE: Ticket title here
```
**Recommendation**: Follow both patterns. The graphite detection section should be structured like AutoMode Detection (keyword scan, boolean flag). The metadata append should follow the Linear Ticket pattern (append `USE_GRAPHITE: true` to arguments). The plan's proposed location (between "Branch Check" and "Classification Task") is correct.

### 4.2 Modify `feature.md` — branch creation step
**Classification**: RELATED
**Existing code**: `embed/workflows/feature.md:49-55` — Step 3 "Create Branch" currently says:
```markdown
3. **Create Branch**
   - Derive a branch name: `feat/{initiative-slug}/{feature-slug}`
   - Create and check out the branch via `mcp__zombiekit__git`
   - If already on a non-main branch that matches the initiative: Skip, use current branch
```
The initiative create call is in Step 1, not Step 3. Step 1 says "Create one with an auto-generated name" but doesn't mention `use_graphite`. Step 3 does branch creation via `mcp__zombiekit__git`, not via `initiative create`.

**Recommendation**: The plan says to update Step 3 to pass `use_graphite` when calling initiative create. However, the current workflow creates the initiative in Step 1 and creates the branch in Step 3 using `mcp__zombiekit__git` directly (not via initiative create). The initiative tool's `handleCreate` does its own `EnsureBranch` call. There's a disconnect: either the branch creation in feature.md Step 3 is redundant with the initiative tool's branching, or they serve different purposes. The plan needs to reconcile this. The simplest approach: pass `use_graphite: true` in the Step 1 initiative create call, and remove or conditionalize the Step 3 manual branch creation.

### 4.3 Modify `feature-light.md` — branch creation
**Classification**: RELATED
**Existing code**: `embed/workflows/feature-light.md:52-59` — identical pattern to `feature.md` Step 3.
**Recommendation**: Same modification needed as feature.md. Also has the same disconnect between initiative create (Step 1) and explicit branch creation (Step 3).

### 4.4 Modify `bug.md` — branch creation
**Classification**: RELATED
**Existing code**: `embed/workflows/bug.md:49-55` — identical Step 3 "Create Branch" pattern.
**Recommendation**: Same USE_GRAPHITE passthrough needed.

### 4.5 Modify `refactor.md` — branch creation
**Classification**: RELATED
**Existing code**: `embed/workflows/refactor.md:49-55` — identical Step 3 "Create Branch" pattern.
**Recommendation**: Same USE_GRAPHITE passthrough needed.

### 4.6 Modify `unmanaged.md` — branch creation
**Classification**: RELATED
**Existing code**: `embed/workflows/unmanaged.md:43-74` — Step 3 "Create Branch" has a different pattern (infers branch prefix from user input, uses AskUserQuestion for ambiguous types). It creates branches via `mcp__zombiekit__git`, not via initiative create's built-in branching.
**Recommendation**: Same USE_GRAPHITE passthrough applies, but needs more care due to the custom branch type inference logic. The plan doesn't explicitly address unmanaged.md.

### 4.7 Anti-stacking keyword detection
**Classification**: NONE
**Existing code**: No anti-keyword detection pattern exists anywhere. AutoMode detection only checks for presence, not absence/negation.
**Recommendation**: New pattern. Simple and self-contained in `new.md`.

---

## Cross-Cutting Concerns

### Duplicated MCP helper functions
**Classification**: OVERLAP
**Existing code**: Three identical copies of `getStringArg`, `marshalResponse`, `ToolError`:
- `internal/mcp/tools/git/types.go`
- `internal/mcp/tools/ghpr/types.go`
- `internal/mcp/tools/initiative/tool.go`

Two identical copies of `getBoolArg`:
- `internal/mcp/tools/git/types.go:94-102`
- `internal/mcp/tools/ghpr/types.go:75-83`

**Recommendation**: Out of scope for this feature, but worth noting: adding `getBoolArg` to initiative creates a third copy. A shared `internal/mcp/tools/helpers` package would eliminate all duplication but is a separate refactor.

### Handler workDir vs event.CWD
**Classification**: OVERLAP
**Existing code**: `NewHandler` receives `workingDir` (from `event.CWD` at `internal/cli/hook.go:52`). The event object passed to `handleSessionStart` also contains `event.CWD`. These are the same value.
**Recommendation**: Use `event.CWD` directly in `handleSessionStart` instead of adding a `workDir` field to Handler. Simpler, no struct modification needed.

---

## Summary

| Planned Item | Classification | Action |
|---|---|---|
| `isGraphiteAvailable()` (hook) | RELATED | New code, follow existing LookPath pattern |
| `isGraphiteInitialized()` | NONE | New code |
| `isGraphiteTracked()` | NONE | New code |
| `DetectGraphiteStatus()` | NONE | New code |
| `internal/hook/graphite.go` | NONE | New file |
| Handler workDir field | OVERLAP | Skip -- use `event.CWD` directly instead |
| Append status to bodies | RELATED | Follow existing pattern |
| `EnsureBranchGraphite()` | RELATED | New method, reuse existing sub-methods |
| `isGraphiteAvailable()` (step) | RELATED | New code, intentional duplication |
| `createBranchGraphite()` | NONE | New code |
| `getBoolArg()` (initiative) | DUPLICATE | Copy from git/types.go or ghpr/types.go |
| `use_graphite` parameter | NONE | New schema property |
| `BranchingMethod`/`BranchingWarning` fields | NONE | New struct fields |
| `createNewInitiative()` branching logic | OVERLAP | Extend existing code at tool.go:215-216 |
| Graphite detection in new.md | RELATED | Follow AutoMode + Linear Ticket patterns |
| USE_GRAPHITE in feature.md | RELATED | Modify Step 1 initiative create call |
| USE_GRAPHITE in feature-light.md | RELATED | Same as feature.md |
| USE_GRAPHITE in bug.md | RELATED | Same as feature.md |
| USE_GRAPHITE in refactor.md | RELATED | Same as feature.md |
| USE_GRAPHITE in unmanaged.md | RELATED | Same but more complex (custom branch logic) |
| Anti-stacking keywords | NONE | New pattern in new.md |

**Key findings**:
1. No existing graphite/gt code exists anywhere in the codebase -- all graphite code is genuinely new.
2. The `getBoolArg` helper is a straight copy from existing packages (not truly new code).
3. The plan's Option A (add workDir to Handler struct) is unnecessary -- `event.CWD` is already available.
4. The workflow markdown files have a disconnect between initiative-create branching and explicit Step 3 branching that the plan should reconcile.
5. The plan underestimates workflow file changes -- bug.md, refactor.md, and unmanaged.md also need USE_GRAPHITE passthrough, not just feature.md and feature-light.md.
