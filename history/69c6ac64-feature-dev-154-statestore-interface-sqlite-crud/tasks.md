# Tasks: DEV-154 StateStore CRUD

**Complexity**: Simple (3 files, ~200 LOC, 0 cross-module deps)
**Total tasks**: 10
**Parallel opportunities**: T003-T005 can run in parallel after T001+T002

## Dependency Graph

```
T001 (errors) ──┐
                 ├── T003 (job CRUD) [P] ──┐
T002 (struct+iface) ┘   T004 (watermark) [P] ├── T006-T010 (tests)
                     T005 (slots) [P] ────────┘
```

## Tasks

### Phase 1: Types and Interface

- [ ] T001 [FR-003] Add sentinel errors `ErrJobExists` and `ErrJobNotFound` to `internal/state/errors.go`
  - Add two `var` declarations alongside existing `ErrInvalidDBPath`
  - **Accept**: `errors.Is(ErrJobExists, ErrJobExists)` and `errors.Is(ErrJobNotFound, ErrJobNotFound)` both true

- [ ] T002 [FR-001, FR-002] Add `Job` struct and extend `StateStore` interface in `internal/state/store.go`
  - Add `Job` struct with fields: TicketID, WorktreePath, CmuxSession, PRNumber (*int64), Status, CreatedAt, UpdatedAt (time.Time)
  - Add `time` import
  - Extend `StateStore` interface with 7 CRUD method signatures (retain Migrate + Close)
  - Remove DEV-153 scope comment from interface
  - **Accept**: `var _ StateStore = (*SQLiteStore)(nil)` compiles (will fail until T003-T005 implement methods)

### Phase 2: CRUD Implementation (parallelizable)

- [ ] T003 [P] [FR-004, FR-005, FR-006, FR-013] Implement job CRUD methods on `SQLiteStore` in `internal/state/store.go`
  - `CreateJob`: INSERT with status "queued", detect UNIQUE constraint via `strings.Contains(err.Error(), "UNIQUE constraint failed")` -> wrap as `ErrJobExists`
  - `GetJob`: SELECT by ticket_id, `sql.ErrNoRows` -> `nil, nil`, scan `pr_number` via `sql.NullInt64` -> `*int64`
  - `SetPR`: UPDATE pr_number + updated_at, `RowsAffected() == 0` -> `ErrJobNotFound`
  - Add `"database/sql"`, `"strings"`, `"time"` imports as needed
  - **Accept**: CreateJob + GetJob round-trips; duplicate returns ErrJobExists; SetPR on missing returns ErrJobNotFound

- [ ] T004 [P] [FR-007, FR-008, FR-013] Implement watermark methods on `SQLiteStore` in `internal/state/store.go`
  - `GetCommentWatermark`: SELECT last_processed_comment_id, `sql.ErrNoRows` -> `0, nil`
  - `SetCommentWatermark`: INSERT ON CONFLICT DO UPDATE (upsert), set updated_at
  - **Accept**: Untracked PR returns 0; set+get round-trips; overwrite with lower ID works

- [ ] T005 [P] [FR-009, FR-010, FR-011] Implement slot methods on `SQLiteStore` in `internal/state/store.go`
  - `TryAcquireSlot`: Begin transaction, upsert project row (INSERT ON CONFLICT DO NOTHING with seed limit), atomic UPDATE WHERE active_count < slot_limit, check RowsAffected
  - `ReleaseSlot`: UPDATE SET active_count = MAX(active_count - 1, 0), no-op if row missing
  - **Accept**: Auto-creates project; returns false at limit; clamps to 0; no error for missing project

### Phase 3: Tests

- [ ] T006 [FR-004, FR-005] Add job creation and retrieval tests to `internal/state/store_test.go`
  - `TestCreateJob_AndGetJob`: Create job, verify all fields including status "queued" and timestamps
  - `TestCreateJob_Duplicate_ReturnsErrJobExists`: Second create with same ticket_id returns ErrJobExists
  - `TestGetJob_NonExistent_ReturnsNil`: Returns nil, nil for missing ticket

- [ ] T007 [FR-006, FR-013] Add SetPR tests to `internal/state/store_test.go`
  - `TestSetPR_UpdatesJob`: Set PR number, verify GetJob returns updated PRNumber and advanced UpdatedAt
  - `TestSetPR_NonExistent_ReturnsErrJobNotFound`: SetPR on missing ticket returns ErrJobNotFound

- [ ] T008 [FR-007, FR-008] Add watermark tests to `internal/state/store_test.go`
  - `TestGetCommentWatermark_Untracked_ReturnsZero`: Returns 0 for new PR
  - `TestSetCommentWatermark_RoundTrip`: Set and get returns correct value
  - `TestSetCommentWatermark_Overwrite`: Upsert overwrites existing (including lower ID)

- [ ] T009 [FR-009, FR-010, FR-011] Add slot tests to `internal/state/store_test.go`
  - `TestTryAcquireSlot_AutoCreatesProject`: First call creates row, returns true
  - `TestTryAcquireSlot_AtLimit_ReturnsFalse`: Fill slots, next acquire returns false
  - `TestReleaseSlot_Decrements`: Active count goes down by 1
  - `TestReleaseSlot_ClampsToZero`: Release at 0 is no-op
  - `TestReleaseSlot_NonExistentProject_NoOp`: No error for missing project
  - `TestTryAcquireSlot_Concurrent`: N goroutines with WaitGroup, exactly `limit` succeed

- [ ] T010 [FR-001, FR-012] Add cross-cutting tests to `internal/state/store_test.go`
  - Add compile-time interface check: `var _ StateStore = (*SQLiteStore)(nil)`
  - `TestPersistence_AcrossReopen`: Create job + set watermark + acquire slot, close store, reopen, verify all data

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T002, T010 |
| FR-002 | T002 |
| FR-003 | T001 |
| FR-004 | T003, T006 |
| FR-005 | T003, T006 |
| FR-006 | T003, T007 |
| FR-007 | T004, T008 |
| FR-008 | T004, T008 |
| FR-009 | T005, T009 |
| FR-010 | T005, T009 |
| FR-011 | T005, T009 |
| FR-012 | T010 |
| FR-013 | T003, T004, T007 |
