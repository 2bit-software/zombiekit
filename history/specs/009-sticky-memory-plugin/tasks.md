# Tasks: Sticky Memory Web Plugin

**Input**: Design documents from `/specs/009-sticky-memory-plugin/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in feature specification. Tests omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md, this feature uses:
- **Plugin code**: `internal/webplugins/memory/`
- **Templates**: `internal/webplugins/memory/templates/`
- **CLI modification**: `internal/cli/gui.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Plugin skeleton and project initialization

- [x] T001 Create plugin directory structure at `internal/webplugins/memory/`
- [x] T002 Create plugin.go with Plugin struct and interface implementations at `internal/webplugins/memory/plugin.go`
- [x] T003 [P] Create templates directory with embed directive at `internal/webplugins/memory/templates/`
- [x] T004 Register memory plugin in GUI command at `internal/cli/gui.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement handlers struct with Storage dependency at `internal/webplugins/memory/handlers.go`
- [x] T006 Define view model types (ListData, ViewData, FormData, DeleteData, PaginationData) at `internal/webplugins/memory/handlers.go`
- [x] T007 [P] Implement FormatSize helper function at `internal/webplugins/memory/handlers.go`
- [x] T008 [P] Implement FormatTime helper function at `internal/webplugins/memory/handlers.go`
- [x] T009 Mount basic routes in MountRoutes() at `internal/webplugins/memory/plugin.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Browse Memory List (Priority: P1) MVP

**Goal**: Users navigate to the sticky memory section and see a searchable, paginated list of all stored memories with key metadata.

**Independent Test**: Open the memories page at `/memory` and verify the list displays with name, size, version, and timestamps. Empty state should show prompt to create first memory.

### Implementation for User Story 1

- [x] T010 [US1] Implement list handler with pagination at `internal/webplugins/memory/handlers.go`
- [x] T011 [US1] Create list.html template with memory table/cards at `internal/webplugins/memory/templates/list.html`
- [x] T012 [US1] Add pagination controls to list.html template at `internal/webplugins/memory/templates/list.html`
- [x] T013 [US1] Add empty state display when no memories exist at `internal/webplugins/memory/templates/list.html`
- [x] T014 [US1] Add "New Memory" button with HTMX navigation at `internal/webplugins/memory/templates/list.html`

**Checkpoint**: User Story 1 complete - can browse paginated memory list, see empty state, click to create new

---

## Phase 4: User Story 2 - View Memory Content (Priority: P1) MVP

**Goal**: Users click on a memory from the list to view its full content and metadata in a dedicated view page.

**Independent Test**: Click any memory in the list and verify content displays with metadata section showing name, size, version, created date, and last updated date.

### Implementation for User Story 2

- [x] T015 [US2] Implement view handler at `internal/webplugins/memory/handlers.go`
- [x] T016 [US2] Create view.html template with content display at `internal/webplugins/memory/templates/view.html`
- [x] T017 [US2] Add metadata section (name, size, version, timestamps) to view.html at `internal/webplugins/memory/templates/view.html`
- [x] T018 [US2] Add Edit and Delete action buttons to view.html at `internal/webplugins/memory/templates/view.html`
- [x] T019 [US2] Add back to list navigation at `internal/webplugins/memory/templates/view.html`

**Checkpoint**: User Story 2 complete - can view full memory content with metadata

---

## Phase 5: User Story 3 - Create New Memory (Priority: P1) MVP

**Goal**: Users create a new memory by providing a name and content through a simple form.

**Independent Test**: Click "New Memory", fill out the form, and verify the new memory appears in the list at the top.

### Implementation for User Story 3

- [x] T020 [US3] Implement createForm handler at `internal/webplugins/memory/handlers.go`
- [x] T021 [US3] Implement create handler with validation at `internal/webplugins/memory/handlers.go`
- [x] T022 [US3] Create form.html template with name and content fields at `internal/webplugins/memory/templates/form.html`
- [x] T023 [US3] Add validation error display to form.html at `internal/webplugins/memory/templates/form.html`
- [x] T024 [US3] Add cancel button with HTMX navigation back to list at `internal/webplugins/memory/templates/form.html`

**Checkpoint**: User Story 3 complete (MVP COMPLETE) - can create memories, view list, view content

---

## Phase 6: User Story 4 - Edit Existing Memory (Priority: P2)

**Goal**: Users edit the content of an existing memory, creating a new version.

**Independent Test**: Click edit on any memory, modify content, save, and verify the version number increments and updated timestamp changes.

### Implementation for User Story 4

- [x] T025 [US4] Implement editForm handler at `internal/webplugins/memory/handlers.go`
- [x] T026 [US4] Implement update handler at `internal/webplugins/memory/handlers.go`
- [x] T027 [US4] Extend form.html to handle edit mode (pre-populate fields, disable name) at `internal/webplugins/memory/templates/form.html`

**Checkpoint**: User Story 4 complete - can edit memories with version tracking

---

## Phase 7: User Story 5 - Delete Memory (Priority: P2)

**Goal**: Users delete a memory they no longer need, with confirmation to prevent accidents.

