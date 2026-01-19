# Tasks: Smarter Import Synchronization

**Feature**: DEV-87
**Generated**: 2026-01-19
**Complexity**: Medium (7 files, 6 phases)

## Task List

### Phase 1: Database Schema

- [ ] T001 [FR-001] Create migration `internal/database/migrations/postgres/005_recall_import_state.sql` with `recall_import_state` table
- [ ] T002 [FR-005] Add `history_gap` column and partial index to `recall_chunks` in same migration
- [ ] T003 [FR-001] Add `ImportState` type to `internal/recall/types.go`
- [ ] T004 [P] [FR-001] Implement `GetImportState` method in `internal/recall/postgres/storage.go`
- [ ] T005 [P] [FR-001] Implement `SaveImportState` method (UPSERT) in `internal/recall/postgres/storage.go`
- [ ] T006 [P] [FR-009] Implement `DeleteImportState` method in `internal/recall/postgres/storage.go`
- [ ] T007 [P] [FR-009] Implement `CleanupStaleImportStates` method in `internal/recall/postgres/storage.go`
- [ ] T008 [FR-005] Extend `ChunkInput` with `HistoryGap` field and update `SaveWithSource` to persist it

### Phase 2: Parser Extension

- [ ] T009 [FR-003][FR-004] Add `ErrSyncPointNotFound` sentinel error to `internal/recall/claude/parser.go`
- [ ] T010 [FR-003][FR-004] Implement `ParseFileFromUUID` function in `internal/recall/claude/parser.go`

### Phase 3: File Lock Mechanism

- [ ] T011 [FR-010] Create `internal/recall/claude/lock.go` with `ImportLock` type
- [ ] T012 [FR-010][NFR-001] Implement `AcquireLock` function with `syscall.Flock` (non-blocking exclusive)
- [ ] T013 [NFR-001] Implement `Release` method with safe cleanup

### Phase 4: Import Logic Refactor

- [ ] T014 Extend `Storage` interface in `internal/recall/storage.go` with import state methods
- [ ] T015 [FR-010] Integrate lock acquisition at start of `importClaudeHistory` in `internal/cli/recall.go`
- [ ] T016 [FR-009] Add stale state cleanup call after file discovery in `internal/cli/recall.go`
- [ ] T017 [FR-002] Implement mtime-based file skip logic in new `processFile` function
- [ ] T018 [FR-003][FR-004] Implement sync point detection and incremental parsing in `processFile`
- [ ] T019 [FR-005][FR-006] Implement divergence handling with `history_gap` marking and warning output
- [ ] T020 [FR-007] Add `--force` flag to CLI and wire bypass logic

### Phase 5: Output Verbosity

- [ ] T021 [FR-008] Change default output to summary-only (X new from Y files, Z unchanged)
- [ ] T022 [FR-008] Add per-file verbose output when `--verbose` flag present
- [ ] T023 [FR-006] Ensure divergence warnings appear regardless of verbosity level

### Phase 6: Testing

- [ ] T024 [P] Unit tests for `ParseFileFromUUID` (valid UUID, empty UUID, missing UUID, empty file)
- [ ] T025 [P] Unit tests for lock acquisition and release
- [ ] T026 [P] Integration tests for import state CRUD operations
- [ ] T027 Integration test: fresh import creates import state
- [ ] T028 Integration test: unchanged file skipped via mtime check
- [ ] T029 Integration test: changed file imports only new entries
- [ ] T030 Integration test: missing sync point triggers divergence handling
- [ ] T031 Integration test: `--force` flag bypasses state check
- [ ] T032 Integration test: stale states cleaned up for deleted files
- [ ] T033 Integration test: concurrent import blocked with clear error

## Dependency Order

```
T001 ──► T002 ──► T003 ──┬──► T004-T008 (parallel) ──┐
                         │                            │
                         ├──► T009-T010 (sequential) ─┼──► T014 ──► T015-T020 (sequential) ──► T021-T023 (sequential) ──► T024-T033
                         │                            │
                         └──► T011-T013 (sequential) ─┘
```

**Parallel Opportunities:**
- T004, T005, T006, T007 can run in parallel (independent storage methods)
- T024, T025, T026 can run in parallel (independent test suites)
- Phase 2 (T009-T010) and Phase 3 (T011-T013) can run in parallel after T003

**Critical Path:**
T001 → T002 → T003 → T010 → T014 → T018 → T021 → T027

## Traceability Matrix

| Task | FR/NFR | User Story |
|------|--------|------------|
| T001-T008 | FR-001, FR-005, FR-009 | US1, US3 |
| T009-T010 | FR-003, FR-004 | US1 |
| T011-T013 | FR-010, NFR-001 | US1 |
| T014-T020 | FR-002-007, NFR-002 | US1, US2, US3, US4 |
| T021-T023 | FR-006, FR-008 | US1 |
| T024-T033 | All FRs | All USs |

## Completion Summary

- **Total tasks**: 33
- **Parallel opportunities**: 10 tasks can be parallelized
- **Estimated complexity**: Medium
- **Suggested execution order**: Follow dependency graph above

## Next Step

```
/brains.implement
```
