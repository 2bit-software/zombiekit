# Tasks: WebGUI Status Page

**Input**: Design documents from `/specs/014-webgui-status/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: No explicit test requirements in specification. Tests are omitted per template guidelines.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `internal/`, `cmd/` at repository root
- All status types go in `internal/web/`
- Templates in `internal/web/templates/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create foundational types and extend server configuration

- [X] T001 Define StatusInfo, VersionInfo, DatabaseStatus, RuntimeInfo, PluginStatus, ConfigInfo types in internal/web/status.go
- [X] T002 Define StatusConfig struct with required dependencies (port, log level, storage config, start time) in internal/web/status.go
- [X] T003 Add StartTime field to Server struct in internal/web/server.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core status gathering infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Implement GatherStatus(ctx, cfg, registry) function skeleton in internal/web/status.go
- [X] T005 [P] Add sanitizePostgresURL helper function in internal/web/status.go
- [X] T006 [P] Add formatUptime helper function for human-readable duration in internal/web/status.go
- [X] T007 Update ServerConfig to include StatusConfig fields in internal/web/server.go
- [X] T008 Update NewServer to initialize StartTime in internal/web/server.go
- [X] T009 Update gui.go to pass storage config, port, and log level to ServerConfig in internal/cli/gui.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - View System Version Information (Priority: P1) 🎯 MVP

**Goal**: Display application version, git commit, build date, and Go version on the home page

**Independent Test**: Visit home page and verify version details match `brains version` output

### Implementation for User Story 1

- [X] T010 [US1] Implement gatherVersionInfo() that calls version.Get() and maps to VersionInfo in internal/web/status.go
- [X] T011 [US1] Call gatherVersionInfo() from GatherStatus() in internal/web/status.go
- [X] T012 [US1] Add version section to home.html template with version, commit, build date, go version in internal/web/templates/home.html
- [X] T013 [US1] Update homeHandler to call GatherStatus and pass StatusInfo to template in internal/web/server.go

**Checkpoint**: At this point, User Story 1 should be fully functional - version info visible on home page

---

## Phase 4: User Story 2 - View Database Backend Status (Priority: P1)

**Goal**: Display database backend type (SQLite/PostgreSQL), location (sanitized), and connection status

**Independent Test**: Start GUI with SQLite, verify "SQLite" displayed with file path; start with PostgreSQL, verify "PostgreSQL" with host/database only

### Implementation for User Story 2

- [X] T014 [US2] Implement gatherDatabaseStatus(cfg StorageConfig) that extracts backend type and sanitized location in internal/web/status.go
- [X] T015 [US2] For SQLite: return file path directly in gatherDatabaseStatus in internal/web/status.go
- [X] T016 [US2] For PostgreSQL: call sanitizePostgresURL to show host/database only in gatherDatabaseStatus in internal/web/status.go
- [X] T017 [US2] Add Connected field with placeholder true (health check deferred) in gatherDatabaseStatus in internal/web/status.go
- [X] T018 [US2] Call gatherDatabaseStatus() from GatherStatus() in internal/web/status.go
- [X] T019 [US2] Add database section to home.html template with backend, location, and status indicator in internal/web/templates/home.html

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - version and database info visible

---

## Phase 5: User Story 3 - View Runtime Environment Information (Priority: P2)

**Goal**: Display OS, architecture, Go version, uptime, CPU count, and goroutine count

**Independent Test**: Visit home page and verify OS/arch matches `go env GOOS GOARCH`, uptime increases on refresh

### Implementation for User Story 3

- [X] T020 [US3] Implement gatherRuntimeInfo(startTime) using runtime.GOOS, GOARCH, NumCPU, NumGoroutine in internal/web/status.go
- [X] T021 [US3] Calculate uptime from startTime and format with formatUptime helper in gatherRuntimeInfo in internal/web/status.go
- [X] T022 [US3] Call gatherRuntimeInfo() from GatherStatus() in internal/web/status.go
- [X] T023 [US3] Add runtime section to home.html template with platform, uptime, CPU count, goroutines in internal/web/templates/home.html

**Checkpoint**: Runtime environment info visible on home page

