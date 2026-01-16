# Tasks: WebGUI Container Development Environment

**Input**: Design documents from `/specs/016-webgui-container-dev/`
**Prerequisites**: plan.md, spec.md, research.md, quickstart.md

**Tests**: Not requested - this is development tooling with manual verification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the Docker infrastructure for development container

- [X] T001 Create docker directory structure at docker/webgui-dev/
- [X] T002 Add .data/ to .gitignore file at .gitignore

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the Dockerfile and Docker Compose service - MUST be complete before user stories can be verified

**⚠️ CRITICAL**: User story verification requires this infrastructure to be in place

- [X] T003 Create Dockerfile with Go 1.24 + wgo at docker/webgui-dev/Dockerfile
- [X] T004 Add webgui-dev service to docker-compose.yml at docker-compose.yml
- [X] T005 Add webgui:dev task entry to Taskfile.yml at Taskfile.yml

**Checkpoint**: Docker infrastructure ready - user story verification can begin

---

## Phase 3: User Story 1 - Start WebGUI Development Server (Priority: P1) 🎯 MVP

**Goal**: Developer can start the webgui in a containerized development environment with a single command and have hot-reloading working

**Independent Test**: Run `task webgui:dev`, verify webgui is accessible at http://localhost:9981, modify a .go file, verify wgo rebuilds and restarts automatically

### Verification for User Story 1

- [X] T006 [US1] Verify container starts successfully with `task webgui:dev`
- [X] T007 [US1] Verify webgui is accessible at http://localhost:9981
- [X] T008 [US1] Verify wgo detects file changes and triggers rebuild (modify a .go file)
- [X] T009 [US1] Verify rebuild completes within 5 seconds of file save

**Checkpoint**: At this point, User Story 1 should be fully functional - developers can start the container and see hot-reloading working

---

## Phase 4: User Story 2 - Persistent SQLite Data (Priority: P1)

**Goal**: SQLite database files persist between container restarts so test data is not lost

**Independent Test**: Create data in webgui, stop container, restart container, verify data still exists

### Verification for User Story 2

- [X] T010 [US2] Verify .data/ directory is created on first container start
- [X] T011 [US2] Verify SQLite database is created in .data/ directory
- [X] T012 [US2] Verify data persists after container stop and restart (create memory, stop, start, verify memory exists)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work - hot-reload and data persistence

---

## Phase 5: User Story 3 - Stop Development Server (Priority: P2)

**Goal**: Developer can cleanly stop the webgui development container with resources and ports freed

**Independent Test**: Start container, stop it with Ctrl+C, verify port 9981 is released, verify container can restart without conflicts

### Verification for User Story 3

- [X] T013 [US3] Verify container stops gracefully on Ctrl+C
- [X] T014 [US3] Verify port 9981 is released after container stop
- [X] T015 [US3] Verify container can be restarted without conflicts after stop

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and edge case handling

- [X] T016 Update quickstart.md with any discovered issues or corrections at specs/016-webgui-container-dev/quickstart.md
- [X] T017 Verify edge case: container fails with clear error when port 9981 is in use
- [X] T018 Verify edge case: container fails with clear error when Docker is not running

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user story verification
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all user stories being verified

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after User Story 1 verified (uses running container)
- **User Story 3 (P2)**: Can start after User Story 1 verified (needs running container to stop)

### Within Each Phase

- Implementation tasks (T001-T005) must complete before verification tasks (T006+)
- Verification tasks within a story can run sequentially

### Parallel Opportunities

- T001 and T002 can run in parallel (different files)
- T003, T004, T005 must be sequential (T003 creates file T004 references, T005 references T004)
- User Stories 2 and 3 verification could run in parallel if desired (both just need running container)

---

## Parallel Example: Phase 1 Setup

```bash
# These can run in parallel:
Task: "Create docker directory structure at docker/webgui-dev/"
Task: "Add .data/ to .gitignore file at .gitignore"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 2: Foundational (T003-T005)
3. Complete Phase 3: User Story 1 verification (T006-T009)
4. **STOP and VALIDATE**: Developer can start dev environment with hot-reloading
5. Deploy/demo if ready - basic dev workflow is functional

### Incremental Delivery

1. Complete Setup + Foundational → Docker infrastructure ready
2. Verify User Story 1 → Hot-reload working → MVP complete!
3. Verify User Story 2 → Data persistence working
4. Verify User Story 3 → Clean shutdown working
5. Polish → Edge cases documented and tested

---

## Notes

- This feature is infrastructure/tooling - no code changes to the application itself
- All "implementation" is configuration (Dockerfile, docker-compose.yml, Taskfile.yml)
- "Verification" tasks are manual tests per the spec's acceptance scenarios
- T003 Dockerfile must use exact version `golang:1.24-alpine` per go.mod
- T004 must mount both `.:/app` (source) and `.data:/app/.data` (data)
- T005 task should use `docker compose up --build webgui-dev`
