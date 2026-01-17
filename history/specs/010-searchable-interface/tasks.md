# Tasks: Searchable Interface

**Input**: Design documents from `/specs/010-searchable-interface/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/search.go

**Tests**: Included - following existing go test patterns with testify/assert

**Organization**: Tasks grouped by user story for independent implementation and testing

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, etc.)
- All file paths are relative to repository root

---

## Phase 1: Setup

**Purpose**: Create the new internal/search package structure

- [X] T001 Create directory internal/search/
- [X] T002 Create internal/search/search.go with package declaration and imports

---

## Phase 2: Foundational - Core Interface Definition

**Purpose**: Define the Searchable interface and types that ALL user stories depend on

**⚠️ CRITICAL**: All user story implementations depend on this phase

- [X] T003 Define SortOrder type and constants in internal/search/search.go
- [X] T004 Define SearchResult struct in internal/search/search.go
- [X] T005 Define Searchable interface in internal/search/search.go
- [X] T006 Add godoc comments explaining interface contract rules in internal/search/search.go

**Checkpoint**: Interface defined - plugin implementations can now begin

---

## Phase 3: User Story 5 - Interface Composition with WebPlugin (Priority: P1) 🎯 MVP

**Goal**: Verify Searchable is independent of WebPlugin with zero coupling

**Independent Test**: Import internal/search from a test file without importing internal/web; compilation succeeds

### Tests for User Story 5

- [X] T007 [US5] Create internal/search/search_test.go with test package
- [X] T008 [US5] Add TestSearchableInterfaceIndependence verifying no web package imports in internal/search/search_test.go
- [X] T009 [US5] Add TestTypeAssertionToSearchable testing interface satisfaction in internal/search/search_test.go

### Implementation for User Story 5

- [X] T010 [US5] Verify internal/search/search.go has no imports from internal/web (manual review)
- [X] T011 [US5] Add compile-time interface check example in internal/search/search_test.go

**Checkpoint**: Searchable interface is proven independent of WebPlugin

---

## Phase 4: User Story 1 - Plugin Developer Implements Search (Priority: P1)

**Goal**: Enable a plugin to implement the Searchable interface and return results

**Independent Test**: Create mock implementation, call Search(), verify SearchResult items returned

### Tests for User Story 1

- [X] T012 [P] [US1] Add TestSearchReturnsResults with mock implementation in internal/search/search_test.go
- [X] T013 [P] [US1] Add TestSearchWithMaxResults verifying limit is respected in internal/search/search_test.go
- [X] T014 [P] [US1] Add TestSearchEmptyQueryReturnsEmptySlice in internal/search/search_test.go
- [X] T015 [P] [US1] Add TestSearchNoMatchesReturnsEmptySlice in internal/search/search_test.go
- [X] T016 [P] [US1] Add TestSearchResultNeverNil verifying non-nil return in internal/search/search_test.go

### Implementation for User Story 1

- [X] T017 [US1] Add helper function IsValidSortOrder in internal/search/search.go
- [X] T018 [US1] Implement Searchable on profiles.Plugin in internal/webplugins/profiles/plugin.go
- [X] T019 [US1] Add compile-time interface check var _ search.Searchable = (*Plugin)(nil) in internal/webplugins/profiles/plugin.go

**Checkpoint**: profiles.Plugin implements Searchable and returns valid results

---

## Phase 5: User Story 4 - Search Across Names and Content (Priority: P1)

**Goal**: Search matches both item names and content, not just names

**Independent Test**: Create items where one matches by name only, one by content only; both appear in results

### Tests for User Story 4

- [X] T020 [P] [US4] Add TestSearchMatchesName verifying name matching in internal/webplugins/profiles/plugin_test.go
- [X] T021 [P] [US4] Add TestSearchMatchesContent verifying content matching in internal/webplugins/profiles/plugin_test.go
- [X] T022 [P] [US4] Add TestSearchNoDuplicates verifying item matching both appears once in internal/webplugins/profiles/plugin_test.go
- [X] T023 [P] [US4] Add TestSearchCaseInsensitive verifying case-insensitive matching in internal/webplugins/profiles/plugin_test.go

### Implementation for User Story 4

- [X] T024 [US4] Update profiles.Plugin.Search to search profile names in internal/webplugins/profiles/plugin.go
- [X] T025 [US4] Update profiles.Plugin.Search to search profile content in internal/webplugins/profiles/plugin.go
- [X] T026 [US4] Add deduplication logic to prevent duplicate results in internal/webplugins/profiles/plugin.go

**Checkpoint**: Search finds items by name OR content, no duplicates

---

## Phase 6: User Story 2 - Search Results Sorted by Relevance (Priority: P2)

**Goal**: Default sort order is relevance (closest matches first)

**Independent Test**: Search query with varying match quality; exact matches appear before partial

### Tests for User Story 2

- [X] T027 [P] [US2] Add TestSearchDefaultSortIsRelevance in internal/webplugins/profiles/plugin_test.go
- [X] T028 [P] [US2] Add TestSearchExactMatchBeforePartial in internal/webplugins/profiles/plugin_test.go
- [X] T029 [P] [US2] Add TestSearchEmptySortOrderDefaultsToRelevance in internal/webplugins/profiles/plugin_test.go

### Implementation for User Story 2

- [X] T030 [US2] Implement relevance scoring (exact > prefix > contains) in internal/webplugins/profiles/plugin.go
- [X] T031 [US2] Apply relevance sort when sortOrder is empty or SortRelevance in internal/webplugins/profiles/plugin.go

**Checkpoint**: Default sort shows best matches first

---

## Phase 7: User Story 3 - Search Results Sorted by User-Specified Order (Priority: P2)

**Goal**: Support all sort options: created_date, updated_date, last_used, name

**Independent Test**: Same search with different sort_order values produces different orderings

### Tests for User Story 3

- [X] T032 [P] [US3] Add TestSearchSortByName in internal/webplugins/profiles/plugin_test.go
- [X] T033 [P] [US3] Add TestSearchSortByCreatedDate in internal/webplugins/profiles/plugin_test.go
- [X] T034 [P] [US3] Add TestSearchSortByUpdatedDate in internal/webplugins/profiles/plugin_test.go
- [X] T035 [P] [US3] Add TestSearchSortByLastUsed (graceful handling if not tracked) in internal/webplugins/profiles/plugin_test.go
- [X] T036 [P] [US3] Add TestSearchInvalidSortOrderFallsBackToRelevance in internal/webplugins/profiles/plugin_test.go

### Implementation for User Story 3

- [X] T037 [US3] Implement SortName ordering (A-Z by title) in internal/webplugins/profiles/plugin.go
- [X] T038 [US3] Implement SortCreatedDate ordering (newest first) in internal/webplugins/profiles/plugin.go
- [X] T039 [US3] Implement SortUpdatedDate ordering (most recent first) in internal/webplugins/profiles/plugin.go
- [X] T040 [US3] Implement SortLastUsed ordering with fallback in internal/webplugins/profiles/plugin.go
- [X] T041 [US3] Add graceful fallback to relevance for invalid sort orders in internal/webplugins/profiles/plugin.go

**Checkpoint**: All sort orders work correctly

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Complete second plugin implementation and final validation

- [X] T042 [P] Implement Searchable on memory.Plugin in internal/webplugins/memory/plugin.go
- [X] T043 [P] Add compile-time interface check in internal/webplugins/memory/plugin.go
- [X] T044 [P] Add basic search tests for memory.Plugin in internal/webplugins/memory/plugin_test.go
- [X] T045 Run go test ./internal/search/... ./internal/webplugins/... to verify all tests pass
- [X] T046 Run go vet ./internal/search/... to check for issues
- [X] T047 Validate quickstart.md examples compile and work

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1
- **User Story 5 (Phase 3)**: Depends on Phase 2 - establishes independence
- **User Story 1 (Phase 4)**: Depends on Phase 2 - core implementation
- **User Story 4 (Phase 5)**: Depends on Phase 4 - extends search behavior
- **User Story 2 (Phase 6)**: Depends on Phase 4 - adds relevance sort
- **User Story 3 (Phase 7)**: Depends on Phase 6 - adds all sort options
- **Polish (Phase 8)**: Depends on all user stories

### User Story Dependencies

```text
Phase 2 (Foundational)
    ├── US5 (P1): Interface independence ── can start immediately after Phase 2
    ├── US1 (P1): Plugin implements search ── can start immediately after Phase 2
    │       │
    │       └── US4 (P1): Search names + content ── extends US1
    │               │
    │               └── US2 (P2): Relevance sort ── extends US4
    │                       │
    │                       └── US3 (P2): All sort options ── extends US2
    │
    └── Polish ── after all stories complete
