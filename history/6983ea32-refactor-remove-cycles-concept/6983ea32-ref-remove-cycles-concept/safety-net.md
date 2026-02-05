# Safety Net Assessment: Test Coverage Before Refactoring

## Existing Test Coverage

### initiative/ Package

| File | Coverage | Notes |
|------|----------|-------|
| `cycle_test.go` | Good | 5 tests covering cycle creation, ID generation, numbering - **will be deleted** |
| `markdown_test.go` | Good | 11 tests for INITIATIVE.md parsing - **needs update for flat structure** |
| `service_test.go` | Moderate | Tests Create, List, GetActive, Status - **needs update** |
| `state_test.go` | Good | Tests state persistence - **no changes needed** |

### step/ Package

| File | Coverage | Notes |
|------|----------|-------|
| `service_test.go` | Moderate | Tests step execution, prerequisites - **minor updates** |
| `loader_test.go` | Good | Tests step definition loading - **no changes needed** |

### mcp/tools/initiative/

| File | Coverage | Notes |
|------|----------|-------|
| `tool_test.go` | Moderate | Tests MCP tool actions - **needs response field updates** |

## Coverage Gaps to Address Before Refactoring

### Critical: None

The refactor removes functionality rather than adding it. All existing tests that need to pass will be updated to reflect the new structure.

### Recommended: Add Integration Test

Consider adding a test that exercises the full flow:
1. Create initiative
2. Execute steps (feature → plan → tasks → implement)
3. Verify file locations at each stage
4. Complete initiative

This would catch any path resolution issues.

## Test Strategy During Refactor

1. **Delete cycle_test.go first** - These tests test code being removed
2. **Update markdown_test.go** - Change cycle-based assertions to flat structure
3. **Update service_test.go** - Remove cycle expectations from Create/Status
4. **Update tool_test.go** - Remove CycleID/CyclePath from response assertions
5. **Run `go test ./...` after each file change** - Catch regressions early

## Rollback Safety

The git branch `6983ea32-refactor-remove-cycles-concept` provides rollback capability. If tests fail unexpectedly:

```bash
git checkout main
git branch -D 6983ea32-refactor-remove-cycles-concept
```

## Pre-Refactor Checklist

- [x] All existing tests pass on main branch
- [x] Test coverage mapped to affected files
- [x] No coverage gaps that would hide regressions
- [x] Rollback strategy documented
