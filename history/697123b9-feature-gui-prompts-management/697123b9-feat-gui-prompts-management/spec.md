# Feature Specification: GUI Prompts Management

**Feature Branch**: `697123b9-feature-gui-prompts-management`
**Created**: 2026-01-21
**Status**: Approved
**MVP Scope**: P1 (List, Filter, Search, View)
**Input**: User description: "Update the GUI to support workflows as well as profiles. Create a new navbar top-level section called 'Prompts', with sub-sections for workflows, actions/steps, and domain agents. Allow filtering by categories, sorting by date/name, with full CRUD for all prompt types."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse All Prompts (Priority: P1)

As a user, I want to see all available prompts (workflows, profiles, steps) in one unified view so I can discover what's available across the system.

**Why this priority**: Core discovery functionality - users must see what exists before they can manage it.

**Independent Test**: Can be fully tested by navigating to `/prompts` and verifying all prompt types appear with correct metadata.

**Acceptance Scenarios**:

1. **Given** I'm logged into the GUI, **When** I click "Prompts" in the sidebar, **Then** I see a list of all workflows, profiles, and steps with their name, type, source, and description.
2. **Given** I'm on the Prompts page, **When** prompts exist in local, global, and embedded locations, **Then** all three sources are displayed with appropriate badges.
3. **Given** I'm on the Prompts page, **When** a profile is shadowed (local overrides global), **Then** the shadowed profile shows a visual indicator.

---

### User Story 2 - Filter and Search Prompts (Priority: P1)

As a user, I want to filter prompts by category and source, and search by name/description, so I can quickly find what I need.

**Why this priority**: Essential for usability when many prompts exist.

**Independent Test**: Can be tested by applying various filters and verifying the list updates correctly.

**Acceptance Scenarios**:

1. **Given** I'm on the Prompts page, **When** I click the "Workflows" tab, **Then** only workflows are displayed.
2. **Given** I'm on the Prompts page, **When** I select "Local" from the source filter, **Then** only local prompts are shown.
3. **Given** I'm on the Prompts page, **When** I type "feat" in the search box, **Then** only prompts with "feat" in name or description appear.
4. **Given** I have filters applied, **When** I click "Clear filters", **Then** all prompts are shown again.

---

### User Story 3 - View Prompt Details (Priority: P1)

As a user, I want to view the full content of any prompt so I can understand what it does and how it's configured.

**Why this priority**: Users must read prompts to understand/use them.

**Independent Test**: Can be tested by clicking on any prompt and verifying the detail view shows all metadata and content.

**Acceptance Scenarios**:

1. **Given** I'm on the Prompts list, **When** I click on a prompt name, **Then** I see a detail page with all YAML frontmatter fields and the markdown body.
2. **Given** I'm viewing a profile, **When** it has "includes" specified, **Then** I see the list of included profiles.
3. **Given** I'm viewing an embedded prompt, **When** I look at the actions, **Then** the edit/delete buttons are disabled or hidden.

---

### User Story 4 - Sort Prompts (Priority: P2)

As a user, I want to sort prompts by name, type, or source so I can organize my view.

**Why this priority**: Improves usability but not blocking for basic functionality.

**Independent Test**: Can be tested by clicking sort controls and verifying order changes.

**Acceptance Scenarios**:

1. **Given** I'm on the Prompts page, **When** I click the "Name" column header, **Then** prompts are sorted alphabetically.
2. **Given** prompts are sorted by name ascending, **When** I click "Name" again, **Then** sort order reverses (descending).
3. **Given** I'm on the Prompts page, **When** I select "Source" from sort dropdown, **Then** prompts are grouped by source (local, global, embedded).

---

### User Story 5 - Create New Profile (Priority: P2)

As a user, I want to create a new profile so I can add custom domain agents to my project.

**Why this priority**: CRUD completion - users need to add new prompts.

**Independent Test**: Can be tested by filling out the create form and verifying the file is created.

**Acceptance Scenarios**:

1. **Given** I'm on the Prompts page, **When** I click "New Prompt" and select "Profile", **Then** I see a form with fields for name, description, type, includes, inherits, model, color, and content.
2. **Given** I'm on the create form, **When** I fill required fields and submit, **Then** the profile is saved to the selected location (local/global).
3. **Given** I submit a profile with invalid YAML, **When** the server validates, **Then** I see an error message describing the issue.
4. **Given** I create a profile with name "test-agent", **When** it saves successfully, **Then** I'm redirected to the detail view and "test-agent.md" exists in the selected location.

---

### User Story 6 - Create New Workflow (Priority: P2)

As a user, I want to create a new workflow so I can define custom entry points for work.

**Why this priority**: Workflows are key orchestration primitives.

**Independent Test**: Can be tested by creating a workflow and verifying it appears in the list.

**Acceptance Scenarios**:

1. **Given** I'm on the create form with "Workflow" selected, **When** I view the form, **Then** I see fields for name, description, and content (markdown body).
2. **Given** I fill out the workflow form, **When** I submit, **Then** the workflow is saved to the selected location.

---

### User Story 7 - Create New Step (Priority: P2)

As a user, I want to create a new step so I can define custom phases within initiatives.

**Why this priority**: Steps enable workflow customization.

**Independent Test**: Can be tested by creating a step and verifying it appears in the list.

**Acceptance Scenarios**:

1. **Given** I'm on the create form with "Step" selected, **When** I view the form, **Then** I see fields for name, description, profiles (multi-select), files (list input), and content.
2. **Given** I fill out the step form with "profiles: [research, feature]", **When** I submit, **Then** the step is saved with the correct frontmatter.

---

### User Story 8 - Edit Existing Prompt (Priority: P2)

As a user, I want to edit prompts I've created so I can update them over time.

