# Progress Log: MCP Command Idempotency

**Initiative:** 696c144a-feature-mcp-idempotency
**Status:** Complete

## Phase 1: Independent Additions

### T001 - Add FindActiveByNameAndType to service.go
- Status: Complete
- Files: `internal/initiative/service.go`
- Notes: Added method that checks if active initiative matches name+type using existing normalizeName()

### T002 - Add copyTemplateIfNotExists to tool.go
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool.go`
- Notes: Added helper function using bytes.TrimSpace for whitespace detection

### T003 - Add response fields to types.go
- Status: Complete
- Files: `internal/mcp/tools/initiative/types.go`
- Notes: Added AlreadyExisted, SkippedFiles, CopiedFiles fields to CreateResponse

## Phase 2: Unit Tests

### T004 - Add unit tests for FindActiveByNameAndType
- Status: Complete
- Files: `internal/initiative/service_test.go`
- Notes: 5 test cases covering no active, different name, different type, exact match, name normalization

### T005 - Add unit tests for copyTemplateIfNotExists
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool_test.go`
- Notes: 6 test cases covering file doesn't exist, has content, empty, whitespace-only, write error, minimal content

## Phase 3: Integration Changes

### T006 - Add findFirstCycle helper
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool.go`
- Notes: Scans for cycle folder by checking for spec.md or research.md

### T007 - Modify copyTemplatesToCycle signature
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool.go`
- Notes: Changed to return (skipped, copied []string, err error)

### T008 - Modify handleCreate idempotency flow
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool.go`
- Notes: Restructured flow to check matching active first, then different active (error), then create new

### T009 - Run existing tests
- Status: Complete
- Notes: All tests pass

## Phase 4: Integration Tests

### T010 - TestHandleCreate_Idempotent
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool_test.go`
- Notes: Verifies same name+type returns existing initiative, preserves spec.md content

### T011 - TestHandleCreate_DifferentInitiativeActive and TestHandleCreate_SameNameDifferentType
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool_test.go`
- Notes: Verifies error when different initiative active, and when same name but different type

### T012 - TestHandleCreate_AfterComplete
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool_test.go`
- Notes: Verifies new initiative created after completing previous (added time.Sleep for timestamp uniqueness)

---

## Files Changed

| File | Changes |
|------|---------|
| `internal/initiative/service.go` | Added `FindActiveByNameAndType` method |
| `internal/initiative/service_test.go` | Added `TestService_FindActiveByNameAndType` |
| `internal/mcp/tools/initiative/tool.go` | Added `findFirstCycle`, `copyTemplateIfNotExists`, modified `copyTemplatesToCycle` and `handleCreate` |
| `internal/mcp/tools/initiative/tool_test.go` | Created with unit tests and integration tests |
| `internal/mcp/tools/initiative/types.go` | Added idempotency fields to `CreateResponse` |

## Test Results

```
ok      github.com/zombiekit/brains/internal/initiative                 0.178s
ok      github.com/zombiekit/brains/internal/mcp/tools/initiative       1.434s
```

## Suggested Next Command

`/brains.complete` to mark initiative as complete
