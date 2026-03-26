# Technical Requirements: Orchestrator State Schema

## Package Location

`internal/state/` — owns the `StateStore` interface and SQLite implementation. Per the architecture brief: "All orchestrator logic reads/writes through this interface. The SQLite implementation is behind it; nothing else should know SQLite exists."

## Driver

`modernc.org/sqlite` — already in `go.mod`, pure Go, no CGO dependency.

## Schema

### `jobs` table

```sql
CREATE TABLE IF NOT EXISTS jobs (
    ticket_id   TEXT PRIMARY KEY,
    worktree_path TEXT NOT NULL,
    cmux_session  TEXT NOT NULL,
    pr_number     INTEGER,
    status        TEXT NOT NULL DEFAULT 'queued',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Status values: `queued`, `in_progress`, `pr_open`, `completed`, `failed`, `needs_attention`.

### `comment_watermarks` table

```sql
CREATE TABLE IF NOT EXISTS comment_watermarks (
    pr_number                 INTEGER PRIMARY KEY,
    last_processed_comment_id INTEGER NOT NULL,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### `concurrency_slots` table

```sql
CREATE TABLE IF NOT EXISTS concurrency_slots (
    project_id   TEXT PRIMARY KEY,
    active_count INTEGER NOT NULL DEFAULT 0,
    slot_limit   INTEGER NOT NULL DEFAULT 1
);
```

Note: column named `slot_limit` not `limit` (reserved word in SQL).

## Migration Strategy

**Hand-rolled runner** — same pattern as `internal/database/migrations.go`:
- Own `embed.FS` in the `internal/state` package
- Own `schema_migrations` table in the orchestrator database
- Migrations in `internal/state/migrations/` directory
- File naming: `001_initial_schema.sql`

Do NOT reuse the brains migration runner. The orchestrator uses a separate database file.

## Connection Configuration

```go
db.SetMaxOpenConns(1)  // Single writer — prevents SQLITE_BUSY

// Pragmas (set after open):
// PRAGMA journal_mode=WAL
// PRAGMA busy_timeout=5000
// PRAGMA foreign_keys=ON
// PRAGMA synchronous=NORMAL
```

## Interface Shape

Unified `StateStore` interface with optional embedded sub-interfaces:

```go
type StateStore interface {
    // Job operations (DEV-154)
    // Watermark operations (DEV-154)
    // Slot operations (DEV-154)
    Close() error
    Migrate(ctx context.Context) error
}
```

For DEV-153, only `Close()` and `Migrate()` are needed. The CRUD methods are DEV-154's scope.

## Constructor

```go
func NewSQLiteStore(ctx context.Context, dbPath string) (*SQLiteStore, error)
```

- Creates parent directories if needed (`os.MkdirAll` with `0o755`)
- Opens connection with pragmas
- Runs migrations
- Returns ready-to-use store

## DB Path Configuration

Environment variable `ORCHESTRATOR_DB_PATH`. Default: `~/.zombiekit/orchestrator.db`.

Env var resolution happens in `cmd/orchestrator/main.go`, not in the store constructor. The constructor receives the resolved path as a `string` parameter. Use `os.UserHomeDir()` for tilde expansion in the default path.

## Testing Approach

- `t.TempDir()` for isolated test databases
- `testify/require` for assertions
- Test cases: first-run creates tables, restart is idempotent, invalid path fails fast
- No `t.Parallel()` if tests share state

## Error Patterns

Sentinel errors in `internal/state/errors.go`:

```go
var (
    ErrInvalidDBPath = errors.New("invalid database path")
    // CRUD errors go in DEV-154
)
```

Return `ErrInvalidDBPath` when the path is empty or parent directory creation fails. Wrap with context: `fmt.Errorf("open state store: %w", err)`. OS-level errors (permission denied, disk full) propagate as wrapped errors, not as `ErrInvalidDBPath`.

## Files to Create

```
internal/state/
    store.go          — StateStore interface + SQLiteStore struct + constructor
    migrations.go     — embed.FS + migration runner (copy pattern from internal/database)
    errors.go         — sentinel errors
    migrations/
        001_initial_schema.sql  — all three tables
    store_test.go     — initialization tests
```
