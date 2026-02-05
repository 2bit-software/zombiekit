# Progress Log

## Implementation Plan

| Phase | Task | Status |
|-------|------|--------|
| 1 | Remove MCP `feature` tool | Complete |
| 2 | Remove MCP `step` tool | Complete |
| 3 | Remove orphaned `embed/steps/` | Complete |
| 4 | Remove `/brains.step` skill | Complete |
| 5 | Enhance `/brains.next` for step navigation | Complete |
| 6 | Clean up KnownTools | Complete |
| 7 | Verify | Complete |

---

## Phase 1 - Remove MCP `feature` tool
- Status: Complete
- Files modified:
  - `internal/config/tools.go` - removed "feature" from KnownTools
  - `internal/mcp/server.go` - removed feature tool registration and handler
- Files deleted:
  - `internal/mcp/tools/zombiekit/` (entire directory)

## Phase 2 - Remove MCP `step` tool
- Status: Complete
- Files modified:
  - `internal/config/tools.go` - removed "step" from KnownTools
  - `internal/mcp/server.go` - removed step tool registration and handler
- Files deleted:
  - `internal/mcp/tools/step/` (entire directory)

## Phase 3 - Remove orphaned `embed/steps/`
- Status: Complete
- Files modified:
  - `embed.go` - removed EmbeddedSteps declaration and initialization
  - `cmd/brains/main.go` - removed step.SetEmbeddedFS call
  - `internal/step/embedded.go` - removed SetEmbeddedFS/GetEmbeddedFS/HasEmbeddedSteps
  - `internal/step/loader.go` - removed embedded filesystem support
  - `internal/step/service.go` - removed SetEmbeddedFS method
- Files deleted:
  - `embed/steps/` (entire directory with 8 markdown files)
- Tests updated:
  - `internal/step/loader_test.go` - rewrote to test local/global only
  - `internal/step/service_test.go` - rewrote to use local step files

## Phase 4 - Remove `/brains.step` skill
- Status: Complete
- Files deleted:
  - `embed/workflows/step.md`
  - `embed/integrations/claude/commands/brains.step.md`
- Tests updated:
  - `internal/cli/init_test.go` - updated file counts (5 -> 4 command files)

## Phase 5 - Enhance `/brains.next` for step navigation
- Status: Complete
- Files modified:
  - `embed/workflows/next.md` - added explicit step navigation section

## Phase 6 - Clean up KnownTools
- Status: Complete
- Files modified:
  - `internal/config/tools.go` - removed:
    - "feature"
    - "step"
    - "profile-show" (never registered)
    - "profile-validate" (never registered)

## Phase 7 - Verify
- Status: Complete
- `go build ./...` - Success
- `go test ./...` - All tests pass

---

## Summary

### Files Deleted
- `internal/mcp/tools/zombiekit/tool.go`
- `internal/mcp/tools/zombiekit/tool_test.go`
- `internal/mcp/tools/step/tool.go`
- `internal/mcp/tools/step/tool_test.go`
- `embed/steps/audit.md`
- `embed/steps/bug.md`
- `embed/steps/clarify.md`
- `embed/steps/feature.md`
- `embed/steps/implement.md`
- `embed/steps/plan.md`
- `embed/steps/refactor.md`
- `embed/steps/tasks.md`
- `embed/workflows/step.md`
- `embed/integrations/claude/commands/brains.step.md`

### Files Modified
- `internal/config/tools.go`
- `internal/mcp/server.go`
- `embed.go`
- `cmd/brains/main.go`
- `internal/step/embedded.go`
- `internal/step/loader.go`
- `internal/step/service.go`
- `internal/step/loader_test.go`
- `internal/step/service_test.go`
- `internal/cli/init_test.go`
- `internal/mcp/tools/initiative/tool.go`
- `embed/workflows/next.md`

### Tools Removed from MCP Server
- `feature` (deprecated, replaced by initiative workflow)
- `step` (orphaned, never called by any workflow)

### KnownTools Removed
- `feature`
- `step`
- `profile-show`
- `profile-validate`
