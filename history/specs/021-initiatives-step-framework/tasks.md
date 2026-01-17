# Tasks: Initiatives Step Framework

**Input**: Design documents from `/specs/021-initiatives-step-framework/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included as this is a new package with clear interfaces.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `internal/` for packages, following existing zombiekit structure
- Tests colocated with source files (Go convention: `*_test.go`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create new packages and directory structure

- [x] T001 Create `internal/initiative/` package directory
- [x] T002 Create `internal/step/` package directory
- [x] T003 Create `internal/mcp/tools/step/` package directory

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and interfaces that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Types and Interfaces

- [x] T004 [P] Define InitiativeType and InitiativeStatus enums in `internal/initiative/types.go`
- [x] T005 [P] Define Initiative struct in `internal/initiative/types.go`
- [x] T006 [P] Define InitiativeState struct in `internal/initiative/types.go`
- [x] T007 [P] Define StepSource enum in `internal/step/types.go`
- [x] T008 [P] Define Step struct with frontmatter fields in `internal/step/types.go`
- [x] T009 [P] Define StepResponse struct in `internal/step/types.go`
- [x] T010 [P] Define StepFrontmatter struct for YAML parsing in `internal/step/types.go`

### State Management

- [x] T011 Implement StateManager interface in `internal/initiative/state.go` with Load/Save/Lock methods
- [x] T012 Implement file-based state storage with flock locking in `internal/initiative/state.go`
- [x] T013 Write tests for StateManager in `internal/initiative/state_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Execute a Step in an Initiative (Priority: P1) 🎯 MVP

**Goal**: Implement the core MCP tool that returns directive, history folder, files to read, and composed prompt for any step.

**Independent Test**: Call MCP endpoint with step="specify" and dir="/path/to/project", verify all four outputs are returned correctly.

### Tests for User Story 1

- [x] T014 [P] [US1] Write unit tests for step loader in `internal/step/loader_test.go`
- [x] T015 [P] [US1] Write unit tests for step service in `internal/step/service_test.go`
- [x] T016 [P] [US1] Write unit tests for MCP step tool in `internal/mcp/tools/step/tool_test.go`

### Implementation for User Story 1

- [x] T017 [P] [US1] Implement step frontmatter parsing using adrg/frontmatter in `internal/step/frontmatter.go`
- [x] T018 [US1] Implement step loader with resolution order (local→global→embedded) in `internal/step/loader.go`
- [x] T019 [US1] Create embedded default step definitions (specify, plan, tasks, implement, audit, clarify, complete) in `templates/steps/`
- [x] T020 [US1] Implement StepService interface in `internal/step/service.go` with GetStep and Execute methods
- [x] T021 [US1] Integrate StepService with profile.Service.Compose() in `internal/step/service.go`
- [x] T022 [US1] Implement file pattern resolution for step.Files globs in `internal/step/service.go`
- [x] T023 [US1] Implement MCP step tool definition and handler in `internal/mcp/tools/step/tool.go`
- [x] T024 [US1] Register step tool in MCP server in `internal/mcp/server.go`
- [x] T025 [US1] Add error handling for NOT_INITIALIZED, UNKNOWN_STEP, NO_ACTIVE_INITIATIVE in `internal/mcp/tools/step/tool.go`

**Checkpoint**: Step execution works for existing initiatives with default steps

---

## Phase 4: User Story 2 - Start a New Initiative (Priority: P2)

**Goal**: Enable creating new initiatives with proper folder structure in `./history/` and state tracking.

**Independent Test**: Call MCP endpoint with step="init", type="feature", name="user-auth", verify folder created and state updated.

### Tests for User Story 2

- [x] T026 [P] [US2] Write unit tests for initiative service in `internal/initiative/service_test.go`
- [x] T027 [P] [US2] Write integration test for init step in `internal/mcp/tools/step/tool_test.go`

### Implementation for User Story 2

- [x] T028 [US2] Implement InitiativeService interface in `internal/initiative/service.go`
- [x] T029 [US2] Implement Create() method with hex-timestamp folder naming in `internal/initiative/service.go`
- [x] T030 [US2] Implement INITIATIVE.md template generation in `internal/initiative/service.go`
- [x] T031 [US2] Implement List() method to enumerate initiatives from history/ in `internal/initiative/service.go`
- [x] T032 [US2] Implement GetActive() and SetActive() using StateManager in `internal/initiative/service.go`
- [x] T033 [US2] Implement Complete() method to mark initiative as completed in `internal/initiative/service.go`
- [x] T034 [US2] Add special handling for "init" step in MCP tool in `internal/step/service.go`
- [x] T035 [US2] Add special handling for "complete" step in MCP tool in `internal/step/service.go`
- [x] T036 [US2] Add INVALID_TYPE and INITIATIVE_NOT_FOUND error handling in `internal/mcp/tools/step/tool.go`
- [x] T037 [US2] Implement initiative parameter override in MCP tool in `internal/mcp/tools/step/tool.go`

