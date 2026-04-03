# Tasks: Graphite Stack Branching

**Complexity**: Medium (15 files, ~400 LOC, 3 Go packages + 6 markdown files)

## Phase 1: Graphite Detection (hook layer)

- [ ] T001 [P] [US2] Create `internal/hook/graphite.go` with `DetectGraphiteStatus()`, `isGraphiteAvailable()`, `isGraphiteInitialized()`, `isGraphiteTracked()` functions
  - **Accept**: `DetectGraphiteStatus("/path/to/repo")` returns correct status string for each of the 4 states
  - **FR**: FR-001, FR-002, FR-003

- [ ] T002 [P] [US2] Create `internal/hook/graphite_test.go` with unit tests for all detection functions
  - **Accept**: Tests pass for `.graphite/` dir present/absent, PATH lookup, and tracked/untracked states. Skip graphite-specific tests when `gt` not available.
  - **FR**: FR-001, FR-002, FR-003

- [ ] T003 [US2] Modify `internal/hook/handler.go` — call `DetectGraphiteStatus(event.CWD)` in `handleSessionStart()` and append result to `bodies` slice before `FormatOutput()`
  - **Accept**: Startup hook output includes graphite status line in `<system-reminder>` tags
  - **Depends on**: T001
  - **FR**: FR-003a

## Phase 2: GitService Graphite Branch Creation (step layer)

- [ ] T004 [P] [US1] Add `isGraphiteAvailable()` method to `GitService` in `internal/step/git.go` — `exec.LookPath("gt")`
  - **Accept**: Returns true when `gt` in PATH, false otherwise
  - **FR**: FR-008

- [ ] T005 [US1,US6] Add `createBranchGraphite(branchName string) error` method to `GitService` in `internal/step/git.go` — runs `gt create <branchName> --no-interactive` with `cmd.Dir = g.workDir`
  - **Accept**: Creates a graphite-tracked branch in a graphite-initialized repo
  - **Depends on**: T004
  - **FR**: FR-008

- [ ] T006 [US1,US4,US5,US6] Add `EnsureBranchGraphite(initType, name string) (method, warning string, err error)` method to `GitService` in `internal/step/git.go` implementing the full flow: graceful degradation → format name → branch exists check (with `gt track` for existing non-tracked branches) → graphite create → fallback to git
  - **Accept**: Returns `("graphite", "", nil)` on success, `("git", "<warning>", nil)` on fallback, `("", "", nil)` on graceful degradation
  - **Depends on**: T004, T005
  - **FR**: FR-008, FR-009

- [ ] T007 [P] [US1,US6] Add graphite tests to `internal/step/git_test.go` — `TestGitService_EnsureBranchGraphite_FallbackWhenNoGraphite`, `TestGitService_EnsureBranchGraphite_GracefulDegradation`, `TestGitService_IsGraphiteAvailable`
  - **Accept**: All tests pass. Fallback test confirms `method == "git"` when graphite unavailable.
  - **Depends on**: T006
  - **FR**: FR-008, FR-009, FR-012

## Phase 3: Initiative Tool Parameter (MCP layer)

- [ ] T008 [P] [US1] Add `BranchingMethod string` and `BranchingWarning string` fields to `CreateResponse` in `internal/mcp/tools/initiative/types.go` with `json:"branching_method,omitempty"` and `json:"branching_warning,omitempty"` tags
  - **Accept**: JSON response includes new fields when set, omits when empty
  - **FR**: FR-010, FR-011

- [ ] T009 [US1] Add `getBoolArg()` helper and `use_graphite` boolean property to tool `Definition()` InputSchema in `internal/mcp/tools/initiative/tool.go`
  - **Accept**: `initiative create` accepts `use_graphite` boolean parameter. `getBoolArg` extracts bool from args map.
  - **FR**: FR-007

- [ ] T010 [US1,US4,US5,US6] Modify `createNewInitiative()` in `internal/mcp/tools/initiative/tool.go` — add `useGraphite bool` parameter, conditional branching logic calling `EnsureBranchGraphite()` vs `EnsureBranch()`, populate `BranchingMethod` and `BranchingWarning` in response
  - **Accept**: `initiative create` with `use_graphite: true` returns `branching_method: "graphite"` (when graphite available) or `branching_method: "git"` with warning on fallback. Without `use_graphite`, behavior unchanged.
  - **Depends on**: T006, T008, T009
  - **FR**: FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013

