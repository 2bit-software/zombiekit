# Tasks: Workflow Step Tracking

**Plan**: [implementation-plan.md](./implementation-plan.md)
**Spec**: [spec.md](./spec.md)
**Created**: 2026-01-31

## Complexity

- **Classification**: Medium (12 files)
- **Parallel opportunities**: Phase 2+3 can run together; profile updates parallel

## Phase 1: Data Model Changes

- [x] T001 Simplify `InitiativeState` struct in `internal/initiative/types.go` - remove `Cycle`, `LastActivity`, `CurrentStep` fields, add `Status` field
- [x] T002 Add `InitiativeStatus` type with `in_progress`/`complete` constants in `internal/initiative/types.go`
- [x] T003 Update `StateManager.Save()` in `internal/initiative/state.go` - remove automatic `LastActivity` update

## Phase 2: INITIATIVE.md Parser (parallelizable with Phase 3)

- [x] T004 [P] Create `internal/initiative/markdown.go` with `ParsedInitiative`, `ParsedCycle`, `ParsedStep` types
- [x] T005 [P] Implement `ParseInitiativeMD(path string)` function - parse header metadata (Name, Type, Status, Created)
- [x] T006 Implement cycle section parser - regex for `### N. type/name (status)` headers
- [x] T007 Implement step table parser - parse markdown table rows into `ParsedStep` structs
- [x] T008 Implement `ActiveCycle()`, `CurrentStep()`, `NextStep()` helper methods
- [x] T009 Implement `UpdateStepStatus(cycleNum, stepName, status, timestamp)` method
- [x] T010 Implement `AddStep(cycleNum, afterStep, newStep)` method for inserting steps
- [x] T011 Implement `WriteTo(path)` method - preserve non-cycle sections, atomic write

## Phase 3: Workflow Frontmatter (parallelizable with Phase 2)

- [x] T012 [P] Add `WorkflowStep` and `WorkflowMeta` types to `internal/step/types.go`
- [x] T013 [P] Update `embed/profiles/feature.md` frontmatter - add `steps:` array with spec→plan→tasks→implement
- [x] T014 [P] Update `embed/profiles/bug.md` frontmatter - add `steps:` array with investigate→fix→verify
- [x] T015 [P] Update `embed/profiles/refactor.md` frontmatter - add `steps:` array with analyze→plan→implement→verify
- [x] T016 Implement `GetWorkflowSteps(workflowType)` in `internal/step/service.go` - parse profile frontmatter

## Phase 4: Initiative Creation Changes

- [x] T017 Update `createInitiativeMD()` in `internal/initiative/service.go` - new template with Cycles section and step table
- [x] T018 Update `handleCreate()` in `internal/mcp/tools/initiative/tool.go` - load workflow steps, pass to creation

## Phase 5: Status and Step Tools

- [x] T019 Update `StatusResult` struct in `internal/initiative/service.go` - add `CurrentCycle`, `StepStatus`, `StepsCompleted`, `StepsTotal`
- [x] T020 Update `Status()` method in `internal/initiative/service.go` - parse INITIATIVE.md for cycle/step state
- [x] T021 Update step tool `Execute()` in `internal/mcp/tools/step/tool.go` - write step status to INITIATIVE.md after execution

## Phase 6: next.md Workflow Update

- [x] T022 Rewrite `embed/workflows/next.md` - implement complete-or-advance logic using INITIATIVE.md state

## Phase 7: Cleanup and Tests

- [x] T023 Remove deprecated field references in `internal/initiative/service.go` - `state.Cycle`, `state.CurrentStep`, `state.LastActivity`
- [x] T024 Update `internal/initiative/state_test.go` - use new `active.json` format
- [x] T025 Update `internal/initiative/service_test.go` - test new Status() behavior
- [x] T026 Create `internal/initiative/markdown_test.go` - test parse single/multiple cycles, malformed tables, step updates

## Additional Work (discovered during implementation)

- [x] T027 Fix `internal/step/service_test.go` - update tests for new path behavior (no separate cycle folder)

## Dependency Graph

```
T001 ─┬─► T002 ─► T003
      │
      ├─► T004 ─► T005 ─► T006 ─► T007 ─► T008 ─► T009 ─► T010 ─► T011
      │                                                              │
      └─► T012 ─┬─► T013                                             │
                ├─► T014                                             │
                ├─► T015                                             │
                └─► T016 ────────────────────────────────────────────┤
                                                                     │
                    T017 ◄───────────────────────────────────────────┤
                      │                                              │
                      ▼                                              │
                    T018 ◄───────────────────────────────────────────┘
                      │
                      ▼
                    T019 ─► T020 ─► T021
                                     │
                                     ▼
                                   T022
                                     │
                                     ▼
                    T023 ─► T024 ─► T025 ─► T026
```

## Execution Order

**Batch 1** (parallel):
- T001, T002, T003 (data model)
- T004, T005 (parser types)
- T012, T013, T014, T015 (frontmatter)

**Batch 2** (sequential, needs Batch 1):
- T006, T007, T008 (parser implementation)
- T016 (step service)

**Batch 3** (sequential, needs Batch 2):
- T009, T010, T011 (parser writer)
- T017, T018 (creation)

**Batch 4** (sequential, needs Batch 3):
- T019, T020, T021 (status/step tools)

**Batch 5** (sequential, needs Batch 4):
- T022 (next.md workflow)

**Batch 6** (cleanup, needs all):
- T023, T024, T025, T026 (cleanup and tests)

## Spec Traceability

| Spec Requirement | Tasks |
|------------------|-------|
| `active.json` minimal pointer | T001, T002, T003 |
| Workflow frontmatter with steps | T012-T016 |
| INITIATIVE.md with cycles/steps | T004-T011, T017 |
| Agent can modify steps | T009, T010, T011 |
| `/brains.next` reads from INITIATIVE.md | T022 |
| `initiative status` from INITIATIVE.md | T019, T020 |
| Multiple sequential cycles | T006, T017 |