**Why this priority**: Users must iterate on their prompts.

**Independent Test**: Can be tested by editing a local prompt and verifying changes persist.

**Acceptance Scenarios**:

1. **Given** I'm viewing a local prompt, **When** I click "Edit", **Then** I see the edit form pre-populated with current values.
2. **Given** I'm on the edit form, **When** I change the description and save, **Then** the file is updated and I see the new description.
3. **Given** I'm viewing an embedded prompt, **When** I look for the edit button, **Then** it's not visible (embedded is read-only).

---

### User Story 9 - Delete Prompt (Priority: P3)

As a user, I want to delete prompts I no longer need so I can keep my workspace clean.

**Why this priority**: Less common operation, not blocking core usage.

**Independent Test**: Can be tested by deleting a local prompt and verifying it's removed.

**Acceptance Scenarios**:

1. **Given** I'm viewing a local prompt, **When** I click "Delete", **Then** I see a confirmation dialog.
2. **Given** I confirm deletion, **When** the operation completes, **Then** the file is removed and I'm redirected to the list.
3. **Given** I'm viewing a global prompt, **When** I try to delete, **Then** I see a warning that this affects all projects.
4. **Given** I'm viewing an embedded prompt, **When** I look for delete, **Then** it's not available.

---

### User Story 10 - Copy/Fork Embedded Prompt (Priority: P3)

As a user, I want to copy an embedded prompt to local/global so I can customize it.

**Why this priority**: Enables customization workflow without editing embedded.

**Independent Test**: Can be tested by forking an embedded prompt and verifying the copy exists.

**Acceptance Scenarios**:

1. **Given** I'm viewing an embedded prompt, **When** I click "Copy to Local", **Then** a copy is created in `.brains/{type}s/` with the same name.
2. **Given** I copy a prompt, **When** I view the list, **Then** both versions appear (local shadows embedded).

---

### Edge Cases

- What happens when a prompt file has invalid YAML frontmatter? → Show error message, don't crash
- What happens when `.brains/` directory doesn't exist for a new project? → Create it on first save
- What happens when user tries to create a prompt with a name that already exists? → Show error, offer to overwrite or rename
- How does search handle prompts with empty descriptions? → Search by name only
- What happens when profiles have circular includes? → Detect and show validation error

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display all prompts (workflows, profiles, steps) from local, global, and embedded sources in a unified view
- **FR-002**: System MUST allow filtering prompts by category (workflow, profile, step)
- **FR-003**: System MUST allow filtering prompts by source (local, global, embedded)
- **FR-004**: System MUST allow searching prompts by name and description
- **FR-005**: System MUST allow sorting prompts by name (asc/desc), type, or source
- **FR-006**: System MUST display prompt detail including all frontmatter fields and markdown content
- **FR-007**: System MUST allow creating new prompts (profiles, workflows, steps) with appropriate fields
- **FR-008**: System MUST allow editing local and global prompts
- **FR-009**: System MUST prevent editing/deleting embedded prompts
- **FR-010**: System MUST allow deleting local and global prompts with confirmation
- **FR-011**: System MUST validate YAML frontmatter on create/edit
- **FR-012**: System MUST support copying embedded prompts to local/global locations
- **FR-013**: System MUST indicate shadowed prompts (local overriding global/embedded)
- **FR-014**: System MUST create directories if they don't exist on save

### Key Entities

- **Prompt**: Abstract base - has name, description, source (local/global/embedded), path, content
- **Profile**: Prompt with additional fields: type (domain/action/step/skill), includes, inherits, model, color
- **Workflow**: Prompt with name, description, content (simpler structure)
- **Step**: Prompt with profiles (list), files (list), type=step

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can navigate to Prompts page and see all available prompts within 2 seconds
- **SC-002**: Filtering by category/source updates the list within 500ms
- **SC-003**: Search results appear within 500ms of typing
- **SC-004**: Users can complete the create→save workflow without referencing documentation
- **SC-005**: All CRUD operations work correctly for local and global prompts
- **SC-006**: Embedded prompts are clearly distinguished and protected from modification

## Testing Requirements *(mandatory)*

### Test Strategy

- Integration tests for HTTP handlers (primary focus)
- E2E tests for critical user journeys (list, create, edit)
- Unit tests for YAML frontmatter parsing if complex
- Test frameworks: Go testing with httptest for handlers

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | GET /prompts returns all prompts with correct source badges |
| FR-002 | Integration | GET /prompts?category=workflow filters correctly |
| FR-003 | Integration | GET /prompts?source=local filters correctly |
| FR-004 | Integration | GET /prompts?q=feat searches name/description |
| FR-005 | Integration | GET /prompts?sort=name&order=asc sorts correctly |
| FR-006 | Integration | GET /prompts/{type}/{name} returns full detail |
| FR-007 | Integration | POST /prompts/{type} creates new prompt |
| FR-008 | Integration | PUT /prompts/{type}/{name} updates prompt |
| FR-009 | Integration | PUT/DELETE on embedded returns 403 Forbidden |
| FR-010 | Integration | DELETE /prompts/{type}/{name} removes file |
| FR-011 | Integration | POST with invalid YAML returns 400 with error |
| FR-012 | Integration | POST /prompts/{type}/{name}/copy creates copy |
| FR-013 | Integration | List response includes shadowed flag |
| FR-014 | Integration | Save creates parent directories |

### Edge Case Coverage

- Invalid YAML frontmatter → Integration test: POST with malformed YAML, expect 400
- Missing .brains directory → Integration test: Save to new project, verify directory created
- Duplicate name → Integration test: Create with existing name, expect 409 Conflict
- Circular includes → Integration test: Create profile with circular dependency, expect validation error
