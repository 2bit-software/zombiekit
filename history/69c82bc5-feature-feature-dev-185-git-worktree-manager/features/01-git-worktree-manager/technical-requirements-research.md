# Technical Requirements: Git Worktree Manager

These are implementation hints and technical constraints extracted from the ticket and research. They inform HOW the business spec should be implemented but are not part of the business spec itself.

## Implementation Constraints

1. **Shell out to git CLI** — no libgit2, no go-git. Git CLI is simpler and more debuggable on shared-filesystem setups.
2. **Commands**: `git worktree add`, `git worktree remove`, `git worktree list --porcelain`, `git branch -d/-D`, `git worktree prune`
3. **No remote operations** — push/fetch is the agent's responsibility, not this primitive's.

## Execution Patterns

1. **Use `exec.CommandContext`** — accept `context.Context` for timeout/cancellation on every operation
2. **Separate stdout/stderr** — use `cmd.Stdout`/`cmd.Stderr` buffers, not `CombinedOutput()`, to enable error classification by parsing stderr
3. **Set `GIT_TERMINAL_PROMPT=0`** in command environment to prevent blocking on credential prompts
4. **Parse exit codes** — exit 128 = `fatal:` (git internal), exit 1 = regular error, exit 0 = success
5. **No shell wrapping** — pass args directly to `exec.Command("git", args...)`, never use `sh -c`

## Error Classification Strategy

Define sentinel errors and classify by matching stderr content:
- `already used by worktree` -> ErrBranchCheckedOut
- `already exists` (path) -> ErrPathExists
- `a branch named ... already exists` -> ErrBranchExists
- `is not a working tree` -> ErrNotAWorktree
- `contains modified or untracked` -> ErrWorktreeDirty
- `cannot remove a locked` -> ErrWorktreeLocked
- `cannot delete branch` -> ErrBranchInUseByWorktree

## Testing Strategy

- Use real local git repos via `t.TempDir()` + `git init`
- Skip tests when git unavailable: `exec.LookPath("git")`
- Table-driven tests for sanitization, error classification
- Test actual git state after operations (verify branch exists, worktree listed, etc.)

## Package Placement

- `internal/worktree/` following existing patterns
- Files: `doc.go`, `manager.go`, `types.go`, `errors.go`, `manager_test.go`

## Codebase Conventions to Follow

- Receiver names: underscore prefix (`func (_m *Manager)`)
- Custom error type with `Code`, `Message`, `Hint` fields
- Constructor: `New(opts ...Option)` or `New(repoDir string, opts ...Option)`
- Interface for testability if needed downstream
- `testify/assert` + `testify/require` for test assertions
