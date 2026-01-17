# Tasks: Profile Import Subcommand

**Input**: Design documents from `/specs/005-profile-import/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: Included as test tasks are implicit in Go projects (tests alongside implementation).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go CLI application at repository root
- **Source**: `internal/profile/`, `internal/cli/`
- **Tests**: `*_test.go` alongside source files

---

## Phase 1: Setup

**Purpose**: Define new types and prepare the foundation for import functionality

- [X] T001 Add ImportResult and ImportFailure types to internal/profile/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core import service that MUST be complete before CLI integration

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Create Importer struct and NewImporter constructor in internal/profile/importer.go
- [X] T003 Implement convertClaudeTobrains helper function for frontmatter conversion in internal/profile/importer.go
- [X] T004 Implement writeBrainsProfile helper for file writing with directory creation in internal/profile/importer.go
- [X] T005 Implement Import method that loads Claude agents, converts, and writes brains profiles in internal/profile/importer.go
- [X] T006 Add unit tests for frontmatter conversion (name, description, includes preserved; model, color discarded; inherits forced to false) in internal/profile/importer_test.go

**Checkpoint**: Import service ready - CLI integration can now begin

---

## Phase 3: User Story 1 - Import Claude Agents to Brains Profiles (Priority: P1) 🎯 MVP

**Goal**: Convert Claude agents from both local and global directories to brains profiles, preserving scope

**Independent Test**: Create sample Claude agents in `.claude/agents/` and `~/.claude/agents/`, run `brains profiles import claude`, verify profiles exist in `.brains/profiles/` and `~/.brains/profiles/`

### Implementation for User Story 1

- [X] T007 [US1] Add `import` subcommand to profiles command in internal/cli/profile.go with source argument
- [X] T008 [US1] Implement import command handler that calls Importer.Import() in internal/cli/profile.go
- [X] T009 [US1] Format and display text output showing created profiles count and paths in internal/cli/profile.go
- [X] T010 [US1] Add integration test for importing local Claude agents in internal/profile/importer_test.go
- [X] T011 [US1] Add integration test for importing global Claude agents in internal/profile/importer_test.go

**Checkpoint**: Basic import working - can import Claude agents to brains profiles

---

## Phase 4: User Story 2 - Overwrite Existing Profiles on Collision (Priority: P1)

**Goal**: Re-import updated Claude agents, overwriting previous brains versions without prompting

**Independent Test**: Create brains profile with same name as Claude agent, run import, verify brains profile content is replaced

### Implementation for User Story 2

- [X] T012 [US2] Add overwrite detection logic to Import method (check if target exists before write) in internal/profile/importer.go
- [X] T013 [US2] Track created vs overwritten profiles separately in ImportResult in internal/profile/importer.go
- [X] T014 [US2] Update text output to show overwritten profiles separately from created in internal/cli/profile.go
- [X] T015 [US2] Add integration test for overwrite behavior in internal/profile/importer_test.go

**Checkpoint**: Overwrite behavior working - re-import safely overwrites existing profiles

---

## Phase 5: User Story 3 - Preview Import with Dry Run (Priority: P2)

**Goal**: Show what profiles would be created/overwritten without making changes

**Independent Test**: Run `brains profiles import claude --dry-run`, verify output shows planned operations, no files created/modified

### Implementation for User Story 3

- [X] T016 [US3] Add --dry-run flag to import subcommand in internal/cli/profile.go
- [X] T017 [US3] Modify Import method to skip file writes when dryRun=true in internal/profile/importer.go
- [X] T018 [US3] Update text output to indicate dry run mode and planned operations in internal/cli/profile.go
- [X] T019 [US3] Add test for dry run behavior (no files written) in internal/profile/importer_test.go

**Checkpoint**: Dry run working - users can preview import before committing

---

## Phase 6: User Story 4 - Import Summary Report (Priority: P2)

**Goal**: Provide detailed feedback about import results including JSON format option

**Independent Test**: Run import with multiple agents, verify output includes counts and paths; test with --format json

### Implementation for User Story 4

- [X] T020 [US4] Add --format flag (text/json) to import subcommand in internal/cli/profile.go
- [X] T021 [US4] Implement JSON output formatting for ImportResult in internal/cli/profile.go
- [X] T022 [US4] Add error handling for partial failures (continue importing, report failures at end) in internal/profile/importer.go
- [X] T023 [US4] Track and report failed agents in ImportResult.FailedAgents in internal/profile/importer.go
- [X] T024 [US4] Add test for partial failure handling (some agents fail, others succeed) in internal/profile/importer_test.go

**Checkpoint**: Full reporting working - users get complete feedback on import operations

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases and robustness improvements

- [X] T025 Handle edge case: Claude agents directory doesn't exist (graceful message) in internal/profile/importer.go
- [X] T026 Handle edge case: Empty agent body (valid, create profile with empty body) in internal/profile/importer.go
- [X] T027 Add validation for invalid source type argument (only "claude" supported) in internal/cli/profile.go
- [X] T028 Run quickstart.md scenarios to validate end-to-end functionality

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational completion
  - US1 & US2 are both P1 priority but US2 depends on US1 (overwrite extends basic import)
  - US3 & US4 are P2 priority, can proceed after US1/US2
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2 - Core import functionality
- **User Story 2 (P1)**: Depends on US1 - Extends import with overwrite tracking
- **User Story 3 (P2)**: Depends on Phase 2 - Dry run is independent feature
- **User Story 4 (P2)**: Depends on US1/US2 - Reporting builds on import results

### Within Each Phase

- Types before implementation
- Helpers before main methods
- Core logic before CLI integration
- Implementation before tests (Go standard: tests alongside code)

### Parallel Opportunities

- T010 and T011 can run in parallel (different test scenarios)
- T020 and T022 can run in parallel (different files/concerns)
- T025 and T026 can run in parallel (different edge cases)

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (types)
2. Complete Phase 2: Foundational (import service)
3. Complete Phase 3: User Story 1 (basic import)
4. Complete Phase 4: User Story 2 (overwrite)
5. **STOP and VALIDATE**: Test `brains profiles import claude` works end-to-end

### Incremental Delivery

1. Setup + Foundational → Import service ready
2. Add US1 → Basic import works → Can use for simple migrations
3. Add US2 → Overwrite works → Safe for re-imports
4. Add US3 → Dry run → Preview before commit
5. Add US4 → Full reporting → Production-ready

---

## Notes

- All new code in internal/profile/importer.go (service) and internal/cli/profile.go (CLI)
- Uses existing ClaudeSource for reading agents (from 004-source-interface)
- Uses os.WriteFile() directly for overwrite semantics (not BrainsSource.CreateProfile which errors on collision)
- File permissions: 0o644 for files, 0o755 for directories
- Commit after each phase or logical group
