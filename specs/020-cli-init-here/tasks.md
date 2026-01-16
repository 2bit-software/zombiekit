# Tasks: CLI Init Enhancement

**Input**: Design documents from `/specs/020-cli-init-here/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md

**Tests**: Not explicitly requested in specification. Tests are included as part of polish phase for comprehensive coverage.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go CLI project at repository root
- Key files: `embed.go`, `internal/cli/init.go`, `cmd/brains/main.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add embedded filesystem declarations for commands and templates

- [X] T001 Add EmbeddedCommands variable with //go:embed directive for integrations/claude/commands/* in embed.go
- [X] T002 [P] Add EmbeddedTemplates variable with //go:embed directive for templates/templates/* in embed.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Export EmbeddedCommands and EmbeddedTemplates from embed.go for use by internal packages
- [X] T004 Add --force flag definition to init command in internal/cli/init.go
- [X] T005 Create helper function copyEmbeddedFiles(fsys fs.FS, srcPrefix, destDir string, force bool) in internal/cli/init.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Initialize ZombieKit in Current Repository (Priority: P1) 🎯 MVP

**Goal**: Run `brains init` to create .claude/commands/ and .brains/templates/ with all embedded files

**Independent Test**: Run `brains init` in a fresh directory, verify 15 command files in .claude/commands/ and 5 template files in .brains/templates/

### Implementation for User Story 1

- [X] T006 [US1] Modify init command action to call full setup when --global is NOT specified in internal/cli/init.go
- [X] T007 [US1] Implement .claude/ and .claude/commands/ directory creation in internal/cli/init.go
- [X] T008 [US1] Implement .brains/ and .brains/templates/ directory creation in internal/cli/init.go
- [X] T009 [US1] Call copyEmbeddedFiles for EmbeddedCommands to .claude/commands/ in internal/cli/init.go
- [X] T010 [US1] Call copyEmbeddedFiles for EmbeddedTemplates to .brains/templates/ in internal/cli/init.go
- [X] T011 [US1] Implement verbose output: print each file as copied/skipped in internal/cli/init.go
- [X] T012 [US1] Implement skip logic: check if file exists before copying (no --force) in internal/cli/init.go
- [X] T013 [US1] Print summary at end: "Initialized ZombieKit: X files copied, Y skipped" in internal/cli/init.go
- [X] T014 [US1] Register .brains directory in profile registry after successful initialization in internal/cli/init.go

**Checkpoint**: At this point, `brains init` should fully work for fresh directories

---

## Phase 4: User Story 2 - View Init Help and Options (Priority: P2)

**Goal**: `brains init --help` displays all options with clear descriptions

**Independent Test**: Run `brains init --help` and verify --global and --force flags are documented

### Implementation for User Story 2

- [X] T015 [US2] Update Usage string for init command to describe full setup behavior in internal/cli/init.go
- [X] T016 [US2] Add Usage description for --force flag: "Overwrite existing files" in internal/cli/init.go
- [X] T017 [US2] Verify --global flag has existing description preserved in internal/cli/init.go

**Checkpoint**: Help text is complete and accurate

---

## Phase 5: User Story 3 - Force Overwrite Existing Files (Priority: P3)

**Goal**: `brains init --force` overwrites existing command and template files

**Independent Test**: Create directory with modified brains.feature.md, run `brains init --force`, verify file is replaced with embedded version

### Implementation for User Story 3

- [X] T018 [US3] Modify copyEmbeddedFiles to accept force parameter and overwrite when true in internal/cli/init.go
- [X] T019 [US3] Implement verbose output for overwritten files: "Overwrote filename" in internal/cli/init.go
- [X] T020 [US3] Update summary to include overwritten count: "X copied, Y skipped, Z overwritten" in internal/cli/init.go
- [X] T021 [US3] Pass --force flag value to copyEmbeddedFiles calls in init action in internal/cli/init.go

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, edge cases, and validation

- [X] T022 Implement error handling for non-writable directory (return clear error message) in internal/cli/init.go
- [X] T023 [P] Implement validation for empty embedded filesystem (return error suggesting reinstall) in internal/cli/init.go
- [X] T024 [P] Implement continue-on-error for individual file copy failures in internal/cli/init.go
- [X] T025 Create unit tests for init command in internal/cli/init_test.go
- [X] T026 [P] Test fresh directory initialization in internal/cli/init_test.go
- [X] T027 [P] Test skip existing files behavior in internal/cli/init_test.go
- [X] T028 [P] Test --force overwrite behavior in internal/cli/init_test.go
- [X] T029 Run quickstart.md validation: build binary and test in temporary directory
- [X] T030 Verify all 15 command files are embedded correctly by running go build and inspecting

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories should proceed sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Core functionality
- **User Story 2 (P2)**: Can start after US1 - Uses flag definitions from US1
- **User Story 3 (P3)**: Can start after US1 - Extends copy logic from US1

### Within Each User Story

- Directory creation before file copying
- Core implementation before summary output
- Story complete before moving to next priority

### Parallel Opportunities

- Setup: T001 and T002 can run in parallel (different embed directives)
- Polish: T023-T024 can run in parallel, T026-T028 can run in parallel (different test files)
- User stories are sequential due to shared file (internal/cli/init.go)

---

## Parallel Example: Phase 1 Setup

```bash
# Launch both embed declarations together:
Task: "Add EmbeddedCommands variable in embed.go"
Task: "Add EmbeddedTemplates variable in embed.go"
```

## Parallel Example: Phase 6 Tests

```bash
# Launch all test cases together:
Task: "Test fresh directory initialization in internal/cli/init_test.go"
Task: "Test skip existing files behavior in internal/cli/init_test.go"
Task: "Test --force overwrite behavior in internal/cli/init_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (embed directives)
2. Complete Phase 2: Foundational (helper function, --force flag)
3. Complete Phase 3: User Story 1 (core init functionality)
4. **STOP and VALIDATE**: Test `brains init` in fresh directory
5. Deploy/demo if ready - users can now initialize ZombieKit!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP Ready** (users can init)
3. Add User Story 2 → Test independently → Help is complete
4. Add User Story 3 → Test independently → Force overwrite works
5. Add Polish → All edge cases handled, tests pass

### Recommended Execution Order

Since all user stories modify `internal/cli/init.go`, execute sequentially:

1. T001-T002 (parallel) → Setup complete
2. T003-T005 (sequential) → Foundational complete
3. T006-T014 (sequential) → US1 complete → **MVP!**
4. T015-T017 (sequential) → US2 complete
5. T018-T021 (sequential) → US3 complete
6. T022-T030 (mostly parallel) → Polish complete

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- All user stories modify internal/cli/init.go, limiting parallelism
- MVP is achievable after completing Phase 3 (User Story 1)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
