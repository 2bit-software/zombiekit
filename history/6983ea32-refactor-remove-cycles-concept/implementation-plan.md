# Implementation Plan: Remove Cycles Concept

## Overview

This plan removes the cycles abstraction from initiatives. The existing codebase has these cycle-related components:

- `types.go`: `CycleType`, `CycleStatus`, `Cycle` types (lines 102-178)
- `cycle.go`: `CreateCycle`, `generateCycleID`, `getNextCycleNumber` functions
- `markdown.go`: `ParsedCycle` struct, `ActiveCycle()` method, cycle-based parsing
- MCP tool: `CreateCycle` call, `mapInitTypeToCycleType`, `findFirstCycle`, `copyTemplatesToCycle`
- Response types: `CycleID`, `CyclePath` fields

## Implementation Order

The phases are ordered for compilation safety - each phase leaves the codebase compilable and testable.

---

## Phase 1: Update Markdown Parsing (Foundation)

**Goal**: Flatten `ParsedInitiative` to use steps directly instead of nested cycles.

### 1.1 Update ParsedInitiative struct

File: `internal/initiative/markdown.go`

**Changes**:
```go
// BEFORE
type ParsedInitiative struct {
    Name    string
    Type    string
    Status  string
    Created time.Time
    Cycles  []ParsedCycle  // Remove this
}

// AFTER
type ParsedInitiative struct {
    Name    string
    Type    string
    Status  string
    Created time.Time
    Steps   []ParsedStep   // Add this - flat steps list
}
```

### 1.2 Remove ParsedCycle struct

Delete the `ParsedCycle` struct entirely.

### 1.3 Update parsing logic

**Changes to `ParseInitiativeMD()`**:
- Remove `cycleHeaderRe` regex
- Parse step table directly after `## Steps` header (not `## Cycles` with nested `###` headers)
- Store steps in `parsed.Steps` directly

**New INITIATIVE.md format to parse**:
```markdown
## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-02-04 10:00 |
| plan | pending | - |
```

### 1.4 Update navigation methods

- `ActiveCycle()` → DELETE (no longer needed)
- `CurrentStep()` → Update to iterate `p.Steps` directly
- `NextStep()` → Update to iterate `p.Steps` directly
- `UpdateStepStatus()` → Remove `cycleNum` parameter, iterate `p.Steps`
- `AddStep()` → Remove `cycleNum` parameter

### 1.5 Update WriteTo()

- Replace `formatCycle()` with `formatSteps()`
- Look for `## Steps` section instead of `## Cycles`
- Write flat step table

### 1.6 Update markdown_test.go

- Update all test markdown strings: `## Cycles` → `## Steps`
- Remove `### N. type/name (status)` cycle headers
- Update assertions to check `parsed.Steps` instead of `parsed.Cycles`

**Verification**: `go test ./internal/initiative/...` passes

---

## Phase 2: Update Initiative Service

**Goal**: Service creates flat INITIATIVE.md, no cycle folder creation.

### 2.1 Update createInitiativeMD()

File: `internal/initiative/service.go`

**Changes**:
- Change `## Cycles` section to `## Steps`
- Remove the `### 1. type/name (active)` cycle header generation
- Write step table directly under `## Steps`

### 2.2 Update StatusResult

Remove or deprecate `CycleID` and `CurrentCycle` fields from `StatusResult`:
```go
type StatusResult struct {
    Active         bool
    InitiativeID   string
    InitiativeType string
    CurrentStep    string
    StepStatus     string
    // CycleID        string   // REMOVE
    // CurrentCycle   int      // REMOVE
    StepsCompleted int
    StepsTotal     int
    AvailableDocs  []string
    SuggestedNext  string
    HistoryPath    string
    InitiativeFile string
    Files          []string
}
```

### 2.3 Update Status()

- Remove `cycleID`, `currentCycle` variables
- Replace `parsed.ActiveCycle()` with direct `parsed.Steps` iteration

### 2.4 Update service_test.go

Update tests that create/check INITIATIVE.md content.

**Verification**: `go test ./internal/initiative/...` passes

---

## Phase 3: Delete Cycle Implementation

**Goal**: Remove cycle types and CreateCycle function.

### 3.1 Delete cycle.go

Remove the entire file:
- `CreateCycle()`
- `generateCycleID()`
- `getNextCycleNumber()`

### 3.2 Delete cycle_test.go

Remove the entire test file.

### 3.3 Remove cycle types from types.go

Delete lines 102-178:
- `CycleType` type and constants
- `CycleStatus` type and constants
- `Cycle` struct and all methods

**Verification**: `go build ./internal/initiative` passes (no remaining references)

