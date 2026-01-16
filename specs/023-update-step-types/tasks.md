# Tasks: Update Step Types & MCP Tool Interface

**Input**: Design documents from `/specs/023-update-step-types/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project type**: Single Go CLI application
- **Source**: `internal/` for implementation, `templates/` for step definitions
- **Tests**: `*_test.go` files alongside source

---

## Phase 1: Setup (Template Cleanup)

**Purpose**: Remove legacy templates and rename implement to eat

- [x] T001 Delete legacy template templates/steps/init.md
- [x] T002 [P] Delete legacy template templates/steps/specify.md
- [x] T003 [P] Rename templates/steps/implement.md to templates/steps/eat.md (update name in frontmatter)
- [x] T004 [P] Delete templates/steps/complete.md (complete is now an initiative action, not a step)

---

## Phase 2: Foundational (Initiative Tool Infrastructure)

**Purpose**: Create the new `initiative` MCP tool to handle lifecycle operations (create, status, complete, list)

**Critical**: User story implementation depends on this phase - the initiative tool must exist before steps can require active initiatives

### Initiative Service Extensions

- [x] T005 Add Create(type, name, dir) method to internal/initiative/service.go that creates initiative folder, cycle folder, git branch, copies templates (already exists)
- [x] T006 Add CreateCycle(initiativeID) method to internal/initiative/service.go for creating new cycles within an existing initiative (already exists in cycle.go)
- [x] T007 Add Complete(dir) method to internal/initiative/service.go that marks initiative complete and clears active state (already exists)
- [x] T008 Add List(dir) method to internal/initiative/service.go that returns all initiatives with status (already exists)
- [x] T009 Add Status(dir) method to internal/initiative/service.go that returns active initiative info, current step, available docs

### Initiative MCP Tool

- [x] T010 Create internal/mcp/tools/initiative/types.go with InitiativeRequest, InitiativeCreateResponse, InitiativeStatusResponse, InitiativeCompleteResponse, InitiativeListResponse per data-model.md
- [x] T011 Create internal/mcp/tools/initiative/tool.go with Tool struct and Definition() returning InputSchema per contracts/initiative-tool.md
- [x] T012 Implement Handle() method in internal/mcp/tools/initiative/tool.go dispatching to create/status/complete/list based on action param
- [ ] T013 Create internal/mcp/tools/initiative/tool_test.go with tests for all four actions

### Step Tool Simplification

- [x] T014 Remove type, name, description, new_initiative, phase parameters from InputSchema in internal/mcp/tools/step/tool.go per contracts/step-tool.md
- [x] T015 Update internal/step/types.go ExecuteOptions to remove Type, Name, Description, NewInitiative fields
- [x] T016 Add active initiative check to Execute() in internal/step/service.go - return NO_ACTIVE_INITIATIVE error if no active initiative

### MCP Server Registration

- [x] T017 Register initiative tool in internal/mcp/server.go alongside step tool

**Checkpoint**: Initiative tool ready - step tool simplified - MCP server registers both tools

---

## Phase 3: User Story 1 - Start New Feature Workflow (Priority: P1)

**Goal**: User creates feature initiative via `initiative create`, then runs `step feature` for specification guidance

**Independent Test**: Call `initiative(action="create", type="feature", name="user-auth")` then `step(step="feature")` and verify folder structure created with templates, step provides specify workflow directive

### Implementation for User Story 1

- [ ] T018 [US1] Add test in internal/mcp/tools/initiative/tool_test.go for initiative create action creating folder structure with spec.md, research.md templates
- [ ] T019 [US1] Verify templates/steps/feature.md contains complete specify workflow directive (research-create-audit-highlight phases)
- [ ] T020 [US1] Add integration test verifying initiative create → step feature flow works end-to-end

**Checkpoint**: Feature workflow entry point fully functional

---

## Phase 4: User Story 2 - Create Bug Investigation (Priority: P2)

**Goal**: User creates bug initiative via `initiative create type=bug`, then runs `step bug` for investigation guidance

**Independent Test**: Call `initiative(action="create", type="bug", name="payment-timeout")` then `step(step="bug")` and verify bug-type initiative created with appropriate templates

### Implementation for User Story 2

- [ ] T021 [P] [US2] Create templates/steps/bug.md with bug investigation directive (reproduction, root cause, fix spec)
- [ ] T022 [US2] Add test in internal/mcp/tools/initiative/tool_test.go for initiative create action with type=bug
- [ ] T023 [US2] Add test in internal/step/service_test.go for bug step providing investigation guidance

**Checkpoint**: Bug workflow fully functional

---

## Phase 5: User Story 3 - Create Refactor Specification (Priority: P2)

**Goal**: User creates refactor initiative via `initiative create type=refactor`, then runs `step refactor` for refactoring guidance

**Independent Test**: Call `initiative(action="create", type="refactor", name="extract-auth")` then `step(step="refactor")` and verify refactor-type initiative created

### Implementation for User Story 3

- [ ] T024 [P] [US3] Create templates/steps/refactor.md with refactor directive (before/after, behavior preservation)
- [ ] T025 [US3] Add test in internal/mcp/tools/initiative/tool_test.go for initiative create action with type=refactor
- [ ] T026 [US3] Add test in internal/step/service_test.go for refactor step providing behavior preservation guidance

**Checkpoint**: Refactor workflow fully functional

---

## Phase 6: User Story 4 - Execute Planning Phase (Priority: P1)

**Goal**: Plan step enforces spec approval prerequisite and provides planning guidance

**Independent Test**: Run `step plan` with approved spec and verify step executes; run without approved spec and verify hard block with guidance

### Implementation for User Story 4

- [ ] T027 [US4] Add stepPrerequisites map to internal/step/service.go defining plan→spec.md (approved status required)
- [ ] T028 [US4] Implement YAML frontmatter status parser in internal/step/prereq.go that reads artifact files and extracts `status` field
- [ ] T029 [US4] Add checkPrerequisite method to Service in internal/step/service.go that uses prereq parser to validate artifact exists and status matches
- [ ] T030 [US4] Verify templates/steps/plan.md exists with planning directive
- [ ] T031 [US4] Add test case in internal/step/service_test.go for plan step blocking when spec not approved
- [ ] T032 [US4] Add test case in internal/step/service_test.go for plan step allowing when spec approved

**Checkpoint**: Plan step prerequisite enforcement verified

---

## Phase 7: User Story 5 - Generate Task Breakdown (Priority: P1)

**Goal**: Tasks step enforces plan approval prerequisite and provides task generation guidance

**Independent Test**: Run `step tasks` with approved plan and verify step executes; run without approved plan and verify hard block

### Implementation for User Story 5

- [ ] T033 [US5] Add tasks→plan.md prerequisite to stepPrerequisites map in internal/step/service.go
- [ ] T034 [US5] Verify templates/steps/tasks.md exists with task generation directive
- [ ] T035 [US5] Add test case in internal/step/service_test.go for tasks step blocking when plan not approved
- [ ] T036 [US5] Add test case in internal/step/service_test.go for tasks step allowing when plan approved

**Checkpoint**: Tasks step prerequisite enforcement verified

---

## Phase 8: User Story 6 - Execute Implementation (Priority: P1)

**Goal**: Eat step enforces tasks.md prerequisite and guides task execution

**Independent Test**: Run `step eat` with tasks.md and verify step executes; run without tasks.md and verify hard block

### Implementation for User Story 6

- [ ] T037 [US6] Add eat→tasks.md prerequisite to stepPrerequisites map in internal/step/service.go
- [ ] T038 [US6] Verify templates/steps/eat.md exists with implementation guidance (from T003 rename)
- [ ] T039 [US6] Implement next task detection in eat step handler: parse tasks.md for first unchecked `- [ ]` item
- [ ] T040 [US6] Add test case in internal/step/service_test.go for eat step blocking when tasks.md missing
- [ ] T041 [US6] Add test case in internal/step/service_test.go for eat step allowing when tasks.md exists
- [ ] T042 [US6] Add test case for eat step returning NextTask info with task ID and description
- [ ] T043 [US6] Add test case for eat step indicating "all tasks complete" when all checkboxes are checked

**Checkpoint**: Eat step prerequisite enforcement verified

---

## Phase 9: User Story 7 - Run Audit Check (Priority: P2)

**Goal**: Audit step verifies cross-artifact alignment

**Independent Test**: Run `step audit` on initiative with artifacts and verify audit runs

### Implementation for User Story 7

- [ ] T044 [US7] Verify templates/steps/audit.md exists with audit directive
- [ ] T045 [US7] Add test in internal/step/service_test.go for audit step working with active initiative

**Checkpoint**: Audit step functional

---

## Phase 10: User Story 8 - Request Clarification (Priority: P2)

**Goal**: Clarify step identifies underspecified areas

**Independent Test**: Run `step clarify` on initiative and verify clarification workflow

### Implementation for User Story 8

- [ ] T046 [US8] Verify templates/steps/clarify.md exists with clarification directive
- [ ] T047 [US8] Add test in internal/step/service_test.go for clarify step working with active initiative
- [ ] T048 [US8] Add test verifying clarify step appends Q/A to Clarifications section with session date header

**Checkpoint**: Clarify step functional

---

## Phase 11: User Story 9 - Complete Initiative (Priority: P1)

**Goal**: User calls `initiative complete` to mark initiative finished and clear active state

**Independent Test**: Call `initiative(action="complete")` on active initiative and verify status changed and active state cleared

### Implementation for User Story 9

- [ ] T049 [US9] Add test in internal/mcp/tools/initiative/tool_test.go for complete action updating INITIATIVE.md status
- [ ] T050 [US9] Add test in internal/mcp/tools/initiative/tool_test.go for complete action clearing active state
- [ ] T051 [US9] Verify Complete() method handles case where no active initiative exists (returns NO_ACTIVE_INITIATIVE error per contracts/initiative-tool.md)

**Checkpoint**: Initiative completion fully functional

---

## Phase 12: User Story 10 - Check Initiative Status (Priority: P2)

**Goal**: User calls `initiative status` to see current initiative state

**Independent Test**: Call `initiative(action="status")` and verify it returns current step, available docs, suggested next step

### Implementation for User Story 10

- [ ] T052 [US10] Add test in internal/mcp/tools/initiative/tool_test.go for status action returning active initiative info per contracts/initiative-tool.md
- [ ] T053 [US10] Add test in internal/mcp/tools/initiative/tool_test.go for status action when no active initiative (returns active=false with suggestion)
- [ ] T054 [US10] Add test in internal/mcp/tools/initiative/tool_test.go for list action returning all initiatives from history folder
- [ ] T055 [US10] Verify Status() method includes suggested_next based on available artifacts

**Checkpoint**: Initiative status and list fully functional

---

## Phase 13: Edge Cases & Error Handling

**Purpose**: Ensure proper error handling for invalid states per contracts

- [ ] T056 Add test for step tool returning NO_ACTIVE_INITIATIVE when no initiative is active
- [ ] T057 Add test for initiative create failing with INITIATIVE_ALREADY_ACTIVE when initiative already active (per contracts/initiative-tool.md)
- [ ] T058 Add test for initiative create failing with MISSING_REQUIRED_PARAM when type or name missing
- [ ] T059 Add test for initiative tool returning INVALID_ACTION for unknown actions
- [ ] T060 Add test for step tool returning UNKNOWN_STEP for invalid step names

---

## Phase 14: Legacy Step Removal Verification

**Purpose**: Ensure legacy steps return proper errors per contracts/step-tool.md

- [ ] T061 Add test in internal/step/loader_test.go verifying "init" returns UNKNOWN_STEP error
- [ ] T062 [P] Add test in internal/step/loader_test.go verifying "specify" returns UNKNOWN_STEP error
- [ ] T063 [P] Add test in internal/step/loader_test.go verifying "implement" returns UNKNOWN_STEP error
- [ ] T064 [P] Add test in internal/step/loader_test.go verifying "complete" returns UNKNOWN_STEP error (now initiative action)
- [ ] T065 Remove legacy step cases from Execute() switch in internal/step/service.go if present
- [ ] T066 Remove creation logic from internal/step/feature.go (now in initiative service)

---

## Phase 15: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup and validation

- [ ] T067 Run go test ./internal/step/... to verify all step tests pass
- [ ] T068 [P] Run go test ./internal/initiative/... to verify all initiative tests pass
- [ ] T069 [P] Run go test ./internal/mcp/... to verify all MCP tool tests pass
- [ ] T070 Run go build ./... to verify no compilation errors
- [ ] T071 Verify ListSteps() in internal/step/loader.go returns exactly 8 steps (feature, bug, refactor, plan, tasks, eat, audit, clarify)
- [ ] T072 Manual end-to-end test per quickstart.md: initiative create → step feature → step plan → step tasks → step eat → initiative complete

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-12)**: All depend on Foundational completion
- **Edge Cases (Phase 13)**: Depends on User Stories completion
- **Legacy Removal (Phase 14)**: Depends on Setup completion
- **Polish (Phase 15)**: Depends on all phases being complete

### User Story Dependencies

| Story | Depends On | Can Parallel With |
|-------|------------|-------------------|
| US1 (Feature) | Foundational | None (MVP) |
| US2 (Bug) | Foundational, US1 verified | US3 |
| US3 (Refactor) | Foundational, US1 verified | US2 |
| US4 (Plan) | Foundational | US7, US8 |
| US5 (Tasks) | US4 | - |
| US6 (Eat) | US5 | - |
| US7 (Audit) | Foundational | US4, US8 |
| US8 (Clarify) | Foundational | US4, US7 |
| US9 (Complete) | Foundational | US10 |
| US10 (Status) | Foundational | US9 |

### Parallel Opportunities

**Phase 1 (Setup)**:
- T001, T002, T003, T004 can all run in parallel (different files)

**Phase 2 (Foundational)**:
- T005-T009 (initiative service methods) are sequential (methods build on types)
- T010-T012 (initiative tool) depend on T005-T009
- T014-T016 (step simplification) can run in parallel with T005-T009

**After Foundational**:
- US2 and US3 can run in parallel (bug.md and refactor.md are independent)
- US4, US5, US6 must be sequential (prerequisite chain)
- US7 and US8 can run in parallel
- US9 and US10 can run in parallel

**Phase 14 (Legacy Removal)**:
- T056, T057, T058, T059 can run in parallel (different test cases)

---

## Parallel Example: Initiative Tool Creation

```bash
# Launch step simplification in parallel with initiative service work:
Task: "Remove type, name, description params from InputSchema in internal/mcp/tools/step/tool.go"
Task: "Add Create method to internal/initiative/service.go"

