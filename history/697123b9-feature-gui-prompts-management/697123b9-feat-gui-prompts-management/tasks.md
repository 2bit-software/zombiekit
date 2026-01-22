---
status: complete
updated: 2026-01-21
complexity: medium
total_tasks: 14
parallel_opportunities: 5
---

# Tasks: GUI Prompts Management (MVP - P1)

## Summary

- **Total tasks**: 14
- **Completed**: 14
- **Parallelizable**: 5 groups
- **Critical path**: T001 → T005 → T007 → T010 → T012 → T014
- **Scope**: P1 only (List, Filter, Search, View)

## Task List

### Phase 1: Backend Service Enhancement

- [x] T001 [US1] Add `List()` method to workflow service
  - File: `internal/workflow/service.go`
  - Add `List() ([]*Workflow, error)` method
  - Add `loadAllFromDir()` helper
  - Add `loadAllFromEmbedded()` helper
  - Resolution: local > global > embedded (higher shadows lower)
  - AC: Returns all workflows with correct Source field

### Phase 2: Plugin Infrastructure (Parallelizable with T001)

- [x] T002 [P] Create plugin directory structure
  - Create: `internal/webplugins/prompts/`
  - Create empty files: `plugin.go`, `handlers.go`, `types.go`, `converters.go`
  - Create: `internal/webplugins/prompts/templates/`
  - AC: Directory structure exists

- [x] T003 [P] [US1] Implement type definitions
  - File: `internal/webplugins/prompts/types.go`
  - Define: `PromptCategory`, `PromptSource` with methods
  - Define: `Prompt`, `ListData`, `ViewData` structs
  - Define: `FilterOptions`, `SortOptions` structs
  - AC: Types compile and have Badge/Label helper methods

### Phase 3: Data Layer

- [x] T004 [US1] Implement converters for all prompt types
  - File: `internal/webplugins/prompts/converters.go`
  - Implement: `convertWorkflow()`, `convertWorkflowFull()`
  - Implement: `convertProfile()`, `convertProfileFull()`
  - Implement: `convertStep()`, `convertStepFull()`
  - Implement: source mapping helpers
  - AC: All converters produce valid Prompt structs

- [x] T005 [US1] Implement plugin core with aggregatePrompts
  - File: `internal/webplugins/prompts/plugin.go`
  - Define Plugin struct with service dependencies
  - Implement `NewPlugin()` constructor
  - Implement `SidebarItems()` returning order=15
  - Implement `MountRoutes()` with `/` and `/{category}/{name}`
  - AC: Plugin satisfies web.Plugin interface
  - Depends: T001, T003, T004

### Phase 4: HTTP Handlers

- [x] T006 [P] [US2] Implement filter and sort helpers
  - File: `internal/webplugins/prompts/handlers.go`
  - Implement: `filterPrompts(prompts, FilterOptions) []Prompt`
  - Implement: `sortPrompts(prompts, SortOptions)`
  - Handle: category, source, query (case-insensitive name/desc search)
  - Handle: sort by name, category, source with secondary sort by name
  - AC: Filters and sorts work correctly

- [x] T007 [US1] [US2] Implement list handler
  - File: `internal/webplugins/prompts/handlers.go`
  - Add `handlers` struct with service dependencies
  - Implement `newHandlers()` constructor
  - Implement `aggregatePrompts()` method
  - Implement `list()` handler with query param parsing
  - Default sort: name asc
  - AC: GET /prompts returns filtered/sorted prompts
  - Depends: T005, T006

- [x] T008 [US3] Implement view handler
  - File: `internal/webplugins/prompts/handlers.go`
  - Implement `getPrompt(category, name) (*Prompt, error)`
  - Implement `view()` handler
  - Handle unknown category with error response
  - AC: GET /prompts/{category}/{name} returns full prompt detail
  - Depends: T005

### Phase 5: Templates

- [x] T009 [P] [US1] [US2] Create list template
  - File: `internal/webplugins/prompts/templates/list.html`
  - Header with disabled "New Prompt" button (P2 feature)
  - Category filter tabs: All | Workflows | Profiles | Steps
  - Source filter dropdown
  - Search input with submit button
  - Sort dropdown
  - Results table: Name, Category, Source (badge), Description
  - Empty state with clear filters link
  - Result count footer
  - HTMX: hx-get, hx-target="#content", hx-push-url="true"
  - AC: Template renders without errors, filters update via HTMX

