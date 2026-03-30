# Initiative: mcp-idempotency

**Type**: feature
**Status**: complete
**Created**: 2026-01-17T14:59:22-08:00
**ID**: 696c144a-feature-mcp-idempotency

## Description

Make MCP `initiative create` command idempotent to prevent data loss when the same command is called multiple times. Templates should not overwrite existing files with content.

## Goals

1. Detect when active initiative matches requested name+type and return existing instead of creating duplicate
2. Skip copying template files when destination already has content
3. Provide clear response indicating whether initiative was new or existing

## Completion

**Completed**: 2026-01-17
**Duration**: Same day

### Outcomes
- Feature: Initiative idempotency - Complete (12/12 tasks)
- All acceptance criteria met
- All tests passing

### Files Changed
| File | Changes |
|------|---------|
| `internal/initiative/service.go` | Added `FindActiveByNameAndType` method |
| `internal/initiative/service_test.go` | Added `TestService_FindActiveByNameAndType` |
| `internal/mcp/tools/initiative/tool.go` | Added `findFirstCycle`, `copyTemplateIfNotExists`, modified `copyTemplatesToCycle` and `handleCreate` |
| `internal/mcp/tools/initiative/tool_test.go` | Created with unit tests and integration tests |
| `internal/mcp/tools/initiative/types.go` | Added idempotency fields to `CreateResponse` |

### Test Results
```
ok  github.com/2bit-software/zombiekit/internal/initiative              0.178s
ok  github.com/2bit-software/zombiekit/internal/mcp/tools/initiative    1.434s
```

### Notes
- Idempotency only checks active initiative (not all history) for performance
- Template protection uses `bytes.TrimSpace` to detect empty/whitespace-only files
- Response includes `already_existed`, `skipped_files`, `copied_files` for transparency
