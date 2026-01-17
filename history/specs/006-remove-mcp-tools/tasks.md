# Tasks: Remove profile-show and profile-validate MCP Tools

**Input**: Design documents from `/specs/006-remove-mcp-tools/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)

**Tests**: No test tasks included - spec did not explicitly request TDD approach. Existing tests will be verified.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `internal/`, `cmd/` at repository root
- Paths follow existing project structure from plan.md

---

## Phase 1: Setup (Not Required)

**Purpose**: Project initialization and basic structure

No setup tasks needed - this is a code removal task in an existing project.

---

## Phase 2: Foundational (Not Required)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

No foundational tasks needed - project infrastructure already exists.

---

## Phase 3: User Story 1 - Streamlined MCP Tool Surface (Priority: P1) 🎯 MVP

**Goal**: Remove profile-show and profile-validate from MCP tool registration so they are no longer advertised or callable.

**Independent Test**: Start MCP server and verify that only profile-compose and profile-list appear in the tool list; attempts to call profile-show or profile-validate fail with "unknown tool" error.

### Implementation for User Story 1

- [X] T001 [P] [US1] Remove profile-show tool registration from registerProfileTools() in internal/mcp/server.go
- [X] T002 [P] [US1] Remove profile-validate tool registration from registerProfileTools() in internal/mcp/server.go
- [X] T003 [P] [US1] Remove handleProfileShow handler function in internal/mcp/server.go
- [X] T004 [P] [US1] Remove handleProfileValidate handler function in internal/mcp/server.go

**Checkpoint**: At this point, profile-show and profile-validate should no longer be exposed via MCP

---

## Phase 4: User Story 2 - Retained Essential Tools (Priority: P1)

**Goal**: Verify that profile-compose and profile-list remain fully functional after the removal.

**Independent Test**: Start MCP server and successfully call profile-list to get available profiles, then call profile-compose with valid profile names to get composed content.

### Implementation for User Story 2

- [X] T005 [US2] Verify profile-compose registration and handler remain intact in internal/mcp/server.go
- [X] T006 [US2] Verify profile-list registration and handler remain intact in internal/mcp/server.go
- [X] T007 [US2] Run existing tests to confirm profile-compose functionality: go test ./internal/mcp/...
- [X] T008 [US2] Run existing tests to confirm profile-list functionality: go test ./internal/mcp/...

**Checkpoint**: At this point, all essential profile tools should remain fully functional

---

## Phase 5: Polish & Verification

**Purpose**: Final verification and cleanup

- [X] T009 Build project to verify no compile errors: go build ./...
- [X] T010 Run all MCP tests to verify no regressions: go test ./internal/mcp/...
- [X] T011 Verify MCP server starts without errors: go run ./cmd/brains serve (manual check)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Skipped - not required
- **Foundational (Phase 2)**: Skipped - not required
- **User Story 1 (Phase 3)**: Can start immediately - removal tasks
- **User Story 2 (Phase 4)**: Depends on User Story 1 completion (verification tasks)
- **Polish (Phase 5)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies - pure code removal
- **User Story 2 (P1)**: Depends on User Story 1 - verification of retained functionality

### Within Each User Story

- T001-T004 can all run in parallel (different code sections)
- T005-T008 are sequential verification tasks
- T009-T011 are sequential verification tasks

### Parallel Opportunities

Within User Story 1, all removal tasks (T001-T004) can run in parallel as they modify different sections of the same file:

```bash
# Launch all removal tasks together (they affect different code sections):
Task: "Remove profile-show tool registration from registerProfileTools() in internal/mcp/server.go"
Task: "Remove profile-validate tool registration from registerProfileTools() in internal/mcp/server.go"
Task: "Remove handleProfileShow handler function in internal/mcp/server.go"
Task: "Remove handleProfileValidate handler function in internal/mcp/server.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 3: User Story 1 (T001-T004)
2. **STOP and VALIDATE**: Verify code compiles and profile-show/profile-validate are gone
3. Continue to verification

### Incremental Delivery

1. Complete User Story 1 → Removal complete
2. Complete User Story 2 → Verification complete
3. Complete Polish → Feature done

---

## Notes

- [P] tasks = different code sections, can edit same file in parallel
- [Story] label maps task to specific user story for traceability
- This is a removal task - no new code, only deletions
- Existing HandleShow and HandleValidate methods in internal/mcp/tools/profile/tool.go are intentionally kept for potential CLI use
- Commit after completing each user story