**Independent Test**: Click delete on a memory, confirm the action, and verify it no longer appears in the list.

### Implementation for User Story 5

- [x] T028 [US5] Implement deleteConfirm handler at `internal/webplugins/memory/handlers.go`
- [x] T029 [US5] Implement delete handler at `internal/webplugins/memory/handlers.go`
- [x] T030 [US5] Create delete.html template with confirmation at `internal/webplugins/memory/templates/delete.html`
- [x] T031 [US5] Add cancel button returning to memory view at `internal/webplugins/memory/templates/delete.html`

**Checkpoint**: User Story 5 complete - can delete memories with confirmation

---

## Phase 8: User Story 6 - Search and Filter (Priority: P2)

**Goal**: Users search for specific memories by name or content to quickly find what they need.

**Independent Test**: Enter a search term and verify only matching memories appear in the results. Clear search to show all.

### Implementation for User Story 6

- [x] T032 [US6] Add search input field to list.html with HTMX trigger at `internal/webplugins/memory/templates/list.html`
- [x] T033 [US6] Update list handler to filter by search query at `internal/webplugins/memory/handlers.go`
- [x] T034 [US6] Add empty search results state to list.html at `internal/webplugins/memory/templates/list.html`
- [x] T035 [US6] Add clear search link at `internal/webplugins/memory/templates/list.html`

**Checkpoint**: User Story 6 complete - can search memories by name and content

---

## Phase 9: User Story 7 - Toggle Markdown View Mode (Priority: P3)

**Goal**: Users switch between rendered markdown and raw source view when viewing memory content.

**Independent Test**: View a memory with markdown content, click the toggle button, and verify the view switches between rendered and source.

### Implementation for User Story 7

- [x] T036 [US7] Add marked.js CDN script to shell or view.html at `internal/webplugins/memory/templates/view.html`
- [x] T037 [US7] Add view toggle button to view.html at `internal/webplugins/memory/templates/view.html`
- [x] T038 [US7] Implement toggleView() JavaScript function at `internal/webplugins/memory/templates/view.html`
- [x] T039 [US7] Add htmx:afterSwap handler to re-render markdown at `internal/webplugins/memory/templates/view.html`
- [x] T040 [US7] Add styling for source view (monospace, pre-wrap) at `internal/webplugins/memory/templates/view.html`

**Checkpoint**: User Story 7 complete - all features implemented

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T041 [P] Add keyboard navigation support (tabindex, focus states) across all templates
- [x] T042 [P] Add consistent error styling across all templates
- [x] T043 Verify HTMX navigation works correctly for all routes
- [x] T044 Run quickstart.md validation - verify all features work end-to-end
- [x] T045 [P] Add loading states for HTMX requests (optional enhancement)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - User stories can proceed in priority order (P1 → P2 → P3)
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational - Needs list.html links to work (minimal US1 dependency)
- **User Story 3 (P1)**: Can start after Foundational - Needs form.html independent of US1/US2
- **User Story 4 (P2)**: Depends on US2 (view page) and US3 (form.html template)
- **User Story 5 (P2)**: Depends on US2 (view page has delete button)
- **User Story 6 (P2)**: Depends on US1 (list.html exists)
- **User Story 7 (P3)**: Depends on US2 (view.html exists)

### Within Each User Story

- Handler implementation before template
- Template structure before HTMX integration
- Core features before enhancements

### Parallel Opportunities

- **Setup Phase**: T003 (templates dir) can run parallel to T002 (plugin.go)
- **Foundational Phase**: T007, T008 (helper functions) can run in parallel
- **User Stories 1-3 (P1)**: Can largely run in parallel as they touch different templates
- **Polish Phase**: T041, T042, T045 can run in parallel

---

## Parallel Example: Setup Phase

```bash
# Launch setup tasks in parallel:
Task: "Create plugin.go" in internal/webplugins/memory/plugin.go
Task: "Create templates directory" in internal/webplugins/memory/templates/
```

## Parallel Example: Foundational Phase

```bash
# Launch helper functions in parallel:
Task: "Implement FormatSize helper" in internal/webplugins/memory/handlers.go
Task: "Implement FormatTime helper" in internal/webplugins/memory/handlers.go
```

---

## Implementation Strategy

### MVP First (User Stories 1-3)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (Browse List)
4. Complete Phase 4: User Story 2 (View Content)
5. Complete Phase 5: User Story 3 (Create Memory)
6. **STOP and VALIDATE**: Test all MVP features independently
7. Deploy/demo - users can now browse, view, and create memories

### Incremental Delivery

1. **MVP**: Setup + Foundational + US1 + US2 + US3 → Core CRUD reading and creation
2. **P2 Features**: US4 (Edit) + US5 (Delete) + US6 (Search) → Full CRUD + search
3. **P3 Features**: US7 (Markdown Toggle) → Enhanced viewing experience
4. **Polish**: Cross-cutting improvements → Production ready

---

## Notes

- [P] tasks = different files or independent sections, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Existing `internal/memory.Storage` interface is reused - no storage work needed
- Follow profiles plugin pattern in `internal/webplugins/profiles/` for consistency
