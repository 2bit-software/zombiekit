# Implementation Plan: Conversation History Search GUI (DEV-70)

**Created**: 2026-01-19
**Status**: Draft
**Depends on**: DEV-67 (recall storage), DEV-69 (Claude import) - both Done

## Overview

Implement a web GUI for browsing and searching conversation history. The feature builds on the existing recall subsystem and follows established web plugin patterns from the memory plugin.

## Implementation Phases

### Phase 1: Storage Layer Extension

**Goal**: Add `ListConversations` method to enable browsing conversations.

**Files**:
- `internal/recall/types.go` - Add `ConversationSummary` type
- `internal/recall/storage.go` - Add `ListConversations` to interface
- `internal/recall/postgres/storage.go` - Implement the method
- `internal/recall/postgres/storage_test.go` - Add tests

**Steps**:

1. **Add ConversationSummary type** to `types.go`:
   ```go
   type ConversationSummary struct {
       ConversationID string
       Title          string    // First user message (truncated) or "[No title]"
       MessageCount   int
       FirstMessage   time.Time
       LastMessage    time.Time
       Source         string
       Project        string    // CWD from metadata
   }
   ```

2. **Extend Storage interface** in `storage.go`:
   ```go
   // ListConversations returns conversations ordered by last activity.
   // limit=0 uses default (100), offset supports pagination.
   ListConversations(ctx context.Context, limit, offset int) ([]ConversationSummary, error)
   ```

3. **Implement PostgreSQL method** using GROUP BY with subquery for title:
   - Query groups by `conversation_id`
   - Aggregates: COUNT, MIN/MAX timestamps
   - Subquery fetches first user message as title (truncated to 100 chars)
   - Orders by `last_message DESC`

4. **Add integration tests** for:
   - Empty database returns empty slice
   - Single conversation with multiple messages
   - Multiple conversations ordered by recency
   - Pagination with offset
   - Conversations without user messages (uses "[No title]")

**Validation**: Run `go test ./internal/recall/...` - all tests pass.

---

### Phase 2: Web Plugin Scaffold

**Goal**: Create the recall web plugin with basic structure.

**Files**:
- `internal/webplugins/recall/plugin.go` - Plugin struct and WebPlugin interface
- `internal/webplugins/recall/handlers.go` - HTTP handlers
- `internal/webplugins/recall/templates/` - HTML templates (embedded)
- `internal/cli/gui.go` - Plugin registration

**Steps**:

1. **Create plugin.go** with:
   - `Plugin` struct holding `recall.Storage` and `recall.Embedder`
   - `NewPlugin(storage, embedder)` constructor
   - `SidebarItems()` returning single "Conversations" item
   - `MountRoutes(r)` registering handlers
   - `Templates()` returning embedded FS
   - Interface assertions for `web.TemplatePlugin` and `search.Searchable`

2. **Create handlers.go** with:
   - `handlers` struct wrapping storage and embedder
   - Data types: `ListData`, `ViewData`, `SearchData`, `PaginationData`
   - Empty handler stubs for all routes

3. **Create template directory** with placeholder files:
   - `templates/list.html` - Conversation list
   - `templates/view.html` - Conversation detail
   - `templates/search-results.html` - Search results

4. **Register plugin** in `gui.go`:
   - Check if recall storage and embedder are available
   - Create plugin instance and register as "recall"

**Validation**: `go build ./...` succeeds, plugin appears in sidebar.

---

### Phase 3: Conversation List View (FR-001)

**Goal**: Display paginated list of conversations.

**Files**:
- `internal/webplugins/recall/handlers.go` - Implement `list` handler
- `internal/webplugins/recall/templates/list.html` - List template

**Steps**:

1. **Implement list handler**:
   - Parse `page` and `limit` query params (default: page=1, limit=20)
   - Call `storage.ListConversations(ctx, limit, offset)`
   - Calculate pagination metadata
   - Render template with conversations and pagination

2. **Create list template**:
   - Search input form (HTMX POST to `/recall/search`)
   - Conversation cards showing: title, message count, date range, source badge
   - HTMX links to detail view (`/recall/{id}`)
   - Pagination controls with page size selector

3. **Empty state**: Show "No conversations yet" message.

**Validation**: Navigate to `/recall`, see paginated list, pagination works.

---

### Phase 4: Conversation Detail View (FR-003)

**Goal**: Display full conversation with all messages.

**Files**:
- `internal/webplugins/recall/handlers.go` - Implement `view` handler
- `internal/webplugins/recall/templates/view.html` - Detail template

**Steps**:

1. **Implement view handler**:
   - Extract conversation ID from URL param
   - Call `storage.GetByConversation(ctx, conversationID)`
   - Handle not found (return 404 page)
   - Render template with all messages

2. **Create view template**:
   - Back link to list
   - Conversation header with metadata
   - Message list with role indicators (user/assistant)
   - Timestamps on each message
   - Proper whitespace/code formatting (use `<pre>` or markdown rendering)

3. **Long message handling**: Show full content (no truncation in detail view).

**Validation**: Click conversation from list, see all messages with correct roles.

---

### Phase 5: Semantic Search (FR-002, FR-005)

**Goal**: Search conversations using natural language queries.

**Files**:
- `internal/webplugins/recall/handlers.go` - Implement `search` handler
- `internal/webplugins/recall/templates/search-results.html` - Search results template
- `internal/webplugins/recall/plugin.go` - Implement `search.Searchable` interface

**Steps**:

1. **Implement search handler**:
   - Extract query and project filter from form/query params
   - Check embedder availability (return graceful error if offline)
   - Generate embedding for query
   - Call `storage.Search(ctx, embedding, limit)`
   - Filter results by project if filter is active (post-query filtering)
   - Group results by conversation (multiple chunks from same conversation)
   - Render results with snippets and similarity scores