- [x] T010 [US3] Create view template
  - File: `internal/webplugins/prompts/templates/view.html`
  - Back link to list
  - Header: name, category label, source badge
  - Disabled Edit/Delete buttons for non-embedded (P2 feature)
  - Metadata section with conditional profile/step fields
  - Content section with markdown rendering
  - Toggle between rendered/source view (JavaScript)
  - AC: Template renders all prompt types correctly
  - Depends: T009 (shares styling patterns)

### Phase 6: Integration

- [x] T011 Register templates with renderer
  - File: May require embedding templates in plugin or registering with web.Renderer
  - Verify templates are discovered at `/prompts/list.html` and `/prompts/view.html`
  - AC: Templates render via renderer.Render()
  - Depends: T009, T010

- [x] T012 [US1] Register plugin with GUI server
  - File: `internal/cli/gui.go`
  - Create workflow.Service instance
  - Create prompts.Plugin with all service dependencies
  - Register with `registry.Register("prompts", promptsPlugin)`
  - AC: Prompts appears in sidebar, `/prompts` route works
  - Depends: T005, T007, T008, T011

### Phase 7: Testing

- [x] T013 [P] Write integration tests
  - File: `internal/webplugins/prompts/plugin_test.go`
  - Test `GET /prompts` returns all prompts
  - Test `GET /prompts?category=workflow` filters
  - Test `GET /prompts?source=local` filters
  - Test `GET /prompts?q=feat` searches
  - Test `GET /prompts/workflow/{name}` returns workflow
  - Test `GET /prompts/profile/{name}` returns profile
  - Test `GET /prompts/step/{name}` returns step
  - Test `GET /prompts/invalid/{name}` returns error
  - AC: All tests pass

- [x] T014 Manual E2E verification
  - Start GUI with `brains gui`
  - Navigate to Prompts in sidebar
  - Verify all prompt types displayed
  - Test category filter tabs
  - Test source filter dropdown
  - Test search functionality
  - Test sorting by name/category/source
  - Click through to view detail for each category
  - Verify metadata displays correctly
  - Test rendered/source toggle
  - AC: All user scenarios work as specified

## Dependency Graph

```
T001 ─────────────────┐
                      │
T002 (parallel) ──────┼──→ T005 ──→ T007 ──→ T012 ──→ T014
                      │      ↓
T003 (parallel) ──────┘    T008
                            ↓
T004 ─────────────────────→ ↑

T006 (parallel) ──────────→ T007

T009 (parallel) ──→ T010 ──→ T011 ──→ T012

T013 (parallel with T012-T014)
```

## Execution Order

**Batch 1 (Parallelizable)**:
- T001: workflow.List()
- T002: Directory structure
- T003: Type definitions

**Batch 2 (Parallelizable after Batch 1)**:
- T004: Converters
- T006: Filter/sort helpers
- T009: List template

**Batch 3 (Sequential)**:
- T005: Plugin core (needs T001, T003, T004)
- T010: View template (after T009)

**Batch 4 (Sequential)**:
- T007: List handler (needs T005, T006)
- T008: View handler (needs T005)
- T011: Template registration (needs T009, T010)

**Batch 5 (Sequential)**:
- T012: Plugin registration (needs T005, T007, T008, T011)
- T013: Integration tests (parallel with T012)

**Batch 6 (Final)**:
- T014: Manual E2E verification

## Traceability

| Task | User Stories | Functional Requirements |
|------|--------------|------------------------|
| T001 | US1 | FR-001 |
| T002 | - | - |
| T003 | US1 | FR-001 |
| T004 | US1 | FR-001 |
| T005 | US1 | FR-001 |
| T006 | US2 | FR-002, FR-003, FR-004, FR-005 |
| T007 | US1, US2 | FR-001, FR-002, FR-003, FR-004, FR-005 |
| T008 | US3 | FR-006 |
| T009 | US1, US2 | FR-001, FR-002, FR-003, FR-004, FR-005 |
| T010 | US3 | FR-006 |
| T011 | - | - |
| T012 | US1 | FR-001 |
| T013 | US1, US2, US3 | FR-001 through FR-005 |
| T014 | US1, US2, US3 | FR-001 through FR-006 |