- [ ] T011 [US1,US5] Add tests to `internal/mcp/tools/initiative/tool_test.go` — test `use_graphite` parameter extraction, response field population, and idempotent path (empty branching_method)
  - **Accept**: Tests verify `getBoolArg`, response JSON marshaling with new fields, and idempotent case
  - **Depends on**: T010
  - **FR**: FR-007, FR-010, FR-013

## Phase 4: Workflow Markdown Changes

- [ ] T012 [US1,US4] Add "Graphite Stacking Detection" section to `embed/commands/new.md` between "Branch Check" and "Classification Task" — keyword detection (stack:, use graphite, gt stack, graphite stack), anti-keyword detection (no stack, no graphite, git branch), implicit stacking from startup hook "stacked" signal, uninitialized repo handling (ask about `gt init`), `USE_GRAPHITE: true` metadata append
  - **Accept**: `new.md` detects stacking intent from keywords, implicit signal, or anti-keywords and appends/omits `USE_GRAPHITE: true` metadata accordingly
  - **Depends on**: T003, T010
  - **FR**: FR-004, FR-004a, FR-005, FR-005a, FR-006

- [ ] T013 [P] [US1] Add `USE_GRAPHITE` metadata parsing to Step 1 in `embed/workflows/feature.md` — when `USE_GRAPHITE: true` is in arguments, pass `use_graphite: true` to `initiative create` call
  - **Accept**: `feature.md` passes `use_graphite: true` when metadata present
  - **Depends on**: T012
  - **FR**: FR-005

- [ ] T014 [P] [US1] Add `USE_GRAPHITE` metadata parsing to Step 1 in `embed/workflows/feature-light.md` — same as T013
  - **Accept**: Same as T013
  - **Depends on**: T012

- [ ] T015 [P] [US1] Add `USE_GRAPHITE` metadata parsing to Step 1 in `embed/workflows/bug.md` — same as T013
  - **Accept**: Same as T013
  - **Depends on**: T012

- [ ] T016 [P] [US1] Add `USE_GRAPHITE` metadata parsing to Step 1 in `embed/workflows/refactor.md` — same as T013
  - **Accept**: Same as T013
  - **Depends on**: T012

- [ ] T017 [P] [US1] Add `USE_GRAPHITE` metadata parsing to Step 1 in `embed/workflows/unmanaged.md` — same as T013 but note: unmanaged.md has custom branch type inference logic; USE_GRAPHITE passthrough must work alongside it
  - **Accept**: `unmanaged.md` passes `use_graphite: true` when metadata present, existing custom branch logic unchanged
  - **Depends on**: T012

## Phase 5: Verification

- [ ] T018 Run `go build ./...` and `go test ./...` to verify compilation and all tests pass
  - **Accept**: Zero compilation errors, all existing + new tests pass
  - **Depends on**: T001-T011

## Dependency Graph

```
T001 ─────┐
T002 [P]  ├── T003 ──────────────────────┐
          │                               │
T004 [P] ─┤                               │
          ├── T005 ── T006 ── T007 [P]    │
T008 [P] ─┤                  │            │
T009 ─────┤                  │            │
          └── T010 ── T011   │            │
                      │      │            │
                      └──────┼── T012 ────┤
                             │     │      │
                             │     ├── T013 [P]
                             │     ├── T014 [P]
                             │     ├── T015 [P]
                             │     ├── T016 [P]
                             │     └── T017 [P]
                             │
                             └── T018
```

**Critical path**: T004 → T005 → T006 → T010 → T012 → T013-T017

**Parallel opportunities**:
- T001 + T002 + T004 + T008 can all start simultaneously
- T013-T017 are all parallelizable (identical changes across workflow files)

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T001, T002 |
| FR-002 | T001, T002 |
| FR-003 | T001, T002 |
| FR-003a | T003 |
| FR-004 | T012 |
| FR-004a | T012 |
| FR-005 | T012, T013 |
| FR-005a | T012 |
| FR-006 | T012 |
| FR-007 | T009, T010, T011 |
| FR-008 | T004, T005, T006, T007, T010 |
| FR-009 | T006, T007, T010 |
| FR-010 | T008, T010, T011 |
| FR-011 | T008, T010, T011 |
| FR-012 | T007, T010 |
| FR-013 | T010, T011 |

All FRs covered. No orphan tasks.

## Execution Order

**Round 1** (parallel): T001, T002, T004, T008, T009
**Round 2**: T003, T005
**Round 3**: T006
**Round 4** (parallel): T007, T010
**Round 5**: T011, T012
**Round 6** (parallel): T013, T014, T015, T016, T017
**Round 7**: T018
