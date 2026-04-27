# Reuse Audit

## Summary
- Duplicates: 5 (use existing directly — no new code needed)
- Overlaps: 2 (use as templates)
- Related: 0
- No match: 1 (waitFor helper — create new)

## Findings

### DUPLICATE

#### initTestRepo(t)
- **Existing**: `internal/worktree/manager_test.go:14-33`
- **Decision**: Use existing directly. Same package won't work (different package), but since E2E test is in `internal/orchestrator/`, copy the function. It's 20 lines, standalone.
- **Note**: `runGit` helper at same file :36-43 also needed — copy both.

#### mockArchiver
- **Existing**: `internal/orchestrator/router_test.go:160-168`
- **Decision**: Use existing directly. Same package (`orchestrator`), same `_test.go` scope. The E2E test file can reference it without copying.
- **Caveat**: Existing mock records `ticketID` as a string in `calls []string` but discards `eventKind`. The plan needs eventKind assertion. **Extend** the existing mock to also record eventKind, OR define a new one in the E2E test file.

#### mockAuditor
- **Existing**: `internal/orchestrator/router_test.go:170-178`
- **Decision**: Same as mockArchiver — reuse directly, same caveat about eventKind.

#### stubSession
- **Existing**: `internal/orchestrator/watcher_linear_test.go:123-158`
- **Decision**: Use existing directly. Same package, accessible from E2E test file.

#### runGit(t, dir, args)
- **Existing**: `internal/worktree/manager_test.go:36-43`
- **Decision**: Copy to E2E test file (different package).

### OVERLAP

#### e2eFixture (test fixture struct)
- **Existing**: `routerFixture` (router_test.go:23-66), `commentTestFixture` (watcher_comment_test.go:103-169)
- **Similarity**: Both bundle orchestrator dependencies into a fixture struct with a constructor
- **Decision**: Create new. The E2E fixture needs ALL dependencies (Linear, GitHub, worktree, state, sessions, archiver, auditor, dispatcher, events channel) plus cross-phase mutable state. Neither existing fixture covers this scope.
- **Rationale**: Extending either would require adding fields they don't need for their own tests, violating their single responsibility.

#### Mock call assertion patterns
- **Existing**: `getCalls()` pattern on stubs, `Calls []Call` on MockClient
- **Similarity**: Both record calls; MockClient also records arguments
- **Decision**: Use MockClient's `Calls` pattern for Linear/GitHub (already planned). For archiver/auditor, extend the existing mocks or create new ones that record eventKind.

### NONE

#### waitFor poll helper
- **No existing equivalent found**
- **Decision**: Create new. Consider `require.Eventually` from testify (already a dependency) as an alternative to a custom helper.
