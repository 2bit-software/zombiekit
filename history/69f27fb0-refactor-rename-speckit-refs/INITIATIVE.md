# Initiative: rename-speckit-refs

**Type**: refactor
**Status**: complete
**Created**: 2026-04-29
**ID**: 69f27fb0-refactor-rename-speckit-refs

## Steps

| Step | Status | Updated |
|------|--------|--------|
| analyze | complete | 2026-04-29 15:01 |
| plan | complete | 2026-04-29 15:01 |
| tasks | complete | 2026-04-29 15:02 |
| implement | complete | 2026-04-29 15:03 |

## Description

Replace all "speckit" / "spec-kit" / ".specify" references in active template and profile files (`embed/templates/`, `embed/profiles/`) with correct "brains" or "zombiekit" equivalents. Scope: 4 files, 14 occurrences.

## Goals

- Zero speckit references in active embedded markdown
- Correct mapping of old commands to brains workflow equivalents
- No behavioral changes — purely documentation/commentary updates

## Progress

- [x] Dependency analysis complete — 4 files, 14 occurrences identified
- [x] Command mapping established (speckit.X → brains.next step progression)
- [x] Refactor plan written with exact old→new replacements
- [x] Implementation complete — all 14 references replaced, build passes

## Completion

**Completed**: 2026-04-29
**Duration**: Same session

### Outcomes
- Refactor: rename-speckit-refs - Complete (4 files, 14 replacements)

### Notes
- All `/speckit.*` command references replaced with brains workflow equivalents
- `.specify/` path reference updated to `embed/workflows/`
- `speckit` tool name references updated to `zombiekit`/`brains`
- `go build ./...` passes, zero remaining speckit references in embed/
