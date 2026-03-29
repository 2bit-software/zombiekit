# Technical Requirements & Preferences

Extracted from Linear ticket DEV-186 and research findings. These are implementation hints, not business requirements.

## From Ticket

- Shell out to `cmux` CLI (no library bindings)
- Injectable executor interface wrapping `exec.Command` for testability
- Session naming convention: `{ticket-id}: {title}`

## Research Adjustments

- **Executor interface**: Codebase pattern is direct `exec.CommandContext()` with private `run()` method. No existing packages use executor injection. Recommend following established pattern unless testing without cmux is a hard requirement.
- **Session naming**: cmux uses UUIDs for identification. Names are cosmetic display labels set via `--name` flag. The manager must store and return the UUID, using the name only for human-readable sidebar labels.
- **Env var injection**: No `--env` flag exists. Environment variables must be serialized into the `--command` string as `export K=V && ...` prefixes.

## Recommended Package Location

`internal/cmux/` — following `internal/worktree/` and `internal/callback/` patterns.

## Recommended File Structure

```
internal/cmux/
  doc.go           — package docs with usage example
  types.go         — SessionManager interface, CmuxManager struct, Option type
  errors.go        — ErrorKind enum, Error type, classifyError, Is* helpers
  manager.go       — New(), SpawnSession(), KillSession(), SessionExists()
  manager_test.go  — tests (integration against real cmux if available, skip otherwise)
```

## Key Technical Decisions Needed

1. **Executor injection vs direct exec**: Ticket requests injection; codebase precedent is direct. Trade-off: testability without cmux vs pattern consistency.
2. **UUID storage**: Manager must track ticket-ID-to-workspace-UUID mapping internally or return UUID for caller to store.
3. **Health check**: Should `New()` validate cmux is running (`cmux ping`), or defer to first operation?
