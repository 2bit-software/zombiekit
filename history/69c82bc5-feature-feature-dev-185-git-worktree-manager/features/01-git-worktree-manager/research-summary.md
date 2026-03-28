# Research Summary: Git Worktree Manager

## Codebase Patterns

### Package Structure
- `doc.go` for package-level documentation with usage examples
- Files organized by concern: `service.go` (core), `types.go` (types/constants), `error.go` (custom errors)
- Existing git code in `internal/step/git.go` — branch operations, health checks, graceful degradation

### Interface & DI Patterns
- Small, focused interfaces (3-8 methods)
- Constructors accept interfaces, return concrete types
- Receiver names prefixed with underscore: `func (_g *GitService)`
- All IO dependencies abstracted behind interfaces

### Error Handling
- Custom error types per package with `Code`, `Message`, `Hint` fields
- Wrapping: `fmt.Errorf("context: %w", err)` at each layer
- Comparison: `errors.Is()` / `errors.As()`, never `==`

### Testing Patterns
- Table-driven tests with `t.Run()` subtests
- `t.TempDir()` for temporary git repos
- `t.Helper()` on helper functions, `t.Cleanup()` for teardown
- Skip when git unavailable: `exec.LookPath("git")`
- `testify/assert` (non-fatal) and `testify/require` (fatal)

### Existing Git Execution Pattern (`internal/step/git.go`)
```go
cmd := exec.Command("git", "checkout", branchName)
cmd.Dir = g.workDir
output, err := cmd.CombinedOutput()
```
- Check availability with `exec.LookPath("git")`
- Set `cmd.Dir` for working directory
- Include output in error messages

## Git Worktree CLI Reference

### `git worktree add`
- Syntax: `git worktree add [-f] [-b <branch>] <path> [<commit-ish>]`
- `-b <name>`: Create new branch (fails if exists, exit 128)
- `-f`: Allows re-checking-out a branch used by another worktree (does NOT override path-exists)
- Implicit branch creation from path basename if no `-b` — avoid in automation

### `git worktree remove`
- Syntax: `git worktree remove [-f] <worktree>`
- Clean+unlocked: works. Dirty: needs `-f`. Locked: needs `-f -f`.
- Does NOT delete the branch — separate `git branch -d` required

### `git worktree list --porcelain`
- Blocks separated by blank lines
- Each block: `worktree <path>`, `HEAD <sha>`, `branch refs/heads/<name>` or `detached`
- Optional: `locked [<reason>]`, `prunable <reason>`

### `git branch -d/-D` with Worktrees
- Refuses to delete branch with active worktree (exit 1, not 128)
- No force override — must remove worktree first

### `git worktree prune`
- Cleans stale metadata from manually deleted worktree directories
- Useful as defensive startup step, not routine

### Error Messages (for sentinel error classification)
| Scenario | stderr contains | Exit |
|----------|----------------|------|
| Branch checked out elsewhere | `already used by worktree` | 128 |
| Path exists | `already exists` | 128 |
| Branch name taken | `a branch named` + `already exists` | 128 |
| Not a worktree | `is not a working tree` | 128 |
| Dirty worktree | `contains modified or untracked` | 128 |
| Locked worktree | `cannot remove a locked` | 128 |
| Branch has worktree (delete) | `cannot delete branch` | 1 |

## Key Recommendations

1. Always use explicit `-b <branch>` with `worktree add`
2. Use separate stdout/stderr buffers (not `CombinedOutput`) for error classification
3. Set `GIT_TERMINAL_PROMPT=0` to prevent credential prompts
4. Accept `context.Context` on every operation
5. Define sentinel errors for each failure mode, classify by parsing stderr
6. Full teardown sequence: `worktree remove -f` then `branch -d`
7. Call `worktree prune` once during initialization as defensive cleanup
