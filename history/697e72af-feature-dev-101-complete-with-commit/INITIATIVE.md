# Initiative: dev-101-complete-with-commit

**Type**: feature
**Status**: complete
**Created**: 2026-01-31T13:22:55-08:00
**ID**: 697e72af-feature-dev-101-complete-with-commit

## Description

Enhanced initiative lifecycle with Linear integration. Three features:
1. Ticket capture on creation - records Linear ticket in INITIATIVE.md
2. Commit offer on completion - offers to commit uncommitted changes
3. Linear update on completion - offers to update source ticket and mark Done

## Goals

- [x] Add ticket detection to `/brains.new` workflow
- [x] Add Source section to feature/bug/refactor profiles
- [x] Add commit offer to `/brains.complete` workflow
- [x] Add Linear update offer to `/brains.complete` workflow

## Progress

All 8 implementation tasks completed.

## Completion

**Completed**: 2026-01-31T14:45:00-08:00
**Duration**: ~1.5 hours

### Outcomes
- Feature: Ticket Capture (F1) - Complete
- Feature: Commit Offer (F2) - Complete
- Feature: Linear Update (F3) - Complete

### Files Modified
- `embed/workflows/new.md` - Added Linear ticket detection
- `embed/profiles/feature.md` - Added Source section step
- `embed/profiles/bug.md` - Added Source section step
- `embed/profiles/refactor.md` - Added Source section step
- `embed/workflows/complete.md` - Added commit and Linear update steps
- `embed/profiles/complete.md` - Synced with workflow

### Notes
All features implemented as workflow/profile changes (no Go code).
Graceful error handling for missing Linear MCP or git failures.
