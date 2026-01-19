# Tasks: Conversation History Search GUI (DEV-70)

**Created**: 2026-01-19
**Complexity**: Medium (11 files, ~565 lines)
**Total Tasks**: 18

## Dependency Graph

```
T001 ─┬─→ T002 ─→ T003 ─→ T004 (Storage Layer)
      │
      └─→ T005 ─┬─→ T006 ─→ T007 (Plugin Scaffold)
                │
                ├─→ T008 ─→ T009 (List View)
                │
                ├─→ T010 ─→ T011 (Detail View)
                │
                ├─→ T012 ─→ T013 ─→ T014 (Search)
                │
                └─→ T015 ─→ T016 (Project Filter)

T007 + T009 + T011 + T014 + T016 ─→ T017 (Integration)

T017 ─→ T018 (Tests)
```

## Task List

### Phase 1: Storage Layer

- [x] T001 [US2] Add `ConversationSummary` type to `internal/recall/types.go`
  - Add struct with fields: ConversationID, Title, MessageCount, FirstMessage, LastMessage, Source, Project
  - Add JSON tags
  - **Acceptance**: Type compiles, matches tech spec

- [x] T002 [US2] Add `ListConversations` and `ListDistinctProjects` to storage interface `internal/recall/storage.go`
  - Add method signatures with godoc comments
  - **Acceptance**: Interface compiles

- [x] T003 [US2] Implement `ListConversations` in `internal/recall/postgres/storage.go`
  - Implement SQL with CTE for title extraction
  - Handle limit=0 default (100), offset pagination, project filter
  - **Acceptance**: Method compiles, handles edge cases (empty db, no user messages)

- [x] T004 [P] [US2] Implement `ListDistinctProjects` in `internal/recall/postgres/storage.go`
  - Simple SELECT DISTINCT query
  - **Acceptance**: Returns unique project paths sorted

---

### Phase 2: Plugin Scaffold

- [x] T005 Create `internal/webplugins/recall/plugin.go` with plugin structure
  - Package declaration, imports
  - `//go:embed templates` directive
  - Plugin struct with storage and embedder fields
  - NewPlugin constructor
  - Interface assertions for TemplatePlugin and Searchable
  - **Acceptance**: Compiles with `go build`

- [x] T006 [US2] Implement `SidebarItems()` and `MountRoutes()` in `internal/webplugins/recall/plugin.go`
  - SidebarItems returns "Conversations" item with Order=30
  - MountRoutes registers GET /, /search, /{id}
  - **Acceptance**: Routes compile, sidebar item defined

- [x] T007 [US2] Create handler scaffolding in `internal/webplugins/recall/handlers.go`
  - handlers struct wrapping storage and embedder
  - Data types: ListData, ViewData, SearchData, SearchResultGroup, PaginationData
  - Empty stub methods: list, view, search
  - Constants: DefaultPageLimit, MaxPageLimit, PageLimitOptions
  - **Acceptance**: Compiles, all types match tech spec

---

### Phase 3: Conversation List View (FR-001, FR-008)

- [x] T008 [US2] Implement `list` handler in `internal/webplugins/recall/handlers.go`
  - Parse page, limit, project query params
  - Call storage.ListConversations
  - Calculate pagination metadata
  - Render template with ListData
  - **Acceptance**: Handler returns paginated list ordered by recency

- [x] T009 [US2] Create `internal/webplugins/recall/templates/list.html`
  - Search input form with HTMX
  - Conversation cards with title, count, date, source badge
  - Pagination controls
  - Empty state message
  - **Acceptance**: Template renders, HTMX pagination works

---

### Phase 4: Conversation Detail View (FR-003)

- [x] T010 [US3] Implement `view` handler in `internal/webplugins/recall/handlers.go`
  - Extract conversation ID from chi.URLParam
  - Call storage.GetByConversation
  - Handle not found (404 with error template)
  - Calculate date range, message count
  - Render template with ViewData
  - **Acceptance**: Handler returns all messages with roles and timestamps

- [x] T011 [US3] Create `internal/webplugins/recall/templates/view.html`
  - Back link to list
  - Conversation header with metadata
  - Message list with role indicators and timestamps
  - Use `<pre>` for content formatting
  - **Acceptance**: Messages display with correct roles, timestamps

---

### Phase 5: Semantic Search (FR-002, FR-005, FR-007)

- [x] T012 [US1] Implement `search` handler in `internal/webplugins/recall/handlers.go`
  - Extract query and project params
  - Check embedder availability (nil check)
  - Generate embedding via embedder.Embed
  - Call storage.Search
  - Filter by project if active
  - Group results by conversation with snippets
  - **Acceptance**: Returns relevance-ordered results with snippets

- [x] T013 [US1] Create `internal/webplugins/recall/templates/search-results.html`
  - Back link to list
  - Search query display
  - Result cards with snippets and similarity
  - Empty state for no matches
  - Error state for embedder offline
  - **Acceptance**: Template renders search results correctly

- [x] T014 [US5] Implement `Search()` method (Searchable interface) in `internal/webplugins/recall/plugin.go`
  - Handle empty query (return empty slice)
  - Handle nil embedder (return empty slice silently)
  - Generate embedding, call storage.Search
  - Group by conversation, return SearchResults
  - URL format: /{conversationID} (framework prefixes)
  - **Acceptance**: Global search shows conversation results

---

### Phase 6: Project Filter (FR-004)

- [x] T015 [US4] Update `list` handler to fetch and pass distinct projects
  - Call storage.ListDistinctProjects
  - Pass to template in ListData.Projects
  - **Acceptance**: Project dropdown populated

- [x] T016 [US4] Add project filter dropdown to `internal/webplugins/recall/templates/list.html`
  - Select element with HTMX hx-get
  - Maintain filter across pagination
  - "All Projects" default option
  - **Acceptance**: Filter updates list, persists across pages

---

### Phase 7: Integration

- [x] T017 Register plugin in `internal/cli/gui.go`
  - Import recall web plugin package
  - Create embedder (handle unavailable case)
  - Create plugin with storage and embedder
  - Register as "recall"
  - **Acceptance**: Plugin appears in sidebar, routes work

---

### Phase 8: Testing

- [x] T018 [P] Add integration tests
  - `internal/recall/postgres/storage_test.go`: TestListConversations, TestListDistinctProjects
  - `internal/webplugins/recall/plugin_test.go`: TestSearch (Searchable interface)
  - Test edge cases: empty db, embedder offline, invalid conversation ID
  - **Acceptance**: All tests pass with `go test ./...`

---

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T001-T004, T008-T009 |
| FR-002 | T012-T013 |
| FR-003 | T010-T011 |
| FR-004 | T003-T004, T015-T016 |
| FR-005 | T012-T013 |
| FR-006 | T014, T017 |
| FR-007 | T012-T013 |
| FR-008 | T001, T008-T009 |

## Parallel Execution Opportunities

After T007 (scaffold complete):
- T008-T009 (List View)
- T010-T011 (Detail View)
- T012-T014 (Search)
- T015-T016 (Project Filter)

These 4 streams can be developed in parallel.

## Execution Order (Critical Path)

1. T001 → T002 → T003 + T004 (parallel)
2. T005 → T006 → T007
3. T008 → T009 | T010 → T011 | T012 → T013 → T014 | T015 → T016 (parallel)
4. T017
5. T018

## Notes

- All file paths are relative to project root
- Tasks marked [P] can run in parallel with their siblings
- Each task should be completable in ~15-30 minutes
- Templates reference existing CSS classes from memory plugin