```

### Parallel Opportunities

**Within Phase 2 (Foundational):**
- T003, T004, T005 can be done sequentially in one file

**Within Phase 3 (US5):**
- T007, T008, T009 can run in parallel (same test file, different tests)

**Within Phase 4 (US1):**
- T012, T013, T014, T015, T016 can run in parallel (all test tasks)

**Within Phase 5 (US4):**
- T020, T021, T022, T023 can run in parallel (all test tasks)

**Within Phase 6 (US2):**
- T027, T028, T029 can run in parallel (all test tasks)

**Within Phase 7 (US3):**
- T032, T033, T034, T035, T036 can run in parallel (all test tasks)

**Within Phase 8 (Polish):**
- T042, T043, T044 can run in parallel (different files)

---

## Parallel Example: Phase 4 (User Story 1)

```bash
# Launch all tests for User Story 1 together:
Task: "Add TestSearchReturnsResults in internal/search/search_test.go"
Task: "Add TestSearchWithMaxResults in internal/search/search_test.go"
Task: "Add TestSearchEmptyQueryReturnsEmptySlice in internal/search/search_test.go"
Task: "Add TestSearchNoMatchesReturnsEmptySlice in internal/search/search_test.go"
Task: "Add TestSearchResultNeverNil in internal/search/search_test.go"

# Then implementation tasks sequentially
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 4 + 5)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (interface definition)
3. Complete Phase 3: User Story 5 (verify independence)
4. Complete Phase 4: User Story 1 (basic search works)
5. Complete Phase 5: User Story 4 (search names + content)
6. **STOP and VALIDATE**: Run `go test ./internal/search/... ./internal/webplugins/profiles/...`
7. MVP complete - plugin can implement Searchable

### Incremental Delivery

1. Setup + Foundational → Interface ready
2. Add US5 + US1 + US4 → Core search works (MVP!)
3. Add US2 → Default relevance sort
4. Add US3 → All sort options
5. Polish → Second plugin implementation

---

## Notes

- All test tasks are marked [P] as they touch the same test file but different test functions
- Implementation tasks are sequential within each story as they build on each other
- profiles.Plugin is the primary implementation; memory.Plugin is secondary (Polish phase)
- No dependency between US5 and US1/US4 - they verify different aspects
- Tests use existing testify/assert pattern per go.mod
