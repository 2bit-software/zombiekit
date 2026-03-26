# Research Summary: Orchestrator State Schema

## Codebase Findings

### Existing Patterns to Reuse

**SQLite Driver**: `modernc.org/sqlite v1.41.0` already in `go.mod`. Pure Go, no CGO. Used by `internal/database/sqlite.go` and `internal/memory/sqlite/storage.go`.

**Migration System**: Hand-rolled runner in `internal/database/migrations.go`:
- `embed.FS` for compiled-in SQL files
- `schema_migrations` table for version tracking
- Transactional per-migration application
- File naming: `NNN_descriptive_name.sql`
- `CREATE TABLE IF NOT EXISTS` for defense-in-depth

**SQLite Connection Setup** (`internal/database/sqlite.go`):
- WAL mode enabled
- Foreign keys enabled
- Directory auto-creation
- Wraps `*sql.DB`

**Storage Interface Pattern**: Unified interface per subsystem (e.g., `memory.Storage` with Set/Get/Delete/List/Clear/Close). Context as first parameter, error as last return. Implementations wrap concrete DB types.

**Error Handling**: Sentinel errors in dedicated `errors.go` files. Wrapping with `fmt.Errorf("context: %w", err)`. Comparison via `errors.Is()`/`errors.As()`.

**Testing**: `testify/assert` + `testify/require`. Table-driven tests. `t.TempDir()` for SQLite test databases. `t.Cleanup()` for teardown.

### What Doesn't Exist Yet

- No `cmd/orchestrator/` directory
- No `internal/state/` or `internal/orchestrator/` package
- No orchestrator-related tables or migrations

### Project Structure Context

The orchestrator is a **separate binary** from brains (`cmd/orchestrator/main.go`). Per the architecture brief, it needs its own SQLite database file, separate from the brains database. This means the orchestrator needs its own migration runner (same pattern, different `embed.FS`).

## Domain Research Findings

### SQLite Driver Selection

`modernc.org/sqlite` is ~1.6x slower than `mattn/go-sqlite3` on bulk inserts (1M rows), but comparable on queries. At orchestrator scale (dozens of rows), the difference is irrelevant. Pure Go compilation is the winning factor.

### Connection Management (Critical)

For a Go daemon using SQLite:

| Setting | Value | Why |
|---------|-------|-----|
| `SetMaxOpenConns` | `1` | Prevents `SQLITE_BUSY` — goroutines queue for the single connection |
| `journal_mode` | `WAL` | Concurrent reads don't block writes |
| `busy_timeout` | `5000` | Wait 5s for locks instead of failing immediately |
| `foreign_keys` | `ON` | Enforce referential integrity |
| `synchronous` | `NORMAL` | Safe with WAL, faster than FULL |

The existing `NewSQLiteDB` is missing `busy_timeout`, `synchronous=NORMAL`, and `SetMaxOpenConns(1)`. The orchestrator's DB opener should include all of these.

### Migration Strategy

The existing hand-rolled runner is the right approach. Neither `golang-migrate` nor `goose` provides enough value for <10 tables to justify a new dependency. The orchestrator should copy the pattern with its own `embed.FS`.

### Interface Design

For 3 tables used by a single consumer (the orchestrator), a **unified `StateStore` interface** is pragmatic. The eventual shape (DEV-154) will compose from embedded per-entity sub-interfaces:

```go
// Future shape (DEV-154) — NOT the DEV-153 deliverable
type StateStore interface {
    JobStore
    WatermarkStore
    SlotStore
    Close() error
}
```

For DEV-153, the interface only exposes `Close()` and `Migrate()`. See technical-requirements-research.md for the authoritative DEV-153 interface.

### Idempotent Startup

Use both migration versioning AND `CREATE TABLE IF NOT EXISTS`. The migration runner prevents re-execution; `IF NOT EXISTS` is the safety net for partial-failure edge cases.

## Key Decision Points

1. **Separate DB file** — orchestrator uses its own SQLite file, not the brains database
2. **Package location** — `internal/state/` per the architecture brief
3. **Migration ownership** — orchestrator has its own embedded migrations, not shared with brains
4. **Interface shape** — unified `StateStore` with embedded per-entity sub-interfaces
5. **Connection settings** — `MaxOpenConns(1)` + WAL + busy_timeout for daemon use
