# Tasks: Git Worktree Manager

**Complexity:** Simple (6 files, 0 cross-module deps)
**Total tasks:** 12
**Parallel groups:** 3

## Dependency Graph

```
T001 ─┐
T002 ─┤─> T005 ─> T006 ─┐
T003 ─┤                  ├─> T009 ─> T010 ─> T011 ─> T012
T004 ─┘─> T007 ─> T008 ─┘
```

## Phase 1: Package Foundation (T001-T004, parallelizable)

- [ ] T001 [P] Create `internal/worktree/doc.go` — Package documentation with usage examples for CreateWorktree, DeleteWorktree, CleanBranch. Follow `internal/callback/doc.go` pattern.
  - **File:** `internal/worktree/doc.go`
  - **Spec ref:** Interface Contract
  - **Accept:** Package compiles, `go doc ./internal/worktree` renders correctly

- [ ] T002 [P] Create `internal/worktree/types.go` — Define `Manager` interface (3 methods with `context.Context`), `Option` type, `WithWorktreesRoot` option, `GitManager` struct fields (`repoDir`, `worktreesRoot`, `gitBin`).
  - **File:** `internal/worktree/types.go`
  - **Spec ref:** Interface Contract, Configuration
  - **Accept:** Interface and struct types compile, option function works

- [ ] T003 [P] Create `internal/worktree/errors.go` — Define `ErrorKind` enum (9 kinds), `Error` struct with `Kind`/`Message`/`Err` fields, `Error()`/`Unwrap()` methods, `newError()` constructor, `classifyError(stderr)` function, `Is<Kind>(err)` helpers for all 9 kinds.
  - **File:** `internal/worktree/errors.go`
  - **Spec ref:** Error Design, Error Conditions
  - **Accept:** All error kinds defined, classification covers all stderr patterns including `"not found"` for `ErrBranchNotFound`

- [ ] T004 [P] Create `internal/worktree/sanitize.go` — `sanitizeTitle(title string) string` implementing the 8 ordered rules: lowercase, spaces-to-hyphens, strip non-ASCII/non-alphanum, collapse hyphens, trim hyphens, truncate 40, retrim, fallback "untitled".
  - **File:** `internal/worktree/sanitize.go`
  - **Spec ref:** C1 sanitization rules
  - **Accept:** Function compiles, handles all edge cases per spec

## Phase 2: Core Operations (T005-T008, sequential after Phase 1)

- [ ] T005 Create `internal/worktree/manager.go` — `New(repoDir string, opts ...Option) (*GitManager, error)` constructor: cache git binary path via `exec.LookPath`, apply options, resolve paths to absolute, validate repo with `git rev-parse --git-dir`. Internal `run(ctx, args...) (string, error)` method with separate stdout/stderr, `GIT_TERMINAL_PROMPT=0`, error classification via `classifyError`.
  - **File:** `internal/worktree/manager.go`
  - **Spec ref:** Interface Contract (constructor), Technical Spec (git runner)
  - **Depends on:** T002, T003
  - **Accept:** Constructor creates manager for valid repo, returns `ErrGitUnavailable`/`ErrNotARepository` for invalid inputs

- [ ] T006 Implement `CreateWorktree` on `GitManager` — `os.MkdirAll` for worktrees root, build path as `worktreesRoot/ticketID`, build branch as `ticketID/sanitizeTitle(shortTitle)`, execute `git worktree add -b {branch} {path}`, return absolute path.
  - **File:** `internal/worktree/manager.go`
  - **Spec ref:** C1
  - **Depends on:** T004, T005
  - **Accept:** Creates worktree at correct path with correct branch name

- [ ] T007 Implement `resolveBranch` helper on `GitManager` — Parse `git worktree list --porcelain` output, split on `\n\n`, match `worktree <path>` lines to find the block matching the target path, extract branch from `branch refs/heads/<name>` line. Return `ErrNotAWorktree` if not found.
  - **File:** `internal/worktree/manager.go`
  - **Spec ref:** C2 (branch resolution)
  - **Depends on:** T005
  - **Accept:** Correctly resolves branch name from worktree path

- [ ] T008 Implement `DeleteWorktree` and `CleanBranch` on `GitManager` — `DeleteWorktree`: call `resolveBranch`, then `git worktree remove -f {path}`, then `git branch -D {branch}` (suppress `ErrBranchNotFound`). `CleanBranch`: call `git branch -D {branch}`.
  - **File:** `internal/worktree/manager.go`
  - **Spec ref:** C2, C3
  - **Depends on:** T007
  - **Accept:** DeleteWorktree removes directory and branch; CleanBranch deletes branch only

## Phase 3: Tests (T009-T012, sequential after Phase 2)

- [ ] T009 Create test helpers in `internal/worktree/manager_test.go` — `initTestRepo(t) string` (temp dir, git init, configure user.name/email, create initial commit), `branchExists(t, repoDir, branch) bool`, `worktreeExists(t, repoDir, path) bool`. Skip all tests if git unavailable.
  - **File:** `internal/worktree/manager_test.go`
  - **Spec ref:** Verification
  - **Depends on:** T008
  - **Accept:** Helpers compile and work against real git

- [ ] T010 Add sanitization tests (table-driven) — Test cases: normal input, special chars, unicode stripping, all-special-chars -> "untitled", empty string -> "untitled", long title truncation, leading/trailing hyphens, consecutive hyphens.
  - **File:** `internal/worktree/manager_test.go`
  - **Spec ref:** C1 sanitization rules
  - **Depends on:** T009
  - **Accept:** All 8+ table cases pass

- [ ] T011 Add constructor and CreateWorktree tests — Constructor: valid repo, not-a-repo error, custom worktrees root, default root resolution. CreateWorktree: happy path (verifies path + branch exist), duplicate ticket ID (`ErrPathExists`), branch collision (`ErrBranchExists`), auto-create worktrees root, context cancellation.
  - **File:** `internal/worktree/manager_test.go`
  - **Spec ref:** Interface Contract, C1
  - **Depends on:** T009
  - **Accept:** All test cases pass against real git

- [ ] T012 Add DeleteWorktree and CleanBranch tests — DeleteWorktree: happy path (directory + branch gone), nonexistent path (`ErrNotAWorktree`), dirty worktree (force-removed), locked worktree (`ErrWorktreeLocked`). CleanBranch: happy path, branch with worktree (`ErrBranchInUse`), nonexistent branch (`ErrBranchNotFound`).
  - **File:** `internal/worktree/manager_test.go`
  - **Spec ref:** C2, C3
  - **Depends on:** T009
  - **Accept:** All test cases pass against real git, including locked worktree edge case

## Execution Order

**Critical path:** T002 -> T005 -> T006 -> T009 -> T011

**Recommended execution:**
1. T001 + T002 + T003 + T004 (parallel)
2. T005
3. T006 + T007 (parallel — T006 needs T005+T004, T007 needs T005)
4. T008
5. T009
6. T010 + T011 + T012 (parallel — all depend only on T009)

## Traceability Matrix

| Spec Requirement | Task(s) |
|-----------------|---------|
| C1: CreateWorktree | T006, T011 |
| C2: DeleteWorktree | T007, T008, T012 |
| C3: CleanBranch | T008, T012 |
| Interface Contract | T002 |
| Error Design | T003 |
| Constructor | T005, T011 |
| Sanitization | T004, T010 |
| Verification | T009-T012 |
