# Initiative: dev-87-smarter-claude-importer

**Type**: feature
**Status**: active
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