---

## Phase 4: Update MCP Tool

**Goal**: Tool no longer creates cycles, templates go directly to initiative folder.

### 4.1 Update handleCreate()

File: `internal/mcp/tools/initiative/tool.go`

**Remove**:
- `mapInitTypeToCycleType()` function
- `findFirstCycle()` function
- Call to `initSvc.CreateCycle()`

**Change**:
- `copyTemplatesToCycle()` → `copyTemplatesToInitiative()`
- Copy templates directly to `initiative.Path` instead of `cycle.Path`

### 4.2 Update CreateResponse

File: `internal/mcp/tools/initiative/types.go`

Remove:
- `CycleID` field
- `CyclePath` field

### 4.3 Update StatusResponse

Remove:
- `CycleID` field

### 4.4 Update handleCreate response construction

Remove CycleID/CyclePath assignments.

### 4.5 Update handleStatus response

Remove CycleID assignment from `initSvc.Status()` mapping.

### 4.6 Update tool_test.go

- Remove assertions for CycleID/CyclePath fields
- Update path expectations (no cycle subfolder)

**Verification**: `go test ./internal/mcp/tools/initiative/...` passes

---

## Phase 5: Update Step Service

**Goal**: Step service uses initiative folder directly.

### 5.1 Update StepResponse

File: `internal/step/types.go`

Change:
- Deprecate or remove `CycleFolder` field (initiative_folder serves the same purpose)

### 5.2 Update Execute()

File: `internal/step/service.go`

**Changes**:
- Remove `cyclePath` variable - use `historyFolder` directly
- The code already sets `cyclePath = historyFolder` on line 142, so this simplifies

### 5.3 Update UpdateState()

- Replace `cycle := parsed.ActiveCycle()` with direct `parsed.Steps` iteration
- Update steps directly without cycle indirection

### 5.4 Update service_test.go

If any tests reference cycle paths, update them.

**Verification**: `go test ./internal/step/...` passes

---

## Phase 6: Final Verification

### 6.1 Full test suite

```bash
go test ./...
```

### 6.2 Build verification

```bash
go build ./...
```

### 6.3 Manual smoke test

```bash
# Create fresh initiative and verify folder structure
brains init /tmp/test-project
cd /tmp/test-project

# Create initiative via MCP (or CLI)
# Verify: spec.md and research.md are in history/{id}/ directly
# Verify: No cycle subfolder exists
# Verify: INITIATIVE.md has ## Steps section, not ## Cycles
```

---

## Rollback Strategy

Each phase can be rolled back independently:
```bash
git checkout -- internal/initiative/markdown.go internal/initiative/markdown_test.go  # Phase 1
git checkout -- internal/initiative/service.go internal/initiative/service_test.go   # Phase 2
git checkout internal/initiative/cycle.go internal/initiative/cycle_test.go           # Phase 3
# etc.
```

Full rollback: `git checkout main`

---

## Commit Strategy

| After Phase | Commit |
|-------------|--------|
| Phase 1 | `refactor(initiative): flatten INITIATIVE.md to direct steps` |
| Phase 2 | `refactor(initiative): remove cycle fields from service` |
| Phase 3 | `refactor(initiative): delete cycle types and implementation` |
| Phase 4 | `refactor(mcp): remove cycle from initiative tool responses` |
| Phase 5 | `refactor(step): simplify to use initiative folder directly` |
| Phase 6 | `refactor: complete cycles removal - verification pass` |

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Existing initiatives won't work | Out of scope - documented as breaking change |
| Test updates missed | Each phase ends with test run |
| Hidden cycle references | `grep -r "cycle" internal/` after each phase |
| Markdown parsing breaks | Comprehensive test coverage in markdown_test.go |

---

## Files Changed Summary

| File | Action | Phase |
|------|--------|-------|
| `internal/initiative/markdown.go` | Major update | 1 |
| `internal/initiative/markdown_test.go` | Major update | 1 |
| `internal/initiative/service.go` | Medium update | 2 |
| `internal/initiative/service_test.go` | Medium update | 2 |
| `internal/initiative/cycle.go` | DELETE | 3 |
| `internal/initiative/cycle_test.go` | DELETE | 3 |
| `internal/initiative/types.go` | Remove cycle types | 3 |
| `internal/mcp/tools/initiative/tool.go` | Major update | 4 |
| `internal/mcp/tools/initiative/types.go` | Remove fields | 4 |
| `internal/mcp/tools/initiative/tool_test.go` | Update assertions | 4 |
| `internal/step/service.go` | Minor update | 5 |
| `internal/step/types.go` | Deprecate field | 5 |
