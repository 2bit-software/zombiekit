# Reuse Audit

## Summary
- Duplicates: 1 (directory scan pattern from step loader)
- Overlaps: 1 (StatusResult already computes the fields)
- Related: 2 (help.md structure, profile resolver pattern)
- No match: 0

## Findings

### Item 1: Add 3 fields to StatusResponse — NONE

No reuse opportunity. The struct needs new fields; this is a straightforward schema addition.

### Item 2: Add field mappings to handleStatus() — OVERLAP

**Existing**: `internal/initiative/service.go:282-295` (StatusResult already has StepStatus, StepsCompleted, StepsTotal)
**Decision**: Extend existing — the MCP handler already maps 8 fields from StatusResult to StatusResponse. Adding 3 more follows the identical pattern.
**Plan change**: None needed — plan already describes this approach.

### Item 3: Replace findAvailableDocs() — DUPLICATE pattern

**Existing**: `internal/step/loader.go:107-130` (`loadAllFromDir()`)
**Pattern**: `os.ReadDir()` → filter by `.md` suffix → skip directories. Exact same logic needed.
**Decision**: Copy pattern from step loader. Do NOT import `sort` — the step loader doesn't sort either, and the scan order is consistent enough for this use case.
**Plan change**: Remove `sort.Strings(available)` from the plan. Follow existing codebase pattern.

### Item 4: Rewrite help.md — RELATED

**Related**: Current `embed/commands/help.md` (lines 49-121) already has the two-mode structure (no initiative / active initiative). `determineSuggestedNext()` in service.go mirrors the "suggested actions" logic.
**Note**: Rewrite should preserve the command name and description from frontmatter. The two-mode conditional structure is sound — just needs dynamic data instead of static templates.

## Plan Changes

1. **Step 2**: Remove `sort.Strings(available)` — follow existing `loadAllFromDir` pattern from step loader, which doesn't sort. Remove `sort` import requirement.
