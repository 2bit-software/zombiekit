# Tasks: Profile Type Classification

**Input**: Design documents from `/specs/018-profile-type/`
**Prerequisites**: plan.md, spec.md, data-model.md, research.md

**Tests**: Tests are included for the core parsing functionality as this is foundational.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md:
- Go source: `internal/`
- Profile package: `internal/profile/`
- Web templates: `internal/webplugins/profiles/templates/`

---

## Phase 1: Setup

**Purpose**: No setup needed - this is an extension of existing code

*Phase skipped - all infrastructure already exists*

---

## Phase 2: Foundational (Data Model Extension)

**Purpose**: Add Type field to all data structures - MUST be complete before UI stories

**⚠️ CRITICAL**: US1 and US2 cannot begin until Type field exists in all structs

- [X] T001 [P] Add Type field to ProfileFrontmatter struct in internal/profile/types.go
- [X] T002 [P] Add Type field to Profile struct in internal/profile/types.go
- [X] T003 [P] Add Type field to ListEntry struct in internal/profile/types.go
- [X] T004 [P] Add Type field to ShowResult struct in internal/profile/types.go
- [X] T005 Update ParseProfile to populate Type from frontmatter in internal/profile/frontmatter.go
- [X] T006 Add test for Type parsing in internal/profile/frontmatter_test.go

**Checkpoint**: Type field exists in all structs, parsing works and is tested

---

## Phase 3: User Story 3 - Define Profile Type in Frontmatter (Priority: P1) 🎯 MVP

**Goal**: Profile authors can specify type in YAML frontmatter

**Independent Test**: Create a profile with `type: action` and verify it parses correctly via CLI `brains profile show`

*Note: This is implemented by Phase 2 tasks. No additional tasks needed - the foundational work IS the implementation.*

**Verification**:
- [X] T007 [US3] Verify CLI `brains profile show` outputs type field in JSON in internal/cli/profile.go

**Checkpoint**: Profile type is parsed and visible in CLI output

---

## Phase 4: User Story 1 - View Profile Type in List (Priority: P1)

**Goal**: Users see type badges in the profiles list web page

**Independent Test**: Load profiles list page, verify colored badges appear for profiles with types

### Implementation for User Story 1

- [X] T008 [US1] Add type badge template logic in internal/webplugins/profiles/templates/list.html
- [X] T009 [US1] Add color-coded CSS classes for action (purple), domain (green), step (blue), unknown (gray) badges

**Checkpoint**: Profiles list shows colored type badges

---

## Phase 5: User Story 2 - View Profile Type in Detail View (Priority: P2)

**Goal**: Users see profile type in the detail view metadata section

**Independent Test**: Navigate to a profile detail page, verify type appears in metadata

### Implementation for User Story 2

- [X] T010 [US2] Add type display to metadata section in internal/webplugins/profiles/templates/view.html

**Checkpoint**: Profile detail view shows type in metadata

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and cleanup

- [X] T011 [P] Verify backwards compatibility - profiles without type field still work
- [X] T012 [P] Verify case insensitivity - "Action", "action", "ACTION" all display correctly
- [X] T013 [P] Verify unknown types display with gray styling
- [X] T014 Run quickstart.md validation - test all example profiles

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 2 (Foundational)**: No dependencies - starts immediately
- **Phase 3 (US3)**: Depends on Phase 2 completion
- **Phase 4 (US1)**: Depends on Phase 2 completion
- **Phase 5 (US2)**: Depends on Phase 2 completion
- **Phase 6 (Polish)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 3 (P1)**: Foundational - must complete first (or in parallel with data model)
- **User Story 1 (P1)**: Depends on Phase 2 (Type field in ListEntry)
- **User Story 2 (P2)**: Depends on Phase 2 (Type field in ShowResult)
- **User Story 4 (P3)**: DEFERRED to future iteration (per plan.md)

### Within Each Phase

- T001-T004 can run in parallel (different structs in same file)
- T005 depends on T001-T002 (needs ProfileFrontmatter and Profile structs)
- T006 depends on T005 (tests the parsing logic)

### Parallel Opportunities

```text
Phase 2 parallel execution:
  T001, T002, T003, T004 → all can run together (single file, different sections)

After Phase 2 complete, US1 and US2 can run in parallel:
  T008-T009 (list template)  |  T010 (view template)
```

---

## Parallel Example: Phase 2

```bash
# All struct modifications can be done together:
Task: "Add Type field to ProfileFrontmatter struct in internal/profile/types.go"
Task: "Add Type field to Profile struct in internal/profile/types.go"
Task: "Add Type field to ListEntry struct in internal/profile/types.go"
Task: "Add Type field to ShowResult struct in internal/profile/types.go"

# Then sequential:
Task: "Update ParseProfile to populate Type in internal/profile/frontmatter.go"
Task: "Add test for Type parsing in internal/profile/frontmatter_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 3 + User Story 1)

1. Complete Phase 2: Data model extension (T001-T006)
2. Complete Phase 3: Verify CLI works (T007)
3. Complete Phase 4: List view badges (T008-T009)
4. **STOP and VALIDATE**: Test profiles list with type badges
5. Deploy/demo if ready

### Full Feature

1. Complete MVP above
2. Add Phase 5: Detail view (T010)
3. Complete Phase 6: Polish (T011-T014)

### Task Counts

| Phase | Task Count | Description |
|-------|------------|-------------|
| Phase 2 (Foundational) | 6 | Data model + parsing |
| Phase 3 (US3) | 1 | CLI verification |
| Phase 4 (US1) | 2 | List view badges |
| Phase 5 (US2) | 1 | Detail view |
| Phase 6 (Polish) | 4 | Verification |
| **Total** | **14** | |

---

## Notes

- US4 (Filter by Type) is deferred per plan.md - not included in this task list
- All changes are to existing files - no new files created
- Pattern follows existing Model/Color field implementation
- Type field uses `json:"type,omitempty"` tag for optional serialization
- Templates use case-insensitive comparison for badge colors, preserve original casing for display
