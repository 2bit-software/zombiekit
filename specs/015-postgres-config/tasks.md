# Tasks: PostgreSQL Configuration with SQLite Fallback

**Input**: Design documents from `/specs/015-postgres-config/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Not explicitly requested - minimal test tasks included for critical paths only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Project type**: Single project (CLI tool + MCP server)
- Paths use `internal/` for Go packages per existing structure

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No new setup needed - feature extends existing packages

- [X] T001 Verify existing config package structure in internal/config/

**Note**: This feature extends existing code, no new project initialization required.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data structures and config parsing that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Add FileStorageConfig struct with TOML tags in internal/config/storage.go
- [X] T003 Add Storage field to Config struct in internal/config/config.go
- [X] T004 Add ConnectionTimeout field to StorageConfig in internal/config/storage.go
- [X] T005 [P] Implement storage config merging in internal/config/merger.go
- [X] T006 [P] Add LoadStorageConfigFromFile function in internal/config/loader.go
- [X] T007 Create LoadStorageConfig function that merges file + env configs in internal/config/loader.go

**Checkpoint**: Foundation ready - config loading from TOML files works, user story implementation can begin

---

## Phase 3: User Story 1 - Configure PostgreSQL via Config File (Priority: P1)

**Goal**: Allow developers to configure PostgreSQL connection via `.brains/config.toml` file

**Independent Test**: Create config file with valid PostgreSQL URL, verify application connects to PostgreSQL on startup

### Implementation for User Story 1

- [X] T008 [US1] Add connection timeout parameter to NewPostgresPool in internal/database/postgres.go (already supported via context)
- [X] T009 [US1] Create context with timeout for connection attempts in internal/cli/serve.go
- [X] T010 [US1] Update serve.go to load storage config from file + env in internal/cli/serve.go
- [X] T011 [US1] Add PostgreSQL connection attempt with timeout in internal/cli/serve.go
- [X] T012 [US1] Add logging for successful PostgreSQL connection in internal/cli/serve.go
- [X] T013 [US1] Add unit test for TOML parsing with [storage] section in internal/config/storage_test.go

**Checkpoint**: User Story 1 complete - application connects to PostgreSQL when configured via config file

---

## Phase 4: User Story 2 - Automatic Fallback to SQLite (Priority: P1)

**Goal**: Automatically fall back to SQLite when PostgreSQL is unavailable

**Independent Test**: Configure invalid PostgreSQL credentials, verify application uses SQLite with warning logged

### Implementation for User Story 2

- [X] T014 [US2] Implement fallback logic when PostgreSQL connection fails in internal/cli/serve.go
- [X] T015 [US2] Add warning log with failure reason when fallback occurs in internal/cli/serve.go
- [X] T016 [US2] Update StorageConfig.Backend to reflect actual backend after fallback in internal/cli/serve.go
- [ ] T017 [US2] Add integration test for fallback behavior in tests/integration/config_fallback_test.go (deferred - requires testcontainers)

**Checkpoint**: User Story 2 complete - application falls back to SQLite gracefully when PostgreSQL unavailable

---

## Phase 5: User Story 3 - Environment Variables Override Config File (Priority: P2)

**Goal**: Allow environment variables to override config file settings

**Independent Test**: Set BRAINS_BACKEND=sqlite with postgres config file, verify SQLite is used

### Implementation for User Story 3

- [X] T018 [US3] Ensure env var precedence in LoadStorageConfig function in internal/config/loader.go
- [X] T019 [US3] Add unit test for env var override behavior in internal/config/storage_test.go
- [X] T020 [US3] Verify backward compatibility with existing env var usage in internal/config/storage_test.go

**Checkpoint**: User Story 3 complete - environment variables correctly override config file settings

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T021 [P] Add timeout validation (1-300 seconds range) in internal/config/loader.go
- [X] T022 [P] Update quickstart.md with actual tested examples in specs/015-postgres-config/quickstart.md
- [X] T023 Run all tests and verify no regressions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Verification only - no blocking dependencies
- **Foundational (Phase 2)**: Creates data structures for all user stories - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 priority but can be worked sequentially
  - US3 depends on US1 being complete (needs config file loading to test override)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Builds on US1's PostgreSQL connection code
- **User Story 3 (P2)**: Can start after US1 complete - Tests override of file config with env vars

### Within Each User Story

- Database layer changes before CLI changes
- Config loading before usage
- Implementation before tests

### Parallel Opportunities

- T005 and T006 can run in parallel (different files, no dependencies)
- T021 and T022 can run in parallel (different files)

---

## Parallel Example: Foundational Phase

```bash
# Launch parallel foundational tasks together:
Task: "T005 Implement storage config merging in internal/config/merger.go"
Task: "T006 Add LoadStorageConfigFromFile function in internal/config/loader.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (verification)
2. Complete Phase 2: Foundational (data structures + parsing)
3. Complete Phase 3: User Story 1 (PostgreSQL config from file)
4. Complete Phase 4: User Story 2 (Fallback to SQLite)
5. **STOP and VALIDATE**: Test with real PostgreSQL + fallback scenarios
6. Deploy/demo if ready - core functionality complete

### Incremental Delivery

1. Complete Foundational → Config parsing works
2. Add User Story 1 → PostgreSQL from config file works
3. Add User Story 2 → Fallback works → **Core feature complete**
4. Add User Story 3 → Env override works → **Full feature complete**
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1 and US2 are both P1 priority but share code in serve.go - work sequentially
- Status display already shows backend (no changes needed per plan.md)
- No new dependencies needed - all libraries already in go.mod
