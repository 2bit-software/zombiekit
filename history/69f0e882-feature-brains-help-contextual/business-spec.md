# Business Spec: Contextual /brains.help

## Problem

The `/brains.help` command currently renders a static markdown template regardless of workflow state. It shows placeholder data, lists all commands indiscriminately, and provides no step-specific guidance. Users (both human and AI agents) don't get useful answers to the fundamental questions: "Where am I?", "What can I do?", and "What should I do next?"

## Desired Outcome

A state-aware help command that dynamically renders useful, contextual information based on the current initiative state.

## Functional Requirements

### FR-1: State Detection

The help command MUST call `mcp__zombiekit__initiative` with `action: "status"` to determine the current state before rendering output. The output format MUST differ based on the `active` field.

### FR-2: No-Initiative Mode

When `active: false`, the help command MUST show:
- Available commands with brief descriptions (see command list in FR-5)
- How to start new work (with examples showing auto-classification)
- Recent initiatives (via `mcp__zombiekit__initiative` with `action: "list"`) so users can see history

### FR-3: Active-Initiative Mode

When `active: true`, the help command MUST show:
- **Initiative header**: ID (parsed for display name), type, history path
- **Progress visualization**: Step list with current position marked, progress fraction (e.g., "2/4")
- **Artifact status**: Which files exist in the initiative directory (from `available_docs` and `files` fields), with relative paths
- **Available actions**: Commands filtered to what's relevant NOW, with exact invocation syntax
- **Step context**: One-line description of what the current step is trying to achieve

### FR-4: Workflow-Type Awareness

The step context and progress display MUST use the correct steps per workflow type:

| Workflow | Steps (in order) |
|----------|-----------------|
| **feature** | spec → plan → tasks → implement |
| **bug** | investigate → plan → tasks → fix → verify |
| **refactor** | analyze → plan → tasks → implement |

Note: `feature-light` and `unmanaged` initiatives are created with `initiative_type: "feature"`. They share the feature step table but may have fewer artifacts. The help command should not special-case them.

### FR-5: Command Filtering by State

Available commands:

| Command | Description |
|---------|-------------|
| `/brains.new [desc]` | Start new work (auto-detects feature/bug/refactor) |
| `/brains.next` | Advance to next step in workflow |
| `/brains.complete` | Finish current initiative |
| `/brains.help` | Show this help |

Filtering rules:
- **No initiative**: Show `/brains.new` as primary. Show `/brains.help`. Hide `/brains.next` and `/brains.complete`.
- **Mid-workflow**: Show `/brains.next` as primary. Show `/brains.complete` as secondary. Show `/brains.new` with note that it will close current initiative. Show `/brains.help`.

### FR-6: Source Ticket Integration

If the initiative's INITIATIVE.md contains a `## Source` section (from Linear ticket integration), the help command should read `initiative_file` and parse the Source section to surface the ticket reference and link. This requires reading INITIATIVE.md directly since `initiative status` doesn't include this field.

### FR-7: Progressive Disclosure (Deferred)

Out of scope for v1. Single output mode only. May revisit in a future initiative.

## Prerequisite: MCP StatusResponse Update

The `StatusResponse` struct in `internal/mcp/tools/initiative/types.go` is missing fields that the internal `StatusResult` already computes. The following fields must be added to `StatusResponse` and populated in `handleStatus()`:

| Field | Type | Source |
|-------|------|--------|
| `step_status` | string | `StatusResult.StepStatus` |
| `steps_completed` | int | `StatusResult.StepsCompleted` |
| `steps_total` | int | `StatusResult.StepsTotal` |

The `findAvailableDocs()` function in `internal/initiative/service.go` must be updated to scan for all workflow-specific artifacts, not just the hardcoded generic list. Replace the hardcoded list with a broader scan of all `.md` files in the initiative directory.

## Output Mockup: Active Initiative

```
## brains-help-contextual

**Type**: feature | **Progress**: 1/4 | **Path**: history/69f0e882-feature-brains-help-contextual/

### Progress

    spec        in_progress  <-- current
    plan        pending
    tasks       pending
    implement   pending

**Current step**: spec — Research and write business specification

### Artifacts

    business-spec.md                    (exists)
    technical-requirements-research.md  (exists)
    research-summary.md                 (exists)
    INITIATIVE.md                       (exists)

### Available Actions

    /brains.next        Advance to plan step
    /brains.complete    Finish initiative (3 steps remaining)
    /brains.help        Show this help
    /brains.new [desc]  Start new work (closes current initiative)
```

## Output Mockup: No Initiative

```
## ZombieKit Help

No active initiative.

### Start New Work

    /brains.new add user authentication     (auto-detects: feature)
    /brains.new fix login timeout           (auto-detects: bug)
    /brains.new refactor auth module        (auto-detects: refactor)

### Recent Initiatives

    69f0e882  feature  brains-help-contextual  in_progress
    696ec645  feature  unified-startup         complete

### Other Commands

    /brains.help    Show this help
```

## Acceptance Criteria

- [ ] Running `/brains.help` with no active initiative shows general help with command examples
- [ ] Running `/brains.help` with no active initiative shows recent initiatives from `initiative list`
- [ ] Running `/brains.help` mid-feature shows feature steps (spec/plan/tasks/implement) with current marked
- [ ] Running `/brains.help` mid-bug shows bug steps (investigate/plan/tasks/fix/verify) with current marked
- [ ] Running `/brains.help` mid-refactor shows refactor steps (analyze/plan/tasks/implement) with current marked
- [ ] Progress fraction is displayed (e.g., "1/4")
- [ ] Available actions list only shows commands valid for current state
- [ ] Artifact paths use the relative paths from `files` field
- [ ] Linear ticket reference is surfaced when Source section exists in INITIATIVE.md
- [ ] Output uses consistent `##`/`###` headers and exact command invocations

## Out of Scope

- New MCP tools or endpoints
- Changes to workflow step definitions
- Interactive help (help is read-only, never modifies state)
- Help for non-brains commands (profiles, skills, etc.)
- Progressive disclosure / verbose mode (deferred to future)
- `/brains.step` command (does not exist, removed from help output)

## Resolved Decisions

1. **`/brains.step`**: Removed — command does not exist in the codebase
2. **Progressive disclosure**: Deferred to future — single output mode for v1
3. **Step guidance depth**: One-line description per step
4. **MCP scope**: Small Go changes allowed — add missing fields to StatusResponse, expand findAvailableDocs

## Open Questions

1. **Initiative listing**: How many recent initiatives to show? Default proposal: up to 5, most recent first. The `initiative list` response may not include timestamps for sort ordering — needs verification during planning.
2. **Source section parsing**: Should the help command always read INITIATIVE.md for the Source section, or only when a heuristic suggests it might exist? Simplest: always read it since we already have `initiative_file` path.
