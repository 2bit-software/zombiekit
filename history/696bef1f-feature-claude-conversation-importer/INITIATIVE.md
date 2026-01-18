# Initiative: claude-conversation-importer

**Type**: feature
**Status**: completed
**Created**: 2026-01-17T12:20:47-08:00
**ID**: 696bef1f-feature-claude-conversation-importer
**Linear Ticket**: DEV-69

## Description

Import Claude Code conversation history from `~/.claude/projects/` into the recall system for semantic search. Enables users to search past conversations and navigate to full conversation context from search results.

## Goals

- [x] Import Claude Code JSONL history files
- [x] Support duplicate detection via source tracking
- [x] Enable semantic search across conversations
- [x] Provide conversation retrieval by session ID
- [x] Support both one-time import and watch mode

## Implementation Summary

### Files Created
- `internal/database/migrations/postgres/003_recall_chunks_source_tracking.sql` - Schema migration
- `internal/recall/claude/types.go` - History entry types
- `internal/recall/claude/discovery.go` - File discovery
- `internal/recall/claude/parser.go` - JSONL parser
- `internal/recall/claude/chunker.go` - Message chunking

### Files Modified
- `internal/recall/types.go` - Extended Chunk with source tracking
- `internal/recall/storage.go` - Added SaveWithSource, ExistsBySourceID, GetByConversation
- `internal/recall/postgres/storage.go` - Implemented new methods, updated Search
- `internal/cli/recall.go` - Added watch claude and conversation commands

### Test Files Created
- `internal/recall/claude/parser_test.go` - 14 tests
- `internal/recall/claude/chunker_test.go` - 11 tests
- `internal/recall/claude/discovery_test.go` - 7 tests
- `internal/recall/claude/import_test.go` - 14 integration tests
- `internal/recall/claude/watch_test.go` - 11 integration tests
- `internal/recall/claude/e2e_test.go` - 5 E2E tests
- `internal/recall/postgres/storage_test.go` - Extended with 14 source tracking tests

### CLI Commands
- `brains recall watch claude [--once] [--path] [--project] [--verbose] [--interval]`
- `brains recall conversation <conversation-id>`

## Completion

**Completed**: 2026-01-17T13:25:00-08:00
**Duration**: ~1 hour

### Outcomes
- T001-T013: All implementation tasks - Complete
- T014-T016: Unit tests (parser, chunker, discovery) - Complete (32 tests)
- T017: Storage source tracking tests - Complete (14 tests)
- T018: Import flow integration tests - Complete (14 tests)
- T019: Watch mode integration tests - Complete (4 tests)
- T020: Conversation retrieval tests - Complete (7 tests)
- T021: E2E tests - Complete (5 tests)

**Total Tests**: 83 passing tests

### Notes
- All 21 tasks completed successfully
- Full test coverage for all business requirements (BR-001 through BR-010, excluding deferred BR-007)
- Search results now include source tracking fields for conversation navigation
