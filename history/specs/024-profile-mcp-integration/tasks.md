# Tasks: Profile-MCP Integration

**Input**: Design documents from `/specs/024-profile-mcp-integration/`
**Prerequisites**: research.md (available), data-model.md (available), contracts/ (available), quickstart.md (available)

**Tests**: Not requested - implementation only.

**Organization**: Tasks organized by priority from research.md gap analysis.

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Verify existing infrastructure and understand current state

- [x] T001 Verify step service and profile loading work correctly by running `go test ./internal/step/... -v`
- [x] T002 [P] Verify MCP step tool works correctly by running `go test ./internal/mcp/tools/step/... -v`
- [x] T003 [P] Review existing profile structure in `templates/steps/` for consistency

**Checkpoint**: Tests pass, infrastructure understood

---

## Phase 2: P0 - Major Profile Rewrites

**Purpose**: Address major gaps identified in research.md - plan.md and tasks.md profiles need complete rewrites

**Goal**: These profiles need structured directives with response handling, prerequisites, and clear workflows

### plan.md Profile Rewrite

- [x] T004 Read current `templates/steps/plan.md` and compare against standardized structure from research.md
- [x] T005 Rewrite `templates/steps/plan.md` with standardized sections:
  - Context (what this step does, agent vs system responsibilities)
  - Response Handling (how to interpret MCP response fields)
  - Prerequisites (spec.md with approved status required)
  - Workflow (constitution check, architecture decisions, phased breakdown)
  - Output (plan.md artifact structure)
  - Success Criteria (checkboxes)
  - Behavior Rules (constraints)

### tasks.md Profile Rewrite

- [x] T006 [P] Read current `templates/steps/tasks.md` and identify gaps
- [x] T007 Rewrite `templates/steps/tasks.md` with standardized sections:
  - Context (what this step does)
  - Response Handling (how to interpret MCP response fields)
  - Prerequisites (plan.md with approved status required)
  - Workflow (task ID format, dependency markers, parallel markers, user story organization)
  - Output (tasks.md artifact structure per template)
  - Success Criteria (checkboxes)
  - Behavior Rules (constraints)

**Checkpoint**: P0 profiles rewritten with complete, actionable directives

---

## Phase 3: P1 - Moderate Profile Updates

**Purpose**: Address moderate gaps - eat.md, audit.md, clarify.md need next_task handling, severity classification, taxonomy

### eat.md Profile Enhancement

- [x] T008 Enhance `templates/steps/eat.md` with:
  - Response Handling section (especially `next_task` field usage)
  - Check for null next_task indicating completion
  - Task-by-task execution guidance
  - TDD focus per task
  - Progress tracking in tasks.md

### audit.md Profile Enhancement

- [x] T009 [P] Enhance `templates/steps/audit.md` with:
  - Response Handling section
  - Severity classification (CRITICAL, MAJOR, MINOR, INFO)
  - Cross-artifact alignment checking
  - Structured audit report format

### clarify.md Profile Enhancement

- [x] T010 [P] Enhance `templates/steps/clarify.md` with:
  - Response Handling section
  - Question taxonomy (requirements, edge cases, constraints, dependencies)
  - Question format structure
  - Integration rules for encoding answers back into spec

**Checkpoint**: P1 profiles enhanced with moderate improvements

---

## Phase 4: P2 - Minor Profile Polish

**Purpose**: Polish feature.md, bug.md, refactor.md with Response Handling section

### feature.md Polish

- [x] T011 Add Response Handling section to `templates/steps/feature.md` after Context section:
  - Check prerequisites.met
  - Read files_to_read
  - Parse workflow_phases
  - Follow directive
  - Output to cycle_folder
  - Reference composed_prompt

### bug.md Polish

- [x] T012 [P] Add Response Handling section to `templates/steps/bug.md` (mirror feature.md structure)

### refactor.md Polish

- [x] T013 [P] Add Response Handling section to `templates/steps/refactor.md` (mirror feature.md structure)

**Checkpoint**: All profiles have consistent Response Handling sections

---

## Phase 5: P3 - Validation

**Purpose**: Verify all profile changes work correctly through the MCP tool

- [x] T014 Run `go build -o bin/brains ./cmd/brains` to ensure embedded profiles compile
- [x] T015 Run `go test ./internal/step/... -v` to verify profile loading
- [x] T016 [P] Run `go test ./internal/mcp/tools/step/... -v` to verify MCP tool responses
- [x] T017 Manually test step execution flow per quickstart.md scenarios

**Checkpoint**: All tests pass, profiles load correctly

---

## Phase 6: Polish

**Purpose**: Final verification and documentation

- [x] T018 Validate all profiles against FR-001..FR-005 in `templates/steps/*.md`:
  - FR-001: Each profile has YAML frontmatter with `name`, `description`, `profiles`, `files`, `type` fields
  - FR-002: Multi-phase profiles (feature, bug, refactor) have phase-by-phase directive structure
  - FR-003: Verify loading works via `go test ./internal/step/... -run TestLoader`
  - FR-004: Test local override by placing test profile in `.brains/steps/` temporarily
  - FR-005: Verify `files` field patterns are relative to cycle folder (no absolute paths)
- [x] T019 Run quickstart.md validation scenarios
- [x] T020 Update any references in contracts/ if response handling documentation needs adjustment

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **P0 Major (Phase 2)**: Depends on Setup - blocks nothing but is highest priority
- **P1 Moderate (Phase 3)**: Can start after Setup, independent of P0
- **P2 Minor (Phase 4)**: Can start after Setup, independent of P0/P1
- **Validation (Phase 5)**: Depends on Phases 2-4 completion
- **Polish (Phase 6)**: Depends on Validation

### Parallel Opportunities

Within Phase 2:
- T004/T005 (plan.md) and T006/T007 (tasks.md) can run in parallel

Within Phase 3:
- T008 (eat), T009 (audit), T010 (clarify) can all run in parallel

Within Phase 4:
- T011 (feature), T012 (bug), T013 (refactor) can all run in parallel

Within Phase 5:
- T015 and T016 can run in parallel

---

## Parallel Example: Phase 3

```bash
# Launch all P1 profile updates together:
Task: "Enhance eat.md with next_task handling"
Task: "Enhance audit.md with severity classification"
Task: "Enhance clarify.md with question taxonomy"
```

---

## Implementation Strategy

### MVP First (Phase 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: P0 Major Rewrites (plan.md, tasks.md)
3. **STOP and VALIDATE**: Test profile loading and MCP responses
4. These are the most impactful changes

### Full Delivery

1. Complete all phases in priority order
2. Each profile update is independently testable
3. Final validation confirms all profiles work through MCP

---

## Notes

- No Go code changes required per data-model.md analysis
- Primary change area is profile content (markdown), not types
- Profiles are embedded via Go embed FS - rebuild required after changes
- Test with `go build` to verify embed, `go test` to verify loading
- Response structure documented in contracts/step-tool.md
