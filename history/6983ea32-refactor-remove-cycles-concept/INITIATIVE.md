# Initiative: remove-cycles-concept

**Type**: refactor
**Status**: in_progress
**Created**: 2026-02-04
**ID**: 6983ea32-refactor-remove-cycles-concept

## Cycles

### 1. ref/remove-cycles-concept (active)

| Step | Status | Updated |
|------|--------|--------|
| analyze | completed | 2026-02-04 17:23 |
| plan | completed | 2026-02-04 17:31 |
| implement | in_progress | 2026-02-04 17:31 |
| verify | pending | - |

## Description

Remove the "cycles" abstraction layer from initiatives. Currently each initiative can have multiple cycles (feat/ref/fix passes), creating extra folder nesting and complexity. This refactor flattens to: initiative folder contains artifacts directly, INITIATIVE.md tracks steps without cycle headers.

## Goals

1. Flatten folder structure: `history/{init}/spec.md` instead of `history/{init}/{cycle}/spec.md`
2. Simplify INITIATIVE.md: `## Steps` section replaces `## Cycles` with nested cycle headers
3. Remove cycle types, CreateCycle function, and all cycle-related code
4. Update MCP tool responses to remove cycle_id/cycle_path fields

## Progress

- [x] Analyze phase: Documented goal, constraints, dependencies, safety net
- [x] Plan phase: Implementation plan and technical spec created
- [x] Implement phase: Execute 6-phase refactor plan
- [ ] Verify phase: Full test suite, build, manual smoke test
