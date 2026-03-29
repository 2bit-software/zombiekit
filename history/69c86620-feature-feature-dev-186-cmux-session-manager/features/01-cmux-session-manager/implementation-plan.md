# Implementation Plan: cmux Session Manager

## Overview

Three-phase implementation for `internal/cmux/` package. Each phase compiles and passes tests before the next begins.

## Phase 1: Package Foundation

**Files**: `doc.go`, `types.go`, `errors.go`

### 1.1 Package documentation (`doc.go`)
- Package overview with usage example
- Document cmux version requirement (>= 0.63.0)

### 1.2 Types (`types.go`)
- `SessionManager` interface: `SpawnSession`, `KillSession`, `SessionExists`
- `CmuxManager` struct: cmuxBin string, sessions map, mu sync.RWMutex
- `Option` func type with `WithCommand(cmd string)` option
- `SessionInfo` struct: ref string, ticketID string, name string

### 1.3 Error classification (`errors.go`)
- `ErrorKind` enum: `ErrSessionExists`, `ErrSessionNotFound`, `ErrCmuxUnavailable`, `ErrBinaryNotFound`, `ErrCommandFailed`, `ErrInvalidEnvKey`
- `Error` struct with Kind, Message, Err
- `Is*` helpers for each kind
- `classifyError(stderr string) ErrorKind` parser for cmux error output
- `newError` constructor

**Verification**: `go build ./internal/cmux/`

## Phase 2: Core Operations

**Files**: `manager.go`, `parse.go`

### 2.1 Output parser (`parse.go`)
- `parseNewWorkspace(stdout string) (ref string, err error)` -- extracts ref from `OK workspace:N`
- `parseListWorkspaces(stdout string) []workspaceEntry` -- parses plain text list
- `workspaceEntry` struct: ref, name, selected bool
- `findByNamePrefix(entries []workspaceEntry, prefix string) *workspaceEntry` -- matches `{ticketID}: *`

### 2.2 Constructor (`manager.go`)
- `New(opts ...Option) (*CmuxManager, error)`
  1. `exec.LookPath("cmux")` -- validates binary on PATH
  2. Run `cmux ping` -- validates cmux is running
  3. Initialize sessions map
  4. Apply options (default command: `claude`)

### 2.3 SpawnSession
1. Acquire write lock on sessions map
2. Check internal map for existing ticket ID
3. If not in map: run `cmux list-workspaces`, parse, check for name match `{ticketID}: *`
4. If found in either: return `ErrSessionExists`
5. Build command string: shell-escape env vars, append launch command
6. Run `cmux new-workspace --cwd <path> --command <cmdString>`
7. Parse ref from stdout
8. Run `cmux rename-workspace --workspace <ref> "{ticketID}: {title}"`
9. Store ref in sessions map
10. Return ref

### 2.4 KillSession
1. Acquire write lock
2. Look up ref from sessions map by ticket ID
3. If not found: return `ErrSessionNotFound`
4. Run `cmux close-workspace --workspace <ref>`
5. Remove from sessions map
6. Return nil (or wrapped error on cmux failure)

### 2.5 SessionExists
1. Acquire read lock, check if ticket ID is in map, copy ref if found
2. Release read lock
3. Run `cmux list-workspaces`, parse output
4. Search for name match `{ticketID}: *`
5. If found in cmux: return true, nil
6. If NOT found in cmux but was in map: acquire write lock, remove stale entry, return false, nil
7. If cmux command fails: return false, error

### 2.6 Shell escaping helper
- `buildCommand(env map[string]string, cmd string) (string, error)`
- Validates env keys match `[A-Za-z_][A-Za-z0-9_]*`
- Single-quote wraps values with `'\''` escaping for embedded quotes
- Empty map produces just the command string

### 2.7 Private run helper
- `run(ctx context.Context, args ...string) (stdout string, err error)`
- Uses `exec.CommandContext(ctx, m.cmuxBin, args...)`
- Captures stdout and stderr
- On non-zero exit: classifies error from stderr, returns typed Error

**Verification**: `go build ./internal/cmux/`

## Phase 3: Tests

**Files**: `manager_test.go`, `parse_test.go`

### 3.1 Parse tests (`parse_test.go`)
- `TestParseNewWorkspace` -- happy path, malformed input
- `TestParseListWorkspaces` -- multiple entries, selected marker, empty list, format change detection
- `TestParseListWorkspaces_UnparseableInput` -- non-empty gibberish returns error
- `TestFindByTicketID` -- exact match, no match, multiple matches
- `TestBuildCommand` -- empty env, single var, multiple vars, special characters in values, invalid key

### 3.2 Integration tests (`manager_test.go`)
- Skip if cmux not available: `t.Skip("cmux not available")`
- `TestNew_CmuxNotOnPath` -- validates binary check
- `TestNew_CmuxNotRunning` -- cmux on PATH but not running (ping fails)
- `TestSpawnSession_Success` -- create, verify in list, cleanup
- `TestSpawnSession_Duplicate` -- create, try again, expect ErrSessionExists
- `TestSpawnSession_RenameFailureCleanup` -- verify orphan workspace is closed if rename fails
- `TestKillSession_Success` -- create, kill, verify gone
- `TestKillSession_NotFound` -- kill nonexistent, expect ErrSessionNotFound
- `TestSessionExists_Running` -- create, check exists = true
- `TestSessionExists_NotRunning` -- check exists = false
- `TestSessionExists_StaleTracking` -- create, close manually via cmux, check exists reconciles
- `TestConcurrent_DifferentTickets` -- parallel spawns for different IDs
- `TestConcurrent_SameTicket` -- two goroutines spawn same ticket; exactly one succeeds, other gets ErrSessionExists

All integration tests use `t.Cleanup` to close any workspaces they create.

**Verification**: `go test ./internal/cmux/ -v`

## Dependencies

- No new external dependencies
- Uses only stdlib: `os/exec`, `context`, `strings`, `sync`, `fmt`, `regexp`, `sort`

## Risk Register

| Risk | Mitigation |
|------|-----------|
| `--name` doesn't show in list (spike finding) | Use `rename-workspace` after create |
| Workspace ref reuse after close | Match by name, not ref, in SessionExists |
| 500ms command race | Accept for v1; document as known limitation |
| Plain text parsing fragility | Dedicated parser with tests; fail loudly on unexpected format |
