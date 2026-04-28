# Tasks: Contextual /brains.help

## Complexity: Simple (4 files, ~230 lines)

## Tasks

### Go Prerequisites (Steps 1-2, parallelizable)

- [ ] T001 [P] [FR-1,FR-3] Add StepStatus, StepsCompleted, StepsTotal fields to StatusResponse struct
  - File: `internal/mcp/tools/initiative/types.go` (lines 34-45)
  - Add 3 fields after `CurrentStep`: `StepStatus string`, `StepsCompleted int`, `StepsTotal int`
  - Use `json:"step_status,omitempty"`, `json:"steps_completed,omitempty"`, `json:"steps_total,omitempty"`
  - AC: Struct compiles with new fields

- [ ] T002 [P] [FR-1,FR-3] Add field mappings in handleStatus() response construction
  - File: `internal/mcp/tools/initiative/tool.go` (lines 284-295)
  - Add 3 lines mapping from `status.StepStatus`, `status.StepsCompleted`, `status.StepsTotal`
  - AC: `go build ./internal/mcp/tools/initiative/...` passes

- [ ] T003 [P] [FR-3] Replace findAvailableDocs() with directory scan
  - File: `internal/initiative/service.go` (lines 392-411)
  - Replace hardcoded `knownDocs` list with `os.ReadDir` scan
  - Filter: all `.md` files except `INITIATIVE.md` (use `InitiativeMDFile` constant)
  - Keep `contracts/` directory check
  - Follow pattern from `internal/step/loader.go:loadAllFromDir()`
  - No sort needed — `os.ReadDir` returns sorted entries
  - AC: `go test ./internal/initiative/...` passes; new .md files in initiative dir appear in `available_docs`

- [ ] T004 [FR-1,FR-3] Verify Go changes with tests
  - Run: `go test ./internal/initiative/... ./internal/mcp/tools/initiative/...`
  - Verify existing `TestService_Status` still passes
  - Verify `initiative status` MCP call now returns `step_status`, `steps_completed`, `steps_total`
  - AC: All tests pass, new fields present in JSON output
  - Depends on: T001, T002, T003

### Help Command Rewrite (Step 3)

- [ ] T005 [FR-1,FR-2,FR-3,FR-4,FR-5,FR-6] Rewrite help.md with state-aware instructions
  - File: `embed/commands/help.md`
  - Preserve frontmatter (name: help, description unchanged)
  - Replace execution steps and output templates with:
    1. Instructions to call `mcp__zombiekit__initiative` with `action: "status"`
    2. Branch on `active` field
    3. No-initiative mode: commands, examples, recent initiatives (up to 5 via `action: "list"`)
    4. Active mode: header, progress, step context, artifacts, source ticket, filtered actions
  - Embed step lookup tables (feature/bug/refactor step names and descriptions)
  - Embed command filtering rules (no-initiative vs mid-workflow)
  - Remove all references to `/brains.step` (does not exist)
  - Include instructions to read `initiative_file` for Source section (FR-6)
  - Include edge case handling (no .brains dir, missing INITIATIVE.md, all steps done, unknown type)
  - AC: Matches output mockups in business-spec.md
  - Depends on: T004 (needs new StatusResponse fields available)

### Validation (Step 4)

- [ ] T006 Manual test: `/brains.help` with no active initiative
  - Complete current initiative or test in a clean state
  - Verify: shows command examples, recent initiatives, no step-specific content
  - AC: Output matches no-initiative mockup in business-spec.md
  - Depends on: T005

- [ ] T007 Manual test: `/brains.help` with active initiative
  - Run with current initiative active
  - Verify: shows correct step list (feature: spec/plan/tasks/implement), current step marked, artifacts listed, filtered commands
  - AC: Output matches active-initiative mockup in business-spec.md
  - Depends on: T005

## Dependency Graph

```
T001 ──┐
T002 ──┼── T004 ── T005 ── T006
T003 ──┘                 └── T007
```

T001, T002, T003 are parallelizable.
T004 gates T005.
T006 and T007 are parallelizable after T005.

## Execution Order

1. T001 + T002 + T003 (parallel)
2. T004 (verify)
3. T005 (bulk of work)
4. T006 + T007 (parallel validation)

## Traceability

| FR | Tasks |
|----|-------|
| FR-1 (State Detection) | T001, T002, T004, T005 |
| FR-2 (No-Initiative Mode) | T005, T006 |
| FR-3 (Active-Initiative Mode) | T001, T002, T003, T004, T005, T007 |
| FR-4 (Workflow-Type Awareness) | T005, T007 |
| FR-5 (Command Filtering) | T005, T006, T007 |
| FR-6 (Source Ticket) | T005, T007 |
| FR-7 (Deferred) | — |
