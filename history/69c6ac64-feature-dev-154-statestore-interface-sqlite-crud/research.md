---
status: complete
updated: 2026-03-27
---

# Research: StateStore Interface and SQLite CRUD Operations

## Executive Summary

DEV-153 delivered the SQLite schema (3 tables: jobs, comment_watermarks, concurrency_slots), migration system, and a minimal `StateStore` interface with only `Migrate()` and `Close()`. DEV-154 extends this interface with all CRUD methods and implements them on `SQLiteStore`, which already exposes a `DB()` accessor for direct `*sql.DB` access.

## Findings

### Codebase Context

**Existing state package (`internal/state/`):**
- `errors.go`: Single sentinel error `ErrInvalidDBPath`
- `store.go`: `StateStore` interface (Migrate, Close) + `SQLiteStore` struct with `DB()` accessor
- `store_test.go`: 7 tests covering schema creation, pragmas, idempotency, error paths
- `migrations.go`: Embedded migration runner with `schema_migrations` tracking
- `migrations/001_initial_schema.sql`: Creates jobs, comment_watermarks, concurrency_slots tables

**Schema (from DEV-153):**
- `jobs`: ticket_id (PK), worktree_path, cmux_session, pr_number (nullable), status (default 'queued'), created_at, updated_at
- `comment_watermarks`: pr_number (PK), last_processed_comment_id, updated_at
- `concurrency_slots`: project_id (PK), active_count (default 0), slot_limit (default 1)

**Interface patterns in codebase:**
- `memory.Storage`: Set/Get/Delete/List/Clear/Close -- focused, domain-specific
- `recall.Storage`: Save/Search/Delete with triple returns (id, created, error)
- All methods take `context.Context` first
- Optional results use `mo.Maybe[T]` (project's own generic Option type)
- `sql.ErrNoRows` treated as non-error, returns `Nothing()`

**SQLite patterns:**
- WAL mode + `MaxOpenConns=1` for single-writer safety
- `busy_timeout=5000` for lock contention
- Transactions via `BeginTx` + `defer tx.Rollback()` + `tx.Commit()`
- Embedded migrations with `go:embed`

**Test patterns:**
- `setupTestStore(t)` helper with `t.TempDir()` and `t.Cleanup()`
- `require.NoError` for setup, `assert.Equal` for assertions
- Compile-time interface check: `var _ Interface = (*Impl)(nil)`

### Domain Knowledge

**Concurrency slot management:**
- Must be atomic: check-and-increment in a single transaction to prevent races
- SQLite's single-writer model + WAL mode makes this straightforward -- one transaction at a time
- The `busy_timeout=5000` pragma handles contention by retrying for 5 seconds

**Comment watermark pattern:**
- Standard "high-water mark" for idempotent event processing
- Only the highest processed ID matters -- no need to track individual comments
- Upsert pattern (INSERT ON CONFLICT UPDATE) is the natural fit

## Decision Points

- [x] **D1**: Should `GetJob` return `*Job, error` or `(Job, bool, error)`?
  - Recommendation: Return `*Job, error` -- nil pointer for not-found, consistent with the ticket's "zero/nil value" language. Simpler than Maybe[T] for this use case since Job is a struct (not used in functional chains).

- [x] **D2**: Should `CreateJob` error on duplicate ticket IDs or be idempotent?
  - Recommendation: Error on duplicate. The orchestrator should not silently overwrite a job. Add `ErrJobExists` sentinel error.

- [x] **D3**: Should `SetCommentWatermark` reject lower IDs or always overwrite?
  - Recommendation: Always overwrite. The caller is responsible for only advancing the watermark. Simplifies the store (pure persistence, no business logic per ticket scope).

- [x] **D4**: Should `ReleaseSlot` error when active count is zero?
  - Recommendation: No-op (clamp to zero). Defensive -- a double-release shouldn't crash the system. Log if needed at the orchestrator layer.

- [x] **D5**: Should `TryAcquireSlot` auto-create the project row or require pre-creation?
  - Recommendation: Auto-create via upsert. The orchestrator shouldn't need a separate "register project" step.

## Recommendations

1. **Extend the existing `StateStore` interface** in `store.go` with the new CRUD methods. Don't create a separate interface.
2. **Add sentinel errors** to `errors.go`: `ErrJobExists`, `ErrJobNotFound` (for SetPR on missing job).
3. **Use transactions** for `TryAcquireSlot` (atomic check-and-increment) and `CreateJob` (uniqueness check).
4. **Use `INSERT OR IGNORE` / `ON CONFLICT`** for watermark upserts and slot auto-creation.
5. **Return `*Job` not `Maybe[Job]`** -- the Maybe type is used for value types in this codebase; Job is a reference type.
6. **Add a `Job` struct** to the state package with fields matching the schema.
7. **Test concurrent access** with goroutines + `sync.WaitGroup`, not table-driven subtests (goroutines need the real store, not isolated contexts).

## Sources

- `internal/state/store.go` -- existing StateStore interface and SQLiteStore
- `internal/state/migrations/001_initial_schema.sql` -- schema definition
- `internal/memory/storage.go` -- interface design pattern reference
- `internal/memory/sqlite/storage.go` -- SQLite CRUD implementation pattern reference
- `internal/recall/storage.go` -- alternative interface pattern reference
- DEV-154 Linear ticket -- acceptance criteria and scope
