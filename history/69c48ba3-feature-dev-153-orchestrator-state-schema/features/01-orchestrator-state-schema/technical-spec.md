# Technical Spec: Orchestrator State Schema (DEV-153)

## Package: `internal/state`

Import path: `github.com/zombiekit/brains/internal/state`

---

## errors.go

```go
package state

import "errors"

var (
    ErrInvalidDBPath = errors.New("invalid database path")
)
```

---

## migrations/001_initial_schema.sql

```sql
CREATE TABLE IF NOT EXISTS jobs (
    ticket_id     TEXT PRIMARY KEY,
    worktree_path TEXT NOT NULL,
    cmux_session  TEXT NOT NULL,
    pr_number     INTEGER,
    status        TEXT NOT NULL DEFAULT 'queued',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comment_watermarks (
    pr_number                 INTEGER PRIMARY KEY,
    last_processed_comment_id INTEGER NOT NULL,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS concurrency_slots (
    project_id   TEXT PRIMARY KEY,
    active_count INTEGER NOT NULL DEFAULT 0,
    slot_limit   INTEGER NOT NULL DEFAULT 1
);
```

---

## migrations.go

```go
package state

import (
    "context"
    "database/sql"
    "embed"
    "fmt"
    "sort"
    "strconv"
    "strings"
    "time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, db *sql.DB) error {
    // 1. Bootstrap schema_migrations table
    // 2. ReadDir "migrations"
    // 3. Sort entries, filter *.sql
    // 4. For each: parseMigrationFilename, check if applied, apply in tx, record version
    // Pattern: copy from internal/database/migrations.go RunSQLiteMigrations
}

func parseMigrationFilename(filename string) (int, string) {
    // Same logic as internal/database/migrations.go
}
```

Key differences from `internal/database/migrations.go`:
- No PostgreSQL functions — SQLite only
- No `GetMigrationStatus` — not needed
- No `config.StorageConfig` dependency — takes raw `*sql.DB`
- Reads from `"migrations"` dir (not `"migrations/sqlite"`)

---

## store.go

```go
package state

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

// StateStore defines the interface for orchestrator persistent state.
// DEV-153 scope: initialization and lifecycle only.
// DEV-154 will add CRUD methods.
type StateStore interface {
    Migrate(ctx context.Context) error
    Close() error
}

// SQLiteStore implements StateStore backed by a local SQLite database.
type SQLiteStore struct {
    db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed state store.
// It creates parent directories, opens the connection with
// appropriate pragmas, and runs any pending migrations.
func NewSQLiteStore(ctx context.Context, dbPath string) (*SQLiteStore, error) {
    if dbPath == "" {
        return nil, fmt.Errorf("open state store: %w", ErrInvalidDBPath)
    }

    dir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return nil, fmt.Errorf("create state directory %s: %w", dir, err)
    }

    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open state store: %w", err)
    }

    db.SetMaxOpenConns(1)

    pragmas := []string{
        "PRAGMA journal_mode=WAL",
        "PRAGMA busy_timeout=5000",
        "PRAGMA foreign_keys=ON",
        "PRAGMA synchronous=NORMAL",
    }
    for _, pragma := range pragmas {
        if _, err := db.ExecContext(ctx, pragma); err != nil {
            db.Close()
            return nil, fmt.Errorf("set %s: %w", pragma, err)
        }
    }

    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, fmt.Errorf("state store connection failed: %w", err)
    }

    if err := RunMigrations(ctx, db); err != nil {
        db.Close()
        return nil, fmt.Errorf("run state migrations: %w", err)
    }

    return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB for use by CRUD operations (DEV-154).
func (s *SQLiteStore) DB() *sql.DB {
    return s.db
}

// Migrate runs any pending database migrations.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
    return RunMigrations(ctx, s.db)
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
    if s.db != nil {
        return s.db.Close()
    }
    return nil
}
```

### Design Decisions

1. **`DB()` accessor**: Exposes `*sql.DB` for DEV-154 to build CRUD operations on top. Not part of the `StateStore` interface — it's an implementation detail of `SQLiteStore`.

2. **Pragmas as loop**: Follows the pattern from domain research (River Queue, Jake Gold) rather than the existing `NewSQLiteDB` pattern. Adds `busy_timeout` and `synchronous=NORMAL` that the existing code lacks.

3. **No `config.StorageConfig` dependency**: The store takes a raw `dbPath string`. Config resolution (env var, defaults, tilde expansion) belongs in the binary's main.go, not in the store package. This keeps the store package dependency-free.

4. **Constructor runs migrations**: On every startup. The migration runner is idempotent so this is safe. `Migrate()` is also exposed for explicit re-runs if needed.

---

## store_test.go

```go
package state

import (
    "context"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *SQLiteStore {
    t.Helper()
    dbPath := filepath.Join(t.TempDir(), "test.db")
    store, err := NewSQLiteStore(context.Background(), dbPath)
    require.NoError(t, err)
    t.Cleanup(func() { store.Close() })
    return store
}
```

Test cases verify:
- Tables exist in `sqlite_master` after construction
- Column names and types match expected schema
- Re-opening same DB path succeeds without error
- Empty path returns `ErrInvalidDBPath`
- Unwritable path returns an error (skip on CI if running as root)
- WAL mode is active (`PRAGMA journal_mode` returns `wal`)
- Foreign keys are enabled (`PRAGMA foreign_keys` returns `1`)
- Migration re-runs are idempotent

---

## Traceability Matrix

| Behavior | Files | How Verified |
|----------|-------|-------------|
| B1: First-time startup | store.go, migrations.go, 001_initial_schema.sql | `TestNewSQLiteStore_FirstRun` |
| B2: Idempotent restart | store.go, migrations.go | `TestNewSQLiteStore_IdempotentRestart` |
| B3: Schema migration | migrations.go | `TestRunMigrations_Idempotent` |
| B4: Invalid DB path | store.go, errors.go | `TestNewSQLiteStore_EmptyPath`, `TestNewSQLiteStore_UnwritablePath` |
| B5: DB path config | (cmd/orchestrator/main.go — out of scope) | Verified by constructor accepting `dbPath` parameter |
