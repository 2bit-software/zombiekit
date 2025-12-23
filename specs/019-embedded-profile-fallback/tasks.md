# Tasks: Embedded Profile Fallback

**Input**: Design documents from `/specs/019-embedded-profile-fallback/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Tests are included for unit testing the core embedded functionality. No TDD workflow explicitly requested.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go CLI with internal packages
- Source: `internal/`, `cmd/brains/`
- Profiles: `profiles/` (repository root)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add SourceEmbedded constant and prepare embed.FS infrastructure

- [X] T001 Add SourceEmbedded constant to ProfileSource enum in internal/profile/types.go
- [X] T002 Update ProfileSource.String() to return "embedded" for SourceEmbedded in internal/profile/types.go
- [X] T003 [P] Create embedded.go with SetEmbeddedFS(), GetEmbeddedFS(), HasEmbeddedProfiles() functions in internal/profile/embedded.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the core embedded profile loading infrastructure that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Implement loadEmbeddedProfiles() function to read profiles from embed.FS in internal/profile/embedded.go
- [X] T005 Create loadProfilesFromEmbedded() helper that returns map[string]*Profile in internal/profile/embedded.go
- [X] T006 [P] Add //go:embed profiles/* directive and embeddedProfiles variable in embed.go (root level)
- [X] T007 Call profile.SetEmbeddedFS(embeddedProfiles) in cmd/brains/main.go init or before CLI runs

**Checkpoint**: Foundation ready - embedded profiles can be loaded from binary

---

## Phase 3: User Story 1 - Default Profiles Always Available (Priority: P1) 🎯 MVP

**Goal**: Users can run profile commands immediately without any .brains/profiles/ directories

**Independent Test**: Run `brains profile list` with no profile directories - should show embedded profiles with source "embedded"

### Tests for User Story 1

- [X] T008 [P] [US1] Unit test for SetEmbeddedFS/GetEmbeddedFS/HasEmbeddedProfiles in internal/profile/embedded_test.go
- [X] T009 [P] [US1] Unit test for loadEmbeddedProfiles() returning correct profiles in internal/profile/embedded_test.go
- [X] T010 [P] [US1] Unit test verifying embedded profiles have source=SourceEmbedded and path="[embedded]/name.md" in internal/profile/embedded_test.go

### Implementation for User Story 1

- [X] T011 [US1] Extend BrainsSource.FindProfileDirs() to append embedded ResolvedDirectory as last item in internal/profile/brains_source.go
- [X] T012 [US1] Extend BrainsSource.LoadProfiles() to include embedded profiles when loading in internal/profile/brains_source.go
- [X] T013 [US1] Extend BrainsSource.LoadAllProfiles() to include embedded profiles for list command in internal/profile/brains_source.go
- [X] T014 [US1] Set Profile.Path to "[embedded]/<name>.md" format for embedded profiles in internal/profile/embedded.go
- [X] T015 [US1] Verify `brains profile list` shows embedded profiles with source "embedded"
- [X] T016 [US1] Verify `brains profile show <embedded-profile>` returns content with path "[embedded]/name.md"
- [X] T017 [US1] Verify `brains profile compose <embedded-profile>` returns composed content

**Checkpoint**: User Story 1 complete - profile commands work without any configuration

---

## Phase 4: User Story 2 - Local Profiles Override Embedded (Priority: P2)

**Goal**: Local/global profiles shadow embedded profiles with the same name

**Independent Test**: Create local `.brains/profiles/init.md` and run `brains profile compose init` - should return local version, not embedded

### Tests for User Story 2

- [X] T018 [P] [US2] Unit test verifying local profile shadows embedded profile with same name in internal/profile/brains_source_test.go
- [X] T019 [P] [US2] Unit test verifying global profile shadows embedded profile with same name in internal/profile/brains_source_test.go
- [X] T020 [P] [US2] Unit test verifying precedence order: local > parent > global > embedded in internal/profile/brains_source_test.go

### Implementation for User Story 2

- [X] T021 [US2] Verify embedded profiles are appended AFTER global in FindProfileDirs() return slice in internal/profile/brains_source.go
- [X] T022 [US2] Verify LoadProfiles() respects existing "first wins" shadowing logic for embedded in internal/profile/brains_source.go
- [X] T023 [US2] Verify `brains profile list` marks embedded as shadowed when local/global exists
- [X] T024 [US2] Extend GetInheritanceChain() to include embedded profiles for inherits:true resolution in internal/profile/brains_source.go

**Checkpoint**: User Story 2 complete - shadowing works correctly with embedded profiles

---

## Phase 5: User Story 3 - MCP Tools Use Embedded Fallback (Priority: P2)

**Goal**: MCP profile-compose and profile-list tools include embedded profiles

**Independent Test**: Call MCP profile-list with no profile directories - should return embedded profiles

### Tests for User Story 3

- [X] T025 [P] [US3] Unit test for MCP profile-compose with embedded profile in internal/mcp/tools/profile/tool_test.go
- [X] T026 [P] [US3] Unit test for MCP profile-list returning embedded profiles in internal/mcp/tools/profile/tool_test.go

### Implementation for User Story 3

- [X] T027 [US3] Verify NewService() in MCP tool handlers picks up embedded profiles (no code change expected - uses existing service)
- [X] T028 [US3] Integration test: MCP profile-compose returns embedded content when no filesystem profiles exist
- [X] T029 [US3] Integration test: MCP profile-list includes embedded profiles in response

**Checkpoint**: User Story 3 complete - MCP tools work with embedded profiles

---

## Phase 6: User Story 4 - Profiles Embedded at Build Time (Priority: P3)

**Goal**: The ./profiles/ directory is embedded in the binary at compile time

**Independent Test**: Build binary, move to new directory with no profiles, run `brains profile list` - should show embedded

### Tests for User Story 4

- [X] T030 [P] [US4] Integration test: build binary and verify profiles are embedded in tests/integration/profile_embedded_test.go
- [X] T031 [P] [US4] Test that embedded profile content matches source files at build time

### Implementation for User Story 4

- [X] T032 [US4] Verify //go:embed directive in embed.go (root level) correctly embeds profiles/* directory
- [X] T033 [US4] Verify standard `go build` includes embedded profiles (no special build flags needed)
- [X] T034 [US4] Document embedding in quickstart.md verification section

**Checkpoint**: User Story 4 complete - binary is self-contained with embedded profiles

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, error handling, and validation

- [X] T035 [P] Handle embedded profile parse errors gracefully (skip invalid, continue) in internal/profile/embedded.go
- [X] T036 [P] Handle empty/uninitialized embed.FS gracefully (return empty list, no error) in internal/profile/embedded.go
- [X] T037 [P] Verify profile validation includes embedded profiles in cycle detection in internal/profile/service.go
- [X] T038 [P] Verify includes referencing embedded profiles work correctly
- [X] T039 Run quickstart.md verification checklist
- [X] T040 Update CLAUDE.md if any new patterns introduced (already updated by /speckit.plan)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 (P1): Can start immediately after Foundational
  - US2 (P2): Can start after Foundational (no dependency on US1 for implementation)
  - US3 (P2): Can start after Foundational (uses same service as CLI)
  - US4 (P3): Can start after Foundational (build verification)
- **Polish (Phase 7)**: Can start after Phase 2, but best after US1 complete

### User Story Dependencies

- **User Story 1 (P1)**: After Foundational - core implementation, no cross-story dependencies
- **User Story 2 (P2)**: After Foundational - tests shadowing behavior, independent of other stories
- **User Story 3 (P2)**: After Foundational - MCP uses same service, minimal new code
- **User Story 4 (P3)**: After Foundational - build verification only, no code dependencies

### Within Each User Story

- Tests can run in parallel [P] before implementation
- Implementation tasks execute sequentially within story
- Integration verification after implementation

### Parallel Opportunities

- T001, T002, T003 can run in parallel (different files)
- T006, T007 can run in parallel (embed declaration vs main.go call)
- T008, T009, T010 can run in parallel (different test functions)
- T018, T019, T020 can run in parallel (different test cases)
- T025, T026 can run in parallel (different MCP tool tests)
- T030, T031 can run in parallel (different integration tests)
- T035, T036, T037, T038 can run in parallel (independent edge cases)

---

## Parallel Example: User Story 1

```bash
# Launch all tests for US1 together:
Task: "Unit test for SetEmbeddedFS/GetEmbeddedFS/HasEmbeddedProfiles in internal/profile/embedded_test.go"
Task: "Unit test for loadEmbeddedProfiles() returning correct profiles in internal/profile/embedded_test.go"
Task: "Unit test verifying embedded profiles have source=SourceEmbedded in internal/profile/embedded_test.go"

