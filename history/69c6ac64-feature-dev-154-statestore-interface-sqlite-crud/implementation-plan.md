# Implementation Plan: DEV-154 StateStore CRUD

## Overview

Extend the existing `internal/state/` package (from DEV-153) with CRUD methods on `StateStore` interface and `SQLiteStore` implementation. No new files needed -- all changes go into existing files.

## File Change Summary

| File | Action | What Changes |
|------|--------|-------------|
| `internal/state/errors.go` | Edit | Add `ErrJobExists`, `ErrJobNotFound` |
| `internal/state/store.go` | Edit | Add `Job` struct, extend `StateStore` interface, implement 7 CRUD methods |
| `internal/state/store_test.go` | Edit | Add ~15 integration tests |

## Implementation Steps

### Step 1: Sentinel Errors (errors.go)

Add two new sentinel errors alongside existing `ErrInvalidDBPath`:

```go
var (
    ErrInvalidDBPath = errors.New("invalid database path")
    ErrJobExists     = errors.New("job already exists")
    ErrJobNotFound   = errors.New("job not found")
)
```

**FR**: FR-003
**Dependencies**: None

---

### Step 2: Job Struct + Interface Extension (store.go)

Add `Job` struct before the interface definition:

```go
type Job struct {
    TicketID     string
    WorktreePath string
    CmuxSession  string
    PRNumber     *int64
    Status       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

Extend `StateStore` interface to include all CRUD methods (retain `Migrate` and `Close`).

**FR**: FR-001, FR-002
**Dependencies**: Step 1 (errors referenced in interface docs)

---

### Step 3: Implement Job CRUD Methods (store.go)

#### 3a: CreateJob

```sql
INSERT INTO jobs (ticket_id, worktree_path, cmux_session, status, created_at, updated_at)
VALUES (?, ?, ?, 'queued', ?, ?)
```

- Catch SQLite UNIQUE constraint violation -> wrap as `ErrJobExists`
- Set `created_at` and `updated_at` to `time.Now()`

**FR**: FR-004, FR-013

#### 3b: GetJob

```sql
SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at
FROM jobs WHERE ticket_id = ?
```

- `sql.ErrNoRows` -> return `nil, nil`
- Scan `pr_number` into `sql.NullInt64`, convert to `*int64`

**FR**: FR-005

#### 3c: SetPR

```sql
UPDATE jobs SET pr_number = ?, updated_at = ? WHERE ticket_id = ?
```

- Check `RowsAffected() == 0` -> return `ErrJobNotFound`

**FR**: FR-006, FR-013

---

### Step 4: Implement Watermark Methods (store.go)

#### 4a: GetCommentWatermark

```sql
SELECT last_processed_comment_id FROM comment_watermarks WHERE pr_number = ?
```

- `sql.ErrNoRows` -> return `0, nil`

**FR**: FR-007

#### 4b: SetCommentWatermark

```sql
INSERT INTO comment_watermarks (pr_number, last_processed_comment_id, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(pr_number) DO UPDATE SET
    last_processed_comment_id = excluded.last_processed_comment_id,
    updated_at = excluded.updated_at
```

**FR**: FR-008, FR-013

---

### Step 5: Implement Slot Methods (store.go)

#### 5a: TryAcquireSlot

In a single transaction:

```sql
-- Step 1: Upsert project row (seed limit on first access)
INSERT INTO concurrency_slots (project_id, active_count, slot_limit)
VALUES (?, 0, ?)
ON CONFLICT(project_id) DO NOTHING;

-- Step 2: Atomic check-and-increment
UPDATE concurrency_slots
SET active_count = active_count + 1
WHERE project_id = ? AND active_count < slot_limit;
```

- `RowsAffected() == 0` -> return `false, nil` (at capacity)
- `RowsAffected() == 1` -> return `true, nil` (acquired)

**FR**: FR-009, FR-011

#### 5b: ReleaseSlot

```sql
UPDATE concurrency_slots
SET active_count = MAX(active_count - 1, 0)
WHERE project_id = ?
```

- `RowsAffected() == 0` -> no-op (non-existent project, not an error)

**FR**: FR-010

---

### Step 6: Integration Tests (store_test.go)

Use existing `setupTestStore(t)` helper. Add tests:

**Job tests:**
1. `TestCreateJob_AndGetJob` - round-trip create + retrieve
2. `TestCreateJob_Duplicate_ReturnsErrJobExists` - duplicate ticket ID
3. `TestGetJob_NonExistent_ReturnsNil` - nil, nil for missing
4. `TestSetPR_UpdatesJob` - PR number set and updated_at advances
5. `TestSetPR_NonExistent_ReturnsErrJobNotFound` - missing job

**Watermark tests:**
6. `TestGetCommentWatermark_Untracked_ReturnsZero` - 0 for new PR
7. `TestSetCommentWatermark_RoundTrip` - set and get
8. `TestSetCommentWatermark_Overwrite` - upsert overwrites (including lower ID)

**Slot tests:**
9. `TestTryAcquireSlot_AutoCreatesProject` - upsert on first call
10. `TestTryAcquireSlot_AtLimit_ReturnsFalse` - capacity check
11. `TestReleaseSlot_Decrements` - active count goes down
12. `TestReleaseSlot_ClampsToZero` - no negative count
13. `TestReleaseSlot_NonExistentProject_NoOp` - no error for missing
14. `TestTryAcquireSlot_Concurrent` - goroutines with WaitGroup

**Cross-cutting tests:**
15. `TestPersistence_AcrossReopen` - close + reopen, data survives (extends existing idempotent restart test)

**Compile-time check:**
16. `var _ StateStore = (*SQLiteStore)(nil)` - interface compliance

**FR**: All FRs covered per FR-to-test mapping in spec

## Technical Notes

- **FR-013 exception**: `concurrency_slots` table has no `updated_at` column. FR-013 applies to `jobs` and `comment_watermarks` only. No schema migration needed -- slot counting doesn't benefit from timestamps.
- **SQLite unique constraint detection**: Use `strings.Contains(err.Error(), "UNIQUE constraint failed")` to detect duplicate key violations. The codebase has no existing pattern for this (no prior UNIQUE constraint handling). String matching is simple and sufficient -- the error message is stable across SQLite versions.
- **Nullable int64 scanning**: Use `sql.NullInt64` to scan `pr_number`, then convert to `*int64` in the Go struct.
- **Transaction safety**: `TryAcquireSlot` is the only method requiring an explicit transaction. All other methods are single-statement and inherit SQLite's implicit transaction safety.

## Dependency Order

```
Step 1 (errors) ──┐
                   ├── Step 3 (job CRUD) ──┐
Step 2 (struct) ───┘                       │
                   ┌── Step 4 (watermark) ──├── Step 6 (tests)
                   └── Step 5 (slots) ──────┘
```

Steps 3, 4, 5 are independent of each other and could be implemented in parallel. Step 6 depends on all prior steps.
