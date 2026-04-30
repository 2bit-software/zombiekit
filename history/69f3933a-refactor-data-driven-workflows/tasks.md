# Tasks: data-driven-workflows

**Input**: `refactor-plan.md`
**Complexity**: Medium (15 files, 5 Go packages, sequential critical path)

## Phase 1: Regression Baselines (no code changes)

- [ ] T001 [P] Add `TestParseInitiativeMD_ThreeColumnBaseline` in `internal/initiative/markdown_test.go`
  - Write a 3-column INITIATIVE.md to temp file, parse it, assert steps parsed correctly
- [ ] T002 [P] Add `TestGetWorkflowSteps_CurrentBehavior` in `internal/step/service_test.go`
  - Set up embedded profile with `steps:` frontmatter, call GetWorkflowSteps, assert returns steps
- [ ] T003 [P] Add `TestCreateInitiativeMD_CurrentFormat` in `internal/initiative/service_test.go`
  - Create initiative with steps, read INITIATIVE.md back, assert 3-column table format

**Checkpoint**: All existing + new tests pass. No behavior changed.

---

## Phase 2: Workflow Steps Parsing

- [ ] T004 Add `WorkflowStep` type and `Steps` field to `Workflow` struct in `internal/workflow/service.go`
  - Type: `WorkflowStep{Name string, Profiles []string}` with yaml tags
  - Add `Steps []WorkflowStep` field to `Workflow` struct
- [ ] T005 Update `parseWorkflow()` in `internal/workflow/service.go` to parse `steps:` from frontmatter
  - Extend the YAML unmarshal struct to include `Steps []WorkflowStep`
  - Populate `wf.Steps` from parsed frontmatter
- [ ] T006 [P] Add workflow step parsing tests in `internal/workflow/service_test.go`
  - `TestParseWorkflow_WithSteps`: workflow with steps → Steps populated
  - `TestParseWorkflow_WithoutSteps`: no steps field → Steps is nil
  - `TestParseWorkflow_MultipleProfilesPerStep`: profiles array with 2+ entries
  - `TestParseWorkflow_EmptyStepsArray`: `steps: []` → empty slice
  - `TestService_Load_WithSteps`: integration via Load()

**Checkpoint**: Workflow files can declare step sequences. Nothing reads them yet.

---

## Phase 3: INITIATIVE.md 4-Column Parser

- [ ] T007 Add `Profile` field to `ParsedStep` in `internal/initiative/markdown.go`
- [ ] T008 Add 4-column regex and detection logic in `internal/initiative/markdown.go`
  - New regex `stepRowRe4Col` for `| Step | Profile | Status | Updated |`
  - Detect format by checking if header row contains "Profile"
  - `parseStepRow()` handles both 3-col and 4-col
- [ ] T009 Update `formatSteps()` in `internal/initiative/markdown.go`
  - If any step has non-empty Profile → write 4-column format
  - Otherwise → write 3-column format (backwards compat)
- [ ] T010 [P] Add exhaustive parser tests in `internal/initiative/markdown_test.go`
  - `TestParseInitiativeMD_FourColumnTable`: parse 4-col, Profile field populated
  - `TestParseInitiativeMD_ThreeColumnTable_BackwardsCompat`: 3-col still works, Profile empty
  - `TestParsedInitiative_FormatSteps_WithProfiles`: 4-col output
  - `TestParsedInitiative_FormatSteps_WithoutProfiles`: 3-col output
  - `TestParsedInitiative_RoundTrip_FourColumn`: write → parse → equal
  - `TestParsedInitiative_RoundTrip_ThreeColumn`: write → parse → equal
  - `TestParseStepRow_FourColumn`: individual row
  - `TestParseStepRow_ThreeColumn`: individual row (regression)
  - `TestParsedInitiative_UpdateStepStatus_WithProfile`: preserves Profile
  - `TestParsedInitiative_WriteTo_PreservesProfileColumn`: full round trip

