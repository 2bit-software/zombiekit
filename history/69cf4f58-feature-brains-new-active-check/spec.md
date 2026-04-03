# Feature Specification: Active Initiative Detection in /brains.new

**Feature Branch**: `feat/brains-new-active-check`
**Created**: 2026-04-02
**Status**: Draft
**Input**: The `/brains.new` command should check for an active initiative and handle scenarios where one already exists and is unfinished.

## User Scenarios & Testing

### User Story 1 - Detect Active Initiative and Present Options (Priority: P1)

When a user runs `/brains.new` with a different initiative already active, the system should detect the conflict and present three resolution options instead of returning a raw error.

**Why this priority**: This is the core behavior change — without it, users hit a dead-end error message.

**Independent Test**: Run `/brains.new` with an active initiative. Verify the user sees a prompt with three options and can select one.

**Acceptance Scenarios**:

1. **Given** an active initiative exists, **When** the user runs `/brains.new` with different work, **Then** the system displays the active initiative's name, type, creation date, and current step, and presents three numbered options.
2. **Given** an active initiative exists, **When** the user selects "close out early", **Then** the system completes the active initiative and proceeds to create the new one.
3. **Given** an active initiative exists, **When** the user selects "delete history", **Then** the system removes the initiative folder and state, then creates the new one.
4. **Given** an active initiative exists, **When** the user selects "cancel", **Then** the system stops the new command and reminds the user to finish the existing work (e.g., `/brains.next`).

---

### User Story 2 - Same Initiative Idempotency (Priority: P2)

When a user runs `/brains.new` with the same name+type as the active initiative, the system should silently resume without prompting.

**Why this priority**: Prevents unnecessary friction for users who re-run the same command.

**Independent Test**: Create an initiative, then re-run `/brains.new` with the same input. Verify no conflict prompt appears.

**Acceptance Scenarios**:

1. **Given** an active initiative with name "foo" and type "feature", **When** the user runs `/brains.new` for "foo" as a feature, **Then** the system resumes the existing initiative without prompting.

---

### User Story 3 - Abandon Action in Initiative Tool (Priority: P1)

The initiative MCP tool needs an `abandon` action that clears state AND removes the history folder, supporting the "delete history" option.

**Why this priority**: Required for User Story 1's "delete history" option to work cleanly.

**Independent Test**: Create an initiative, then call `initiative abandon`. Verify `.brains/active.json` is removed and the `history/` folder is deleted.

**Acceptance Scenarios**:

1. **Given** an active initiative, **When** `abandon` is called, **Then** `.brains/active.json` is cleared and the initiative's `history/` folder is removed.
2. **Given** no active initiative, **When** `abandon` is called, **Then** the system returns a `NO_ACTIVE_INITIATIVE` error.

---

### Edge Cases

- What happens if the history folder was already manually deleted but `active.json` still points to it? — `abandon` should clear state regardless, `complete` should also tolerate missing folder.
- What happens if the user runs `/brains.new` and the active initiative's folder is corrupt (missing INITIATIVE.md)? — Present options with whatever metadata is available (at minimum the initiative ID from `active.json`).

## Implementation Scope

### Where the change goes

The conflict detection and user prompt logic goes in **`embed/commands/new.md`** — the shared command that runs before dispatching to any workflow. This means all workflow types (feature, bug, refactor, feature-light, unmanaged) get the check without duplicating logic across each workflow file.

The `abandon` action is added to the Go MCP tool at **`internal/mcp/tools/initiative/tool.go`** and the underlying service at **`internal/initiative/service.go`**.

### Files to modify

| File | Change |
|------|--------|
| `embed/commands/new.md` | Add initiative status check before classification dispatch |
| `internal/mcp/tools/initiative/tool.go` | Add `abandon` action handler |
| `internal/initiative/service.go` | Add `Abandon()` method |
| `internal/mcp/tools/initiative/tool_test.go` | Tests for `abandon` action |

### How the user "selects" an option

The prompt is presented as markdown text with three numbered options. The LLM uses `AskUserQuestion` (deferred tool) to ask the user which option they want. The user replies with a number or keyword, and the LLM proceeds accordingly.

### "Close out early" semantics

"Close out early" performs a **minimal completion**: it calls `initiative complete`, which only clears `.brains/active.json`. It does NOT run end-of-workflow steps (no summary generation, no artifact finalization). The initiative folder remains in `history/` as-is, in whatever partial state it was in. This is intentional — the user is explicitly choosing to move on.

### Idempotency matching

Idempotency uses `FindActiveByNameAndType()` in `service.go` (exact match on normalized name + exact match on type string). Name normalization: lowercase, hyphens for spaces, alphanumeric only. This is existing behavior — no changes needed.

### Concurrency

The initiative tool already uses flock-based locking via `stateManager.Lock()`. Concurrent `/brains.new` calls are serialized at the state file level. No additional locking needed.

## Requirements

### Functional Requirements

- **FR-001**: `new.md` MUST call `initiative status` before classification. If active and different from the new request, present the conflict prompt.
- **FR-002**: Conflict prompt MUST display: initiative ID, name, type, created date, and current step (from status response). If status data is partial (corrupt initiative), show what's available.
- **FR-003**: Conflict prompt MUST present three options via `AskUserQuestion`: (1) close out early, (2) delete history, (3) cancel new request
- **FR-004**: "Close out early" MUST call `initiative complete` (minimal — clears state only, no workflow finalization) then proceed with new initiative creation
- **FR-005**: "Delete history" MUST call `initiative abandon` then proceed with new initiative creation
- **FR-006**: "Cancel" MUST stop execution and tell the user to run `/brains.next` to continue existing work
- **FR-007**: The initiative MCP tool MUST support an `abandon` action that: (a) reads active initiative path from state, (b) removes the history folder via `os.RemoveAll`, (c) clears `.brains/active.json`. Returns JSON with `initiative_id`, `deleted_path`, and `abandoned_at` fields.
- **FR-008**: Same name+type idempotency MUST continue to work without prompting. Matching uses `FindActiveByNameAndType()` — exact match on normalized name + type. No changes to this path.

### Key Entities

- **Initiative**: The active work unit tracked in `.brains/active.json` and stored in `history/{id}/`
- **InitiativeState**: The pointer in `active.json` (path, started timestamp, status)

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users never see the raw `INITIATIVE_ALREADY_ACTIVE` error from the workflow — they see a prompt with options instead
- **SC-002**: All three resolution paths (close, delete, cancel) work correctly and leave the system in a consistent state
- **SC-003**: Existing idempotent creation (same name+type) continues to work without regression

## Testing Requirements

### Test Strategy

Integration tests for the `abandon` action in the Go MCP tool. The workflow markdown changes are tested via manual E2E (run `/brains.new` with an active initiative and verify the prompt).

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-007 | Integration | `abandon` action clears state and removes history folder |
| FR-007 | Integration | `abandon` with no active initiative returns error |
| FR-007 | Integration | `abandon` tolerates missing history folder |
| FR-008 | Integration | Existing idempotency test (same name+type returns success) |

### Edge Case Coverage

- Missing history folder during abandon -> clears state anyway, no error
- Corrupt initiative (missing INITIATIVE.md) -> prompt still shows available metadata
