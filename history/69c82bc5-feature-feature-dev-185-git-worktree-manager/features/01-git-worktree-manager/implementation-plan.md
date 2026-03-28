# Implementation Plan: Git Worktree Manager

## Overview

Three-phase delivery: types/errors first (no git dependency), then core operations (real git), then integration tests. Each phase is independently compilable and testable.

## Phase 1: Package Foundation

**Goal:** Define all types, errors, interfaces, and the constructor — everything except git operations.

### Step 1.1: Package scaffolding
- Create `internal/worktree/` package
- Create `doc.go` with package documentation and usage examples
- Create `types.go` with `Option`, `Manager` interface

### Step 1.2: Error types
- Create `errors.go` with `ErrorKind` enum and `Error` struct
- All error kinds: `ErrPathExists`, `ErrBranchExists`, `ErrNotAWorktree`, `ErrWorktreeLocked`, `ErrBranchInUse`, `ErrBranchNotFound`, `ErrGitUnavailable`, `ErrNotARepository`, `ErrGitCommand`
- Constructor functions: `newError(kind, msg, cause)`
- `Is<Kind>(err) bool` helper functions

### Step 1.3: Constructor and git runner
- Create `manager.go` with `GitManager` struct and `New()` constructor
- Internal `run(ctx, args...) (stdout, error)` method for git execution
  - Separate stdout/stderr capture
  - `GIT_TERMINAL_PROMPT=0` in env
  - Error classification by parsing stderr
- Constructor validates git availability (`exec.LookPath`) and repo validity (`git rev-parse --git-dir`)
- `WithWorktreesRoot` option
- Default worktrees root: `../worktrees` relative to `repoDir`

### Step 1.4: Sanitization
- Create `sanitize.go` with `sanitizeTitle(title string) string`
- Ordered rules as specified in business spec
- Table-driven unit tests for sanitization edge cases

**Deliverable:** Package compiles, constructor works, sanitization tested.

## Phase 2: Core Operations

**Goal:** Implement the three Manager interface methods.

### Step 2.1: CreateWorktree
- Create worktrees root directory if needed (`os.MkdirAll`)
- Build path: `filepath.Join(worktreesRoot, ticketID)`
- Build branch: `ticketID + "/" + sanitizeTitle(shortTitle)`
- Execute: `git worktree add -b {branch} {path}`
- Classify errors from stderr (path exists, branch exists)
- Return absolute path

### Step 2.2: DeleteWorktree
- Resolve branch: `git worktree list --porcelain`, parse to find branch for path
- Remove worktree: `git worktree remove -f {path}`
- Delete branch: `git branch -D {branch}`
- Classify errors (not a worktree, locked)

### Step 2.3: CleanBranch
- Delete branch: `git branch -D {branch}`
- Classify errors (branch in use by worktree, branch not found)

**Deliverable:** All three methods implemented and calling real git.

## Phase 3: Tests

**Goal:** Comprehensive test coverage using real local git repos.

### Step 3.1: Test helpers
- `initTestRepo(t) string` — creates temp dir, runs `git init`, configures user, creates initial commit
- `branchExists(t, repoDir, branch) bool`
- `worktreeExists(t, repoDir, path) bool`

### Step 3.2: Constructor tests
- Valid repo directory
- Invalid directory (not a repo)
- Custom worktrees root
- Default worktrees root resolution

### Step 3.3: CreateWorktree tests
- Happy path: creates worktree and branch
- Duplicate ticket ID: returns `ErrPathExists`
- Branch name collision: returns `ErrBranchExists`
- Sanitization applied correctly to branch name
- Worktrees root auto-created
- Context cancellation

### Step 3.4: DeleteWorktree tests
- Happy path: removes directory and branch
- Nonexistent path: returns `ErrNotAWorktree`
- Dirty worktree: still removed (force)
- Locked worktree: returns `ErrWorktreeLocked`
- Branch correctly resolved and deleted

### Step 3.5: CleanBranch tests
- Happy path: branch deleted
- Branch with active worktree: returns `ErrBranchInUse`
- Nonexistent branch: returns `ErrBranchNotFound`

### Step 3.6: Sanitization tests (table-driven)
- Normal input: `"git worktree manager"` -> `"git-worktree-manager"`
- Special chars: `"hello!! world@@"` -> `"hello-world"`
- Unicode: `"uberfluss"` -> `"berfluss"` (non-ASCII stripped)
- All special: `"!!@@##"` -> `"untitled"`
- Empty string: `""` -> `"untitled"`
- Long title: truncated to 40 chars
- Leading/trailing hyphens trimmed
- Consecutive hyphens collapsed

**Deliverable:** Full test suite passing against real git.

## File Structure

```
internal/worktree/
  doc.go           — package documentation
  types.go         — Manager interface, Option type
  errors.go        — ErrorKind, Error struct, Is* helpers
  manager.go       — GitManager struct, constructor, git runner
  sanitize.go      — sanitizeTitle function
  manager_test.go  — all tests
```

## Dependencies

- `os/exec` — git command execution
- `context` — cancellation/timeout
- `bytes` — stdout/stderr capture
- `strings` — stderr parsing, sanitization
- `path/filepath` — path construction
- `os` — directory creation
- `regexp` — sanitization (optional, could use strings)
- `testing` + `testify` — tests

No external dependencies beyond the standard library and testify.

## Traceability

| Spec Capability | Plan Phase | Plan Step |
|----------------|-----------|-----------|
| C1: CreateWorktree | Phase 2 | Step 2.1 |
| C2: DeleteWorktree | Phase 2 | Step 2.2 |
| C3: CleanBranch | Phase 2 | Step 2.3 |
| Interface Contract | Phase 1 | Step 1.1 |
| Error Design | Phase 1 | Step 1.2 |
| Constructor | Phase 1 | Step 1.3 |
| Sanitization | Phase 1 | Step 1.4 |
| Verification | Phase 3 | All |