2. **Create search-results template**:
   - Conversation cards with match snippets highlighted
   - Similarity indicator (e.g., "95% match" or subtle progress bar)
   - Links to full conversation view
   - Hidden project field to maintain filter across searches

3. **Implement Searchable interface** for global search:
   - Same logic as search handler (without project filter - global search is unfiltered)
   - Return `[]search.SearchResult` with Title=snippet, URL=/{convID}
   - Note: Framework auto-prefixes URLs with plugin name

4. **Handle embedder offline** (FR-007):
   - Check embedder health before embedding
   - Return user-friendly error: "Search unavailable - embedding service offline"

**Validation**: Search for term, see relevance-ordered results with snippets, click through to conversation.

---

### Phase 6: Project Filter (FR-004)

**Goal**: Filter conversations by project directory.

**Files**:
- `internal/recall/storage.go` - Add `ListConversationsByProject` or add `project` param
- `internal/recall/postgres/storage.go` - Implement filtering
- `internal/webplugins/recall/handlers.go` - Add filter to list handler
- `internal/webplugins/recall/templates/list.html` - Add filter dropdown

**Steps**:

1. **Extend storage interface**:
   - Option A: Add `project` param to `ListConversations`
   - Option B: Add separate `ListConversationsByProject` method
   - Decision: Add optional `project string` parameter (empty = all)

2. **Implement filter in PostgreSQL**:
   - Add WHERE clause: `metadata->>'cwd' LIKE $project || '%'`
   - Prefix match allows filtering by parent directory

3. **Add ListDistinctProjects method**:
   - Add to storage interface: `ListDistinctProjects(ctx) ([]string, error)`
   - SQL: `SELECT DISTINCT metadata->>'cwd' FROM recall_chunks WHERE metadata->>'cwd' IS NOT NULL ORDER BY 1`
   - Cache in handler or fetch on each request (low volume, acceptable)

4. **Update list handler**:
   - Parse `project` query param
   - Pass to storage
   - Fetch distinct projects for dropdown

5. **Update list template**:
   - Add project dropdown (populated from distinct CWDs)
   - Maintain filter across pagination

**Validation**: Select project filter, list shows only matching conversations.

---

### Phase 7: Global Search Integration (FR-006)

**Goal**: Conversation results appear in global search bar.

**Files**:
- `internal/webplugins/recall/plugin.go` - Already has `search.Searchable` from Phase 5

**Steps**:

1. **Verify interface implementation**:
   - Plugin already implements `search.Searchable`
   - Web server automatically discovers via type assertion

2. **Test global search**:
   - Type query in global search bar
   - Verify conversations appear under "Conversations" section
   - Click result, navigate to detail view

**Validation**: Global search shows conversation results.

---

### Phase 8: Error Handling & Polish (FR-007, Edge Cases)

**Goal**: Handle error states gracefully.

**Steps**:

1. **Database connection errors**:
   - List/view handlers catch errors and render error template
   - Include retry button

2. **Invalid conversation ID**:
   - View handler returns 404 page with back link

3. **Empty conversations**:
   - Filter out in storage query (WHERE message_count > 0)
   - Or handle in handler

4. **Long messages**:
   - Detail view: Show full content with scroll
   - List view: Truncate title to 100 chars

**Validation**: Test each error scenario, verify graceful handling.

---

## Dependency Order

```
Phase 1 (Storage)
    ↓
Phase 2 (Scaffold)
    ↓
Phase 3 (List) ─────────────────────┐
    ↓                               │
Phase 4 (Detail)                    │
    ↓                               │
Phase 5 (Search) ───────────────────┼── Can be parallelized
    ↓                               │
Phase 6 (Filter)                    │
    ↓                               │
Phase 7 (Global Search) ────────────┘
    ↓
Phase 8 (Polish)
```

Phases 3-7 can be developed in any order after Phase 2, but dependencies within each phase must be respected.

## Risk Areas

| Risk | Mitigation |
|------|------------|
| SQL performance with large datasets | Add index on conversation_id (already exists), monitor query times |
| Embedder unavailability during search | Graceful degradation with user-facing error message |
| Template complexity | Follow memory plugin patterns exactly |

## Testing Strategy

- **Unit tests**: ConversationSummary type construction
- **Integration tests**: Storage methods with test PostgreSQL
- **Handler tests**: HTTP responses using `httptest`
- **E2E tests**: Manual testing of user flows (search, browse, view)

### Required Test Cases (from spec)

| FR | Test | Type |
|----|------|------|
| FR-001 | Handler returns paginated list with HTMX compatibility | Integration |
| FR-002 | Search returns semantically matched results | Integration |
| FR-003 | Detail handler returns full conversation | Integration |
| FR-004 | List handler filters by project query param | Integration |
| FR-005 | Search results include snippets, relevance-ordered | Integration |
| FR-006 | Plugin Search method returns valid SearchResults | Unit |
| FR-007 | Search returns graceful error when embedder unavailable | Integration |

### Edge Case Tests

- Empty database → empty slice, no error
- Invalid conversation ID → 404 response
- Embedder offline → 503 with error message
- Page/limit params invalid → defaults applied

## Out of Scope (Per Spec)

- Keyword search (semantic only)
- Export functionality
- Conversation editing/deletion
- Data ingestion (handled by DEV-69)

## Known Limitations (MVP)

- **Code block rendering**: Messages display with `<pre>` for whitespace preservation but don't render markdown code blocks with syntax highlighting. Acceptable for MVP; consider markdown rendering as future enhancement.
- **"Show more" for long messages**: Detail view shows full content with scroll. Expand/collapse for >10KB messages not implemented in MVP.
