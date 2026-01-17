# Research: SQLite to PostgreSQL Migration Tool

**Feature**: 013-sqlite-postgres-import
**Date**: 2025-12-22

## Research Topics

### 1. SQLite Exclusive Locking (FR-013)

**Decision**: Use `PRAGMA locking_mode=EXCLUSIVE` for the duration of the import.

**Rationale**:
- SQLite's WAL mode allows concurrent readers but the spec requires exclusive access (FR-013)
- `PRAGMA locking_mode=EXCLUSIVE` prevents other processes from reading or writing
- Lock is acquired on first database access and held until connection closes
- This ensures consistent snapshot during import

**Alternatives considered**:
- `BEGIN EXCLUSIVE TRANSACTION` - only locks during transaction, not sufficient for full import duration
- File-based flock - would require additional coordination, SQLite's built-in locking is simpler

**Implementation**:
```go
// After opening SQLite connection:
_, err := db.ExecContext(ctx, "PRAGMA locking_mode=EXCLUSIVE")
// Then access tables to acquire the lock
_, err = db.QueryContext(ctx, "SELECT 1 FROM memories LIMIT 1")
```

### 2. Import Metadata Storage (FR-002)

**Decision**: Create `import_metadata` table in PostgreSQL target database.

**Rationale**:
- Storing in PostgreSQL keeps import state with the target data
- Allows tracking multiple source SQLite databases (by path hash)
- Persists across tool restarts (SC-006)

**Schema**:
```sql
CREATE TABLE IF NOT EXISTS import_metadata (
    id SERIAL PRIMARY KEY,
    source_path_hash TEXT NOT NULL UNIQUE,  -- SHA256 of absolute SQLite path
    source_path TEXT NOT NULL,               -- Original path for display
    last_import_at TIMESTAMPTZ NOT NULL,     -- When last import completed
    last_imported_updated_at TIMESTAMPTZ,    -- Max updated_at from source at import time
    items_imported INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Alternatives considered**:
- Store in SQLite source - violates FR-004 (read-only source)
- Store in separate file - more complex, another file to track
- Store in-memory only - doesn't persist (violates SC-006)

### 3. Incremental Import Strategy (FR-003)

**Decision**: Use `updated_at` timestamp comparison for incremental detection.

**Rationale**:
- Existing schema has `updated_at` on all memory records
- Compare `updated_at > last_imported_updated_at` for efficient filtering
- Handle new records (created_at > last_import) and updates (updated_at > last_import)

**Query for pending items**:
```sql
SELECT name, version, content, deleted, created_at, updated_at
FROM memories
WHERE updated_at > ?  -- last_imported_updated_at
ORDER BY updated_at ASC, name ASC, version ASC
```

**Alternatives considered**:
- Track each imported item individually - scales poorly, adds complexity
- Use content hashing - expensive for large content, doesn't handle deletions
- Sequence numbers - not in existing schema

### 4. Version Conflict Resolution (FR-012)

**Decision**: Version-based comparison with soft-delete of superseded records.

**Rationale**:
- Spec clarification: "Compare versions; if SQLite has higher version, import it and mark old PostgreSQL version as deleted"
- This preserves history while ensuring latest version is active

**Algorithm**:
1. For each item to import, check if name exists in PostgreSQL
2. If exists with same version, skip (already imported)
3. If exists with lower version, soft-delete all PostgreSQL versions and import new one
4. If exists with higher version in PostgreSQL, skip (PostgreSQL has newer data - edge case)
5. If doesn't exist, insert new record

**Implementation**:
```go
// In transaction:
// 1. Get max version for name in PostgreSQL
// 2. If srcVersion > pgVersion: soft-delete existing, insert new
// 3. If srcVersion == pgVersion: skip (idempotent)
// 4. If srcVersion < pgVersion: skip (target is ahead)
```

### 5. Batch Import Performance (SC-001)

**Decision**: Batch inserts with configurable batch size, default 100 items.

**Rationale**:
- 1000 items in 30 seconds = ~33 items/second minimum
- Batching reduces network round-trips
- PostgreSQL COPY would be faster but harder to handle per-item errors
- Batch size of 100 balances memory usage and performance

**Implementation approach**:
- Use `pgx.Batch` for batched prepared statements
- Process in batches, commit after each batch
- Track progress per batch for resumability

**Alternatives considered**:
- COPY FROM - fastest but doesn't support per-item error handling
- Single-row inserts - too slow
- Parallel workers - adds complexity, not needed for 1000 items

### 6. Progress Reporting (FR-007)

**Decision**: Callback-based progress reporting with item count and percentage.

**Rationale**:
- CLI can display progress bar or simple count
- Same interface supports verbose and quiet modes
- Callback pattern allows flexibility without coupling

**Interface**:
```go
type ProgressFunc func(imported, total int, currentItem string)

type ImportOptions struct {
    DryRun       bool
    OnProgress   ProgressFunc
    BatchSize    int
}
```

### 7. Timezone Handling (FR-011)

**Decision**: Normalize timestamps to UTC during import.

**Rationale**:
- SQLite stores timestamps as TEXT without timezone info (assumed local or UTC)
- PostgreSQL uses TIMESTAMPTZ for proper timezone handling
- Converting to UTC ensures consistent comparison

**Implementation**:
- Parse SQLite timestamps as UTC (Go's time.Parse without location = UTC)
- PostgreSQL TIMESTAMPTZ with NOW() already uses UTC

### 8. CLI Integration Pattern

**Decision**: Add `import` subcommand under existing `db` command.

**Rationale**:
- Follows existing pattern (`brains db migrate`, `brains db status`)
- Related to database operations
- Consistent with project structure

**Command structure**:
```
brains db import [flags]
  --from, -f     SQLite database path (required)
  --to, -t       PostgreSQL connection URL (required, or from env)
  --dry-run      Preview without importing
  --batch-size   Items per batch (default: 100)
  --verbose      Show detailed progress
```

## Dependencies

No new dependencies required. All functionality uses existing:
- `modernc.org/sqlite` - SQLite driver
- `jackc/pgx/v5` - PostgreSQL driver with batch support
- `urfave/cli/v2` - CLI framework

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Long-running import locks SQLite | Document expected duration; add timeout option |
| PostgreSQL connection drops mid-import | Batch commits ensure partial progress saved; re-run continues |
| Timestamp parsing differences | Use consistent UTC parsing; test with edge cases |
| Very large content fields | Already TEXT type; no practical limit difference |
