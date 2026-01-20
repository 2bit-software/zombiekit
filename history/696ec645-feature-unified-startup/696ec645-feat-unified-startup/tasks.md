---
status: draft
created: 2026-01-19
plan: implementation-plan.md
spec: spec.md
---

# Task List: Unified Startup Command

**Complexity**: Medium (11 files)
**Parallel opportunities**: Phase 1 and Phase 2 can execute in parallel

## Phase 1: Shutdown Manager (Independent)

- [ ] T001 [P] Create `internal/shutdown/manager.go` with Manager struct
- [ ] T002 [P] Implement `Manager.Run()` with errgroup and context cancellation
- [ ] T003 Implement signal handling (SIGINT/SIGTERM) with buffered channel
- [ ] T004 Implement 10-second shutdown timeout with force exit
- [ ] T005 Implement double Ctrl+C force exit behavior
- [ ] T006 Add unit tests `internal/shutdown/manager_test.go` for Manager

## Phase 2: Configuration System (Independent, Parallel with Phase 1)

- [ ] T007 [P] Create `internal/config/startup.go` with StartupConfig types
- [ ] T008 [P] Implement config file discovery (local → global → env)
- [ ] T009 Implement `LoadStartupConfig()` with YAML parsing
- [ ] T010 Implement `Validate()` with port/interval checks
- [ ] T011 Add default values matching current `task up` behavior
- [ ] T012 Add unit tests `internal/config/startup_test.go` for config loading and validation

## Phase 3: Service Runners (Depends on Phase 2)

- [ ] T013 Create `internal/startup/service.go` with Service interface
- [ ] T014 Create `internal/startup/gui_service.go` - wrapper for GUI server
- [ ] T015 Create `internal/startup/recall_service.go` - wrapper for recall watch
- [ ] T016 Implement service-prefixed logging via slog WithGroup
- [ ] T017 Add unit tests `internal/startup/service_test.go` for service wrappers

## Phase 4: Start Command CLI (Depends on Phases 1, 2, 3)

- [ ] T018 Create `internal/cli/start.go` with start command implementation
- [ ] T019 Integrate shutdown manager with service errgroup
- [ ] T020 Register start command in `internal/cli/root.go`

## Phase 5: Integration Testing (Depends on Phase 4)

- [ ] T021 Add integration test: start command launches services
- [ ] T022 Add integration test: Ctrl+C triggers graceful shutdown
- [ ] T023 Add integration test: disabled services are skipped
- [ ] T024 Add integration test: invalid config produces clear error

---

## Traceability Matrix

| Task | FR | US |
|------|----|----|
| T001-T006 | FR-005, FR-009 | US2 |
| T007-T012 | FR-002, FR-008, FR-010 | US4 |
| T013-T017 | FR-003, FR-004 | US1 |
| T018-T020 | FR-001, FR-006, FR-007 | US1, US3 |
| T021-T024 | All | All |

## Execution Order

```
T001, T002, T007, T008  (parallel start)
         ↓
   T003, T004, T005     (Phase 1 completion)
         ↓
   T009, T010, T011     (Phase 2 completion)
         ↓
      T006, T012        (unit tests)
         ↓
   T013, T014, T015     (service wrappers)
         ↓
      T016, T017        (logging + tests)
         ↓
   T018, T019, T020     (CLI command)
         ↓
   T021, T022, T023, T024 (integration tests)
```

## Suggested Next Command

```
/brains.implement
```
