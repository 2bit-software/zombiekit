# Implementation Plan: MCP Interface Cleanup

## Overview

Remove orphaned and deprecated MCP tools, consolidate workflow commands, and clean up the codebase.

## Phase 1: Remove MCP `feature` Tool

**Files:**
- `internal/config/tools.go` - remove "feature" from KnownTools
- `internal/mcp/server.go` - remove zombiekitTool field, instantiation, registration, handler
- `internal/mcp/tools/zombiekit/` - delete entire directory

**Steps:**
1. Edit `internal/config/tools.go`: Remove `"feature"` from KnownTools slice
2. Edit `internal/mcp/server.go`:
   - Remove import: `"github.com/zombiekit/brains/internal/mcp/tools/zombiekit"`
   - Remove field: `zombiekitTool *zombiekit.Tool`
   - Remove instantiation: `zombiekitTool := zombiekit.NewTool()`
   - Remove struct assignment: `zombiekitTool: zombiekitTool,`
   - Remove registration block (lines 161-168)
   - Remove `handleFeature` method (lines 180-193)
3. Delete directory: `internal/mcp/tools/zombiekit/`

## Phase 2: Remove MCP `step` Tool

**Files:**
- `internal/config/tools.go` - remove "step" from KnownTools
- `internal/mcp/server.go` - remove stepTool field, instantiation, registration, handler
- `internal/mcp/tools/step/` - delete entire directory

**Steps:**
1. Edit `internal/config/tools.go`: Remove `"step"` from KnownTools slice
2. Edit `internal/mcp/server.go`:
   - Remove import: `steptool "github.com/zombiekit/brains/internal/mcp/tools/step"`
   - Remove field: `stepTool *steptool.Tool`
   - Remove instantiation: `stepToolInst := steptool.NewTool()`
   - Remove struct assignment: `stepTool: stepToolInst,`
   - Remove `registerStepTool()` method call
   - Remove `registerStepTool()` method
   - Remove `handleStep()` method
3. Delete directory: `internal/mcp/tools/step/`

## Phase 3: Remove Orphaned `embed/steps/`

**Files:**
- `embed/steps/` - delete entire directory
- `embed/embed.go` - remove steps embedding if present

**Steps:**
1. Delete directory: `embed/steps/`
2. Check `embed/embed.go` for steps-related embed directives and remove

## Phase 4: Remove `/brains.step` Skill

**Files:**
- `embed/workflows/step.md` - delete
- `embed/integrations/claude/commands/brains.step.md` - delete

**Steps:**
1. Delete: `embed/workflows/step.md`
2. Delete: `embed/integrations/claude/commands/brains.step.md`

## Phase 5: Enhance `/brains.next` for Backwards Navigation

**Files:**
- `embed/workflows/next.md` - add support for jumping to any step by name

**Steps:**
1. Edit `embed/workflows/next.md`:
   - Add section for explicit step navigation
   - If argument is a valid step name (spec, plan, tasks, implement, audit, clarify), jump to that step
   - Keep existing alternate path handling (audit, clarify, research)

## Phase 6: Clean Up KnownTools

**Files:**
- `internal/config/tools.go`

**Steps:**
1. Remove from KnownTools:
   - `"feature"` (done in Phase 1)
   - `"step"` (done in Phase 2)
   - `"profile-show"` (never registered)
   - `"profile-validate"` (never registered)

## Phase 7: Verify

**Steps:**
1. `go build ./...` - ensure compilation
2. `go test ./...` - ensure tests pass
3. Manual test: start MCP server, verify tool list is correct

## Risk Mitigation

- **Low risk**: All removed code is orphaned/unused
- **Rollback**: All changes are deletions, easily reverted via git

## Dependencies

| Phase | Depends On |
|-------|------------|
| 2 | 1 (clean compile between phases) |
| 3 | 2 |
| 4 | None |
| 5 | 4 (brains.step removed before enhancing next) |
| 6 | 1, 2 |
| 7 | All |
