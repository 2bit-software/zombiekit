# Research Summary: cmux Session Manager

## Codebase Patterns

### Package Structure

Established pattern from `internal/worktree/` and `internal/callback/`:

```
doc.go           — package overview with usage examples
types.go         — interfaces, structs, Option type
errors.go        — ErrorKind enum, Error type, Is* helpers, classifyError()
manager.go       — New() constructor, core methods
manager_test.go  — tests using testify/assert+require
```

### Command Execution

Direct `exec.CommandContext()` usage via private `run()` method on the manager struct. No executor interface injection. Captures stdout+stderr, classifies errors from stderr content.

### Constructor Pattern

`New(requiredArgs, opts ...Option) (*Struct, error)` — validates prerequisites (binary on PATH via `exec.LookPath`), sets defaults, applies options, post-validates.

### Error Pattern

Custom `Error` struct with `Kind ErrorKind`, `Message string`, `Err error`. Public `Is*()` helpers for each error kind. Private `classifyError(stderr)` maps CLI output to error kinds.

### Test Pattern

- `testify/assert` and `testify/require`
- `t.Helper()` on all helper functions
- `t.TempDir()` for temp directories
- `t.Skip()` when prerequisites missing
- Integration-style tests against real CLIs

## cmux CLI Behavior

### Terminology

cmux uses **"workspace"** not "session". Workspaces are sidebar tabs in a macOS GUI application. The CLI is a thin client communicating via Unix socket (`/tmp/cmux.sock`).

### Key Commands

| Operation | Command | Notes |
|-----------|---------|-------|
| Create | `cmux new-workspace --cwd DIR --name NAME --command CMD --json` | Returns `{"workspace_id": "UUID"}` |
| Close | `cmux close-workspace --workspace UUID` | Requires UUID, not name |
| List | `cmux list-workspaces --json` | Array of workspace objects |
| Ping | `cmux ping` | Health check — is cmux running? |
| Send | `cmux send "text\n" --surface SURFACE_ID` | Send keystrokes to workspace |
| Rename | `cmux rename-workspace "NEW NAME"` | Cosmetic label only |

### Critical Design Facts

1. **UUID-centric**: All programmatic operations use UUIDs, not names. Names are display labels only.
2. **No `--env` flag**: Environment variables must be set via `--command "export K=V && cmd"`.
3. **No atomic exists check**: Must parse `list-workspaces --json` output.
4. **GUI required**: cmux is a macOS app, no headless/daemon mode.
5. **Socket auth**: `CMUX_SOCKET_MODE` must be `automation` or `allowall` for external CLI access.
6. **`--command` races**: The `--command` flag waits 500ms for shell init before sending text.

### Auto-set Environment Variables

cmux sets these in every spawned terminal:
- `CMUX_WORKSPACE_ID` — workspace UUID
- `CMUX_SURFACE_ID` — surface UUID
- `CMUX_SOCKET_PATH` — socket path

### Error Behavior

Not well documented. Expected patterns:
- Socket not found → connection error
- Workspace UUID not found → error response
- cmux not running → connection refused

## Divergences from Linear Ticket

| Ticket Assumption | Reality |
|-------------------|---------|
| Session naming `{ticket-id}: {title}` as unique identifier | Names are cosmetic; UUID is the identifier |
| Injectable executor interface | Codebase uses direct exec.Command, no injection |
| "cmux session" terminology | cmux uses "workspace" |
| Env vars as map parameter | Must be embedded in --command string |
| Session exists by name check | Must list all workspaces and filter by UUID |

## Existing Codebase References

- `internal/state/store.go` has `cmux_session TEXT` column in jobs table
- `CreateJob` accepts `cmuxSession string` parameter
- No existing cmux package implementation
