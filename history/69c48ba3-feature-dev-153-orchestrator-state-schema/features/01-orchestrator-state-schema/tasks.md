# Tasks: Orchestrator State Schema (DEV-153)

**Complexity**: Simple (5 files, 1 package, ~250 lines)
**Critical path**: T001 → T003 → T004 → T005

## Dependency Graph

```
T001 (errors.go) ──┐
T002 (SQL file)  ──┤── T004 (store.go) ── T005 (tests)
T003 (migrations)──┘
```

T001 and T002 are parallelizable. T003 depends on T002. T004 depends on T001+T003. T005 depends on T004.

## Tasks

- [ ] T001 [P] [B4] Create `internal/state/errors.go`
  - Package declaration: `package state`
  - Sentinel error: `ErrInvalidDBPath = errors.New("invalid database path")`
  - **Acceptance**: File compiles (`go build ./internal/state/...`)

- [ ] T002 [P] [B1] Create `internal/state/migrations/001_initial_schema.sql`
  - Three `CREATE TABLE IF NOT EXISTS` statements
  - Tables: `jobs` (7 cols), `comment_watermarks` (3 cols), `concurrency_slots` (3 cols)
  - Copy DDL exactly from technical-spec.md Schema section
  - **Acceptance**: Valid SQL syntax (no Go compilation needed)

- [ ] T003 [B1,B2,B3] Create `internal/state/migrations.go`
  - `//go:embed migrations/*.sql` directive
  - `RunMigrations(ctx context.Context, db *sql.DB) error` function
  - `parseMigrationFilename(filename string) (int, string)` helper
  - Bootstrap `schema_migrations` table
  - Apply migrations in transactions, record versions
  - Pattern source: `internal/database/migrations.go` lines 112-186 (SQLite portion only)
  - **Acceptance**: Compiles with T001 and T002 present

- [ ] T004 [B1,B2,B4,B5] Create `internal/state/store.go`
  - `StateStore` interface: `Migrate(ctx) error`, `Close() error`
  - `SQLiteStore` struct wrapping `*sql.DB`
  - `NewSQLiteStore(ctx, dbPath) (*SQLiteStore, error)` constructor:
    - Empty path check → `ErrInvalidDBPath`
    - `os.MkdirAll(dir, 0o755)`
    - `sql.Open("sqlite", dbPath)`
    - `db.SetMaxOpenConns(1)`
    - Pragmas: WAL, busy_timeout=5000, foreign_keys=ON, synchronous=NORMAL
    - `db.PingContext(ctx)`
    - `RunMigrations(ctx, db)`
  - `DB() *sql.DB` accessor
  - `Migrate(ctx) error` method
  - `Close() error` method
  - **Acceptance**: `go build ./internal/state/...` succeeds

- [ ] T005 [B1,B2,B3,B4] Create `internal/state/store_test.go`
  - `setupTestStore(t) *SQLiteStore` helper using `t.TempDir()`
  - Test: `TestNewSQLiteStore_FirstRun` — tables exist in sqlite_master
  - Test: `TestNewSQLiteStore_IdempotentRestart` — open/close/reopen same path
  - Test: `TestNewSQLiteStore_EmptyPath` — returns `ErrInvalidDBPath`
  - Test: `TestNewSQLiteStore_UnwritablePath` — returns error
  - Test: `TestNewSQLiteStore_PragmasSet` — WAL mode + foreign keys active
  - Test: `TestRunMigrations_Idempotent` — double migration run succeeds
  - **Acceptance**: `go test ./internal/state/...` all pass

- [ ] T006 Verify clean build
  - Run `go build ./internal/state/...`
  - Run `go test ./internal/state/...`
  - Run `go vet ./internal/state/...`
  - **Acceptance**: All three pass with zero warnings

## Traceability

| Spec Behavior | Tasks |
|---------------|-------|
| B1: First-time startup | T002, T003, T004, T005 |
| B2: Idempotent restart | T003, T004, T005 |
| B3: Schema migration | T003, T005 |
| B4: Invalid DB path | T001, T004, T005 |
| B5: DB path config | T004 (constructor takes dbPath) |

## Execution Order

**Sequential (recommended for single session):**
T001 → T002 → T003 → T004 → T005 → T006

**Parallel opportunities:**
T001 and T002 can run simultaneously (no shared dependencies).
