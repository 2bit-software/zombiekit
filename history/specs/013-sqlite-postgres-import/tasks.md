# Tasks: SQLite to PostgreSQL Migration Tool

**Input**: Design documents from `/specs/013-sqlite-postgres-import/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Tests are included as this project follows test-first principles per the constitution check.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- Core logic: `internal/memory/importer/`
- CLI commands: `internal/cli/`
- Integration tests: `tests/integration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create importer package structure and core types

- [X] T001 Create importer package directory at internal/memory/importer/
- [X] T002 [P] Define ImportOptions and ImportResult types in internal/memory/importer/types.go
- [X] T003 [P] Define ImportMetadata type and repository interface in internal/memory/importer/metadata.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Implement ImportMetadata PostgreSQL table creation in internal/memory/importer/metadata.go
- [X] T005 Implement GetLastImport and SaveImportMetadata operations in internal/memory/importer/metadata.go
- [X] T006 Implement SQLite exclusive locking helper (PRAGMA locking_mode=EXCLUSIVE) in internal/memory/importer/importer.go
- [X] T007 Implement timestamp normalization to UTC helper in internal/memory/importer/importer.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - First-time Migration (Priority: P1) 🎯 MVP

**Goal**: Import all memory items from SQLite to PostgreSQL on first run

**Independent Test**: Run import command against SQLite with sample memories, verify all data appears in PostgreSQL

### Tests for User Story 1

- [X] T008 [P] [US1] Write test for basic import (10 items) in internal/memory/importer/importer_test.go
- [X] T009 [P] [US1] Write test for Unicode/special character preservation in internal/memory/importer/importer_test.go
- [X] T010 [P] [US1] Write test for empty source database in internal/memory/importer/importer_test.go

### Implementation for User Story 1

- [X] T011 [US1] Implement fetchAllMemories() to read from SQLite in internal/memory/importer/importer.go
- [X] T012 [US1] Implement insertMemory() to write single item to PostgreSQL in internal/memory/importer/importer.go
- [X] T013 [US1] Implement Import() main method for full import in internal/memory/importer/importer.go
- [X] T014 [US1] Add basic CLI command structure for 'brains db import' in internal/cli/import.go
- [X] T015 [US1] Wire --from and --to flags in CLI command in internal/cli/import.go
- [X] T016 [US1] Add basic completion summary output (items imported) in internal/cli/import.go

**Checkpoint**: First-time import working - can migrate all data from SQLite to PostgreSQL

---

## Phase 4: User Story 2 - Incremental Migration (Priority: P1)

**Goal**: Import only items added/updated since last import

**Independent Test**: Run initial import, add new memories to SQLite, run import again, verify only new items transferred

### Tests for User Story 2

- [X] T017 [P] [US2] Write test for incremental import (skip already-imported) in internal/memory/importer/importer_test.go
- [X] T018 [P] [US2] Write test for new version import with soft-delete of old in internal/memory/importer/importer_test.go
- [X] T019 [P] [US2] Write test for zero items when no changes in internal/memory/importer/importer_test.go

### Implementation for User Story 2

- [X] T020 [US2] Implement fetchMemoriesSince(timestamp) to read only new items in internal/memory/importer/importer.go
- [X] T021 [US2] Implement version conflict resolution (compare and soft-delete) in internal/memory/importer/importer.go
- [X] T022 [US2] Implement checkExistingVersion() for name/version lookup in PostgreSQL in internal/memory/importer/importer.go
- [X] T023 [US2] Update Import() to check last import timestamp and use incremental path in internal/memory/importer/importer.go
- [X] T024 [US2] Update Import() to save new import metadata after completion in internal/memory/importer/importer.go

**Checkpoint**: Incremental import working - subsequent runs only transfer new data

---

## Phase 5: User Story 3 - Import Status Visibility (Priority: P2)

**Goal**: Preview pending imports and show progress during import

**Independent Test**: Run with --dry-run flag, verify items listed without changes; run with --verbose, verify progress shown

### Tests for User Story 3

