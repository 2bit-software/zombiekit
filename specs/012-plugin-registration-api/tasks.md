# Tasks: Simplified Plugin Registration API

**Input**: Design documents from `/specs/012-plugin-registration-api/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested - tests omitted from task list.

**Organization**: Tasks grouped by user story. Note: User Stories 1-3 are all P1 priority and form the core architectural change. User Story 4 is P2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

This is a single Go project:
- **Source**: `internal/` at repository root
- **Web core**: `internal/web/`
- **Plugins**: `internal/webplugins/`

---

## Phase 1: Setup (No Changes Required)

**Purpose**: Project already exists with structure in place

- [X] T001 Verify existing project structure and dependencies are in place

**Note**: No setup tasks needed - this feature modifies existing code rather than adding new structure.

---

## Phase 2: Foundational (Core Interface Changes)

**Purpose**: Core infrastructure changes that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. These changes break the existing plugin interface.

- [X] T002 Add `ValidatePluginName(name string) error` function in `internal/web/plugin.go`
- [X] T003 Add `InvalidPluginNameError` type in `internal/web/plugin.go`
- [X] T004 Add `PrefixURL(pluginName, url string) string` helper function in `internal/web/plugin.go`
- [X] T005 Add `registeredPlugin` struct (name + plugin pair) in `internal/web/plugin.go`
- [X] T006 Modify `PluginRegistry` struct to use `[]registeredPlugin` and `byName` map in `internal/web/plugin.go`
- [X] T007 Add `logger *slog.Logger` field to `PluginRegistry` in `internal/web/plugin.go`
- [X] T008 Update `NewPluginRegistry` to accept logger parameter in `internal/web/plugin.go`
- [X] T009 Rename `DuplicatePluginError.ID` field to `Name` in `internal/web/plugin.go`

**Checkpoint**: Foundation ready - interface and registry structure updated. Old Register() method temporarily broken.

---

## Phase 3: User Story 1 - Register Plugin with Explicit Name (Priority: P1) 🎯 MVP

**Goal**: Developers can register plugins by calling `registry.Register("name", plugin)` with validation and logging.

**Independent Test**: Register a plugin with name "test-plugin" and verify no panic occurs, registration is logged, and plugin is retrievable by name.

### Implementation for User Story 1

- [X] T010 [US1] Remove `ID() string` from `WebPlugin` interface in `internal/web/plugin.go`
- [X] T011 [US1] Implement new `Register(name string, plugin WebPlugin)` method with name validation, duplicate check, and panic on error in `internal/web/plugin.go`
- [X] T012 [US1] Add Info-level logging to `Register` method in `internal/web/plugin.go`
- [X] T013 [US1] Update `Get(name string)` method to use `byName` map in `internal/web/plugin.go`
- [X] T014 [US1] Update `All()` method to return `[]registeredPlugin` with names in `internal/web/plugin.go`

**Checkpoint**: User Story 1 complete - plugins can be registered with explicit names. Old ID() pattern removed.

---

## Phase 4: User Story 2 - Plugin-Relative URLs (Priority: P1)

**Goal**: System automatically prefixes plugin-relative URLs with the plugin name when constructing navigation links.

**Independent Test**: Call `PrefixURL("memory", "/notes")` and verify result is "/memory/notes". Verify absolute URLs pass through unchanged.

### Implementation for User Story 2

- [X] T015 [US2] Add absolute URL detection to `PrefixURL` (http://, https://) in `internal/web/plugin.go`
- [X] T016 [US2] Add double-prefix prevention to `PrefixURL` in `internal/web/plugin.go`
- [X] T017 [US2] Add URL normalization (handle leading slash or not) to `PrefixURL` in `internal/web/plugin.go`

**Checkpoint**: User Story 2 complete - PrefixURL correctly handles all URL patterns.

---

## Phase 5: User Story 3 - Automatic Route Mounting (Priority: P1)

**Goal**: Plugin routes are automatically mounted under `/{pluginName}/...` and handlers receive requests with prefix stripped.

**Independent Test**: Register a plugin as "memory" with route "/list", make request to "/memory/list", verify plugin handler receives the request.

### Implementation for User Story 3

- [X] T018 [US3] Update `setupRouter` to iterate `registry.All()` and use stored names instead of `plugin.ID()` in `internal/web/server.go`
- [X] T019 [US3] Update `NewServer` to pass logger to `NewPluginRegistry` in `internal/web/server.go`

**Checkpoint**: User Story 3 complete - routes mounted correctly with names from registration.

---

## Phase 6: User Story 4 - Sidebar Configuration (Priority: P2)

**Goal**: Sidebar item paths from plugins are automatically prefixed with the plugin name.

**Independent Test**: Register a plugin with sidebar item path "/", verify `SidebarItems()` returns path "/{pluginName}/".

### Implementation for User Story 4

- [X] T020 [US4] Update `SidebarItems()` to iterate with name access and prefix each item's Path in `internal/web/plugin.go`
- [X] T021 [US4] Review `internal/web/render.go` for any sidebar rendering that needs update

**Checkpoint**: User Story 4 complete - sidebar items display with correct prefixed paths.

---

## Phase 7: Plugin Migration

**Purpose**: Migrate existing plugins to the new registration pattern

### Memory Plugin Migration

- [X] T022 [P] Remove `ID() string` method from `Plugin` struct in `internal/webplugins/memory/plugin.go`
- [X] T023 [P] Update `SidebarItems()` to return relative path "/" instead of "/memory" in `internal/webplugins/memory/plugin.go`
- [X] T024 [P] Update `Search()` to return relative URLs (remove "/memory" prefix) in `internal/webplugins/memory/plugin.go`
- [X] T025 [P] Review `internal/webplugins/memory/handlers.go` for hardcoded redirect URLs

### Profiles Plugin Migration

- [X] T026 [P] Remove `ID() string` method from `Plugin` struct in `internal/webplugins/profiles/plugin.go`
- [X] T027 [P] Update `SidebarItems()` to return relative path "/" instead of "/profiles" in `internal/webplugins/profiles/plugin.go`
- [X] T028 [P] Update `Search()` to return relative URLs (remove "/profiles" prefix) in `internal/webplugins/profiles/plugin.go`
- [X] T029 [P] Review `internal/webplugins/profiles/handlers.go` for hardcoded redirect URLs

**Checkpoint**: Both plugins migrated to new pattern - all use relative URLs.

---

## Phase 8: Registration Site Update

**Purpose**: Update where plugins are registered to use new API

- [X] T030 Find and update plugin registration call site (likely in `cmd/` or main.go) to use `registry.Register("name", plugin)` pattern
- [X] T031 Verify correct plugin names used: "memory" and "profiles"

**Checkpoint**: Registration site updated - application starts and plugins work correctly.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [X] T032 Run `go build ./...` to verify no compilation errors
- [X] T033 Run `go test ./...` to verify existing tests pass
- [X] T034 Verify web server starts and serves plugins correctly
- [X] T035 Test sidebar navigation displays correct links
- [X] T036 Test direct URL access to plugin routes (e.g., /memory/, /profiles/)
- [X] T037 Update any tests that reference old `ID()` method or absolute plugin paths

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - verification only
- **Phase 2 (Foundational)**: Depends on Phase 1 - BLOCKS all user stories
- **Phases 3-6 (User Stories)**: All depend on Phase 2 completion
  - US1 → US2 → US3 → US4 (sequential, as each builds on prior)
- **Phase 7 (Migration)**: Depends on all user stories complete
- **Phase 8 (Registration)**: Depends on Phase 7 complete
- **Phase 9 (Polish)**: Depends on Phase 8 complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational - core registration change
- **User Story 2 (P1)**: Depends on US1 - uses PrefixURL
- **User Story 3 (P1)**: Depends on US1 - uses registry.All()
- **User Story 4 (P2)**: Depends on US2 - uses PrefixURL in SidebarItems()

### Within Phases

- Phase 2 (Foundational): Tasks T002-T009 can largely be done in sequence as they build the type system
- Phase 7 (Migration): Memory and Profiles plugin tasks are fully parallel ([P] marked)

### Parallel Opportunities

```text
Phase 7 - Full parallelism between plugins:
  Memory Plugin: T022, T023, T024, T025 (can run together)
  Profiles Plugin: T026, T027, T028, T029 (can run together)
