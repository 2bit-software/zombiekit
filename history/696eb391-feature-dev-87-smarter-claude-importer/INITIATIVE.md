# Initiative: dev-87-smarter-claude-importer

**Type**: feature
**Status**: complete
**Created**: 2026-01-19T14:43:29-08:00
**ID**: 696eb391-feature-dev-87-smarter-claude-importer

## Description

The conversation importer currently scans all conversations from the beginning on every startup, checking each one against the database. For users with large histories, this means hundreds of redundant "already imported" checks before reaching new content. This initiative adds import state tracking per conversation file, enabling efficient incremental imports that skip unchanged files entirely.

## Goals

- Import times proportional to new content, not total history size
- Skip unchanged files entirely via timestamp-based change detection
- Backwards scanning to locate sync point in changed files
- Auto-reconciliation with divergence markers when sync fails
- Reduced/eliminated "skipped" output noise during imports

## Progress

### 2026-01-19: Planning Complete

**Completed:**
- [x] Research existing importer architecture
- [x] Spike: Validated backwards scanning approach (forward scan is sufficient)
- [x] Created implementation-plan.md
- [x] Created technical-spec.md
- [x] Audit passed (100% FR coverage, 100% test coverage)

**Key Technical Decisions:**
- Forward single-pass for sync point (not backwards byte scanning)
- mtime + UUID tracking per file
- history_gap flag for divergence detection
- File lock for concurrent import prevention

**Next Step:** `/brains.tasks` to generate task list

### 2026-01-19: Implementation Complete

**Completed:**
- [x] All 33 tasks from implementation plan
- [x] Database migration for import state tracking
- [x] Parser extension with sync point detection
- [x] File lock mechanism for concurrent import prevention
- [x] Import logic refactor with mtime-based skip
- [x] Verbose/summary output modes
- [x] Unit and integration tests

**Commits:**
- `caca935` feat: add incremental import with mtime tracking (DEV-87)
- `cd6c050` feat: add recall:reset task and fix taskfile issues
- `6785d29` fix: use portable signal names in task up trap
- `b0613fc` fix: remove verbose flag from background importer in task up

**Bug Fixes During Development:**
- 001-sigint-lock-cleanup: Fixed zsh trap signal compatibility (INT → SIGINT)
- Removed --verbose from background importer to prevent interleaved output

## Completion

**Completed**: 2026-01-19T15:35:00-08:00
**Duration**: ~1 day

### Outcomes
- Feature: Incremental import with mtime tracking - Complete
- Feature: Sync point detection and divergence handling - Complete
- Feature: File locking for concurrent import prevention - Complete
- Feature: recall:reset task for database cleanup - Complete
- Bug: SIGINT handling in task up - Fixed
- Bug: Interleaved verbose output - Fixed

### Files Changed
- `internal/database/migrations/postgres/005_recall_import_state.sql` - NEW
- `internal/recall/types.go` - ImportState type, HistoryGap field
- `internal/recall/storage.go` - Import state interface methods
- `internal/recall/postgres/storage.go` - Storage implementation
- `internal/recall/claude/parser.go` - ParseFileFromUUID, ErrSyncPointNotFound
- `internal/recall/claude/lock.go` - NEW
- `internal/cli/recall.go` - Refactored import logic
- `internal/recall/claude/parser_test.go` - Tests
- `internal/recall/claude/lock_test.go` - NEW
- `Taskfile.yml` - recall:reset task, signal fixes
- `Taskfile.dev.yml` - recall:reset task
