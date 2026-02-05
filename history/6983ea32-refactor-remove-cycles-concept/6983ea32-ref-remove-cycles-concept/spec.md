---
status: draft
---

# Refactor Specification: Remove Cycles Concept

**Branch**: `6983ea32-refactor-remove-cycles-concept`
**Created**: 2026-02-04
**Status**: Draft
**Type**: Refactor

## Problem Statement

The current initiative system has a "cycles" abstraction layer where each initiative can contain multiple cycles (feat/ref/fix passes). This creates:

1. **Extra folder nesting**: `history/{init}/{cycle}/spec.md` instead of `history/{init}/spec.md`
2. **Complex state tracking**: INITIATIVE.md tracks cycles within cycles
3. **Code complexity**: Multiple types, functions, and tests dedicated to cycle management
4. **Mental overhead**: Users must understand "initiative > cycle > step" hierarchy

In practice, the multi-cycle feature is unused. If work needs to pivot (e.g., refactor after a feature), users should complete the current initiative and start a new one.

## Refactor Goals

1. **Flatten the folder structure**: Artifacts live directly in initiative folder
2. **Simplify INITIATIVE.md**: Steps tracked directly, no cycle headers
3. **Remove cycle code**: Delete types, functions, and tests related to cycles
4. **Maintain public interfaces**: MCP tools continue to work (minus cycle-specific fields)

## Behavior Changes

### Before (Current)

```
history/
  abc123-feature-user-auth/
    INITIATIVE.md          # Contains "## Cycles" section
    def456-feat-user-auth/ # Cycle folder
      spec.md
      plan.md
      tasks.md
      audit/
```

### After (Target)

```
history/
  abc123-feature-user-auth/
    INITIATIVE.md          # Contains "## Steps" section (flat)
    spec.md                # Directly in initiative folder
    plan.md
    tasks.md
    audit/
```

## Requirements

### FR-001: Initiative Creation

System MUST create initiative folders with artifacts at the root level (no cycle subfolder).

### FR-002: INITIATIVE.md Format

System MUST generate INITIATIVE.md with a flat `## Steps` section:

```markdown
## Steps

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-02-04 10:00 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
```

### FR-003: Step Execution

System MUST execute steps using the initiative folder as the working directory.

### FR-004: MCP Response Simplification

CreateResponse and StatusResponse MUST NOT include `cycle_id` or `cycle_path` fields.

### FR-005: Template Copying

System MUST copy templates (spec.md, research.md) directly to the initiative folder.

## Non-Functional Requirements

- **NFR-001**: All existing tests must pass (after updating expectations)
- **NFR-002**: No backwards compatibility layer - this is a breaking change
- **NFR-003**: Existing initiatives in history/ are not migrated

## Testing Requirements

### Test Strategy

Update existing tests to expect flat structure. No new tests required - this removes functionality.

### FR to Test Mapping

| FR | Test File | Changes |
|----|-----------|---------|
| FR-001 | `service_test.go` | Update Create test to verify flat structure |
| FR-002 | `markdown_test.go` | Update all tests to parse flat Steps section |
| FR-003 | `step/service_test.go` | Update Execute tests for initiative folder |
| FR-004 | `mcp/.../tool_test.go` | Remove cycle field assertions |
| FR-005 | `mcp/.../tool_test.go` | Update template path assertions |

## Success Criteria

- [ ] `go test ./...` passes
- [ ] `go build ./...` succeeds
- [ ] New initiative creates flat folder structure
- [ ] Step execution works without cycle folder
- [ ] INITIATIVE.md displays flat step table
