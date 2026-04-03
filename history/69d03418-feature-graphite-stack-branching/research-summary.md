# Research Summary: Graphite Stack Branching

## Codebase Findings

### Current Branch Creation Flow

Branch creation during initiative creation follows this path:

1. **Entry**: `initiative create` MCP tool → `handleCreate()` → `createNewInitiative()` (`internal/mcp/tools/initiative/tool.go:178-232`)
2. **Branch creation**: `step.NewGitService(dir).EnsureBranch(initType, name)` (line 216) — **best-effort, errors silently ignored**
3. **Implementation**: `internal/step/git.go` — `EnsureBranch()` checks git availability, formats branch name, creates or switches to branch
4. **Branch naming**: `formatBranchName()` maps `feature→feat/`, `bug→fix/`, `refactor→ref/` + slugified name

Key characteristic: branch creation is **decoupled from the workflow markdown**. The Go code creates the branch; the workflow markdown (feature.md step 3) merely documents what should happen. The actual `git checkout -b` happens inside the MCP tool.

### Existing Branch Check (new.md)

The `new.md` command already has a "Pre-Classification: Branch Check" (lines 37-57) that:
- Detects when user is on a non-main branch
- Warns about stacking
- Offers: switch to main, switch to develop, type a branch, or stay (stack intentionally)

This is the natural integration point for a graphite stacking question.

### Configuration System

- TOML-based: `.brains/config.toml` (local) and `~/.config/brains/config.toml` (global)
- Currently only supports `[tools]` section with `enabled` booleans
- Would need extension to support git/branching preferences like `use_graphite`

### MCP Tool Parameters

The `initiative create` action accepts: `action`, `dir`, `type`, `name`, `description`. No branching-related parameters exist.

## Graphite Findings

### Availability

- **Installed**: `gt` v1.8.3 at `/opt/homebrew/bin/gt`
- **Repo not initialized**: No `.graphite` directory exists in zombiekit
- Initialization required: `gt init` (one-time, creates `.graphite/` config)

### Relevant Commands for Branch Creation

| Command | Purpose | Requires staged changes? |
|---------|---------|--------------------------|
| `gt create branch-name` | Create branch tracked by graphite | Typically yes (creates commit) |
| `gt create -m "msg"` | Create with commit message | Yes |
| `gt create --no-commit` | May not exist — needs verification | N/A |
| `gt track` | Track existing branch in graphite | No |

### Key Concern: Empty Branches

`gt create` typically expects a commit. For initiative creation, there are no changes to commit yet. Two approaches:
1. **`gt create` with `--allow-empty`** — if graphite supports it
2. **`git checkout -b` then `gt track`** — create normally, then register with graphite

The `gt track` approach is more robust since it works regardless of staged changes.

### Graphite Gotchas Relevant to This Feature

- `gt rename` breaks PR links — naming must be right from the start
- Never use `git rebase`, `git push`, `git merge`, `git branch -D` on graphite-tracked branches
- After graphite branching, all subsequent git operations should go through `gt` commands
- `gt submit` (not `git push`) for pushing, `gt modify` (not `git commit --amend`) for amending

## Existing Patterns to Leverage

1. **Graceful degradation**: `EnsureBranch()` already returns nil when git is unavailable — same pattern should apply when graphite is unavailable
2. **Best-effort**: Branch creation failure doesn't block initiative creation (line 216: `_ = gitSvc.EnsureBranch(...)`)
3. **Workflow-driven UX**: User-facing questions happen in workflow markdown, not in Go code — the graphite question should follow this pattern
4. **Config precedence**: Local > global > defaults — graphite preference should respect this