# After service methods, launch MCP tool implementation:
Task: "Create internal/mcp/tools/initiative/types.go"
Task: "Create internal/mcp/tools/initiative/tool.go"
```

---

## Implementation Strategy

### MVP First (Phase 1-3 Only)

1. Complete Phase 1: Setup (remove legacy templates)
2. Complete Phase 2: Foundational (initiative tool infrastructure)
3. Complete Phase 3: User Story 1 (feature workflow)
4. **STOP and VALIDATE**: Test `initiative create` → `step feature` flow
5. This confirms the new tool architecture works

### Incremental Delivery

1. Setup + Foundational → Initiative tool created, step tool simplified
2. Add US1 → Feature workflow works with new architecture (MVP!)
3. Add US4 + US5 + US6 → Full lifecycle: plan → tasks → eat
4. Add US9 → Can complete initiatives
5. Add US2 + US3 → Bug and refactor workflows work
6. Add US7 + US8 + US10 → Audit, clarify, and status
7. Edge cases + Legacy removal → Error handling verified
8. Polish → Full workflow validated per quickstart.md

### Critical Path

```
Setup → Foundational → US1 → US4 → US5 → US6 → US9 → Polish
```

This path delivers the minimum viable feature workflow. US2, US3, US7, US8, US10 add breadth but aren't blocking.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Major architectural change: initiative lifecycle separated from step execution
- The `step` tool now requires an active initiative (created via `initiative create`)
- Complete is now an initiative action, not a step
- Commit after each task or logical group
- All response/error schemas per contracts/initiative-tool.md and contracts/step-tool.md
