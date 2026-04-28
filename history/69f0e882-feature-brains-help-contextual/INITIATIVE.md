# Initiative: brains-help-contextual

**Type**: feature
**Status**: complete
**Created**: 2026-04-28
**ID**: 69f0e882-feature-brains-help-contextual

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-28 10:04 |
| plan | completed | 2026-04-28 10:25 |
| tasks | completed | 2026-04-28 10:45 |
| implement | completed | 2026-04-28 10:52 |

## Description

Augment the `/brains.help` command to provide rich, contextual information based on current workflow state — what step the user is on, what commands are available, what artifacts exist, and what to do next.

## Goals

- Make `/brains.help` state-aware (shows different info depending on active initiative, current step)
- Surface available commands and their relevance to current state
- Show artifact status and paths
- Provide genuinely useful guidance rather than a static command list

## Progress

- [x] Initiative created, spec phase started
- [x] Business spec written, audited, revised (2 audit agents, 1 revision cycle)
- [x] Implementation plan created with reuse audit
- [x] Tasks generated (7 tasks, simple complexity)
- [x] Go prerequisite changes (StatusResponse fields, findAvailableDocs scan)
- [x] help.md rewritten with state-aware instructions
- [x] Build verified, all tests pass

## Completion

**Completed**: 2026-04-28
**Duration**: Same day (spec through implement)

### Outcomes

- Feature: contextual /brains.help — Complete
  - StatusResponse now includes step_status, steps_completed, steps_total
  - findAvailableDocs scans all .md files instead of hardcoded list
  - help.md renders state-aware output with progress, artifacts, filtered commands

### Files Changed

- `internal/mcp/tools/initiative/types.go` — Added 3 fields to StatusResponse
- `internal/mcp/tools/initiative/tool.go` — Added 3 field mappings in handleStatus()
- `internal/initiative/service.go` — Replaced findAvailableDocs() with os.ReadDir scan
- `embed/commands/help.md` — Full rewrite with state-aware rendering instructions
