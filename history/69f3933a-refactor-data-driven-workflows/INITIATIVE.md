# Initiative: data-driven-workflows

**Type**: refactor
**Status**: complete
**Created**: 2026-04-30
**ID**: 69f3933a-refactor-data-driven-workflows

## Steps

| Step | Status | Updated |
|------|--------|--------|
| analyze | complete | 2026-04-30 10:36 |
| plan | complete | 2026-04-30 10:37 |
| tasks | complete | 2026-04-30 10:38 |
| implement | complete | 2026-04-30 10:45 |

## Description

Refactor the workflow system to separate data (step sequences) from behavior (profile instructions).
Workflows become pure YAML frontmatter definitions of step order and profile mapping.
Profiles become pure instruction content with no routing awareness.
INITIATIVE.md step table gains a Profile column so /brains.next can resolve profiles deterministically.

## Goals

- Workflows own step sequencing (data)
- Profiles own step execution (behavior)
- Commands own orchestration (control flow)
- Custom workflows via user-defined markdown files — no Go changes needed

## Progress

- [x] Goal and constraints defined
- [x] Dependency analysis: 8 components, ~15 files affected
- [x] Safety net: existing test coverage mapped, gaps identified
- [x] Refactor plan: 7 atomic steps with exhaustive test specifications
- [x] Task breakdown: 20 tasks across 6 phases, 7 commits
- [x] Implementation complete — all tasks done, tests pass, build clean

## Completion

**Completed**: 2026-04-30
**Duration**: Same session

### Outcomes
- Refactor: data-driven-workflows - Complete

### Notes
- `workflow.WorkflowStep` type added with `Name` and `Profiles []string` fields
- `parseWorkflow()` now extracts `steps:` from workflow YAML frontmatter
- `ParsedStep` gained `Profile` field; parser handles 3-col and 4-col step tables
- `loadWorkflowSteps()` reads workflows first, falls back to profile frontmatter
- `createInitiativeMD()` writes 4-column table when steps have profiles
- All 5 embedded workflows populated with step sequences
- `steps:`/`handoffs:` removed from feature/bug/refactor profiles
- `next` command updated to read Profile column
- 14 new test cases covering parsing, round-trips, backwards compat
- All affected tests pass, build clean
