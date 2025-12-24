# Feature Specification: Initiative Feature Workflow

**Feature Branch**: `022-initiative-feature-workflow`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "New initiative/feature workflow: builds initiative folders in ./history with nested structure, creates state files, copies templates from source, and guides LLM through spec workflow via MCP step endpoint"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start a New Feature Initiative (Priority: P1)

A developer wants to start working on a new feature. They invoke the MCP step endpoint with `step="feature"` and a name for their feature. The system creates the complete initiative structure, copies relevant templates, and returns the composed profile prompt that guides the LLM through the specification workflow.

**Why this priority**: This is the core functionality - the single entry point that orchestrates the entire "new feature" experience. Without this, developers must manually create folders and invoke multiple steps.

**Independent Test**: Can be fully tested by calling `mcp_zombiekit__step` with `step="feature"` and `name="user-auth"`, verifying it creates the expected folder structure in `./history/`, copies templates, updates state, and returns a response containing the spec workflow profile.

**Acceptance Scenarios**:

1. **Given** a project with a `.brains` folder, **When** the developer calls the MCP step endpoint with `step="feature"` and `name="user-auth"`, **Then** a new initiative folder is created at `./history/{hex-timestamp}-feat-user-auth/` containing: (1) INITIATIVE.md with metadata, (2) copied spec template ready for use, and the state file is updated to track this as the active initiative.

2. **Given** the feature step is invoked successfully, **When** the response is returned, **Then** it includes: (1) directive text explaining the spec workflow, (2) the history folder path, (3) list of template files now available, (4) composed profile prompt for the "specify" workflow.

3. **Given** an existing active initiative, **When** the developer invokes `step="feature"` with a new name, **Then** the new initiative is created and becomes the active initiative, with the previous one remaining in history but no longer active.

---

### User Story 2 - Create Sub-Initiative Within Parent Initiative (Priority: P2)

A developer has an existing initiative (e.g., a large feature) and wants to break it into smaller sub-initiatives. They invoke the feature step with the parent initiative context, creating a nested structure.

**Why this priority**: Sub-initiatives enable breaking large features into manageable pieces, but the basic single-initiative workflow must work first.

**Independent Test**: Can be tested by first creating a parent initiative, then invoking `step="feature"` with `parent` parameter pointing to the parent, verifying nested folder creation.

**Acceptance Scenarios**:

1. **Given** an active initiative at `./history/{id}-feat-big-feature/`, **When** the developer calls the step endpoint with `step="feature"`, `name="sub-part"`, and `parent` parameter, **Then** a sub-initiative folder is created at `./history/{id}-feat-big-feature/{new-id}-feat-sub-part/`.

2. **Given** a sub-initiative is created, **When** the state file is updated, **Then** it tracks the sub-initiative as active while preserving the parent relationship.

---

### User Story 3 - Customize Template Copying (Priority: P3)

A developer or team has customized their workflow templates. When creating a new feature initiative, the system should use local templates if available, falling back to embedded defaults.

**Why this priority**: Customization enables teams to adapt the workflow to their needs, but the core workflow must work with defaults first.

**Independent Test**: Can be tested by placing custom templates in `.brains/templates/`, invoking `step="feature"`, and verifying the custom templates are copied instead of defaults.

**Acceptance Scenarios**:

1. **Given** custom templates exist in `.brains/templates/`, **When** a new feature initiative is created, **Then** the system copies templates from `.brains/templates/` with fallback to embedded defaults for missing files.

2. **Given** no custom templates exist, **When** a new feature initiative is created, **Then** the system copies all templates from the embedded defaults.

---

### Edge Cases

- What happens when `./history` folder does not exist? The system creates it automatically.
- What happens when a feature with the same name already exists? A new unique ID is generated (timestamp-based), so naming collisions are impossible.
- What happens if template copying fails partway through? The system reports the error but leaves the initiative folder created (partial state is better than nothing).
- What happens if the step is called without a name? Return a clear error requiring the `name` parameter.
- How does the system handle very long feature names? Names are normalized (slugified) with a reasonable maximum length.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a "feature" step accessible via `mcp_zombiekit__step` endpoint with `step="feature"`.
- **FR-002**: System MUST accept a `name` parameter (required) specifying the feature name/slug.
- **FR-003**: System MUST create an initiative folder in `./history/` with naming format `{hex-timestamp}-feat-{normalized-name}`.
- **FR-004**: System MUST create an `INITIATIVE.md` file in the new folder containing: name, type (feature), status (active), created timestamp, and ID.
- **FR-005**: System MUST copy spec template files to the new initiative folder from: (1) `.brains/templates/` if present, or (2) embedded defaults.
- **FR-006**: System MUST update the state file (`.brains/active.json`) to track the new initiative as active.
- **FR-007**: System MUST return a StepResponse containing: directive text, history folder path, list of files created, and composed profile prompt for the spec workflow.
- **FR-008**: System MUST support an optional `parent` parameter to create sub-initiatives nested within a parent initiative folder.
- **FR-009**: System MUST normalize feature names to slug format (lowercase, alphanumeric, hyphens only).
- **FR-010**: System MUST validate that the working directory contains a `.brains` folder before execution.
- **FR-011**: System MUST return clear error messages when required parameters are missing or invalid.
- **FR-012**: System MUST compose the profile prompt using the existing profile composition system from `internal/profile`.
- **FR-013**: The directive returned MUST guide the LLM to perform the specification workflow (research, spec writing, auditing).

### Key Entities

- **Feature Initiative**: A specialized initiative with type "feature" that includes template files copied for the spec workflow.
- **Sub-Initiative**: A child initiative nested within a parent initiative's folder, enabling hierarchical organization.
- **Template Source**: The location from which templates are copied, with local (`.brains/templates/`) taking precedence over embedded defaults.
- **Feature Step Response**: The structured output from the "feature" step, extending StepResponse with feature-specific guidance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new feature initiative can be created and ready for use with a single call completing in under 2 seconds.
- **SC-002**: 100% of feature step invocations with valid parameters result in a complete initiative structure with all required files.
- **SC-003**: The returned profile prompt enables the LLM to proceed with the specification workflow without requiring additional manual guidance.
- **SC-004**: Template copying uses local customizations when present, with 100% fallback coverage to embedded defaults.
- **SC-005**: Error messages for invalid inputs clearly describe the problem and suggest corrective action.

## Assumptions

- The existing initiative service (`internal/initiative`) and step service (`internal/step`) from spec 021 are implemented and available.
- The profile composition system (`internal/profile`) is available for creating the composed prompt.
- Templates will be stored in `templates/templates/` (embedded) and optionally overridden in `.brains/templates/` (local).
- The "feature" step builds on top of the "init" step functionality but adds template copying and profile composition.
- The naming convention uses `feat` as a short prefix (consistent with git branch naming conventions like `feat/`, `fix/`, `refactor/`).
