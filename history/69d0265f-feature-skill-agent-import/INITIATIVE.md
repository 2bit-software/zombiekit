# Initiative: skill-agent-import

**Type**: feature
**Status**: complete
**Created**: 2026-04-03
**ID**: 69d0265f-feature-skill-agent-import

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | complete | 2026-04-03 |
| plan | complete | 2026-04-03 |
| tasks | complete | 2026-04-03 |
| implement | complete | 2026-04-03 |

## Description

Import Claude Code skills and agents into zombiekit's profile system with optional shim generation.

## Completion

**Completed**: 2026-04-03
**Duration**: 1 day

### Outcomes
- Feature: skill-agent-import — Complete (all 4 steps)
  - 2 new MCP tools: `skill-import-list` and `skill-import`
  - Discovery of Claude skills and agents with shim exclusion
  - Import with frontmatter transformation (skill + agent)
  - Optional shim generation back to Claude locations
  - 12 new tests, all passing

### Files Created
- `internal/skill/discover.go` — Discovery logic
- `internal/skill/importskill.go` — Import, transformation, shim generation
- `internal/mcp/tools/skillimport/tool.go` — MCP tool handlers
- `internal/skill/discover_test.go` — 5 discovery tests
- `internal/skill/importskill_test.go` — 7 import tests

### Files Modified
- `internal/skill/install.go` — Added `IsShim()` helper
- `internal/mcp/server.go` — Registered new tools
