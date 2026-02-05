# Initiative: mcp-interface-audit

**Type**: refactor
**Status**: completed
**Created**: 2026-02-04
**ID**: 6983e225-refactor-mcp-interface-audit

## Cycles

### 1. ref/mcp-interface-audit (completed)

| Step | Status | Updated |
|------|--------|--------|
| analyze | completed | 2026-02-04 16:45 |
| plan | completed | 2026-02-04 17:02 |
| implement | completed | 2026-02-04 17:45 |
| verify | completed | 2026-02-04 17:48 |

## Source

**Linear Ticket**: [DEV-94](https://linear.app/heinsight/issue/DEV-94/audit-the-mcp-interface-and-remove-olddeprecatedunused-functions)
**Title**: audit the MCP interface and remove old/deprecated/unused functions

## Description

Audit the MCP interface and remove old/deprecated/unused functions. Per the ticket: "I think we still have 'feature' as a tool? isn't that replaced by a 'step:feature'?"

## Goals

1. Remove deprecated MCP `feature` tool and `zombiekit/` package
2. Remove orphaned MCP `step` tool (workflows use `profile-compose` directly)
3. Remove orphaned `embed/steps/` directory (only used by MCP step tool)
4. Remove `/brains.step` skill (consolidate into `/brains.next`)
5. Clean up `KnownTools` list (remove feature, step, profile-show, profile-validate)
6. Enhance `/brains.next` to support backwards navigation

## Progress

### Analysis Complete (2026-02-04)

**Architecture Discovery:**
- All user commands (`/brains.new`, `/brains.step`, `/brains.next`) use `profile-compose` → `embed/profiles/`
- MCP `step` tool uses `internal/step/Loader` → `embed/steps/` but NOTHING calls it
- The `embed/steps/*.md` files reference the MCP `step` tool (circular, orphaned)

**Removal List:**

| Item | Status | Action |
|------|--------|--------|
| MCP `feature` tool | Deprecated | Remove from server.go |
| `internal/mcp/tools/zombiekit/` | Unused | Delete directory |
| MCP `step` tool | Orphaned | Remove from server.go |
| `internal/mcp/tools/step/` | Orphaned | Delete directory |
| `embed/steps/` | Orphaned | Delete directory |
| `/brains.step` skill | Redundant | Delete workflow + command |
| `profile-show` in KnownTools | Never registered | Remove |
| `profile-validate` in KnownTools | Never registered | Remove |
| `feature` in KnownTools | Being removed | Remove |
| `step` in KnownTools | Being removed | Remove |

**Keep (still used):**
- `internal/step/` package - used by initiative tool, GUI, git ops (NOT the MCP tool)

## Completion

**Completed**: 2026-02-04 17:50
**Duration**: ~1.5 hours

### Outcomes
- ✓ Removed MCP `feature` tool and `internal/mcp/tools/zombiekit/` package
- ✓ Removed MCP `step` tool and `internal/mcp/tools/step/` package
- ✓ Removed orphaned `embed/steps/` directory (8 markdown files)
- ✓ Removed `/brains.step` skill (workflow + command)
- ✓ Cleaned up KnownTools (removed feature, step, profile-show, profile-validate)
- ✓ Enhanced `/brains.next` to support explicit step navigation

### Files Changed
- 14 files deleted
- 12 files modified
- All tests passing
- Build successful
