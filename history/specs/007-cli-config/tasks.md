# Tasks: CLI Configuration System

**Input**: Design documents from `/specs/007-cli-config/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested in feature specification - tests omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Project Type**: Single Go CLI application
- **Source**: `internal/` at repository root
- **CLI**: `internal/cli/`
- **Config**: `internal/config/`
- **MCP**: `internal/mcp/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and dependencies

- [X] T001 Add BurntSushi/toml dependency via `go get github.com/BurntSushi/toml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core config infrastructure that MUST be complete before ANY user story can be implemented

**Note**: These tasks establish the config loading and merging foundation used by all stories.

- [X] T002 Define Config and ToolConfig structs with TOML tags in internal/config/config.go
- [X] T003 Implement ToolCategory() function to derive category from tool name in internal/config/tools.go
- [X] T004 [P] Implement IsToolEnabled() method on Config struct in internal/config/tools.go
- [X] T005 [P] Implement Merge() method for Config struct in internal/config/merger.go
- [X] T006 Implement NewDefaultConfig() constructor returning all-tools-enabled default in internal/config/config.go

**Checkpoint**: Foundation ready - config structures and merge logic available for all stories

---

## Phase 3: User Story 1 - Disable Specific MCP Tool via Local Config (Priority: P1)

**Goal**: Allow per-project tool customization via `.brains/config.toml` local config file

**Independent Test**: Create `.brains/config.toml` that disables a tool, start MCP server, verify tool not exposed

### Implementation for User Story 1

- [X] T007 [US1] Implement LocalConfigPath() function returning `.brains/config.toml` in internal/config/loader.go
- [X] T008 [US1] Implement LoadFile() method to parse TOML config from path in internal/config/loader.go
- [X] T009 [US1] Implement LoadLocalConfig() function with error handling and debug logging in internal/config/loader.go
- [X] T010 [US1] Modify MCP server.go to accept Config and filter tool registration based on IsToolEnabled() in internal/mcp/server.go
- [X] T011 [US1] Integrate local config loading into serve command startup in internal/cli/serve.go

**Checkpoint**: User Story 1 complete - local config disables tools for specific projects

---

## Phase 4: User Story 2 - Global Default Configuration (Priority: P2)

**Goal**: Enable global default tool preferences across all projects via `~/.config/brains/config.toml`

**Independent Test**: Create only global config, start server in directory without local config, verify global settings applied

### Implementation for User Story 2

- [X] T012 [US2] Implement GlobalConfigPath() with XDG support and platform detection in internal/config/loader.go
- [X] T013 [US2] Handle XDG_CONFIG_HOME environment variable in GlobalConfigPath() in internal/config/loader.go
- [X] T014 [US2] Handle macOS fallback to ~/.config (not ~/Library/Application Support) in internal/config/loader.go
- [X] T015 [US2] Handle Windows %APPDATA% path in GlobalConfigPath() in internal/config/loader.go
- [X] T016 [US2] Implement LoadGlobalConfig() function with error handling in internal/config/loader.go
- [X] T017 [US2] Integrate global config loading into serve command (load before local) in internal/cli/serve.go

**Checkpoint**: User Story 2 complete - global defaults applied when no local config exists

---

## Phase 5: User Story 3 - Override Global with Local Config (Priority: P3)

**Goal**: Local config settings override global settings, more specific tool settings override category settings

**Independent Test**: Create global and local configs with conflicting settings, verify local takes precedence

### Implementation for User Story 3

- [X] T018 [US3] Implement LoadConfig() orchestrator that loads global, then merges local in internal/config/loader.go
- [X] T019 [US3] Ensure Merge() correctly handles category vs tool-specific precedence in internal/config/merger.go
- [X] T020 [US3] Add debug logging for loaded config file paths (FR-013) in internal/config/loader.go
- [X] T021 [US3] Update serve command to use LoadConfig() orchestrator in internal/cli/serve.go

**Checkpoint**: User Story 3 complete - precedence chain global < local works correctly

---

## Phase 6: User Story 4 - Command Line Override (Priority: P4)

**Goal**: CLI flags `--enable-tool` and `--disable-tool` override all config file settings

**Independent Test**: Run server with CLI flags that contradict config settings, verify CLI takes precedence

### Implementation for User Story 4

- [X] T022 [US4] Add --enable-tool StringSlice flag to serve command in internal/cli/serve.go
- [X] T023 [US4] Add --disable-tool StringSlice flag to serve command in internal/cli/serve.go
- [X] T024 [US4] Implement ApplyCLIOverrides() to apply CLI flags to merged config in internal/config/merger.go
- [X] T025 [US4] Integrate CLI flag processing into serve command after config loading in internal/cli/serve.go
- [X] T026 [US4] Add warning logging for unknown tool names (FR-011) in internal/config/tools.go

**Checkpoint**: User Story 4 complete - full precedence chain CLI > local > global > defaults working

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, validation, and refinements

- [X] T027 [P] Add warning logging with file path and line number for invalid TOML syntax in internal/config/loader.go
- [X] T028 [P] Validate config loading completes under 10ms (performance goal) via manual testing
- [X] T029 Run quickstart.md validation scenarios manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 (Phase 3): Can start after Foundational - No dependencies on other stories
  - US2 (Phase 4): Can start after US1 OR in parallel - Independent of US1
  - US3 (Phase 5): Depends on US1 and US2 (needs both local and global loading to test precedence)
  - US4 (Phase 6): Depends on US3 (needs full config loading to add CLI override layer)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Foundational only - can start immediately after Phase 2
- **User Story 2 (P2)**: Foundational only - can start in parallel with US1 or after
- **User Story 3 (P3)**: Requires US1 and US2 complete (merges local over global)
- **User Story 4 (P4)**: Requires US3 complete (adds final CLI override layer)

### Within Each User Story

- Config path functions before loading functions
- Loading functions before integration with CLI
- Core implementation before logging/error handling refinements

### Parallel Opportunities

- T004 and T005 can run in parallel (different files)
- T027 and T028 can run in parallel (independent concerns)
- US1 and US2 can theoretically run in parallel (different config sources)

---

## Parallel Example: Foundational Phase

```bash
# After T002, T003, T006 complete:
Task: "Implement IsToolEnabled() method on Config struct in internal/config/tools.go" (T004)
Task: "Implement Merge() method for Config struct in internal/config/merger.go" (T005)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (add TOML dependency)
2. Complete Phase 2: Foundational (config structs, merge logic)
3. Complete Phase 3: User Story 1 (local config loading)
4. **STOP and VALIDATE**: Create `.brains/config.toml`, disable a tool, verify it's not exposed
5. Deploy/demo if ready - local config works!

### Incremental Delivery

1. Setup + Foundational --> Foundation ready
2. Add User Story 1 --> Local config works --> Can demo per-project config
3. Add User Story 2 --> Global config works --> Can demo global defaults
4. Add User Story 3 --> Precedence works --> Full layered config
5. Add User Story 4 --> CLI overrides work --> Complete feature
6. Each story adds value without breaking previous stories

### Recommended Order

This feature benefits from sequential delivery due to precedence dependencies:
1. **MVP**: Phase 1-3 (local config only) - immediately useful
2. **Enhancement**: Phase 4 (add global config)
3. **Full Feature**: Phase 5-6 (add precedence and CLI overrides)
4. **Polish**: Phase 7 (error handling refinements)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently testable after completion
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Graceful degradation: invalid configs log warnings, don't fail startup
