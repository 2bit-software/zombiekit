# Technical Spec: DEV-154 StateStore CRUD

## Architecture

Pure extension of the existing `internal/state/` package. No new packages, no new files, no new dependencies. All changes are additive to `errors.go`, `store.go`, and `store_test.go`.

```
internal/state/
├── errors.go              # + ErrJobExists, ErrJobNotFound
├── store.go               # + Job struct, extended interface, 7 CRUD methods
├── store_test.go          # + ~15 integration tests
├── migrations.go          # unchanged
└── migrations/
    └── 001_initial_schema.sql  # unchanged
```

## Interface Design

```go
type StateStore interface {
    // Lifecycle (existing from DEV-153)
    Migrate(ctx context.Context) error
    Close() error

    // Job CRUD
    CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
    GetJob(ctx context.Context, ticketID string) (*Job, error)
    SetPR(ctx context.Context, ticketID string, prNumber int64) error

    // Comment watermarks
    GetCommentWatermark(ctx context.Context, prNumber int64) (int64, error)
    SetCommentWatermark(ctx context.Context, prNumber int64, commentID int64) error

    // Concurrency slots
    TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
    ReleaseSlot(ctx context.Context, projectID string) error
}
```

## SQL Statements

### CreateJob
```sql
INSERT INTO jobs (ticket_id, worktree_path, cmux_session, status, created_at, updated_at)
VALUES (?, ?, ?, 'queued', ?, ?)
```
Error detection: `strings.Contains(err.Error(), "UNIQUE constraint failed")` -> `ErrJobExists`

### GetJob
```sql
SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at
FROM jobs WHERE ticket_id = ?
```
`sql.ErrNoRows` -> `nil, nil`. Scan `pr_number` into `sql.NullInt64`.

### SetPR
```sql
UPDATE jobs SET pr_number = ?, updated_at = ? WHERE ticket_id = ?
```
`result.RowsAffected() == 0` -> `ErrJobNotFound`

### GetCommentWatermark
```sql
SELECT last_processed_comment_id FROM comment_watermarks WHERE pr_number = ?
```
`sql.ErrNoRows` -> `0, nil`

### SetCommentWatermark
```sql
INSERT INTO comment_watermarks (pr_number, last_processed_comment_id, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(pr_number) DO UPDATE SET
    last_processed_comment_id = excluded.last_processed_comment_id,
    updated_at = excluded.updated_at
```

### TryAcquireSlot (in transaction)
```sql
-- Upsert: seed row if not exists
INSERT INTO concurrency_slots (project_id, active_count, slot_limit)
VALUES (?, 0, ?)
ON CONFLICT(project_id) DO NOTHING;

-- Atomic check-and-increment
UPDATE concurrency_slots
SET active_count = active_count + 1
WHERE project_id = ? AND active_count < slot_limit;
```
`result.RowsAffected() == 0` -> `false, nil`
`result.RowsAffected() == 1` -> `true, nil`

### ReleaseSlot
```sql
UPDATE concurrency_slots
SET active_count = MAX(active_count - 1, 0)
WHERE project_id = ?
```
No-op if row doesn't exist (RowsAffected == 0 is not an error).

## Concurrency Model

- SQLite `MaxOpenConns=1` serializes all database access
- WAL mode allows concurrent reads while writes are serialized
- `busy_timeout=5000` handles lock contention with retries
- `TryAcquireSlot` uses an explicit transaction (BeginTx) for the upsert+update pair
- All other methods are single-statement and safe without explicit transactions

## Timestamp Handling

- Uses `time.Now()` directly (consistent with existing `internal/memory/sqlite/storage.go` pattern)
- `created_at` set once on `CreateJob`
- `updated_at` set on `CreateJob`, advanced on `SetPR` and `SetCommentWatermark`
- `concurrency_slots` has no `updated_at` column -- no timestamp management needed

## Nullable Field Handling

- `pr_number` in jobs table is nullable INTEGER
- Scanned via `sql.NullInt64`, converted to `*int64` in Job struct
- `nil` means no PR has been associated yet
