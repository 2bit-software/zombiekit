# Tasks: Plugin-Style Web GUI Architecture

**Input**: Design documents from `/specs/008-plugin-web-gui/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested - implementation tasks only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md, this project uses:
- `internal/web/` - Core web infrastructure
- `internal/webplugins/` - Plugin implementations
- `internal/cli/` - CLI commands

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add Chi dependency and create directory structure

- [x] T001 Add Chi router dependency via `go get github.com/go-chi/chi/v5`
- [x] T002 [P] Create directory structure: `internal/web/templates/`, `internal/web/static/css/`, `internal/web/static/js/`
- [x] T003 [P] Create directory structure: `internal/webplugins/profiles/templates/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core web infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Define SidebarItem struct in `internal/web/plugin.go`
- [x] T005 Define WebPlugin interface in `internal/web/plugin.go`
- [x] T006 Define TemplatePlugin interface in `internal/web/plugin.go`
- [x] T007 Implement PluginRegistry with Register, Get, All, SidebarItems methods in `internal/web/plugin.go`
- [x] T008 Define PageData struct in `internal/web/render.go`
- [x] T009 Define ServerConfig struct in `internal/web/server.go`
- [x] T010 Implement Renderer with HTMX detection in `internal/web/render.go`
- [x] T011 [P] Create logging middleware with slog in `internal/web/middleware.go`
- [x] T012 [P] Create recovery middleware in `internal/web/middleware.go`
- [x] T013 [P] Create renderer context middleware in `internal/web/middleware.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - View Registered Tools in Sidebar (Priority: P1) 🎯 MVP

**Goal**: Users see a sidebar listing all registered plugins with correct sorting

**Independent Test**: Start server with plugins, verify sidebar displays items sorted by Order field

### Implementation for User Story 1

- [x] T014 [US1] Create shell.html template with sidebar layout in `internal/web/templates/shell.html`
- [x] T015 [US1] Add HTMX and Tailwind CSS CDN links in shell.html head section
- [x] T016 [US1] Create isActive template function for sidebar highlighting in `internal/web/render.go`
- [x] T017 [US1] Add template parsing with go:embed in `internal/web/render.go`
- [x] T018 [US1] Implement sidebar rendering loop in shell.html with SidebarItems iteration
- [x] T019 [US1] Handle empty sidebar state with appropriate messaging in shell.html

**Checkpoint**: Sidebar displays registered plugins sorted by Order

---

## Phase 4: User Story 2 - Navigate to Tool Content (Priority: P1)

**Goal**: Users click sidebar items and content area updates via HTMX without full page reload

**Independent Test**: Click sidebar items, verify content updates, URL changes, sidebar remains stable

### Implementation for User Story 2

- [x] T020 [US2] Implement Server struct with Chi router setup in `internal/web/server.go`
- [x] T021 [US2] Add plugin mounting logic with route groups in `internal/web/server.go`
- [x] T022 [US2] Implement static asset handler with go:embed in `internal/web/server.go`
- [x] T023 [US2] Add HTMX attributes (hx-get, hx-target, hx-push-url) to sidebar links in shell.html
- [x] T024 [US2] Implement Render method with HTMX header detection in `internal/web/render.go`
- [x] T025 [US2] Create content div with id="content" in shell.html for HTMX targeting
- [x] T026 [P] [US2] Create minimal app.js for sidebar active state updates in `internal/web/static/js/app.js`
- [x] T027 [P] [US2] Create minimal app.css for custom styles in `internal/web/static/css/app.css`

**Checkpoint**: HTMX navigation works, browser back/forward preserved

---

## Phase 5: User Story 3 - View Tool Detail Page (Priority: P2)

**Goal**: Users click list items to see detail views, with back navigation

**Independent Test**: Navigate to list, click item, verify detail loads, click back link

*Note: This story is demonstrated through the profiles plugin (US5)*

### Implementation for User Story 3

- [x] T028 [US3] Ensure Renderer.Render handles nested template paths like "profiles/view.html"
- [x] T029 [US3] Document template naming convention for plugins in quickstart.md

**Checkpoint**: Detail view pattern established for plugins to use

---

## Phase 6: User Story 4 - Full Page Load Support (Priority: P2)

**Goal**: Direct URL navigation renders complete page with sidebar

**Independent Test**: Enter plugin URL directly in browser, verify full page with sidebar renders

### Implementation for User Story 4

- [x] T030 [US4] Implement two-pass rendering in Renderer: content first, then shell wrapper
- [x] T031 [US4] Add RenderedContent field to shell template data for embedded content
- [x] T032 [US4] Create 404.html template in `internal/web/templates/404.html`
- [x] T033 [US4] Add NotFound handler returning 404.html in `internal/web/server.go`

**Checkpoint**: Bookmarks and direct URLs work correctly

---

## Phase 7: User Story 5 - Example Plugin: Profiles (Priority: P1)

**Goal**: Working profiles plugin demonstrating list and detail views (read-only)

**Independent Test**: Start server, navigate to /profiles, see list, click profile, see details

### Implementation for User Story 5

- [x] T034 [US5] Create profiles Plugin struct implementing WebPlugin in `internal/webplugins/profiles/plugin.go`
- [x] T035 [US5] Implement ID() returning "profiles" in `internal/webplugins/profiles/plugin.go`
- [x] T036 [US5] Implement SidebarItems() returning profiles nav item in `internal/webplugins/profiles/plugin.go`
- [x] T037 [US5] Implement Templates() with go:embed in `internal/webplugins/profiles/plugin.go`
- [x] T038 [US5] Implement MountRoutes with list and view routes in `internal/webplugins/profiles/plugin.go`
- [x] T039 [US5] Create list handler using profile.Service.List() in `internal/webplugins/profiles/handlers.go`
- [x] T040 [US5] Create view handler using profile.Service.Show() in `internal/webplugins/profiles/handlers.go`
- [x] T041 [P] [US5] Create list.html template with profile iteration in `internal/webplugins/profiles/templates/list.html`
- [x] T042 [P] [US5] Create view.html template with profile details in `internal/webplugins/profiles/templates/view.html`
- [x] T043 [US5] Handle empty profiles list with appropriate message in list.html
- [x] T044 [US5] Add HTMX attributes to list items for partial navigation

**Checkpoint**: Profiles plugin fully functional as reference implementation

---

## Phase 8: Dashboard & CLI Integration

**Goal**: Home page dashboard and CLI serve command wiring

### Implementation

- [x] T045 Create home.html dashboard template in `internal/web/templates/home.html`
- [x] T046 Add welcome message and plugin links to home.html
- [x] T047 Implement home handler rendering dashboard in `internal/web/server.go`
- [x] T048 Implement health check endpoint returning JSON in `internal/web/server.go`
- [x] T049 Implement graceful shutdown with context cancellation in `internal/web/server.go`
- [x] T050 Wire PluginRegistry and profiles plugin in `internal/cli/gui.go` (new gui command)
- [x] T051 Create and start Server in gui command in `internal/cli/gui.go`
- [x] T052 Add --port flag to gui command in `internal/cli/gui.go`

**Checkpoint**: `brains serve` starts web GUI with profiles plugin

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, logging, and validation

- [x] T053 Create error.html template for handler errors in `internal/web/templates/error.html`
- [x] T054 Verify structured logging includes path, status, duration
- [x] T055 Run go build to verify compilation
- [x] T056 Run go test ./internal/web/... to verify any existing tests pass
- [x] T057 Manual verification: start server, test all navigation paths
- [x] T058 Verify quickstart.md instructions work end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - sidebar display
- **User Story 2 (Phase 4)**: Depends on US1 - navigation requires sidebar
- **User Story 3 (Phase 5)**: Depends on US2 - detail views require navigation
- **User Story 4 (Phase 6)**: Depends on US2 - full page uses same renderer
- **User Story 5 (Phase 7)**: Depends on US1, US2 - plugin uses core infrastructure
- **Dashboard (Phase 8)**: Depends on US5 - needs working plugin to display
- **Polish (Phase 9)**: Depends on all phases

### User Story Dependencies

- **User Story 1 (P1)**: Foundation only - first story to implement
- **User Story 2 (P1)**: Requires US1 (sidebar must exist to navigate)
- **User Story 3 (P2)**: Requires US2 (navigation must work for detail views)
- **User Story 4 (P2)**: Requires US2 (renderer logic)
- **User Story 5 (P1)**: Requires US1, US2 (needs working sidebar and navigation)

### Parallel Opportunities

- T002, T003 can run in parallel (different directories)
- T011, T012, T013 can run in parallel (different middleware, same file but independent functions)
- T026, T027 can run in parallel (different files)
- T041, T042 can run in parallel (different templates)

---

## Parallel Example: Phase 2 (Foundational)

```bash
# These can run in parallel (different middleware functions):
Task: "Create logging middleware with slog in internal/web/middleware.go"
Task: "Create recovery middleware in internal/web/middleware.go"
Task: "Create renderer context middleware in internal/web/middleware.go"
```

## Parallel Example: User Story 5 (Profiles Plugin)

```bash
# These can run in parallel (different template files):
Task: "Create list.html template in internal/webplugins/profiles/templates/list.html"
Task: "Create view.html template in internal/webplugins/profiles/templates/view.html"
```

---

## Implementation Strategy

### MVP First (Sidebar + Navigation + Profiles)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL)
3. Complete Phase 3: User Story 1 (Sidebar)
4. Complete Phase 4: User Story 2 (Navigation)
5. Complete Phase 7: User Story 5 (Profiles Plugin)
6. **STOP and VALIDATE**: Test full flow
7. Complete remaining phases

### Suggested MVP Scope

**MVP = Phases 1-4 + Phase 7**

After completing these phases, you have:
- Working sidebar with registered plugins
- HTMX navigation between plugins
- Profiles plugin demonstrating list/detail pattern
- Enough to validate the architecture

### Incremental Delivery

1. Phase 1-2 → Foundation ready
2. Phase 3 → Sidebar displays (visible progress)
3. Phase 4 → Navigation works (interactive)
4. Phase 7 → Profiles plugin (usable feature)
5. Phase 5-6 → Detail views + direct URLs (polish)
6. Phase 8 → Dashboard + CLI (complete)
7. Phase 9 → Polish (production-ready)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Verify go build passes after each phase
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
