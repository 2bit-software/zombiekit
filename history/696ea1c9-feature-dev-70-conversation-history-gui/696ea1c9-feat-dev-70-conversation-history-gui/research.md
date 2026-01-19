---
status: complete
updated: 2026-01-19
---

# Research: Conversation History Search GUI (DEV-70)

## Executive Summary

The ZombieKit web platform already has a mature HTMX-based plugin architecture with search integration. The recall subsystem provides semantic search via PostgreSQL/pgvector but lacks a `ListConversations` API needed for browsing. The GUI can be implemented as a new web plugin following established patterns from the memory plugin.

## Findings

### Codebase Context

**Web Architecture:**
- Go chi v5 router with HTMX frontend (no React/Vue - server-rendered partials)
- Plugin architecture: `WebPlugin` interface with `SidebarItems()` and `MountRoutes()`
- Template rendering: Go `html/template` with embedded FS
- Existing plugins: `memory`, `profiles` - both provide list/view/search patterns
- Global search bar aggregates results from all `search.Searchable` plugins
- Pagination pattern: query params `?page=X&limit=Y` with server-side slicing

**Recall Subsystem (DEV-67/69 deliverables):**
- Storage: PostgreSQL + pgvector (768-dim embeddings via Ollama)
- Table: `recall_chunks` with `conversation_id`, `source`, `source_id`, `metadata` (JSONB)
- Metadata fields: `role`, `timestamp`, `git_branch`, `cwd`, `parent_id`
- Source tracking: Deduplication via `(source, source_id)` unique constraint
- Available APIs:
  - `Search(embedding, limit) -> []SearchResult` (semantic search)
  - `GetByConversation(conversationID) -> []Chunk` (single conversation)
  - `List(limit) -> []Chunk` (raw chunks, no grouping)
  - `ExistsBySourceID(source, sourceID) -> bool`

**Gap Identified:**
- No `ListConversations()` method to enumerate distinct conversations
- No conversation-level metadata (title, message count, date range, source breakdown)
- CLI `recall list` shows raw chunks, not conversations

**Existing Search Integration:**
- `search.Searchable` interface: `Search(query, maxResults, sortOrder) -> []SearchResult`
- `SearchResult`: `Title`, `Description`, `URL` fields
- Plugins implementing this appear in global search results
- Memory plugin uses in-memory filtering (not semantic search)

### Domain Knowledge

**Conversation Display Patterns (industry standard):**
- List view: Show conversations with title/summary, participant indicator (user/assistant ratio), timestamp, source badge
- Detail view: Threaded messages with role indicators, timestamps, expandable content
- Search: Keyword highlighting in snippets, relevance scores optional

**Semantic vs Keyword Search:**
- Recall system does semantic (embedding) search - good for "what did we discuss about X?"
- Users also expect exact text search - "find messages containing 'TypeScript'"
- Both are valuable; semantic is more powerful but slower

**Conversation Grouping:**
- Claude Code uses `session_id` as conversation boundary
- Conversations can span days (persistent session)
- Natural title = first user message or working directory

## Decision Points

- [x] **D1**: Plugin architecture - Follow existing `WebPlugin` pattern (no alternative considered)
- [ ] **D2**: Search type - Semantic only vs hybrid (semantic + keyword)
- [ ] **D3**: Conversation listing - Add storage method vs aggregate in handler
- [ ] **D4**: Conversation metadata - Compute on-demand vs denormalize into new table
- [ ] **D5**: Project scope selector - Query param filter vs separate views

## Recommendations

1. **Add `ListConversations()` to storage interface** - Required for browsing; SQL `GROUP BY conversation_id` with aggregates

2. **Implement as web plugin** following memory plugin patterns exactly - proven architecture, consistent UX

3. **Start with semantic search only** - Already working; keyword search is enhancement

4. **Compute conversation metadata on-demand** (P1) - Avoid schema changes; optimize with materialized view later if slow

5. **Project filter via query param** - `?project=/path/to/project` matches `metadata->>'cwd'` prefix

## Sources

- `internal/webplugins/memory/` - Reference plugin implementation
- `internal/recall/storage.go` - Storage interface
- `internal/recall/postgres/storage.go` - PostgreSQL implementation
- `internal/web/search.go` - Global search integration
- `internal/cli/recall.go` - CLI patterns for recall operations
- Linear DEV-67, DEV-69 - Dependency tickets (both Done)
