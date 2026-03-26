# Implementation Plan: Orchestrator State Schema (DEV-153)

## Overview

5 files in `internal/state/`, no external dependencies to add. Estimated: straightforward implementation following established codebase patterns.

## Dependency Graph

```
errors.go ─────────────────────┐
migrations/001_initial_schema.sql ──┤
migrations.go (depends on SQL file) ┤
store.go (depends on errors, migrations) ┤
store_test.go (depends on store) ──────┘
```

## Steps

### Step 1: Create package structure and sentinel errors

**File**: `internal/state/errors.go`
**Depends on**: nothing
**Covers**: Foundation for B4

Create the `internal/state/` directory and `errors.go` with:
- Package declaration
- `ErrInvalidDBPath` sentinel error

This is the simplest file and establishes the package.

### Step 2: Write the migration SQL

**File**: `internal/state/migrations/001_initial_schema.sql`
**Depends on**: nothing
**Covers**: B1 (schema definition)

Single migration file containing all three `CREATE TABLE IF NOT EXISTS` statements:
- `jobs` — 7 columns (ticket_id PK, worktree_path, cmux_session, pr_number, status, created_at, updated_at)
- `comment_watermarks` — 3 columns (pr_number PK, last_processed_comment_id, updated_at)
- `concurrency_slots` — 3 columns (project_id PK, active_count, slot_limit)

### Step 3: Write the migration runner

**File**: `internal/state/migrations.go`
**Depends on**: Step 2 (SQL file must exist for embed)
**Covers**: B1, B2, B3

Copy the migration runner pattern from `internal/database/migrations.go` (SQLite portion only, ~70 lines):
- `//go:embed migrations/*.sql` directive
- `RunMigrations(ctx, db)` function
- Bootstrap `schema_migrations` table
- Parse filenames, check applied versions, apply in transactions

Differences from the source:
- No PostgreSQL support (SQLite only)
- No `GetMigrationStatus` function (not needed for DEV-153)
- Simpler — single backend, no config switching

### Step 4: Write the store (interface + constructor)

**File**: `internal/state/store.go`
**Depends on**: Steps 1, 3
**Covers**: B1, B2, B4, B5

Contents:
- `StateStore` interface (`Close()`, `Migrate()`)
- `SQLiteStore` struct wrapping `*sql.DB`
- `NewSQLiteStore(ctx, dbPath)` constructor:
  1. Validate path (empty string → `ErrInvalidDBPath`)
  2. `os.MkdirAll` parent directory with `0o755`
  3. `sql.Open("sqlite", dbPath)`
  4. `db.SetMaxOpenConns(1)`
  5. Set pragmas: WAL, busy_timeout=5000, foreign_keys=ON, synchronous=NORMAL
  6. `RunMigrations(ctx, db)`
  7. Return `&SQLiteStore{db: db}`
- `Close()` method
- `Migrate()` method (delegates to `RunMigrations`)

### Step 5: Write tests

**File**: `internal/state/store_test.go`
**Depends on**: Step 4
**Covers**: B1, B2, B3, B4

Test cases (table-driven):

| Test | What it verifies |
|------|-----------------|
| `TestNewSQLiteStore_FirstRun` | B1: Creates file, all 3 tables exist (query `sqlite_master`) |
| `TestNewSQLiteStore_IdempotentRestart` | B2: Open, close, re-open same path — no error, data preserved |
| `TestNewSQLiteStore_EmptyPath` | B4: Empty string returns `ErrInvalidDBPath` |
| `TestNewSQLiteStore_UnwritablePath` | B4: `/root/forbidden/db.sqlite` returns error |
| `TestNewSQLiteStore_PragmasSet` | Verify WAL mode and foreign keys are active via PRAGMA queries |
| `TestRunMigrations_Idempotent` | B2/B3: Running migrations twice doesn't error |

Each test uses `t.TempDir()` for isolation. No `t.Parallel()`.

## Verification

After all steps:
1. `go build ./internal/state/...` compiles
2. `go test ./internal/state/...` passes all tests
3. `go vet ./internal/state/...` clean

## Not in This Plan

- `cmd/orchestrator/main.go` — the binary entrypoint is a separate ticket
- CRUD operations on any table — DEV-154
- Any Linear, GitHub, or cmux integration
