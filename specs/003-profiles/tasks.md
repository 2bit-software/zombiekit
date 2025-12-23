# Tasks: Profile Composition System

**Input**: Design documents from `/specs/003-profiles/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Tests**: Not explicitly requested in specification - tests excluded per template guidelines.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1-US7) this task belongs to
- Paths follow Go project structure from plan.md

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependencies and create base package structure

- [X] T001 Add github.com/gofrs/flock dependency for OS-level file locking
- [X] T002 Add github.com/adrg/frontmatter dependency for YAML frontmatter parsing
- [X] T003 [P] Create internal/profile/ package directory structure
- [X] T004 [P] Create internal/mcp/tools/profile/ package directory structure

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define Profile, ProfileSource, ProfileFrontmatter, CompositionResult, Registry types in internal/profile/types.go
- [X] T006 Implement YAML frontmatter parsing with GetInherits() default-true logic in internal/profile/frontmatter.go
- [X] T007 Implement directory walking for .brains/ directories (CWD to git root + global) in internal/profile/resolver.go
- [X] T008 Implement profile file discovery and loading from resolved directories in internal/profile/resolver.go
- [X] T009 Implement registry file management with flock in internal/profile/registry.go

**Checkpoint**: Foundation ready - profile resolution and parsing infrastructure complete

---

## Phase 3: User Story 1 - Compose Profiles from Multiple Sources (Priority: P1) 🎯 MVP

**Goal**: Compose multiple profiles with proper DAG resolution, deduplication, and inheritance

**Independent Test**: Create profiles in local/global dirs, run compose, verify merged content with correct precedence

### Implementation for User Story 1

- [X] T010 [US1] Implement DAG building with cycle detection using DFS path tracking in internal/profile/composer.go
- [X] T011 [US1] Implement depth-first includes resolution with deduplication in internal/profile/composer.go
- [X] T012 [US1] Implement inheritance resolution (prepend parent content when inherits=true) in internal/profile/composer.go
- [X] T013 [US1] Implement Compose() method returning CompositionResult with metadata in internal/profile/service.go
- [X] T014 [US1] Implement profile compose subcommand with comma/space-separated args in internal/cli/profile.go
- [X] T015 [US1] Implement --format json output for compose command in internal/cli/profile.go

**Checkpoint**: Profile composition fully functional via CLI - MVP complete

---

## Phase 4: User Story 2 - List Available Profiles (Priority: P1)

**Goal**: Discover all available profiles from all sources with source attribution

**Independent Test**: Create profiles in various locations, verify list shows all with correct sources

### Implementation for User Story 2

- [X] T016 [US2] Implement List() method returning all discovered profiles with metadata in internal/profile/service.go
- [X] T017 [US2] Implement precedence marking for duplicate profile names in internal/profile/service.go
- [X] T018 [US2] Implement profile list subcommand with tabular text output in internal/cli/profile.go
- [X] T019 [US2] Implement --format json output for list command in internal/cli/profile.go

**Checkpoint**: Profile listing functional - users can discover available profiles

---

## Phase 5: User Story 3 - Show Individual Profile Content (Priority: P1)

**Goal**: Inspect profile content with or without inheritance resolved

**Independent Test**: Create profile with inheritance, verify show displays resolved content and --raw shows original

### Implementation for User Story 3

- [X] T020 [US3] Implement Show() method with resolved content and inherited_from tracking in internal/profile/service.go
- [X] T021 [US3] Implement profile show subcommand in internal/cli/profile.go
- [X] T022 [US3] Implement --raw flag to display original file content in internal/cli/profile.go
- [X] T023 [US3] Implement --format json output with raw_content and inherited_from fields in internal/cli/profile.go
- [X] T024 [US3] Implement profile-not-found error with similar name suggestions in internal/profile/service.go

**Checkpoint**: Profile inspection functional - users can examine individual profiles

---

## Phase 6: User Story 4 - Create New Profile (Priority: P2)

**Goal**: Create new profiles with proper template frontmatter

**Independent Test**: Run create command, verify file created with correct frontmatter template

### Implementation for User Story 4

- [X] T025 [US4] Implement Create() method with name normalization (lowercase, hyphens) in internal/profile/service.go
- [X] T026 [US4] Implement profile template generation with all frontmatter fields in internal/profile/service.go
- [X] T027 [US4] Implement profile create subcommand in internal/cli/profile.go
- [X] T028 [US4] Implement --global flag to create in ~/.brains/profiles/ in internal/cli/profile.go
- [X] T029 [US4] Implement already-exists check with error (no overwrite) in internal/profile/service.go
- [X] T030 [US4] Implement not-initialized error with suggestion to run brains init in internal/profile/service.go

**Checkpoint**: Profile creation functional - users can create properly formatted profiles

---

## Phase 7: User Story 5 - Validate Profile Configuration (Priority: P2)

**Goal**: Check profiles for circular dependencies and missing references

**Independent Test**: Create profiles with intentional errors, verify validate detects them

### Implementation for User Story 5

- [X] T031 [US5] Implement Validate() method returning all errors found in internal/profile/service.go
- [X] T032 [US5] Implement missing includes detection with similar name suggestions in internal/profile/service.go
- [X] T033 [US5] Implement cycle detection error reporting with full cycle path in internal/profile/service.go
- [X] T034 [US5] Implement profile validate subcommand with text output in internal/cli/profile.go
- [X] T035 [US5] Implement --format json output with structured error details in internal/cli/profile.go

**Checkpoint**: Profile validation functional - users can catch errors early

---

## Phase 8: User Story 6 - MCP Tool Interface (Priority: P2)

**Goal**: Expose profile functionality via MCP tools for AI agents

**Independent Test**: Start MCP server, invoke profile tools with parameters, verify responses

### Implementation for User Story 6

- [X] T036 [US6] Implement profile-compose MCP tool handler in internal/mcp/tools/profile/tool.go
- [X] T037 [US6] Implement profile-list MCP tool handler in internal/mcp/tools/profile/tool.go
- [X] T038 [US6] Implement profile-show MCP tool handler with raw parameter in internal/mcp/tools/profile/tool.go
- [X] T039 [US6] Implement profile-validate MCP tool handler in internal/mcp/tools/profile/tool.go
- [X] T040 [US6] Implement working_directory parameter handling for all tools in internal/mcp/tools/profile/tool.go
- [X] T041 [US6] Register profile tools in MCP server in internal/mcp/server.go
- [X] T042 [US6] Implement MCP error responses with isError flag in internal/mcp/tools/profile/tool.go

**Checkpoint**: MCP interface functional - AI agents can use profile tools

---

## Phase 9: User Story 7 - Initialize Brains Directory (Priority: P2)

**Goal**: Bootstrap the profile system in projects or globally

**Independent Test**: Run init in new directory, verify .brains/profiles/ created

### Implementation for User Story 7

- [X] T043 [US7] Implement brains init command creating .brains/profiles/ directory in internal/cli/init.go
- [X] T044 [US7] Implement --global flag creating ~/.brains/profiles/ in internal/cli/init.go
- [X] T045 [US7] Implement idempotent behavior (succeed if already exists) in internal/cli/init.go
- [X] T046 [US7] Register new directories in registry on init in internal/cli/init.go

**Checkpoint**: Initialization functional - users can bootstrap profile system

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, edge cases, and cleanup

- [X] T047 [P] Implement permission error handling with specific file path in error message in internal/profile/resolver.go
- [X] T048 [P] Implement YAML parse error handling with line number in internal/profile/frontmatter.go
- [X] T049 [P] Implement home directory inaccessible fallback (use local only, log warning) in internal/profile/resolver.go
- [X] T050 [P] Implement profile name normalization and validation (allowed character set) in internal/profile/service.go
- [X] T051 [P] Implement duplicate profile deduplication in compose command arguments in internal/cli/profile.go
- [X] T052 Wire up profile subcommand to root CLI in internal/cli/root.go
- [ ] T053 Run quickstart.md validation scenarios manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phases 3-9)**: All depend on Foundational completion
  - US1-US3 (P1): Core functionality - complete in order
  - US4-US7 (P2): Can proceed after P1 or in parallel if staffed
- **Polish (Phase 10)**: Depends on core user stories being complete

### User Story Dependencies

- **US1 (Compose)**: Primary MVP - no dependencies on other stories
- **US2 (List)**: Can start after Foundational, independent of US1
- **US3 (Show)**: Can start after Foundational, independent of US1/US2
- **US4 (Create)**: Requires US7 (init) to be meaningful, but can be built independently
- **US5 (Validate)**: Uses DAG from US1, best done after US1
- **US6 (MCP Tools)**: Wraps CLI functionality, best done after US1-US5
- **US7 (Init)**: Independent foundation, but US4 depends on it functionally

### Parallel Opportunities

Setup phase:
- T003 and T004 can run in parallel (different directories)

Foundational phase:
- T006-T009 can potentially run in parallel (different files) after T005

User stories (with multiple developers):
- US1, US2, US3 can be developed in parallel after Foundational
- US4 and US7 can be developed in parallel
- US6 should follow US1-US5 for proper abstraction

---

## Parallel Example: Foundational Phase

```bash
# After T005 (types.go) is complete:
Task: "Implement YAML frontmatter parsing in internal/profile/frontmatter.go"
Task: "Implement directory walking in internal/profile/resolver.go"
Task: "Implement registry file management in internal/profile/registry.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T009)
3. Complete Phase 3: User Story 1 - Compose (T010-T015)
4. **STOP and VALIDATE**: Test compose command with real profiles
5. Demo/deploy basic profile composition

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Compose) → Test → MVP ready!
3. Add US2 (List) + US3 (Show) → Discovery complete
4. Add US7 (Init) + US4 (Create) → User can bootstrap
5. Add US5 (Validate) → Error detection ready
6. Add US6 (MCP Tools) → AI integration ready
7. Polish phase → Production ready

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story
- Each user story is independently testable per spec
- Commit after each task or logical group
- Error codes defined in contracts/cli.md
- MCP response format defined in contracts/mcp-tools.md
