# Tasks: Core Repository Setup

**Input**: Design documents from `/specs/001-core-repo-setup/`
**Prerequisites**: plan.md, spec.md, research.md

**Tests**: Not explicitly requested in spec - using test harnesses as placeholders per FR-017.

**Organization**: Tasks grouped by user story to enable independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, etc.)
- All paths relative to repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Go module and project structure

- [x] T001 Initialize Go module with `go mod init github.com/2bit-software/zombiekit` in go.mod
- [x] T002 [P] Create .gitignore with Go-specific patterns
- [x] T003 [P] Create configs/.golangci.yml with linter configuration
- [x] T004 [P] Create migrations/.gitkeep placeholder
- [x] T005 [P] Create profiles/.gitkeep placeholder

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CLI skeleton and core packages that all user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Create cmd/brains/main.go with CLI entry point using urfave/cli/v2
- [x] T007 Create internal/cli/root.go with root command and global flags
- [x] T008 Create internal/cli/version.go with version command (ldflags: version, commit)
- [x] T009 [P] Create internal/config/config.go placeholder with package doc
- [x] T010 [P] Create internal/profile/service.go placeholder with package doc
- [x] T011 [P] Create internal/spec/service.go placeholder with package doc
- [x] T012 [P] Create internal/conversation/service.go placeholder with package doc
- [x] T013 [P] Create internal/mcp/server.go placeholder with package doc
- [x] T014 [P] Create internal/web/server.go placeholder with package doc
- [x] T015 Run `go mod tidy` to sync dependencies

**Checkpoint**: CLI skeleton compiles and runs `./bin/brains --help`

---

## Phase 3: User Story 1 - Developer Clones and Builds Project (Priority: P1) MVP

**Goal**: Developer can clone repo and build working binary with `task init && task build`

**Independent Test**: Run `task build` and verify `./bin/brains --help` and `./bin/brains version` work

### Implementation for User Story 1

- [x] T016 [US1] Create Taskfile.yml with `default` task that lists available tasks
- [x] T017 [US1] Add `init` task to Taskfile.yml that runs `go mod download`
- [x] T018 [US1] Add `build` task to Taskfile.yml with ldflags for version/commit embedding
- [x] T019 [US1] Verify `task init && task build` produces working binary at ./bin/brains
- [x] T020 [US1] Verify `./bin/brains --help` displays usage information
- [x] T021 [US1] Verify `./bin/brains version` displays version and commit hash

**Checkpoint**: User Story 1 complete - developers can build the project

---

## Phase 4: User Story 2 - Developer Runs Tests (Priority: P2)

**Goal**: Developer can run test suite with coverage using `task test`

**Independent Test**: Run `task test` and verify coverage.out is generated

### Implementation for User Story 2

- [x] T022 [P] [US2] Create internal/cli/root_test.go with placeholder test
- [x] T023 [P] [US2] Create internal/cli/version_test.go with placeholder test
- [x] T024 [P] [US2] Create internal/profile/service_test.go with placeholder test
- [x] T025 [P] [US2] Create internal/spec/service_test.go with placeholder test
- [x] T026 [P] [US2] Create internal/conversation/service_test.go with placeholder test
- [x] T027 [P] [US2] Create internal/mcp/server_test.go with placeholder test
- [x] T028 [P] [US2] Create internal/web/server_test.go with placeholder test
- [x] T029 [US2] Add `test` task to Taskfile.yml that runs tests with coverage
- [x] T030 [US2] Verify `task test` executes all test harnesses and generates coverage.out

**Checkpoint**: User Story 2 complete - test infrastructure established

---

## Phase 5: User Story 3 - Developer Starts Database Services (Priority: P3)

**Goal**: Developer can start PostgreSQL with pgvector using `task db:up`

**Independent Test**: Run `task db:up` and verify PostgreSQL accepts connections on port 9432

### Implementation for User Story 3

- [x] T031 [US3] Create docker-compose.yml with PostgreSQL service (pgvector/pgvector:pg16, port 9432)
- [x] T032 [US3] Add `db:up` task to Taskfile.yml that starts PostgreSQL via docker-compose
- [x] T033 [US3] Add `db:down` task to Taskfile.yml that stops and removes containers
- [x] T034 [US3] Add `db:migrate` task to Taskfile.yml (placeholder - logs message)
- [x] T035 [US3] Verify `task db:up` starts container and PostgreSQL accepts connections
- [x] T036 [US3] Verify `task db:down` stops container cleanly

**Checkpoint**: User Story 3 complete - database services available

---

