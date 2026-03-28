# Feature Specification: StateStore Interface and SQLite CRUD Operations

**Feature Branch**: `morganhein/dev-154-implement-statestore-interface-and-sqlite-crud-operations`
**Created**: 2026-03-27
**Status**: Draft
**Input**: DEV-154 - Implement StateStore interface and SQLite CRUD operations
**Parent**: DEV-146 (Autonomous Dev Pipeline)

## Interface Definition

The `StateStore` interface extends the existing DEV-153 interface (which provides `Migrate` and `Close`) with CRUD methods. All methods take `context.Context` as the first parameter.

```go
type StateStore interface {
    Migrate(ctx context.Context) error
    Close() error

    CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
    GetJob(ctx context.Context, ticketID string) (*Job, error)
    SetPR(ctx context.Context, ticketID string, prNumber int64) error

    GetCommentWatermark(ctx context.Context, prNumber int64) (int64, error)
    SetCommentWatermark(ctx context.Context, prNumber int64, commentID int64) error

    TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
    ReleaseSlot(ctx context.Context, projectID string) error
}
```

## Data Types

```go
type Job struct {
    TicketID     string
    WorktreePath string
    CmuxSession  string
    PRNumber     *int64     // nil when no PR has been set
    Status       string     // "queued" on creation; status updates are out of scope for DEV-154
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

Field-to-schema mapping:
| Go Field | Schema Column | Go Type | Notes |
|----------|---------------|---------|-------|
| TicketID | ticket_id | string | Primary key |
| WorktreePath | worktree_path | string | |
| CmuxSession | cmux_session | string | |
| PRNumber | pr_number | *int64 | nil = no PR yet |
| Status | status | string | Default "queued", no update method in this ticket |
| CreatedAt | created_at | time.Time | Set on creation |
| UpdatedAt | updated_at | time.Time | Set on creation, updated on SetPR |

## Sentinel Errors

```go
var (
    ErrInvalidDBPath = errors.New("invalid database path")       // existing
    ErrJobExists     = errors.New("job already exists")          // new
    ErrJobNotFound   = errors.New("job not found")               // new
)
```

## User Scenarios & Testing

### User Story 1 - Job Lifecycle Management (Priority: P1)

The orchestrator creates a job record when it picks up a Linear ticket for autonomous development, then updates that record as the job progresses through PR submission.

**Why this priority**: Job creation and retrieval is the foundational operation. Without it, the orchestrator cannot track what it's working on.

**Independent Test**: Can be fully tested by creating a job, retrieving it, setting its PR number, and verifying the update persists.

**Acceptance Scenarios**:

1. **Given** a new ticket ID, **When** `CreateJob` is called with worktree path and cmux session, **Then** the record is persisted and retrievable via `GetJob` with status "queued"
2. **Given** an existing job record, **When** `SetPR` is called with a PR number, **Then** `GetJob` returns the job with `PRNumber` set and `UpdatedAt` advanced
3. **Given** a non-existent ticket ID, **When** `GetJob` is called, **Then** `nil, nil` is returned (not an error)
4. **Given** an existing job for ticket X, **When** `CreateJob` is called again for ticket X, **Then** `ErrJobExists` is returned and the existing record is unchanged
5. **Given** a non-existent ticket ID, **When** `SetPR` is called, **Then** `ErrJobNotFound` is returned

---

### User Story 2 - Comment Watermark Tracking (Priority: P2)

The orchestrator polls PR comments to respond to reviewer feedback. It tracks which comments it has already processed via a watermark so it doesn't re-process old comments after restart.

**Why this priority**: Without watermark tracking, the orchestrator would re-process all comments on every poll cycle or lose track after restart.

**Independent Test**: Can be tested by setting a watermark, retrieving it, updating it, and verifying persistence across store reopen.

**Acceptance Scenarios**:

1. **Given** a PR with no prior watermark, **When** `GetCommentWatermark` is called, **Then** `0, nil` is returned
2. **Given** a PR with a watermark set to 50, **When** `SetCommentWatermark` is called with comment ID 100, **Then** `GetCommentWatermark` returns `100`
3. **Given** a PR with a watermark set to 50, **When** `SetCommentWatermark` is called with comment ID 30, **Then** `GetCommentWatermark` returns `30` (always overwrite; caller is responsible for monotonic advancement)
4. **Given** a process restart, **When** the store is reopened, **Then** previously set watermarks are still retrievable

---

### User Story 3 - Concurrency Slot Management (Priority: P2)

The orchestrator enforces a per-project concurrency limit. Before starting a new job, it must acquire a slot. If the project is at capacity, the job waits. When a job completes, the slot is released.

**Why this priority**: Prevents resource exhaustion and ensures the orchestrator respects configured parallelism limits.

**Independent Test**: Can be tested by acquiring slots up to the limit, verifying the next acquire fails, releasing one, and verifying acquire succeeds again.

**Acceptance Scenarios**:

1. **Given** a project with no existing row, **When** `TryAcquireSlot(ctx, projectID, 2)` is called, **Then** the row is auto-created via upsert with `slot_limit=2`, `active_count=1`, and `true` is returned
2. **Given** a project with `active_count=1, slot_limit=2`, **When** `TryAcquireSlot` is called, **Then** `active_count` becomes 2 and `true` is returned
3. **Given** a project at its concurrency limit (`active_count == slot_limit`), **When** `TryAcquireSlot` is called, **Then** `false, nil` is returned without modifying state
4. **Given** a running job, **When** `ReleaseSlot` is called, **Then** `active_count` decrements by 1
5. **Given** `active_count` is already 0, **When** `ReleaseSlot` is called, **Then** no-op (count stays at 0, no error)
6. **Given** a project with no existing row, **When** `ReleaseSlot` is called, **Then** no-op (no error)
7. **Given** two goroutines calling `TryAcquireSlot` simultaneously with one slot remaining, **Then** exactly one returns true and the other returns false (no double-counting)

**`limit` parameter semantics**: The `limit` argument to `TryAcquireSlot` is the **initial seed value** used when auto-creating the project row. If the row already exists, the stored `slot_limit` is authoritative and `limit` is ignored. To change a project's limit, update the row directly (out of scope for DEV-154).

---

### Edge Cases (all resolved)

| Edge Case | Behavior | Rationale |
|-----------|----------|-----------|
| `CreateJob` with duplicate ticket ID | Return `ErrJobExists` | Orchestrator should not silently overwrite |
| `ReleaseSlot` when active count is 0 | No-op, no error | Defensive; double-release shouldn't crash |
| `ReleaseSlot` for non-existent project | No-op, no error | No row to decrement |
| `SetPR` for non-existent job | Return `ErrJobNotFound` | Caller has a bug if the job doesn't exist |
| `SetCommentWatermark` with lower ID | Overwrite | Pure persistence; caller owns monotonic logic |
| `GetJob` for non-existent ticket | Return `nil, nil` | Not-found is not an error |
| `GetCommentWatermark` for untracked PR | Return `0, nil` | Zero value signals "no watermark" |

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide a `StateStore` interface as defined above, retaining existing `Migrate()` and `Close()` methods, adding all CRUD methods
- **FR-002**: System MUST define a `Job` struct with fields as specified above
- **FR-003**: System MUST add `ErrJobExists` and `ErrJobNotFound` sentinel errors to `errors.go`
- **FR-004**: `CreateJob` MUST persist a new job with status "queued" and return `ErrJobExists` on duplicate
- **FR-005**: `GetJob` MUST return `*Job, error` where nil Job means not found (not an error)
- **FR-006**: `SetPR` MUST update an existing job's PR number and `updated_at`, returning `ErrJobNotFound` if missing
- **FR-007**: `GetCommentWatermark` MUST return `int64, error` with 0 for untracked PRs
- **FR-008**: `SetCommentWatermark` MUST upsert (insert or update) the watermark for a PR
- **FR-009**: `TryAcquireSlot` MUST atomically check-and-increment in a single transaction, auto-creating the project row via upsert if needed
- **FR-010**: `ReleaseSlot` MUST decrement active count, clamping to 0, no-op for non-existent projects
- **FR-011**: All write operations MUST handle concurrent access safely (SQLite WAL mode + single-writer)
- **FR-012**: All state MUST persist across process restarts
- **FR-013**: All methods MUST set `updated_at` timestamps on mutation operations

### Scope Exclusions

- Job status updates (no `SetStatus` method) -- next ticket
- Crash-recovery reconciliation logic -- next ticket
- Linear API integration
- Any business logic -- pure state persistence only

## Success Criteria

- **SC-001**: All CRUD operations complete successfully with correct data round-tripping
- **SC-002**: Concurrent goroutine access does not produce data races or incorrect slot counts
- **SC-003**: Data persists across store close/reopen cycles
- **SC-004**: Interface is clean enough that orchestrator-core can mock it without leaking SQLite details
- **SC-005**: All sentinel errors are testable via `errors.Is()`

## Testing Requirements

### Test Strategy

Integration tests against a real SQLite database using `t.TempDir()` for isolation. No mocks. Use `testify/require` for setup assertions and `testify/assert` for test assertions. Concurrent access tests use goroutines with `sync.WaitGroup`. Follow existing `setupTestStore(t)` helper pattern.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Compile-time | `var _ StateStore = (*SQLiteStore)(nil)` |
| FR-002 | Compile-time | Job struct compiles with expected fields |
| FR-003 | Unit | `errors.Is(err, ErrJobExists)` and `errors.Is(err, ErrJobNotFound)` |
| FR-004 | Integration | CreateJob persists, retrievable; duplicate returns ErrJobExists |
| FR-005 | Integration | GetJob returns *Job or nil,nil for missing |
| FR-006 | Integration | SetPR updates PR number; returns ErrJobNotFound for missing |
| FR-007 | Integration | GetCommentWatermark returns 0 for untracked |
| FR-008 | Integration | SetCommentWatermark upserts correctly; overwrite with lower ID works |
| FR-009 | Integration | TryAcquireSlot atomic check-and-increment; auto-creates row; returns false at limit |
| FR-010 | Integration | ReleaseSlot decrements; clamps to 0; no-op for missing project |
| FR-011 | Integration | Concurrent TryAcquireSlot with goroutines does not double-count |
| FR-012 | Integration | Data survives store close/reopen |
| FR-013 | Integration | updated_at advances on SetPR and SetCommentWatermark |
