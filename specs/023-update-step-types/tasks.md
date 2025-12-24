# Tasks: Update Step Types

**Input**: Design documents from `/specs/023-update-step-types/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `internal/` for packages, `templates/` for step templates
- All paths relative to repository root

---

## Phase 1: Setup (Template Cleanup)

**Purpose**: Remove legacy templates and rename implement to eat

- [X] T001 Delete legacy template templates/steps/init.md
- [X] T002 [P] Delete legacy template templates/steps/specify.md
- [X] T003 [P] Rename templates/steps/implement.md to templates/steps/eat.md (update name in frontmatter)

---

## Phase 2: Foundational (Prerequisite System)

**Purpose**: Core prerequisite checking infrastructure that MUST be complete before step handlers can enforce requirements

**CRITICAL**: User story implementation depends on this phase

- [X] T004 Add StepPrerequisite type to internal/step/types.go with RequiredArtifact, RequiredStatus, Hint, BlockingStep fields
- [X] T005 Add stepPrerequisites map to internal/step/service.go defining: plan→spec.md, tasks→plan.md, eat→tasks.md
- [X] T006 Add checkPrerequisite method to Service in internal/step/service.go that validates artifact exists and status matches
- [X] T007 Add prerequisite check call at start of Execute() method in internal/step/service.go
- [X] T008 Update error hint in internal/step/loader.go from legacy steps to new step names

**Checkpoint**: Prerequisite system ready - step handler implementation can begin

---

## Phase 3: User Story 1 - Start New Feature Workflow (Priority: P1)

**Goal**: Ensure feature step creates initiative with spec.md template and provides specify workflow guidance

**Independent Test**: Run `/brains.feature "test-feature"` and verify initiative folder created with templates

### Implementation for User Story 1

- [X] T009 [US1] Verify feature step in internal/step/service.go correctly handles "feature" case (already exists, confirm behavior)
- [X] T010 [US1] Verify templates/steps/feature.md contains complete specify workflow directive (research-create-audit-highlight phases)
- [X] T011 [US1] Add test case in internal/step/service_test.go for feature step creating initiative with templates

**Checkpoint**: Feature step fully functional and testable independently

---

## Phase 4: User Story 2 - Create Bug Investigation (Priority: P2)

**Goal**: Bug step creates bug-type initiative with bug-specific directive

**Independent Test**: Run `/brains.bug "test-bug"` and verify bug-type initiative created

### Implementation for User Story 2

- [X] T012 [P] [US2] Create templates/steps/bug.md with bug investigation directive (reproduction, root cause, fix spec)
- [X] T013 [US2] Add executeBugStep method to internal/step/service.go that sets Type="bug" and delegates to executeFeatureStep
- [X] T014 [US2] Add "bug" case to switch in Execute() method in internal/step/service.go
- [X] T015 [US2] Add test case in internal/step/service_test.go for bug step creating bug-type initiative

**Checkpoint**: Bug step fully functional and testable independently

---

## Phase 5: User Story 3 - Create Refactor Specification (Priority: P2)

**Goal**: Refactor step creates refactor-type initiative with behavior preservation focus

**Independent Test**: Run `/brains.refactor "test-refactor"` and verify refactor-type initiative created

### Implementation for User Story 3

- [X] T016 [P] [US3] Create templates/steps/refactor.md with refactor directive (before/after, behavior preservation)
- [X] T017 [US3] Add executeRefactorStep method to internal/step/service.go that sets Type="refactor" and delegates to executeFeatureStep
- [X] T018 [US3] Add "refactor" case to switch in Execute() method in internal/step/service.go
- [X] T019 [US3] Add test case in internal/step/service_test.go for refactor step creating refactor-type initiative

**Checkpoint**: Refactor step fully functional and testable independently

---

## Phase 6: User Story 4 - Execute Planning Phase (Priority: P1)

**Goal**: Plan step enforces spec approval prerequisite and provides planning guidance

**Independent Test**: Run `/brains.plan` with approved spec and verify plan step executes; run without approved spec and verify hard block

### Implementation for User Story 4

- [X] T020 [US4] Verify plan step prerequisite (spec.md approved) is enforced via T005-T007 foundational work
- [X] T021 [US4] Verify templates/steps/plan.md exists with planning directive
- [X] T022 [US4] Add test case in internal/step/service_test.go for plan step blocking when spec not approved
- [X] T023 [US4] Add test case in internal/step/service_test.go for plan step allowing when spec approved

**Checkpoint**: Plan step prerequisite enforcement verified

---

## Phase 7: User Story 5 - Generate Task Breakdown (Priority: P1)

**Goal**: Tasks step enforces plan approval prerequisite and provides task generation guidance

**Independent Test**: Run `/brains.tasks` with approved plan and verify tasks step executes; run without approved plan and verify hard block

### Implementation for User Story 5

- [X] T024 [US5] Verify tasks step prerequisite (plan.md approved) is enforced via T005-T007 foundational work
- [X] T025 [US5] Verify templates/steps/tasks.md exists with task generation directive
- [X] T026 [US5] Add test case in internal/step/service_test.go for tasks step blocking when plan not approved
- [X] T027 [US5] Add test case in internal/step/service_test.go for tasks step allowing when plan approved

**Checkpoint**: Tasks step prerequisite enforcement verified

---

## Phase 8: User Story 6 - Execute Implementation (Priority: P1)

**Goal**: Eat step (renamed from implement) enforces tasks.md prerequisite and guides task execution

**Independent Test**: Run `/brains.eat` with tasks.md and verify step executes; run without tasks.md and verify hard block

### Implementation for User Story 6

- [X] T028 [US6] Verify eat step prerequisite (tasks.md exists) is enforced via T005-T007 foundational work
- [X] T029 [US6] Verify templates/steps/eat.md exists with implementation guidance (from T003 rename)
- [X] T030 [US6] Add test case in internal/step/service_test.go for eat step blocking when tasks.md missing
- [X] T031 [US6] Add test case in internal/step/service_test.go for eat step allowing when tasks.md exists

**Checkpoint**: Eat step prerequisite enforcement verified

---

## Phase 9: User Story 7 - Run Audit Check (Priority: P2)

**Goal**: Audit step verifies cross-artifact alignment

**Independent Test**: Run `/brains.audit` on initiative with artifacts and verify audit runs

### Implementation for User Story 7

- [X] T032 [US7] Verify templates/steps/audit.md exists with audit directive
- [X] T033 [US7] Verify audit step works with existing implementation in internal/step/service.go

**Checkpoint**: Audit step functional

---

## Phase 10: User Story 8 - Request Clarification (Priority: P2)

**Goal**: Clarify step identifies underspecified areas

**Independent Test**: Run `/brains.clarify` on initiative and verify clarification workflow

### Implementation for User Story 8

- [X] T034 [US8] Verify templates/steps/clarify.md exists with clarification directive
- [X] T035 [US8] Verify clarify step works with existing implementation in internal/step/service.go

**Checkpoint**: Clarify step functional

---

## Phase 11: User Story 9 - Complete Initiative (Priority: P1)

**Goal**: Complete step marks initiative finished and clears active state

**Independent Test**: Run `/brains.complete` on active initiative and verify status changed and active state cleared

### Implementation for User Story 9

- [X] T036 [US9] Verify complete step in internal/step/service.go correctly handles "complete" case (already exists)
- [X] T037 [US9] Verify templates/steps/complete.md exists with completion directive
- [X] T038 [US9] Add test case in internal/step/service_test.go for complete step clearing active state

**Checkpoint**: Complete step fully functional

---

## Phase 12: Legacy Step Removal Verification

**Purpose**: Ensure legacy steps return proper errors

- [X] T039 Add test case in internal/step/loader_test.go verifying "init" returns UNKNOWN_STEP error
- [X] T040 [P] Add test case in internal/step/loader_test.go verifying "specify" returns UNKNOWN_STEP error
- [X] T041 [P] Add test case in internal/step/loader_test.go verifying "implement" returns UNKNOWN_STEP error
- [X] T042 Remove "init" case from Execute() switch in internal/step/service.go if present

**Checkpoint**: Legacy steps properly rejected

---

## Phase 13: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup and validation

- [X] T043 Run go test ./internal/step/... to verify all tests pass
- [X] T044 Run go build ./... to verify no compilation errors
- [X] T045 Verify ListSteps() in internal/step/loader.go returns all nine new steps
- [ ] T046 Manual end-to-end test: feature → plan → tasks → eat → complete workflow

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS prerequisite-dependent stories
- **User Story 1 (Phase 3)**: Depends on Setup only (feature step already exists)
- **User Stories 2-3 (Phases 4-5)**: Depend on Foundational completion (need new step handlers)
- **User Stories 4-6 (Phases 6-8)**: Depend on Foundational completion (need prerequisite checking)
- **User Stories 7-9 (Phases 9-11)**: Depend on Setup only (steps already exist)
- **Legacy Removal (Phase 12)**: Depends on Setup completion
- **Polish (Phase 13)**: Depends on all stories being complete

### User Story Dependencies

| Story | Depends On | Can Parallel With |
|-------|------------|-------------------|
| US1 (Feature) | Setup | US7, US8, US9 |
| US2 (Bug) | Foundational | US3 |
| US3 (Refactor) | Foundational | US2 |
| US4 (Plan) | Foundational | US5, US6 |
| US5 (Tasks) | Foundational | US4, US6 |
| US6 (Eat) | Foundational | US4, US5 |
| US7 (Audit) | Setup | US1, US8, US9 |
| US8 (Clarify) | Setup | US1, US7, US9 |
| US9 (Complete) | Setup | US1, US7, US8 |

### Parallel Opportunities

**Phase 1 (Setup)**:
- T001, T002, T003 can all run in parallel (different files)

**Phase 2 (Foundational)**:
- T004 must complete before T005-T007 (type definition needed)
- T005-T008 can run in parallel after T004

**After Foundational**:
- US2 and US3 can run in parallel (bug.md and refactor.md are independent)
- US4, US5, US6 prerequisite tests can run in parallel
- US7, US8, US9 can run in parallel with each other and with US1

**Phase 12 (Legacy Removal)**:
- T039, T040, T041 can run in parallel (different test cases)

---

## Parallel Example: Bug and Refactor Steps

```bash
# Launch new step template creation in parallel:
Task: "Create templates/steps/bug.md with bug investigation directive"
Task: "Create templates/steps/refactor.md with refactor directive"