---

## Phase 6: User Story 4 - View Plugin Status (Priority: P2)

**Goal**: Display count and list of registered plugins with health indicators

**Independent Test**: Visit home page and verify plugin count matches registered plugins (profiles, memory)

### Implementation for User Story 4

- [X] T024 [US4] Implement gatherPluginStatus(registry) that iterates registry.All() and builds []PluginStatus in internal/web/status.go
- [X] T025 [US4] Set Healthy=true for all registered plugins (V1 simplification) in gatherPluginStatus in internal/web/status.go
- [X] T026 [US4] Call gatherPluginStatus() from GatherStatus() in internal/web/status.go
- [X] T027 [US4] Enhance plugins section in home.html to show count and health indicators in internal/web/templates/home.html

**Checkpoint**: Plugin status visible with count and health indicators

---

## Phase 7: User Story 5 - View Configuration Summary (Priority: P3)

**Goal**: Display HTTP port, log level, and profile paths

**Independent Test**: Start GUI on custom port with custom log level, verify values displayed correctly

### Implementation for User Story 5

- [X] T028 [US5] Implement gatherConfigInfo(cfg) that extracts port and log level from StatusConfig in internal/web/status.go
- [X] T029 [US5] Add ProfilePaths placeholder (empty for V1, can enhance later) in gatherConfigInfo in internal/web/status.go
- [X] T030 [US5] Call gatherConfigInfo() from GatherStatus() in internal/web/status.go
- [X] T031 [US5] Add configuration section to home.html template with port and log level in internal/web/templates/home.html

**Checkpoint**: Configuration summary visible on home page

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T032 Verify all status sections render correctly with HTMX partial updates in internal/web/templates/home.html
- [X] T033 [P] Add visual styling consistency - use Tailwind card components for each section in internal/web/templates/home.html
- [X] T034 [P] Add graceful degradation for missing data (show "N/A" or "Unknown" if data unavailable) in internal/web/status.go
- [X] T035 Verify home page loads within 500ms performance target (SC-002) - Measured: ~0.5ms (well under 500ms target)
- [X] T036 Run quickstart.md validation steps - All tests pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed) or sequentially by priority
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories, parallelizable with US1
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Gather function before GatherStatus integration
- GatherStatus integration before template updates
- Template updates last

### Parallel Opportunities

- T005 and T006 can run in parallel (different helper functions, no dependencies)
- All user stories (US1-US5) can run in parallel after Foundational phase
- T033 and T034 can run in parallel (different concerns)

---

## Parallel Example: User Stories 1 and 2

```bash
# After Foundational phase completes, launch User Stories 1 and 2 in parallel:

# Developer A works on User Story 1 (version info):
Task: "T010 [US1] Implement gatherVersionInfo() in internal/web/status.go"
Task: "T011 [US1] Call gatherVersionInfo() from GatherStatus() in internal/web/status.go"
Task: "T012 [US1] Add version section to home.html template"
Task: "T013 [US1] Update homeHandler to call GatherStatus"

# Developer B works on User Story 2 (database status):
Task: "T014 [US2] Implement gatherDatabaseStatus() in internal/web/status.go"
Task: "T015 [US2] For SQLite: return file path directly"
Task: "T016 [US2] For PostgreSQL: call sanitizePostgresURL"
# Note: T017-T019 sequential after T014-T016
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T009) **CRITICAL**
3. Complete Phase 3: User Story 1 (T010-T013)
4. **STOP and VALIDATE**: Visit home page, verify version info displays
5. Deploy/demo if ready

### Recommended Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 + User Story 2 (both P1) → Test → Deploy (MVP!)
3. Add User Story 3 + User Story 4 (both P2) → Test → Deploy
4. Add User Story 5 (P3) → Test → Deploy
5. Polish phase → Final deployment

### Single Developer Strategy

Execute phases sequentially in priority order:
1. Setup → Foundational → US1 → US2 → US3 → US4 → US5 → Polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- No test tasks included (not explicitly requested)
- All template updates assume Tailwind CSS (existing pattern)
- Status data is computed per-request (no caching needed)
