# Dependency Analysis: Files Affected by Cycles Removal

## Core Files to Modify

### internal/initiative/

| File | Impact | Changes Required |
|------|--------|------------------|
| `types.go` | HIGH | Remove `Cycle`, `CycleType`, `CycleStatus` types and constants |
| `cycle.go` | DELETE | Entire file removed - `CreateCycle`, `generateCycleID`, `getNextCycleNumber` |
| `cycle_test.go` | DELETE | All cycle-related tests |
| `markdown.go` | HIGH | Remove `ParsedCycle`, update `ParsedInitiative` to hold steps directly |
| `markdown_test.go` | HIGH | Update all tests to use flat step structure |
| `service.go` | MEDIUM | Remove cycle references from `createInitiativeMD`, `Status`, `findAvailableDocs` |
| `service_test.go` | MEDIUM | Update tests that expect cycle tracking |

### internal/mcp/tools/initiative/

| File | Impact | Changes Required |
|------|--------|------------------|
| `tool.go` | HIGH | Remove `CreateCycle` call, `findFirstCycle`, `mapInitTypeToCycleType`, update `handleCreate` |
| `types.go` | MEDIUM | Remove `CycleID`, `CyclePath` from `CreateResponse` and `StatusResponse` |
| `tool_test.go` | MEDIUM | Update tests for flat structure responses |

### internal/step/

| File | Impact | Changes Required |
|------|--------|------------------|
| `types.go` | LOW | Remove `CycleFolder` field from `StepResponse` |
| `service.go` | LOW | Remove `cyclePath` variable (use `historyFolder` directly) |
| `service_test.go` | LOW | Update tests if they reference cycle paths |

### embed/profiles/

| File | Impact | Changes Required |
|------|--------|------------------|
| `feature.md` | REVIEW | Check if frontmatter references cycles |
| `bug.md` | REVIEW | Check if frontmatter references cycles |
| `refactor.md` | REVIEW | Check if frontmatter references cycles |

## Dependency Graph

```
MCP initiative tool
       │
       ├── initiative.Service.Create()
       │      │
       │      └── createInitiativeMD() ── generates INITIATIVE.md
       │
       ├── initiative.Service.CreateCycle() ← REMOVE
       │      │
       │      └── generateCycleID() ← REMOVE
       │
       └── copyTemplatesToCycle() ← rename to copyTemplatesToInitiative()

step.Service.Execute()
       │
       ├── stateManager.Load() ── gets initiative path
       │
       ├── initiative.ParseInitiativeMD() ← UPDATE (no cycles)
       │
       └── resolveFiles() ── uses initiative folder directly
```

## Test Impact Summary

| Test File | Tests to Delete | Tests to Update |
|-----------|-----------------|-----------------|
| `cycle_test.go` | ALL (5 tests) | - |
| `markdown_test.go` | - | 10 tests (cycle → step structure) |
| `service_test.go` | - | 3-4 tests (cycle references) |
| `tool_test.go` | - | 2-3 tests (response fields) |
| `step/service_test.go` | - | 1-2 tests (cycle path) |

## Files with No Changes Required

- `internal/initiative/state.go` - Only tracks initiative pointer
- `internal/initiative/errors.go` - Generic error types
- `internal/profile/*` - Profiles don't reference cycles
- `internal/step/loader.go` - Step loading is cycle-agnostic
- `internal/workflow/*` - Workflow composition doesn't use cycles