# After templates created, launch handler implementation in parallel:
Task: "Add executeBugStep method to internal/step/service.go"
Task: "Add executeRefactorStep method to internal/step/service.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (remove legacy templates)
2. Complete Phase 3: User Story 1 (verify feature step)
3. **STOP and VALIDATE**: Test feature step independently
4. This confirms the core workflow entry point works

### Incremental Delivery

1. Setup + US1 → Feature workflow works
2. Add Foundational → Prerequisite system in place
3. Add US2 + US3 → Bug and refactor workflows work
4. Add US4 + US5 + US6 → Prerequisite enforcement verified
5. Verify US7 + US8 + US9 → Audit, clarify, complete work
6. Legacy removal → Old steps properly rejected
7. Polish → Full workflow validated

### Suggested Order (Single Developer)

1. T001-T003 (Setup - parallel)
2. T004-T008 (Foundational - sequential with some parallel)
3. T009-T011 (US1)
4. T012-T015 (US2)
5. T016-T019 (US3)
6. T020-T023 (US4)
7. T024-T027 (US5)
8. T028-T031 (US6)
9. T032-T038 (US7, US8, US9 - can parallel)
10. T039-T042 (Legacy removal)
11. T043-T046 (Polish)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Constitution requires tests before implementation, but this feature modifies existing tested code
- Commit after each task or logical group
- Feature step already exists and works - mostly verification tasks for US1
