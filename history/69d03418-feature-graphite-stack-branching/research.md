---
status: complete
updated: 2026-04-03
---

# Research: Graphite Stack Branching

## Executive Summary

Graphite stacking support can be added by extending two layers: the `new.md` workflow command (user-facing question) and the `GitService` in `step/git.go` (branch creation). The safest branch creation approach is `git checkout -b` followed by `gt track`, which avoids the empty-commit problem with `gt create`. The feature should be gated on graphite availability and repo initialization, with graceful fallback.

## Findings

### Codebase Context

**Branch creation architecture** (two layers):
- **Workflow layer** (`embed/commands/new.md`): Handles user interaction — branch check, question presentation. Claude reads this markdown and executes it.
- **Service layer** (`internal/step/git.go`): Handles actual `git checkout -b`. Called from `internal/mcp/tools/initiative/tool.go:216` during `initiative create`.

**Key integration points**:
- `step.GitService.EnsureBranch()` — needs graphite-aware variant
- `initiative.Tool.createNewInitiative()` — needs to pass `useGraphite` flag to GitService
- `initiative.Tool.Definition()` — needs new `use_graphite` input parameter
- `new.md` branch check section — needs graphite option in the AskUserQuestion
- `feature.md` step 3 — needs to pass graphite preference when calling initiative create

**Existing patterns to follow**:
- Graceful degradation: `EnsureBranch()` returns nil on failure (line 24-26)
- Best-effort: `_ = gitSvc.EnsureBranch(...)` (tool.go line 216)
- Detection pattern: `isGitAvailable()` uses `exec.LookPath` — same pattern for `gt`

**Configuration**: TOML-based config exists but only supports tool enable/disable. A persistent graphite preference would require extending the config schema. Out of scope for v1 — ask each time instead.

### Domain Knowledge

**Graphite branch creation options**:

| Approach | Command | Pros | Cons |
|----------|---------|------|------|
| `gt create -m "msg"` | Creates branch + commit | Clean graphite tracking from start | Requires something to commit; creates empty/meaningless commit |
| `git checkout -b` + `gt track` | Two-step | Works with empty branches; no phantom commits | Two commands instead of one; `gt track` behavior needs verification |
| `gt create --allow-empty` | If supported | Single command | May not be supported; still creates empty commit |

**Graphite repo initialization**:
- `gt init` creates `.graphite/` directory with repo metadata
- One-time operation, safe to run multiple times (idempotent)
- Can be automated, but changes repo state (adds `.graphite/` directory which should be gitignored)

**Graphite version considerations**:
- Current installed version: 1.8.3
- `gt track` and `gt create` are stable commands present since early versions
- No version gating needed

## Decision Points

- [x] **D1**: Branch creation method — `gt create` vs `git checkout -b` + `gt track`
  - **Recommendation**: `git checkout -b` + `gt track` — avoids empty commit problem, more robust
  - **Alternative**: Verify if `gt create branch-name` (no `-m` flag) works without committing — if yes, prefer single command

- [ ] **D2**: Auto-initialization — Should we auto-run `gt init` if graphite is installed but repo isn't initialized?
  - **Option A**: Auto-init silently — convenient but opinionated
  - **Option B**: Ask user before initializing — safe but adds another question
  - **Option C**: Don't offer graphite if not initialized — simplest, user must init manually
  - **Recommendation**: Option C for v1. Keep scope small. Document that user needs `gt init` first.

- [ ] **D3**: Trunk branch graphite option — Should graphite stacking be offered when on main/master/develop?
  - **Option A**: Yes, always show when graphite available — consistent UX
  - **Option B**: Only on non-trunk branches — stacking is the use case, trunk is default parent anyway
  - **Recommendation**: Option A — graphite tracking has value even from trunk (enables `gt submit`, `gt modify`)

- [ ] **D4**: Where does the graphite question go in `new.md`?
  - **Option A**: Add to existing branch check options — one question, more options
  - **Option B**: Separate question after branch check — two questions, cleaner separation
  - **Recommendation**: Option A — minimize question count. Add "Stack with Graphite" as an option alongside existing choices.

## Recommendations

1. **Use two-step branch creation**: `git checkout -b` + `gt track --parent {current-branch}`. Verify `gt track` supports `--parent` flag. Fall back to `gt track` without parent if not supported.

2. **Gate on both `gt` in PATH and `.graphite/` directory**: Two simple checks, both must pass to show graphite option.

3. **Add `use_graphite` boolean parameter to `initiative create`**: Minimal API change. The workflow markdown sets it based on user's answer to the branch check question.

4. **Extend `GitService` with `EnsureBranchGraphite()`**: Separate method rather than flag on existing method — keeps the code paths clean and independently testable.

5. **Modify `new.md` branch check**: Add "Stack with Graphite" option. When selected, pass `use_graphite: true` to the initiative create call downstream in the workflow.

6. **Modify `feature.md` step 3**: Accept graphite preference from the new.md flow and pass it to initiative create. If the initiative was already created (idempotent case), skip — branch was already created with the right method.

## Sources

- Graphite CLI documentation (loaded via graphite skill)
- `internal/step/git.go` — current branch creation implementation
- `internal/mcp/tools/initiative/tool.go` — initiative create flow
- `embed/commands/new.md` — branch check workflow
- `internal/config/` — configuration system