# Then implementation sequentially:
Task: "Extend BrainsSource.FindProfileDirs() to append embedded ResolvedDirectory"
Task: "Extend BrainsSource.LoadProfiles() to include embedded profiles"
# ... etc
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T007)
3. Complete Phase 3: User Story 1 (T008-T017)
4. **STOP and VALIDATE**: Test `brains profile list` on fresh system
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Embed infrastructure ready
2. Add User Story 1 → Test independently → Embedded profiles accessible (MVP!)
3. Add User Story 2 → Test independently → Shadowing works correctly
4. Add User Story 3 → Test independently → MCP tools work with embedded
5. Add User Story 4 → Test independently → Build verification complete
6. Polish → Edge cases and validation

### Single Developer Strategy

Recommended order:
1. T001-T007 (Setup + Foundational)
2. T011-T017 (US1 Implementation - skip tests initially for faster MVP)
3. T008-T010 (US1 Tests - after verifying manually)
4. T021-T024 (US2 - shadowing)
5. T027-T029 (US3 - MCP)
6. T032-T034 (US4 - build verification)
7. T035-T040 (Polish)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- The embedded.go file is the main new file (~80-100 LOC)
- brains_source.go modifications are minimal (~20-30 LOC)
- cmd/brains/embed.go is just the //go:embed directive (~5 LOC)
- No changes needed to service.go, CLI, or MCP handlers (they use existing interfaces)
