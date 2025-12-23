# Tasks: Web GUI Search Bar

**Input**: Design documents from `/specs/011-webgui-search/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/search-api.md

**Tests**: Tests are included as specified in plan.md (Go testing with testify)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## User Story Mapping

| Story | Spec Priority | Description |
|-------|---------------|-------------|
| US1   | P1            | Search Across Plugins |
| US2   | P1            | Debounced Search Input |
| US3   | P1            | Navigate to Search Result |
| US4   | P2            | Search Result Display |
| US5   | P2            | Empty State and Loading |
| US6   | P3            | Keyboard Navigation |

**Note**: US1, US2, US3 are tightly coupled (all P1, core search flow) and will be implemented together as the foundational search feature. US4-US6 build on top.

---

## Phase 1: Setup

**Purpose**: Project initialization - no new files needed, existing structure is sufficient

- [x] T001 Verify existing search interface in internal/search/search.go matches requirements
- [x] T002 Verify HTMX is loaded in internal/web/templates/shell.html

---

## Phase 2: Foundational (Core Search Backend)

**Purpose**: Backend search aggregation that ALL user stories depend on

**⚠️ CRITICAL**: No UI work can begin until this phase is complete

- [x] T003 Create PluginSearchResult and SearchResponse types in internal/web/search.go
- [x] T004 Implement search aggregation function in internal/web/search.go that iterates PluginRegistry.All() and type-asserts to Searchable
- [x] T005 Add getPluginLabel helper in internal/web/search.go to extract human-readable name from SidebarItems()[0].Label
- [x] T006 Add search handler function in internal/web/search.go following pattern from research.md
- [x] T007 Register GET /search route in internal/web/server.go setupRouter()
- [x] T008 [P] Create search results template in internal/web/templates/search-results.html per contracts/search-api.md

**Checkpoint**: `curl "http://localhost:8080/search?q=test"` returns HTML results

---

## Phase 3: User Story 1+2+3 - Core Search Flow (Priority: P1) 🎯 MVP

**Goal**: Working search bar with debounce that displays results and navigates on click

**Independent Test**: Type a query, see results from memory/profiles plugins, click a result, verify navigation works and URL updates

### Tests for Core Search Flow

- [x] T009 [P] [US1] Write TestSearchAggregation in internal/web/search_test.go - test multiple plugins, 3 result limit, empty query
- [x] T010 [P] [US1] Write TestSearchHandler in internal/web/search_test.go - test HTTP endpoint returns HTML
- [x] T011 [P] [US3] Write TestSearchResultURLPrefixing in internal/web/search_test.go - verify PrefixURL applied to results

### Implementation for Core Search Flow

- [x] T012 [US1] Add search bar HTML to header in internal/web/templates/shell.html with HTMX attributes per research.md
- [x] T013 [US2] Configure hx-trigger="keyup changed delay:300ms" on search input in internal/web/templates/shell.html
- [x] T014 [US1] Add search results dropdown container (id="search-results") in internal/web/templates/shell.html
- [x] T015 [US3] Add HTMX navigation attributes to search result links (hx-get, hx-target, hx-push-url) in internal/web/templates/search-results.html
- [x] T016 [US1] Style search bar and dropdown with Tailwind classes in internal/web/templates/shell.html
- [x] T017 [US3] Add hx-on:htmx:after-swap handler to close dropdown after navigation in internal/web/static/js/app.js

**Checkpoint**: Full search-to-navigation flow works - type query, see results, click result, content updates, URL changes, dropdown closes

---

## Phase 4: User Story 4 - Search Result Display (Priority: P2)

**Goal**: Results show title with plugin source indicator, grouped by plugin

**Independent Test**: Search returns results from multiple plugins, each group has header with plugin name

### Implementation for User Story 4

- [x] T018 [US4] Ensure search-group-header displays PluginName in internal/web/templates/search-results.html
- [x] T019 [US4] Add CSS classes for visual grouping (borders, spacing) in internal/web/templates/search-results.html
- [x] T020 [US4] Add text truncation (truncate class) for long titles in internal/web/templates/search-results.html

**Checkpoint**: Multi-plugin search shows grouped results with clear visual separation

---

## Phase 5: User Story 5 - Empty State and Loading (Priority: P2)

**Goal**: Loading indicator during search, appropriate messages for empty/no-results states

**Independent Test**: Type query and observe loading spinner; search with no matches shows message; empty input shows nothing

### Implementation for User Story 5

- [x] T021 [US5] Add hx-indicator attribute to search input pointing to loading element in internal/web/templates/shell.html
- [x] T022 [US5] Create loading spinner element (hidden by default, shown via htmx-request class) in internal/web/templates/shell.html
- [x] T023 [US5] Add "No results found" template section in internal/web/templates/search-results.html (already in contract)
- [x] T024 [US5] Handle empty query in search handler - return empty response in internal/web/search.go

**Checkpoint**: Loading indicator appears during search; empty/no-match states handled gracefully

---

## Phase 6: User Story 6 - Keyboard Navigation (Priority: P3)

**Goal**: Arrow keys navigate results, Enter selects, Escape closes dropdown

**Independent Test**: Use keyboard only to search, navigate, and select a result

### Implementation for User Story 6

- [x] T025 [US6] Add keyboard event listener for search input focus in internal/web/static/js/app.js
- [x] T026 [US6] Implement ArrowDown/ArrowUp handlers to cycle through [data-search-result] elements in internal/web/static/js/app.js
- [x] T027 [US6] Add visual highlight class for selected result in internal/web/static/js/app.js
- [x] T028 [US6] Implement Enter key handler to trigger HTMX navigation on selected result in internal/web/static/js/app.js
- [x] T029 [US6] Implement Escape key handler to clear search results in internal/web/static/js/app.js
- [x] T030 [US6] Add click-outside handler to close dropdown in internal/web/static/js/app.js

**Checkpoint**: Full keyboard accessibility - navigate, select, close all work without mouse

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup and validation

- [x] T031 [P] Add "/" hotkey to focus search bar from anywhere in internal/web/static/js/app.js
- [x] T032 [P] Ensure search bar is responsive on mobile viewports in internal/web/templates/shell.html
- [x] T033 Run manual validation per quickstart.md test scenarios
- [x] T034 Verify all acceptance scenarios from spec.md pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - verification only
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **US1+2+3 (Phase 3)**: Core flow - must be done together as MVP
- **US4 (Phase 4)**: Enhances US1 results display - can parallelize with US5/US6
- **US5 (Phase 5)**: Enhances US1/2 UX - can parallelize with US4/US6
- **US6 (Phase 6)**: Adds accessibility - can parallelize with US4/US5

### Within Each Phase

- Tests MUST be written and FAIL before implementation (for phases with tests)
- Backend before frontend
- Templates before JavaScript
- Core functionality before styling

### Parallel Opportunities

Within Phase 2 (Foundational):
- T003-T006 are sequential (types → function → helper → handler)
- T007 and T008 can run in parallel after T006

Within Phase 3 (Core Search):
- T009, T010, T011 can all run in parallel (different test functions)
- T012-T014 are sequential (template modifications)
- T015-T017 can be parallelized after T014

Within Phase 4-6:
- All three phases (US4, US5, US6) can run in parallel since they touch different aspects

---

## Parallel Example: Phase 3 Tests

```bash
# Launch all tests for Core Search Flow together:
Task: "Write TestSearchAggregation in internal/web/search_test.go"
Task: "Write TestSearchHandler in internal/web/search_test.go"
Task: "Write TestSearchResultURLPrefixing in internal/web/search_test.go"
```

## Parallel Example: User Stories 4, 5, 6

```bash
# After Phase 3 completes, these phases can run in parallel:
Developer A: Phase 4 (US4 - Result Display)
Developer B: Phase 5 (US5 - Empty/Loading States)
Developer C: Phase 6 (US6 - Keyboard Navigation)
```

---

## Implementation Strategy

### MVP First (Phase 3 Only)

1. Complete Phase 1: Setup (verification)
2. Complete Phase 2: Foundational (backend search)
3. Complete Phase 3: Core Search Flow (US1+2+3)
4. **STOP and VALIDATE**: Test full search-navigate flow
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Backend ready
2. Add Core Search Flow → Full search works (MVP!)
3. Add Result Display (US4) → Better grouping/labels
4. Add Empty/Loading (US5) → Better UX feedback
5. Add Keyboard Nav (US6) → Accessibility complete
6. Polish → Production ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1, US2, US3 are combined in Phase 3 because they form an inseparable core flow
- Both memory and profiles plugins already implement Searchable
- HTMX handles debouncing natively (no custom JS needed for US2)
- PrefixURL() already exists in internal/web/plugin.go