- [X] T025 [P] [US3] Write test for dry-run mode (no data changes) in internal/memory/importer/importer_test.go
- [X] T026 [P] [US3] Write test for progress callback invocation in internal/memory/importer/importer_test.go

### Implementation for User Story 3

- [X] T027 [US3] Add DryRun field to ImportOptions in internal/memory/importer/types.go
- [X] T028 [US3] Add OnProgress callback field to ImportOptions in internal/memory/importer/types.go
- [X] T029 [US3] Implement Preview() method for dry-run scanning in internal/memory/importer/importer.go
- [X] T030 [US3] Add progress callback invocation in Import() batch loop in internal/memory/importer/importer.go
- [X] T031 [US3] Wire --dry-run flag in CLI command in internal/cli/import.go
- [X] T032 [US3] Wire --verbose flag with progress display in internal/cli/import.go
- [X] T033 [US3] Add --format json support for CI/CD usage in internal/cli/import.go

**Checkpoint**: Preview and progress working - users can see what will happen before import

---

## Phase 6: User Story 4 - Error Recovery (Priority: P3)

**Goal**: Handle failures gracefully and allow resume without data corruption

**Independent Test**: Simulate failure mid-import, re-run, verify no duplicates and remaining items imported

### Tests for User Story 4

- [X] T034 [P] [US4] Write test for partial failure recovery (re-run skips completed) in internal/memory/importer/importer_test.go
- [X] T035 [P] [US4] Write test for per-item error handling (continue on single item failure) in internal/memory/importer/importer_test.go
- [X] T036 [P] [US4] Write test for PostgreSQL unavailable error message in internal/memory/importer/importer_test.go

### Implementation for User Story 4

- [X] T037 [US4] Implement batch-level commits for partial progress in internal/memory/importer/importer.go
- [X] T038 [US4] Add --batch-size flag support in internal/cli/import.go
- [X] T039 [US4] Implement per-item error collection in ImportResult.Errors in internal/memory/importer/importer.go
- [X] T040 [US4] Add connection validation with clear error messages in internal/memory/importer/importer.go
- [X] T041 [US4] Implement exit code 2 for partial failure in internal/cli/import.go
- [X] T042 [US4] Add error details to JSON output format in internal/cli/import.go

**Checkpoint**: Error recovery working - failed imports can be resumed safely

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Integration tests, documentation, and final validation

- [X] T043 [P] Write end-to-end integration test with testcontainers in tests/integration/import_test.go
- [X] T044 [P] Add import command to CLI help and db subcommand in internal/cli/db.go
- [X] T045 Validate quickstart.md scenarios work as documented
- [X] T046 Run all tests and ensure passing

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Depends on US1 (Import() method must exist to extend)
- **User Story 3 (P2)**: Can start after US1 (needs Import() structure, but adds independent features)
- **User Story 4 (P3)**: Depends on US1 and US2 (extends existing import logic)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Type definitions before service logic
- Service logic before CLI wiring
- Core implementation before output formatting

### Parallel Opportunities

- T002, T003 (Setup types) can run in parallel
- All test tasks within a story marked [P] can run in parallel
- T043, T044 (Polish phase) can run in parallel

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all tests for User Story 1 together:
Task: "Write test for basic import (10 items) in internal/memory/importer/importer_test.go"
Task: "Write test for Unicode/special character preservation in internal/memory/importer/importer_test.go"
Task: "Write test for empty source database in internal/memory/importer/importer_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test full import works end-to-end
5. Can deploy with basic import capability

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Basic import works (MVP!)
3. Add User Story 2 → Incremental import works
4. Add User Story 3 → Preview and progress added
5. Add User Story 4 → Error recovery added
6. Each story adds value without breaking previous stories

### Suggested Order

Since US2 extends US1, recommended sequential order:
1. Phase 1: Setup
2. Phase 2: Foundational
3. Phase 3: US1 (First-time Migration)
4. Phase 4: US2 (Incremental Migration)
5. Phase 5: US3 (Status Visibility) - can run in parallel with US4
6. Phase 6: US4 (Error Recovery) - can run in parallel with US3
7. Phase 7: Polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
