# Implementation Plan: Active Initiative Detection in /brains.new

## Overview

Two parallel workstreams: (A) Go backend — add `abandon` action to initiative MCP tool, (B) Markdown — update `new.md` command with conflict detection prompt.

## Step Order

### Step 1: Add `Abandon()` to initiative service

**File**: `internal/initiative/service.go`
**Depends on**: Nothing
**FR**: FR-007

Add `Abandon()` method modeled after `Complete()` (lines 225-241):
1. Load state via `stateManager.Load()`
2. If empty, return `InitiativeError` with code `NO_ACTIVE_INITIATIVE`
3. Resolve initiative folder path: `filepath.Join(s.workDir, state.Initiative)`
4. Remove the folder via `os.RemoveAll(initPath)` — tolerate `ErrNotExist`
5. Call `stateManager.Clear()` to remove `.brains/active.json`
6. Return the initiative ID and deleted path for the response

### Step 2: Add `abandon` action to MCP tool

**File**: `internal/mcp/tools/initiative/tool.go`
**Depends on**: Step 1
**FR**: FR-007

1. Update tool definition (line 43): add "abandon" to description
2. Update enum (line 49): add `"abandon"` to actions list
3. Add `case "abandon"` to Execute switch (line 95-111)
4. Implement `handleAbandon(ctx, dir)` following `handleComplete` pattern (lines 283-323):
   - Create service, get active initiative
   - If no active: return `NO_ACTIVE_INITIATIVE` ToolError
   - Call `initSvc.Abandon()`
   - Return JSON with `initiative_id`, `deleted_path`, `abandoned_at`

### Step 3: Add tests for `abandon`

**File**: `internal/mcp/tools/initiative/tool_test.go`
**Depends on**: Step 2
**FR**: FR-007

Tests modeled after existing patterns (lines 130-294):
1. `TestHandleAbandon_Success`: Create initiative, abandon it, verify state cleared and history folder removed
2. `TestHandleAbandon_NoActive`: Call abandon with no active initiative, expect `NO_ACTIVE_INITIATIVE` error
3. `TestHandleAbandon_MissingHistoryFolder`: Create initiative, manually delete folder, abandon — should clear state without error

### Step 4: Update `new.md` command with conflict detection

**File**: `embed/commands/new.md`
**Depends on**: Steps 1-2 (abandon action must exist)
**FR**: FR-001 through FR-006

Insert a new section before "### Classification Rules" (or as a preamble step before classification):

**Logic to add:**
1. Before classification, call `mcp__zombiekit__initiative` with `action: "status"`
2. If `active: false` — proceed normally (no change)
3. If `active: true` — display initiative details and present three options via `AskUserQuestion`:
   - **Option 1: Close out early** — call `initiative complete`, then proceed with classification
   - **Option 2: Delete history** — call `initiative abandon`, then proceed with classification
   - **Option 3: Cancel** — stop execution, suggest `/brains.next`
4. Idempotency is handled downstream by the MCP tool (same name+type returns success) — no special handling needed in `new.md`

## Dependency Graph

```
Step 1 (service.Abandon)
  └── Step 2 (MCP tool handler)
        └── Step 3 (tests)
        └── Step 4 (new.md update)
```

Steps 3 and 4 are independent of each other and can be done in parallel.

## Reuse Notes

- `Abandon()` reuses the same `stateManager.Load()` / `stateManager.Clear()` pattern as `Complete()`
- `handleAbandon()` follows the exact same structure as `handleComplete()`
- Test setup reuses existing `tmpDir` + `testEmbeddedFS()` pattern
- No new dependencies or libraries needed
