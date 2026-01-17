# Initiative: semantic-search-foundation

**Type**: feature
**Status**: complete
**Created**: 2026-01-17T11:27:29-08:00
**Completed**: 2026-01-17T12:05:00-08:00
**ID**: 696be2a1-feature-semantic-search-foundation
**Linear Ticket**: DEV-72

## Description

RAG Core Infrastructure & CLI - Implement semantic search foundation using PostgreSQL with pgvector for vector storage and Ollama for embedding generation.

## Goals

- [x] Store text chunks with vector embeddings
- [x] Search by semantic similarity
- [x] CLI commands for save, list, search
- [x] Duplicate detection via content hash

## Completion

**Duration**: ~40 minutes

### Outcomes

| Task | Status |
|------|--------|
| T001 - pgvector migration | Complete |
| T002 - Go dependencies | Complete |
| T003 - Core types | Complete |
| T004 - Hash utility | Complete |
| T005 - Storage/Embedder interfaces | Complete |
| T006 - Ollama config fields | Complete |
| T007 - Environment variables | Complete |
| T008 - OllamaEmbedder implementation | Complete |
| T009 - PostgreSQL Storage | Complete |
| T010 - CLI command structure | Complete |
| T011 - CLI subcommand implementations | Complete |
| T012 - Integration tests | Complete |

### Files Created (8)
- `/internal/database/migrations/postgres/002_recall_chunks.sql`
- `/internal/recall/types.go`
- `/internal/recall/hash.go`
- `/internal/recall/storage.go`
- `/internal/recall/embedder.go`
- `/internal/recall/postgres/storage.go`
- `/internal/recall/postgres/storage_test.go`
- `/internal/cli/recall.go`

### Files Modified (3)
- `/go.mod` - Added ollama and pgvector dependencies
- `/internal/config/storage.go` - Ollama config fields and env vars
- `/internal/cli/root.go` - Registered recall command

### Verification
- All 12 tasks completed
- Build passes
- Integration tests pass
- Manual E2E testing verified:
  - `brains recall save` stores content
  - `brains recall list` shows chunks
  - `brains recall search` returns semantically ranked results
  - Duplicate detection works (silent skip)

### Taskfile Tasks Added
- `task db:migrate` - Run all migrations
- `task db:migrate:recall` - Run recall migration
- `task ollama:pull` - Pull nomic-embed-text model
- `task recall:demo` - Demo the feature
