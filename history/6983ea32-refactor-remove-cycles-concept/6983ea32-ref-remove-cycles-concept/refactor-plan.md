# Refactor Plan: Remove Cycles Concept

## Overview

This refactor removes the cycles abstraction layer from initiatives. Each step is atomic and independently committable.

---

## Phase 1: Remove Cycle Types (internal/initiative/types.go)

### Step 1.1: Delete Cycle Types

Remove from `types.go`:
- `CycleType` type and constants (`CycleFeat`, `CycleRef`, `CycleFix`)
- `CycleStatus` type and constants
- `Cycle` struct
- Associated `IsValid()` and `String()` methods

**Verification**: `go build ./internal/initiative` should fail (expected - dependent code not yet updated)

---

## Phase 2: Remove Cycle Implementation

### Step 2.1: Delete cycle.go

Delete the entire file:
- `internal/initiative/cycle.go`

### Step 2.2: Delete cycle_test.go

Delete the entire test file:
- `internal/initiative/cycle_test.go`

**Verification**: Files removed, build still fails (expected)

---

## Phase 3: Update Markdown Parsing (internal/initiative/markdown.go)

### Step 3.1: Simplify ParsedInitiative

Replace cycle-based structure:

**Before**:
```go
type ParsedInitiative struct {
    Cycles []ParsedCycle
}
```

**After**:
```go
type ParsedInitiative struct {
    Steps []ParsedStep
}
```

Remove:
- `ParsedCycle` struct
- `cycleHeaderRe` regex
- `ActiveCycle()` method

Update:
- `ParseInitiativeMD()` to parse steps directly from a flat table
- `CurrentStep()` to work with `Steps` directly (no cycle indirection)
- `NextStep()` to work with `Steps` directly
- `UpdateStepStatus()` to update steps directly
- `AddStep()` to add to flat steps list
- `WriteTo()` to write flat step table
- `formatCycle()` → `formatSteps()`

### Step 3.2: Update INITIATIVE.md Format

**Before**:
```markdown
## Cycles

### 1. feat/user-auth (active)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 |
```

**After**:
```markdown
## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 |
```

### Step 3.3: Update markdown_test.go

Update all tests to expect flat step structure instead of cycles.

**Verification**: `go test ./internal/initiative` passes

---

## Phase 4: Update Initiative Service (internal/initiative/service.go)

### Step 4.1: Update createInitiativeMD()

Remove the Cycles section generation. Generate a flat Steps section instead.

### Step 4.2: Update Status()

Remove cycle-related variables:
- `cycleID`
- `currentCycle`

Update to read steps directly from `ParsedInitiative.Steps`.

### Step 4.3: Remove CreateCycle Call Site

The MCP tool calls `CreateCycle`. Update `service.go` if it has any remaining cycle references.

### Step 4.4: Update service_test.go

Update tests to not expect cycle fields in status results.

**Verification**: `go test ./internal/initiative` passes

---

## Phase 5: Update MCP Tool (internal/mcp/tools/initiative/)

### Step 5.1: Update tool.go handleCreate()

Remove:
- `CreateCycle` call
- `mapInitTypeToCycleType()` function
- `findFirstCycle()` function

Change:
- `copyTemplatesToCycle()` → `copyTemplatesToInitiative()` (copies to initiative folder)

### Step 5.2: Update types.go

Remove from `CreateResponse`:
- `CycleID`
- `CyclePath`

Remove from `StatusResponse`:
- `CycleID`

### Step 5.3: Update tool_test.go

Remove assertions for cycle fields in responses.

**Verification**: `go test ./internal/mcp/tools/initiative` passes

---

## Phase 6: Update Step Service (internal/step/)

### Step 6.1: Update types.go

Remove from `StepResponse`:
- `CycleFolder` field

### Step 6.2: Update service.go

Remove `cyclePath` variable - use `historyFolder` directly for all operations.

Update `UpdateState()` to work with flat step structure.

### Step 6.3: Update service_test.go

Update any tests that reference cycle paths.

**Verification**: `go test ./internal/step` passes

---

## Phase 7: Final Verification

### Step 7.1: Full Test Suite

```bash
go test ./...
```

### Step 7.2: Build Verification

```bash
go build ./...
```

### Step 7.3: Manual Smoke Test

```bash
# Create a fresh initiative
brains init /tmp/test-project
cd /tmp/test-project
# Via MCP or CLI, create initiative and verify folder structure
```

---

## Rollback Strategy

Each step modifies a limited set of files. If a step fails:

1. `git diff` to see changes
2. `git checkout -- <file>` to revert specific files
3. Re-run tests to confirm rollback

If the entire refactor needs to be abandoned:

```bash
git checkout main
```

---

## Commit Points

| After Phase | Commit Message |
|-------------|----------------|
| Phase 2 | `refactor(initiative): remove cycle.go and cycle types` |
| Phase 3 | `refactor(initiative): flatten INITIATIVE.md structure` |
| Phase 4 | `refactor(initiative): update service for flat steps` |
| Phase 5 | `refactor(mcp): remove cycle fields from initiative tool` |
| Phase 6 | `refactor(step): remove cycle folder from step service` |
| Phase 7 | `refactor: remove cycles concept - complete` |