**Checkpoint**: Parser handles both formats. Write detects which to use automatically.

---

## Phase 4: Wire Workflow→Initiative Pipeline

- [ ] T011 Rewrite `loadWorkflowSteps()` in `internal/mcp/tools/initiative/tool.go`
  - Use `workflow.NewServiceForSubdir(dir, "workflows")` instead of `step.NewService(dir)`
  - Load workflow by type name, extract Steps, convert to `initiative.WorkflowStep`
  - Join multiple profiles with comma for Profile field
- [ ] T012 Update `createInitiativeMD()` in `internal/initiative/service.go`
  - Write 4-column step table when steps have Profile values
  - Format: `| %s | %s | %s | %s |` (Name, Profile, Status, Updated)
- [ ] T013 [P] Add integration tests for the new pipeline
  - `TestLoadWorkflowSteps_FromWorkflowFile` in tool_test.go
  - `TestLoadWorkflowSteps_NoWorkflowFile` → returns nil, nil
  - `TestLoadWorkflowSteps_WorkflowWithoutSteps` → returns nil, nil
  - `TestCreateInitiativeMD_WritesProfileColumn` in service_test.go
  - `TestCreateInitiativeMD_Integration_RoundTrip` — create → parse → verify

**Checkpoint**: Initiative creation reads steps from workflow files, writes 4-column table.

---

## Phase 5: Populate Workflow Frontmatter

- [ ] T014 [P] Add `steps:` frontmatter to `embed/workflows/feature.md`
  - Steps: spec→[feature], plan→[plan], tasks→[tasks], implement→[implement]
- [ ] T015 [P] Add `steps:` frontmatter to `embed/workflows/bug.md`
  - Steps: investigate→[bug], plan→[plan], tasks→[tasks], fix→[implement], verify→[audit]
- [ ] T016 [P] Add `steps:` frontmatter to `embed/workflows/refactor.md`
  - Steps: analyze→[refactor], plan→[plan], tasks→[tasks], implement→[implement]
- [ ] T017 [P] Add `steps:` frontmatter to `embed/workflows/feature-light.md` and `unmanaged.md`
- [ ] T018 [P] Add `TestEmbeddedWorkflows_HaveSteps` in `internal/workflow/service_test.go`
  - Load each embedded workflow, verify Steps non-empty, verify profiles exist

**Checkpoint**: `go build ./...` passes. End-to-end: create initiative → 4-column table with correct profiles.

---

## Phase 6: Command and Cleanup

- [ ] T019 Update `embed/commands/next.md` — add Profile column reading instructions
  - "Read Profile column from step table row"
  - "If missing (legacy): use step name as profile"
  - "If comma-separated: compose all profiles"
- [ ] T020 Remove `steps:` and `handoffs:` from `embed/profiles/{feature,bug,refactor}.md` frontmatter
  - Keep all instruction body content intact
  - Only strip the routing metadata from YAML frontmatter

**Checkpoint**: Full end-to-end working. Profiles are pure instructions, workflows own sequencing.

---

## Dependencies

```
T001-T003: Parallel (no deps)
T004-T005: Sequential (T005 depends on T004)
T006: Parallel with T004-T005 (tests can be written alongside)
T007-T009: Sequential (T008 depends on T007, T009 depends on T008)
T010: Parallel with T007-T009
T011-T012: Sequential after T005 and T009 (depends on both features existing)
T013: After T011-T012
T014-T018: Parallel, after T005 (workflow parsing must exist)
T019-T020: After T013 (pipeline must work before cleanup)
```

## Critical Path

T004 → T005 → T011 → T012 → T013 → T019 → T020

## Summary

- **Total tasks**: 20
- **Parallel opportunities**: T001-T003, T006, T010, T014-T018
- **Commits**: 7 (one per plan step)
- **Test cases**: ~25 new tests
