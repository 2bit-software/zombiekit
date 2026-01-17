# Tasks: Initiative Feature Workflow (022)

**Input**: Design documents from `/specs/022-initiative-feature-workflow/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Unit tests are included as the quickstart.md references specific test files.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project type**: Go CLI with internal packages
- **Source**: `internal/` for packages, `cmd/brains/` for CLI entry
- **Templates**: `templates/` for embedded filesystem
- **Tests**: Same directory as implementation (`*_test.go`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and type definitions

- [X] T001 Add CycleType and CycleStatus enums to internal/initiative/types.go
- [X] T002 Add Cycle struct with ID, Type, Name, Path, Status, InitiativeID, Number, timestamps to internal/initiative/types.go
- [X] T003 Simplify InitiativeState struct to pointer-only (Initiative, Cycle paths, timestamps, CurrentStep - NO status fields) in internal/initiative/types.go
- [X] T004 [P] Add Phase struct to internal/step/types.go with Name, Description, Agents, Outputs, Parallel fields
- [X] T005 [P] Extend StepResponse with InitiativeFolder, CycleFolder, WorkflowPhases fields in internal/step/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Create internal/initiative/cycle.go with CreateCycle method skeleton
- [X] T007 Add generateCycleID function using hex timestamp format in internal/initiative/cycle.go
- [X] T008 Add getNextCycleNumber function to count existing cycles in internal/initiative/cycle.go
- [X] T009 [P] Create internal/step/git.go with GitService struct
- [X] T010 Add isGitAvailable and isGitRepository methods to GitService in internal/step/git.go
- [X] T011 Add branchExists, switchToBranch, createBranch methods to GitService in internal/step/git.go
- [X] T012 Add formatBranchName method with prefix mapping (feature→feat, bug→fix, refactor→ref) in internal/step/git.go
- [X] T013 Add EnsureBranch public method with graceful degradation in internal/step/git.go
- [X] T014 [P] Create templates/templates/research-template.md with frontmatter and section structure
- [X] T014a [P] Create templates/templates/initiative-template.md with YAML frontmatter (status, type, created, updated)
- [X] T015 [P] Create templates/steps/feature.md with step definition frontmatter and directive content

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Start a New Feature Initiative (Priority: P1) 🎯 MVP

**Goal**: Developer invokes "feature" step via MCP, system creates initiative structure, copies templates, returns directive for research-create-audit cycle

**Independent Test**: Call `mcp_zombiekit__step` with `step="feature"`, `name="user-auth"`, verify folder creation, template copying, state update, and directive contains workflow phases

### Tests for User Story 1

- [X] T016 [P] [US1] Create internal/initiative/cycle_test.go with TestCreateCycle_ValidInput test
- [X] T017 [P] [US1] Add TestCreateCycle_InvalidCycleType test in internal/initiative/cycle_test.go
- [X] T018 [P] [US1] Add TestGetNextCycleNumber test in internal/initiative/cycle_test.go

### Implementation for User Story 1

- [X] T019 [US1] Implement CreateCycle method body with directory creation and audit subfolder in internal/initiative/cycle.go
- [X] T020 [US1] Add CycleType.IsValid() validation method in internal/initiative/types.go
- [X] T021 [US1] Create internal/step/feature.go with executeFeatureStep method skeleton
- [X] T022 [US1] Add parameter validation for name requirement in internal/step/feature.go
- [X] T023 [US1] Add initiative type validation and default handling in internal/step/feature.go
- [X] T024 [US1] Implement new initiative + first cycle creation path in internal/step/feature.go
- [X] T024a [US1] Create writeInitiativeMetadata function to generate INITIATIVE.md with YAML frontmatter (status: active, type, created, updated) and body (name, ID, description, empty cycles table) in internal/step/feature.go
- [X] T024b [US1] Call writeInitiativeMetadata after initiative folder creation in internal/step/feature.go
- [X] T025 [US1] Add copyTemplatesToCycle method using copyEmbeddedFiles pattern in internal/step/feature.go
- [X] T025a [US1] Add resolveTemplatePath helper that checks .brains/templates/{name} first, falls back to embedded filesystem in internal/step/feature.go
- [X] T025b [P] [US1] Add TestResolveTemplatePath_LocalOverride test verifying local templates take precedence in internal/step/feature_test.go
- [X] T026 [US1] Add buildWorkflowPhases function returning research/create/audit/highlight phases in internal/step/feature.go
- [X] T027 [US1] Wire up state update after cycle creation (path only, no status) in internal/step/feature.go
- [X] T027a [US1] Add function to read initiative status from INITIATIVE.md frontmatter in internal/initiative/cycle.go
- [X] T028 [US1] Build StepResponse with all required fields in internal/step/feature.go
- [X] T029 [US1] Update internal/mcp/tools/step/tool.go input schema with name, type, description, new_initiative parameters
- [X] T030 [US1] Route "feature" step to executeFeatureStep in internal/step/service.go

**Checkpoint**: User Story 1 complete - new feature initiatives can be created with full directive

---

## Phase 4: User Story 2 - Research Phase Execution (Priority: P1)

**Goal**: Directive instructs spawning parallel research agents, specifies how findings are collated into research.md

**Independent Test**: With active initiative, verify directive includes research phase with agent spawning and collation instructions

### Implementation for User Story 2

- [X] T031 [US2] Add research phase content to feature step directive in templates/steps/feature.md
- [X] T032 [US2] Include parallel agent spawning instructions (research-codebase, research-domain) in templates/steps/feature.md
- [X] T033 [US2] Add research.md population guidance with sections in templates/steps/feature.md
- [X] T034 [US2] Add success criteria checklist for research phase in templates/steps/feature.md

**Checkpoint**: Research phase documented in directive

---

## Phase 5: User Story 3 - Specification Creation Phase (Priority: P1)

**Goal**: After research, single agent creates specification using template and research findings

**Independent Test**: Verify directive guides creation of spec.md following template structure

### Implementation for User Story 3

- [X] T035 [US3] Add create phase content to feature step directive in templates/steps/feature.md
- [X] T036 [US3] Include spec.md section requirements (user scenarios, requirements, success criteria) in templates/steps/feature.md
- [X] T037 [US3] Add validation rules (no implementation details, no placeholders) in templates/steps/feature.md
- [X] T038 [US3] Add success criteria checklist for create phase in templates/steps/feature.md

**Checkpoint**: Create phase documented in directive

---

## Phase 6: User Story 4 - Audit Phase Execution (Priority: P1)

**Goal**: Audit agents check for completeness, AI-readiness, quality with loop-back on critical/major issues

**Independent Test**: Verify directive instructs audit checks with severity classification and retry logic

### Implementation for User Story 4

- [X] T039 [US4] Add audit phase content to feature step directive in templates/steps/feature.md
- [X] T040 [US4] Include severity classification (CRITICAL, MAJOR, MINOR, INFO) in templates/steps/feature.md
- [X] T041 [US4] Add conditional transition logic with 3-loop limit in templates/steps/feature.md
- [X] T042 [US4] Add audit output format for audit/{date}.md in templates/steps/feature.md
- [X] T043 [US4] Add success criteria checklist for audit phase in templates/steps/feature.md

**Checkpoint**: Audit phase documented in directive with loop-back conditions

---

## Phase 7: User Story 5 - Highlight and User Approval (Priority: P2)

**Goal**: Present key decisions for user review, require approval before proceeding to planning

**Independent Test**: Verify directive includes highlight presentation format and approval gate

### Implementation for User Story 5

- [X] T044 [US5] Add highlight phase content to feature step directive in templates/steps/feature.md
- [X] T045 [US5] Include user approval gate presentation format in templates/steps/feature.md
- [X] T046 [US5] Add rejection handling with feedback loop-back in templates/steps/feature.md
- [X] T047 [US5] Add behavior rules section (max iterations, never skip phases, cite sources) in templates/steps/feature.md
- [X] T048 [US5] Add phase flow diagram to directive in templates/steps/feature.md

**Checkpoint**: User Story 5 complete - full workflow directive ready

---

## Phase 8: User Story 6 - Add New Cycle to Existing Initiative (Priority: P2)

**Goal**: Create new cycle folder within existing initiative without changing git branch

**Independent Test**: Create feature initiative, complete it, invoke refactor step, verify new cycle folder in same initiative

### Tests for User Story 6

- [X] T049 [P] [US6] Create internal/step/feature_test.go with TestExecuteFeatureStep_NewInitiative test
- [X] T050 [P] [US6] Add TestExecuteFeatureStep_AddCycleToExisting test in internal/step/feature_test.go
- [X] T051 [P] [US6] Add TestExecuteFeatureStep_NewInitiativeFlag test in internal/step/feature_test.go

### Implementation for User Story 6

- [X] T052 [US6] Add mapInitTypeToCycleType helper function in internal/step/feature.go
- [X] T053 [US6] Implement add-cycle-to-existing path when active initiative exists in internal/step/feature.go
- [X] T054 [US6] Update INITIATIVE.md with cycle list when adding new cycle in internal/initiative/cycle.go
- [X] T055 [US6] Add files_to_read population from previous cycles in internal/step/feature.go
- [X] T056 [US6] Ensure git branch is NOT changed when adding cycle to existing initiative in internal/step/feature.go

**Checkpoint**: Multi-cycle initiatives supported

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Testing, validation, and integration

- [X] T057 [P] Create internal/step/git_test.go with TestEnsureBranch_GitNotAvailable test
- [X] T058 [P] Add TestEnsureBranch_CreatesBranch test in internal/step/git_test.go
- [X] T059 [P] Add TestEnsureBranch_SwitchesExistingBranch test in internal/step/git_test.go
- [X] T060 [P] Add TestFormatBranchName tests for all initiative types in internal/step/git_test.go
- [X] T061 Verify performance: feature step completes in under 2 seconds (SC-001)
- [X] T062 Run go test ./internal/initiative/... -v and fix any failures
- [X] T063 Run go test ./internal/step/... -v and fix any failures
- [X] T064 Create .claude/commands/brains.feature.md skill for Claude Code integration
- [ ] T065 Run quickstart.md validation steps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories 1-4 (Phase 3-6)**: All depend on Foundational phase, are P1 priority
- **User Story 5 (Phase 7)**: Depends on Foundational, P2 priority, builds on directive from US1-4
- **User Story 6 (Phase 8)**: Depends on Foundational and US1 implementation, P2 priority
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Core - creates initiative and cycle structure, BLOCKS all other stories
- **User Story 2 (P1)**: Directive content - depends on US1 for file locations
- **User Story 3 (P1)**: Directive content - depends on US1 for file locations
- **User Story 4 (P1)**: Directive content - depends on US1 for file locations
- **User Story 5 (P2)**: Directive content - depends on US1-4 for complete workflow
- **User Story 6 (P2)**: Multi-cycle - depends on US1 for single-cycle implementation

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Types before implementation
- Core logic before integration
- Story complete before moving to next priority

### Parallel Opportunities

**Setup Phase (Phase 1)**:
- T004 and T005 can run in parallel

**Foundational Phase (Phase 2)**:
- T009 (git.go creation) independent of cycle.go tasks
- T014 and T015 (template files) independent of Go code

**User Story 1 Tests**:
```bash
# Run in parallel
Task: T016 "Create cycle_test.go with TestCreateCycle_ValidInput"
Task: T017 "Add TestCreateCycle_InvalidCycleType"
Task: T018 "Add TestGetNextCycleNumber test"
Task: T025b "Add TestResolveTemplatePath_LocalOverride"
```

**User Stories 2-5 (Directive Content)**:
- After US1 implementation complete, US2-US5 directive tasks can potentially run in parallel as they edit different sections of templates/steps/feature.md

**User Story 6 Tests**:
```bash
# Run in parallel
Task: T049 "Create feature_test.go with TestExecuteFeatureStep_NewInitiative"
Task: T050 "Add TestExecuteFeatureStep_AddCycleToExisting"
Task: T051 "Add TestExecuteFeatureStep_NewInitiativeFlag"
```

**Polish Phase Tests**:
```bash
# Run in parallel
Task: T057 "Create git_test.go with TestEnsureBranch_GitNotAvailable"
Task: T058 "Add TestEnsureBranch_CreatesBranch"
Task: T059 "Add TestEnsureBranch_SwitchesExistingBranch"
Task: T060 "Add TestFormatBranchName tests"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (types)
2. Complete Phase 2: Foundational (cycle management, git service, templates)
3. Complete Phase 3: User Story 1 (feature step implementation)
4. **STOP and VALIDATE**: Test `mcp_zombiekit__step(step="feature", name="test")`
5. Verify folder structure, state update, and basic directive returned