**Checkpoint**: Can create and manage initiatives via init/complete steps

---

## Phase 5: User Story 3 - Define Custom Steps (Priority: P3)

**Goal**: Allow projects to override default steps with custom configurations in `.brains/steps/`.

**Independent Test**: Create `.brains/steps/specify.md` with custom profiles/files, verify custom config is used instead of default.

### Tests for User Story 3

- [x] T038 [P] [US3] Write unit tests for custom step override resolution in `internal/step/loader_test.go`
- [x] T039 [P] [US3] Write integration test for custom step loading in `internal/step/service_test.go`

### Implementation for User Story 3

- [x] T040 [US3] Implement local step discovery from `.brains/steps/` in `internal/step/loader.go`
- [x] T041 [US3] Implement global step discovery from `~/.brains/steps/` in `internal/step/loader.go`
- [x] T042 [US3] Implement step precedence: local > global > embedded in `internal/step/loader.go`
- [x] T043 [US3] Add step source tracking (SourceLocal, SourceGlobal, SourceEmbedded) in `internal/step/loader.go`
- [x] T044 [US3] Update StepService to use loader with override resolution in `internal/step/service.go`

**Checkpoint**: Custom steps override defaults correctly

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T045 [P] Add config.IsToolEnabled("step") check for step tool registration in `internal/mcp/server.go`
- [x] T046 [P] Add "step" to default enabled tools in `internal/config/tools.go`
- [x] T047 [P] Update embedded templates with step definitions in `templates/steps/` (embed.go)
- [x] T048 Validate all error messages include actionable suggestions in `internal/mcp/tools/step/tool.go`
- [x] T049 Run quickstart.md validation - implementation matches documentation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - US1 can start immediately after Foundational
  - US2 depends on US1 (uses same MCP tool infrastructure)
  - US3 can run in parallel with US2 (different files)
- **Polish (Final Phase)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Depends on US1 MCP tool infrastructure being complete
- **User Story 3 (P3)**: Can start after Foundational - enhances loader from US1 but modifies different methods

### Within Each User Story

- Tests SHOULD be written and FAIL before implementation
- Types before services
- Services before MCP tool handlers
- Core implementation before error handling

### Parallel Opportunities

- All Setup tasks can run in parallel
- All Foundational type definitions (T004-T010) can run in parallel
- Tests for each user story (T014-T016, T026-T027, T038-T039) can run in parallel
- US2 and US3 can be worked on in parallel by different developers after US1 core is done

---

## Parallel Example: Foundational Phase

```bash
# Launch all type definitions together:
Task: "Define InitiativeType and InitiativeStatus enums in internal/initiative/types.go"
Task: "Define Initiative struct in internal/initiative/types.go"
Task: "Define InitiativeState struct in internal/initiative/types.go"
Task: "Define StepSource enum in internal/step/types.go"
Task: "Define Step struct in internal/step/types.go"
Task: "Define StepResponse struct in internal/step/types.go"
Task: "Define StepFrontmatter struct in internal/step/types.go"
```

## Parallel Example: User Story 1 Tests

```bash
# Launch all tests for User Story 1 together:
Task: "Write unit tests for step loader in internal/step/loader_test.go"
Task: "Write unit tests for step service in internal/step/service_test.go"
Task: "Write unit tests for MCP step tool in internal/mcp/tools/step/tool_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (3 tasks)
2. Complete Phase 2: Foundational (10 tasks)
3. Complete Phase 3: User Story 1 (12 tasks)
4. **STOP and VALIDATE**: Test step execution with existing initiatives
5. Deploy/demo if ready - can execute steps with default definitions

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP: Step execution works**
3. Add User Story 2 → Test independently → **Can create new initiatives**
4. Add User Story 3 → Test independently → **Can customize steps**
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Developer A: User Story 1 (critical path)
3. Once US1 core is done:
   - Developer A: User Story 2
   - Developer B: User Story 3
4. Stories complete and integrate independently

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| Setup | 3 | Create package directories |
| Foundational | 10 | Types, interfaces, state management |
| US1 (P1) | 12 | Step execution MVP |
| US2 (P2) | 12 | Initiative creation |
| US3 (P3) | 7 | Custom step overrides |
| Polish | 5 | Config, templates, validation |
| **Total** | **49** | |

**MVP Scope**: Phases 1-3 (25 tasks) - Step execution with default steps

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Follow existing patterns from `internal/profile/` and `internal/mcp/tools/`
