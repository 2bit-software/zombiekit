# Tasks: Active Initiative Detection in /brains.new

**Complexity**: Simple (4 files, ~120 lines)
**Critical path**: T001 → T002 → T003 (sequential), T004 parallel with T003

## Traceability

| User Story | Tasks |
|------------|-------|
| US1 (Detect & prompt) | T004 |
| US2 (Idempotency) | No change needed — existing behavior |
| US3 (Abandon action) | T001, T002, T003 |

## Tasks

- [ ] T001 [US3] Add `AbandonResult` type to `internal/initiative/types.go`
  - Add struct with `InitiativeID string` and `DeletedPath string` fields
  - **Acceptance**: Type compiles and is importable from service package

- [ ] T002 [US3] Add `Abandon()` method to `internal/initiative/service.go`
  - Load state, check not empty, resolve init path, `os.RemoveAll`, `stateManager.Clear()`
  - Return `AbandonResult` with initiative ID and deleted path
  - Return `InitiativeError{Code: "NO_ACTIVE_INITIATIVE"}` if no active initiative
  - Tolerate missing history folder (`os.RemoveAll` is already idempotent)
  - **Depends on**: T001
  - **Acceptance**: `Abandon()` compiles, removes folder, clears state

- [ ] T003 [US3] Add `abandon` action to MCP tool at `internal/mcp/tools/initiative/tool.go`
  - Update tool description (line ~43) to mention abandon
  - Add `"abandon"` to actions enum (line ~49)
  - Add `case "abandon"` to Execute switch (line ~95-111)
  - Implement `handleAbandon(ctx, dir)` following `handleComplete` pattern
  - Return JSON with `action`, `initiative_id`, `deleted_path`, `abandoned_at`
  - **Depends on**: T002
  - **Acceptance**: MCP tool accepts `abandon` action and returns expected JSON

- [ ] T004 [P] [US3] Add tests for abandon action at `internal/mcp/tools/initiative/tool_test.go`
  - `TestHandleAbandon_Success`: Create initiative, abandon, verify state cleared + folder removed
  - `TestHandleAbandon_NoActive`: No active initiative → `NO_ACTIVE_INITIATIVE` error
  - `TestHandleAbandon_MissingHistoryFolder`: Delete folder manually, abandon still clears state
  - Follow existing test patterns (`tmpDir`, `testEmbeddedFS()`)
  - **Depends on**: T003
  - **Acceptance**: All 3 tests pass

- [ ] T005 [P] [US1] Update `embed/commands/new.md` with initiative conflict detection
  - Insert "Pre-Classification: Active Initiative Check" section before classification rules
  - Instructions: call `initiative status`, if active display details, present 3 options via `AskUserQuestion`
  - Option 1 (close out): call `initiative complete`, proceed
  - Option 2 (delete history): call `initiative abandon`, proceed
  - Option 3 (cancel): stop, suggest `/brains.next`
  - **Depends on**: T003 (abandon action must exist in schema)
  - **Acceptance**: Manual E2E — run `/brains.new` with active initiative, see prompt with 3 options

## Execution Order

```
T001 → T002 → T003 → T004 (parallel with T005)
                    → T005 (parallel with T004)
```

## Parallel Opportunities

T004 and T005 are independent once T003 is done. They can be implemented in parallel.
