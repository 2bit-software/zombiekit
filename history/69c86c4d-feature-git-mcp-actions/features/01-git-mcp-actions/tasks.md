# Tasks: Git MCP Actions

**Complexity:** Medium (11 files, 4 cross-module deps)
**Total tasks:** 9
**Parallel opportunities:** 3 groups

## Dependency Graph

```
T001 (runner) ──┬──► T003 (git read) ──┬──► T007 (server) ──► DONE
                │                       │
T002 (types) ──┬┤                       ├──► T008 (git tests)
               ││                       │
               │└──► T004 (git write) ──┘
               │
T005 (pr types) ──► T006 (gh-pr) ──┬──► T007 (server)
                                   ├──► T009 (pr tests)
```

## Execution Waves

**Wave 1** (parallel, no deps):
- [ ] T001 `internal/git/runner.go` - Git command runner
- [ ] T002 `internal/mcp/tools/git/types.go`, `validation.go` - Types and validation
- [ ] T005 `internal/mcp/tools/ghpr/types.go` - PR types

**Wave 2** (parallel, depends on Wave 1):
- [ ] T003 `internal/mcp/tools/git/tool.go` - Read actions (status, log, diff)
- [ ] T004 `internal/mcp/tools/git/tool.go` - Write actions (stage, commit, push)
- [ ] T006 `internal/mcp/tools/ghpr/tool.go` - PR actions (view, create, comment)

**Wave 3** (parallel, depends on Wave 2):
- [ ] T008 `internal/mcp/tools/git/tool_test.go`, `internal/git/runner_test.go` - Git tests
- [ ] T009 `internal/mcp/tools/ghpr/tool_test.go` - PR tests

**Wave 4** (depends on all above):
- [ ] T007 `internal/config/tools.go`, `internal/mcp/server.go` - Registration and wiring

## Task Details

### T001: Create git command runner package
**Files:** `internal/git/runner.go`
**Acceptance:**
- Runner.Run() executes git commands via exec.CommandContext
- GIT_TERMINAL_PROMPT=0 set in environment
- GitError wraps stderr output
- NewRunner fails gracefully if git not in PATH

### T002: Create git tool types and validation
**Files:** `internal/mcp/tools/git/types.go`, `internal/mcp/tools/git/validation.go`
**Acceptance:**
- All 6 response structs with JSON tags
- ToolError with Code/Message/Hint
- validateFiles rejects flag-like paths, checks file existence
- validateBranch rejects main/master
- validateMessage rejects empty strings

### T003: Implement git tool - read actions
**Files:** `internal/mcp/tools/git/tool.go`
**Acceptance:**
- status action returns branch, status lines, staged flag, tracking info
- log action returns commits with configurable count and base ref
- diff action supports all/staged/unstaged scopes, stat_only mode, path filtering

### T004: Implement git tool - write actions
**Files:** `internal/mcp/tools/git/tool.go` (same file as T003)
**Acceptance:**
- stage validates files then runs git add
- commit writes message to temp file, runs git commit -F, cleans up
- push validates branch protection, supports set_upstream flag

### T005: Create gh-pr tool types
**Files:** `internal/mcp/tools/ghpr/types.go`
**Acceptance:**
- ViewResponse, CreateResponse, CommentResponse structs with JSON tags

### T006: Implement gh-pr tool
**Files:** `internal/mcp/tools/ghpr/tool.go`
**Acceptance:**
- view returns PR details or {exists: false}
- create validates title/body, checks no existing PR, creates via gh CLI
- comment validates pr_number/body, posts via gh CLI
- Graceful error if gh not in PATH

### T007: Register tools in config and server
**Files:** `internal/config/tools.go`, `internal/mcp/server.go`
**Acceptance:**
- "git" and "gh-pr" in KnownTools
- Tools created in NewServer with working directory
- Tools conditionally registered based on config
- Handler methods route to tool.Execute()

### T008: Integration tests for git tool
**Files:** `internal/mcp/tools/git/tool_test.go`, `internal/git/runner_test.go`
**Acceptance:**
- Tests use real git repo in t.TempDir()
- Cover status, log, diff, stage, commit actions
- Cover validation: flag rejection, empty message, branch protection
- Push tested with local bare repo (no network)

### T009: Tests for gh-pr tool
**Files:** `internal/mcp/tools/ghpr/tool_test.go`
**Acceptance:**
- Validation tests (empty title/body, missing pr_number)
- Graceful error when gh binary not found
- No network-dependent tests

## Spec Traceability

| Spec Requirement | Tasks |
|-----------------|-------|
| git-info (status, log, diff-stat) | T001, T002, T003 |
| git-diff (scoped diffs) | T001, T002, T003 |
| git-stage (validated staging) | T001, T002, T004 |
| git-commit (safe commit) | T001, T002, T004 |
| git-push (branch-protected push) | T001, T002, T004 |
| gh-pr (view, create, comment) | T005, T006 |
| Config enable/disable | T007 |
| Input validation | T002, T008, T009 |
