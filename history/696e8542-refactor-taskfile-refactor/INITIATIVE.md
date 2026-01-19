# Initiative: taskfile-refactor

**Type**: refactor
**Status**: complete
**Created**: 2026-01-19T11:25:54-08:00
**ID**: 696e8542-refactor-taskfile-refactor

## Description

Refactor the ZombieKit Taskfile from a single file into a two-file architecture, separating user-facing commands from internal development tools.

## Goals

- Improve discoverability - Users see only relevant tasks
- Reduce cognitive load - Daily tasks vs developer internals separated
- Enable future patterns - CI-aware execution, Docker-based commands
- Follow conventions - Match patterns used in production codebases

## Completion

**Completed**: 2026-01-19
**Duration**: Same day

### Outcomes

| Work Item | Status |
|-----------|--------|
| Create Taskfile.dev.yml | Complete |
| Add dev entry point task | Complete |
| Add silent: true to default | Complete |
| Rename db:up to up | Complete |
| Rename db:down to down | Complete |
| Convert test to delegation | Complete |
| Convert ci to delegation | Complete |
| Convert init to status: pattern | Complete |
| Remove migrated tasks | Complete |
| Verify task counts | Complete |
| Verify delegated tasks | Complete |
| Verify renamed lifecycle tasks | Complete |
| Verify idempotent init | Complete |

### Files Changed

- `Taskfile.yml` - Refactored from 17 to 9 user-facing tasks
- `Taskfile.dev.yml` - New file with 12 development tasks

### Acceptance Criteria Verified

- AC-1: `task` shows 9 user-facing tasks
- AC-2: `task dev` shows 12 dev tasks
- AC-3: `task dev -- fmt` formats code
- AC-6: `task test` runs tests via delegation
- AC-7: `task ci` runs CI pipeline via delegation
- AC-8: `task init` skips golangci-lint if installed
- AC-12: `db:up` and `db:down` no longer exist (breaking change)

### Notes

Clean refactor with no issues. All tasks completed and verified. Breaking change (db:up/db:down removal) is intentional per spec.