```

---

## Parallel Example: Plugin Migration

```bash
# Launch all memory plugin migration tasks together:
Task: "Remove ID() method from memory Plugin in internal/webplugins/memory/plugin.go"
Task: "Update SidebarItems() in internal/webplugins/memory/plugin.go"
Task: "Update Search() in internal/webplugins/memory/plugin.go"
Task: "Review handlers.go for hardcoded URLs in internal/webplugins/memory/handlers.go"

# Launch all profiles plugin migration tasks together:
Task: "Remove ID() method from profiles Plugin in internal/webplugins/profiles/plugin.go"
Task: "Update SidebarItems() in internal/webplugins/profiles/plugin.go"
Task: "Update Search() in internal/webplugins/profiles/plugin.go"
Task: "Review handlers.go for hardcoded URLs in internal/webplugins/profiles/handlers.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1-3)

1. Complete Phase 1: Setup (verification)
2. Complete Phase 2: Foundational (interface/type changes)
3. Complete Phase 3: User Story 1 (registration API)
4. Complete Phase 4: User Story 2 (URL prefixing)
5. Complete Phase 5: User Story 3 (route mounting)
6. **STOP and VALIDATE**: Core API complete, plugins need migration

### Full Feature Delivery

1. Complete Phases 1-5 (MVP)
2. Complete Phase 6: User Story 4 (sidebar prefixing)
3. Complete Phase 7: Migrate both plugins
4. Complete Phase 8: Update registration site
5. Complete Phase 9: Polish and validation

### Note on Breaking Changes

This feature is a breaking change. The WebPlugin interface changes mid-implementation:
- After Phase 2: Interface changed, old plugins won't compile
- After Phase 7: Plugins updated to new pattern
- After Phase 8: Full system working again

Plan for continuous integration: work through all phases in one session or branch to avoid broken intermediate states.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- No tests explicitly requested in spec - validation via manual testing and existing test suite
- Commit after each phase to track progress
- Breaking change: plan for continuous work through all phases
