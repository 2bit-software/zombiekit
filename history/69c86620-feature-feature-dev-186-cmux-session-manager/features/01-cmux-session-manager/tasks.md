# Tasks: cmux Session Manager

## Complexity

- **Files**: 7 (all new, in `internal/cmux/`)
- **LOC**: ~500
- **Dependencies**: 0 cross-module, stdlib only
- **Classification**: Simple

## Dependency Graph

```
T001 ─┐
T002 ─┤── T004 ─── T006
T003 ─┘     │
            T005 ─── T007
```

- Phase 1 (T001-T003): All parallelizable, no dependencies
- Phase 2 (T004-T005): Depend on Phase 1
- Phase 3 (T006-T007): Depend on Phase 2

## Tasks

### Phase 1: Package Foundation (parallelizable)

- [ ] T001 [P] Create package documentation -- `internal/cmux/doc.go`
  - Package overview explaining cmux workspace lifecycle management
  - Usage example showing New(), SpawnSession(), KillSession(), SessionExists()
  - Document cmux >= 0.63.0 requirement
  - **AC**: File compiles, `go vet ./internal/cmux/` passes

- [ ] T002 [P] Define types and interface -- `internal/cmux/types.go`
  - `SessionManager` interface with 3 methods (SpawnSession, KillSession, SessionExists)
  - `CmuxManager` struct with cmuxBin, command, mu sync.Mutex, sessions map
  - `sessionEntry` struct with ref and name fields
  - `Option` func type, `WithCommand` option
  - **AC**: File compiles, interface is implementable

- [ ] T003 [P] Implement error classification -- `internal/cmux/errors.go`
  - `ErrorKind` enum: ErrSessionExists, ErrSessionNotFound, ErrCmuxUnavailable, ErrBinaryNotFound, ErrCommandFailed, ErrInvalidEnvKey
  - `Error` struct implementing `error` and `Unwrap()`
  - `classifyError(stderr string) ErrorKind` -- matches `not_found`, `connection refused`, `No such file`, `could not connect`
  - `Is*` helper functions for each ErrorKind
  - `newError` constructor
  - **AC**: File compiles, all Is* helpers return correct bool for each kind

### Phase 2: Core Operations

- [ ] T004 Implement output parsers and command builder -- `internal/cmux/parse.go`
  - Depends on: T003 (error types for ErrInvalidEnvKey)
  - `parseNewWorkspace(stdout string) (string, error)` -- extracts ref from `OK workspace:N`
  - `parseListWorkspaces(stdout string) ([]workspaceEntry, error)` -- parses plain text, returns error on non-empty unparseable input
  - `findByTicketID(entries []workspaceEntry, ticketID string) *workspaceEntry`
  - `buildCommand(env map[string]string, cmd string) (string, error)` -- shell-escapes values, validates keys
  - `workspaceEntry` struct (ref, name, selected)
  - **AC**: All parsers handle spike-verified output formats; buildCommand produces correct shell escaping

- [ ] T005 Implement manager lifecycle operations -- `internal/cmux/manager.go`
  - Depends on: T001, T002, T003, T004
  - `New(opts ...Option) (*CmuxManager, error)` -- LookPath + ping validation
  - `run(ctx, args) (string, error)` -- exec.CommandContext, captures stdout/stderr, classifies errors
  - `SpawnSession` -- check map, check live state, build command, new-workspace, rename-workspace, track
  - `KillSession` -- check map, close-workspace, remove tracking
  - `SessionExists` -- list-workspaces, parse, reconcile stale tracking
  - **AC**: `go build ./internal/cmux/` passes

### Phase 3: Tests

- [ ] T006 [P] Write parse and command builder unit tests -- `internal/cmux/parse_test.go`
  - Depends on: T004
  - `TestParseNewWorkspace` -- happy path (`OK workspace:9`), malformed input
  - `TestParseListWorkspaces` -- multiple entries with selected marker, empty list, format change detection (non-empty gibberish returns error)
  - `TestFindByTicketID` -- exact match, no match, prefix matching
  - `TestBuildCommand` -- empty env, single var, multiple vars (sorted output), single-quote escaping, embedded quotes, invalid key rejection
  - **AC**: All tests pass, edge cases covered

- [ ] T007 [P] Write integration tests -- `internal/cmux/manager_test.go`
  - Depends on: T005
  - Skip if cmux not available: `t.Skip("cmux not available")`
  - All tests use `t.Cleanup` to close created workspaces
  - Tests:
    - `TestNew_CmuxNotOnPath` -- binary check
    - `TestNew_CmuxNotRunning` -- ping check
    - `TestSpawnSession_Success` -- create, verify in list, cleanup
    - `TestSpawnSession_Duplicate` -- expect ErrSessionExists
    - `TestSpawnSession_RenameFailureCleanup` -- verify orphan cleanup
    - `TestKillSession_Success` -- create, kill, verify gone
    - `TestKillSession_NotFound` -- expect ErrSessionNotFound
    - `TestSessionExists_Running` -- returns true
    - `TestSessionExists_NotRunning` -- returns false
    - `TestSessionExists_StaleTracking` -- manual close, reconcile
    - `TestConcurrent_DifferentTickets` -- parallel spawns
    - `TestConcurrent_SameTicket` -- exactly one succeeds
  - **AC**: All tests pass with real cmux; skipped gracefully without it

## Spec Traceability

| Spec Requirement | Tasks |
|-----------------|-------|
| SpawnSession creates workspace | T005, T007 |
| SpawnSession duplicate error | T005, T007 |
| KillSession terminates | T005, T007 |
| KillSession not-found error | T005, T007 |
| SessionExists true/false | T005, T007 |
| SessionExists stale reconciliation | T005, T007 |
| Health check at init | T005, T007 |
| Shell escaping | T004, T006 |
| Env key validation | T004, T006 |
| Error classification | T003 |
| Configurable command | T002, T005 |
| rename-workspace after create | T005, T007 |

## Execution Order

1. T001 + T002 + T003 (parallel)
2. T004 (depends on T003)
3. T005 (depends on all above)
4. T006 + T007 (parallel, depend on T004/T005 respectively)

**Total**: 7 tasks, 3 parallel opportunities