## Phase 6: User Story 4 - Developer Runs Code Quality Checks (Priority: P3)

**Goal**: Developer can lint, format, and vet code using task commands

**Independent Test**: Run `task lint` and verify golangci-lint executes with project config

### Implementation for User Story 4

- [x] T037 [US4] Add `fmt` task to Taskfile.yml that runs `go fmt ./...`
- [x] T038 [US4] Add `vet` task to Taskfile.yml that runs `go vet ./...`
- [x] T039 [US4] Add `lint` task to Taskfile.yml that runs golangci-lint with configs/.golangci.yml
- [x] T040 [US4] Update `init` task to install golangci-lint if not present
- [x] T041 [US4] Verify `task fmt` formats code without errors
- [x] T042 [US4] Verify `task vet` runs go vet successfully
- [x] T043 [US4] Verify `task lint` runs golangci-lint with project configuration

**Checkpoint**: User Story 4 complete - code quality tooling available

---

## Phase 7: User Story 5 - CI Pipeline Runs All Checks (Priority: P4)

**Goal**: CI can run all quality gates with single `task ci` command

**Independent Test**: Run `task ci` and verify all checks execute in sequence

### Implementation for User Story 5

- [x] T044 [US5] Add `ci` task to Taskfile.yml that runs fmt, vet, lint, test, build in sequence
- [x] T045 [US5] Verify `task ci` fails fast if any check fails
- [x] T046 [US5] Verify `task ci` exits 0 when all checks pass

**Checkpoint**: User Story 5 complete - CI automation ready

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and documentation

- [x] T047 [P] Verify `task --list` shows all tasks with descriptions
- [x] T048 [P] Verify project structure matches MASTER-DESIGN.md layout
- [x] T049 Run quickstart.md validation (clone → init → build workflow)
- [x] T050 Commit all changes with summary of repository setup

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001-T005)
- **User Stories (Phase 3-7)**: All depend on Foundational (T006-T015)
  - US1 (Phase 3): Can start after Foundational
  - US2 (Phase 4): Can start after US1 (needs Taskfile.yml)
  - US3 (Phase 5): Can start after US1 (needs Taskfile.yml)
  - US4 (Phase 6): Can start after US1 (needs Taskfile.yml)
  - US5 (Phase 7): Depends on US2 and US4 (needs test and lint tasks)
- **Polish (Phase 8)**: Depends on all user stories

### User Story Dependencies

- **US1 (Build)**: Foundational only - creates Taskfile.yml that others extend
- **US2 (Tests)**: US1 - extends Taskfile.yml with test task
- **US3 (Database)**: US1 - extends Taskfile.yml with db tasks
- **US4 (Quality)**: US1 - extends Taskfile.yml with lint/fmt/vet tasks
- **US5 (CI)**: US2 + US4 - composes test and lint tasks into ci task

### Parallel Opportunities

- Setup tasks T002-T005 can run in parallel
- Foundational tasks T009-T014 can run in parallel (package placeholders)
- US2 test harness tasks T022-T028 can run in parallel
- US3 and US4 can run in parallel after US1 (different concerns)

---

## Parallel Example: User Story 2 (Test Harnesses)

```bash
# Launch all test harness tasks in parallel:
Task: "Create internal/cli/root_test.go with placeholder test"
Task: "Create internal/cli/version_test.go with placeholder test"
Task: "Create internal/profile/service_test.go with placeholder test"
Task: "Create internal/spec/service_test.go with placeholder test"
Task: "Create internal/conversation/service_test.go with placeholder test"
Task: "Create internal/mcp/server_test.go with placeholder test"
Task: "Create internal/web/server_test.go with placeholder test"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T015)
3. Complete Phase 3: User Story 1 (T016-T021)
4. **STOP and VALIDATE**: Run `task build` and verify binary works
5. Commit and deploy MVP

### Incremental Delivery

1. Complete Setup + Foundational → CLI compiles
2. Add User Story 1 → `task build` works (MVP!)
3. Add User Story 2 → `task test` works
4. Add User Story 3 → `task db:up` works
5. Add User Story 4 → `task lint` works
6. Add User Story 5 → `task ci` works
7. Each story adds developer capability without breaking previous

---

## Notes

- [P] tasks = different files, no dependencies
- [USn] label maps task to specific user story
- All verification tasks (T019-T021, T030, T035-T036, T041-T043, T045-T046) are manual checks
- Test harnesses use `t.Skip("not implemented")` pattern from research.md
- Taskfile.yml grows incrementally with each user story
- Commit after each phase checkpoint
