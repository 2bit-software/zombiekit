# Business Specification: cmux Session Manager

## Purpose

Provide a Go package that manages cmux workspace lifecycles for agent sessions. The orchestrator uses this to spawn isolated Claude Code instances in worktrees, check their status, and terminate them.

## Interface Contract

```go
type SessionManager interface {
    SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string) (workspaceRef string, err error)
    KillSession(ctx context.Context, ticketID string) error
    SessionExists(ctx context.Context, ticketID string) (bool, error)
}
```

## Capabilities

### 1. Spawn a Session

**Given** a ticket ID, human-readable title, worktree path, and a map of environment variables,
**When** `SpawnSession` is called,
**Then** a new cmux workspace is created with:
- Working directory set to the worktree path
- Display name set to `{ticket-id}: {title}` for human identification
- All provided environment variables exported before the main command runs
- The command specified at construction time (default: `claude`) started in the workspace
- The workspace ref (e.g., `workspace:9`) returned to the caller for future reference

Before creating, `SpawnSession` checks both internal tracking AND live cmux state (via `list-workspaces`). If a workspace for this ticket ID already exists in either, it returns a "session exists" error. This prevents duplicate workspaces after manager restarts.

**Error conditions:**
- A workspace for this ticket ID already exists (tracked or live) -> return a "session exists" error
- cmux is not running or unreachable -> return a "cmux unavailable" error
- cmux binary not found on PATH -> return a "binary not found" error
- Workspace creation fails (cmux returns error) -> return a "command failed" error with details

### 2. Kill a Session

**Given** a ticket ID with a tracked workspace,
**When** `KillSession` is called,
**Then** the associated cmux workspace is closed and the tracking entry is removed.

**Error conditions:**
- No tracked workspace for this ticket ID -> return a "session not found" error (not a silent no-op)
- cmux close command fails -> return a "command failed" error with details

### 3. Check Session Existence

**Given** a ticket ID,
**When** `SessionExists` is called,
**Then** it returns whether a workspace is currently running for that ticket ID.

This checks actual cmux state (live workspace list), not just internal tracking. If the internal tracker says a session exists but cmux reports it doesn't (e.g., user manually closed it), the result is `false` and internal tracking is cleaned up.

**Error conditions:**
- cmux is not running or unreachable -> return error (do NOT return `false, nil`)
- `list-workspaces` returns unparseable output -> return error
- cmux is reachable and workspace genuinely not listed -> return `false, nil`

### 4. Health Check at Initialization

**When** the session manager is constructed,
**Then** it validates:
- The cmux binary exists on PATH
- cmux is running and reachable (socket responds to ping)

If either check fails, construction returns an error with a clear message.

## Command Construction

### Launch Command

The command to run inside the workspace is configurable at construction time via an option. Default: `claude`. The caller can override this (e.g., `claude --resume`, `claude --dangerously-skip-permissions`).

### Environment Variable Serialization

The manager serializes the env map into a shell command prefix: `export K1='V1' K2='V2' && <command>`.

**Shell escaping**: The manager is responsible for escaping all env var values using single-quote wrapping with internal single-quote escaping (`'val'\''ue'` pattern). This is not the caller's responsibility. Keys are validated to contain only `[A-Za-z_][A-Za-z0-9_]*` characters; invalid keys return an error.

An empty or nil env map is valid and produces no `export` prefix -- just the launch command directly.

### The 500ms Race

cmux's `--command` flag waits 500ms for shell initialization before sending the text. This is a known limitation of cmux's design. The manager uses `--command` and accepts this delay as sufficient. If this proves unreliable in practice, the mitigation path is switching to a two-step approach: `new-workspace` followed by `cmux send` with a configurable delay. This mitigation is out of scope for the initial implementation.

## Internal Tracking

The manager maintains an in-memory `map[ticketID]sessionEntry` (where sessionEntry holds the workspace ref and display name) protected by a `sync.Mutex`.

- **Locking strategy**: `SpawnSession` holds the mutex for its entire duration (map check + cmux CLI calls + map write) to ensure atomicity of the check-create-rename sequence. This means concurrent spawns for different tickets block each other, which is acceptable because spawns are infrequent (~seconds apart). `KillSession` also holds the lock for its duration. `SessionExists` only acquires the lock briefly for map cleanup.
- **Same-ticket races**: Two concurrent `SpawnSession` calls for the same ticket ID serialize at the mutex. The first succeeds; the second gets "session exists".
- **Post-restart state**: The map is empty on construction. `SpawnSession` guards against duplicates by checking live cmux state (workspace display name match), not just the map. `SessionExists` also checks live state. The caller (orchestrator) is responsible for reconciling its persistent state with the manager after restart.

## cmux CLI Integration Details (Verified via Spike, v0.63.0)

### Workspace Identification

cmux identifies workspaces by refs (`workspace:N`). Display names are set via `rename-workspace` after creation (the `--name` flag on `new-workspace` exists but doesn't reliably appear in `list-workspaces`). The manager matches workspaces by display name prefix `{ticket-id}: ` when checking live state.

### CLI Output Formats

- `new-workspace` returns: `OK workspace:N` (plain text)
- `close-workspace` returns: `OK workspace:N` (success) or `Error: not_found: Workspace not found` (exit 1)
- `list-workspaces` returns: plain text, one line per workspace, format `[*] workspace:N  name  [selected]`
- `ping` returns: `PONG`
- No `--json` flag available on any workspace command

### Error Classification

cmux CLI errors follow the format `Error: <category>: <message>` on stderr with exit code 1. Known categories:
- `not_found` -- workspace doesn't exist
- Connection errors (socket not found, refused) -- cmux not running

The classifier starts with these known patterns and falls back to generic `ErrCommandFailed`.

## Non-Functional Requirements

### Naming Convention

Workspace display names follow `{ticket-id}: {title}` format. This is cosmetic -- the workspace ref is the operational identifier. Names are set via `rename-workspace` after creation.

### Concurrency

Multiple goroutines may call the manager concurrently. All operations must be safe for concurrent use. See "Internal Tracking" section for locking details.

### Context Support

All operations accept `context.Context` for cancellation and timeout propagation.

## Out of Scope

- Reading output from sessions -- cmux handles visibility natively
- Deciding when to spawn or kill sessions -- orchestrator's responsibility
- Knowledge of what runs inside the session beyond launching the configured command
- Reconnecting to sessions after manager restart (caller must re-check via `SessionExists`)
- Managing cmux configuration (socket mode, auth, etc.)
- Orphan workspace cleanup after crashes (orchestrator's responsibility -- it can use `SessionExists` with known ticket IDs from its state store)

## Dependencies

- cmux macOS application must be running with socket access enabled
- cmux CLI binary must be on PATH
- Socket mode must be `automation` or `allowall` for external CLI access

## Integration Points

- **Caller**: Orchestrator core passes ticket ID, title, worktree path, and env vars
- **State store**: Caller stores the returned workspace ref in the jobs table (`cmux_session` column)
- **Callback server**: `WORK_CALLBACK_URL` is passed as one of the env vars -- this package just sets it, doesn't construct it