### Incremental Directive (User Stories 2-5)

1. Add Research phase content to directive → US2 complete
2. Add Create phase content to directive → US3 complete
3. Add Audit phase content with loop-back → US4 complete
4. Add Highlight phase with user approval → US5 complete
5. Full directive ready for autonomous LLM execution

### Multi-Cycle Support (User Story 6)

1. Add path for existing initiative detection
2. Implement cycle addition without git branch change
3. Update INITIATIVE.md with cycle history
4. Include previous cycle artifacts in files_to_read

---

## Summary

| Metric | Count |
|--------|-------|
| Total Tasks | 72 |
| Setup Phase | 5 |
| Foundational Phase | 11 |
| User Story 1 (P1) | 23 |
| User Story 2 (P1) | 4 |
| User Story 3 (P1) | 4 |
| User Story 4 (P1) | 5 |
| User Story 5 (P2) | 5 |
| User Story 6 (P2) | 8 |
| Polish Phase | 9 |
| Parallelizable [P] | 18 |

### MVP Scope (Recommended)

Complete through Phase 3 (User Story 1) for minimal viable feature step that creates initiatives and returns a basic directive. Then incrementally add directive content phases.

### Format Validation

✅ All tasks follow checklist format: `- [ ] [TaskID] [P?] [Story?] Description with file path`
