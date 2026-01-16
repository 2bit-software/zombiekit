# Feature Specification: Profile Type Classification

**Feature Branch**: `018-profile-type`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "I need to introduce another 'type' of profile. The three types are: 1. Action 2. Domain 3. Step. Steps are the profiles that will exist for 'specify' or 'research' or 'clarify'. Action/Domain profiles are the prompts we compose to do the work *inside* a step. I want to add the property to the profiles frontmatter, and expose the property to the frontend."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Profile Type in List (Priority: P1)

As a user browsing profiles in the web interface, I want to see the type of each profile (Action, Domain, or Step) so I can quickly understand what role each profile plays in the workflow composition system.

**Why this priority**: This is the core visibility feature that enables users to distinguish between different kinds of profiles at a glance. Without this, users cannot effectively organize or select appropriate profiles for their needs.

**Independent Test**: Can be fully tested by loading the profiles list page and verifying type badges appear next to profile names for profiles that have a type defined.

**Acceptance Scenarios**:

1. **Given** a profile file with `type: action` in its frontmatter, **When** I view the profiles list, **Then** I see an "Action" badge displayed next to the profile name.
2. **Given** a profile file with `type: step` in its frontmatter, **When** I view the profiles list, **Then** I see a "Step" badge displayed next to the profile name.
3. **Given** a profile file with `type: domain` in its frontmatter, **When** I view the profiles list, **Then** I see a "Domain" badge displayed next to the profile name.
4. **Given** a profile file with no type defined in its frontmatter, **When** I view the profiles list, **Then** no type badge is displayed (the field is optional).

---

### User Story 2 - View Profile Type in Detail View (Priority: P2)

As a user viewing a specific profile's details, I want to see the profile's type prominently displayed in the metadata section so I understand how this profile is intended to be used.

**Why this priority**: Once users can see types in the list, they need the same information in the detail view for consistency and to confirm their selection.

**Independent Test**: Can be fully tested by navigating to a profile detail page and verifying the type is displayed in the metadata section.

**Acceptance Scenarios**:

1. **Given** a profile with `type: step` in its frontmatter, **When** I view the profile detail page, **Then** I see "Type: Step" in the metadata section.
2. **Given** a profile with no type defined, **When** I view the profile detail page, **Then** the Type field is not shown in the metadata section.

---

### User Story 3 - Define Profile Type in Frontmatter (Priority: P1)

As a profile author, I want to specify a type (Action, Domain, or Step) in my profile's YAML frontmatter so that the system and other users understand the profile's purpose.

**Why this priority**: This is the foundational data entry mechanism that enables all other features. Without the ability to define types, the display features have nothing to show.

**Independent Test**: Can be fully tested by creating a profile with the type field and verifying it is parsed correctly via CLI or API.

**Acceptance Scenarios**:

1. **Given** I create a profile file with frontmatter `type: action`, **When** the profile is loaded, **Then** the profile's type property is set to "action".
2. **Given** I create a profile file with frontmatter `type: domain`, **When** the profile is loaded, **Then** the profile's type property is set to "domain".
3. **Given** I create a profile file with frontmatter `type: step`, **When** the profile is loaded, **Then** the profile's type property is set to "step".
4. **Given** I create a profile file without a type field, **When** the profile is loaded, **Then** the profile's type property is empty (no default is applied).

---

### User Story 4 - Filter Profiles by Type (Priority: P3)

As a user with many profiles, I want to filter the profiles list by type so I can quickly find profiles that serve a specific purpose.

**Why this priority**: This is a convenience feature that builds on the core type visibility. It becomes valuable once users have enough profiles that browsing is inefficient.

**Independent Test**: Can be fully tested by applying a type filter and verifying only profiles of that type are shown.

**Acceptance Scenarios**:

1. **Given** I am on the profiles list with profiles of various types, **When** I select the "Step" filter, **Then** only profiles with type "step" are displayed.
2. **Given** I am on the profiles list with a filter active, **When** I clear the filter, **Then** all profiles are displayed.
3. **Given** I filter by "Action", **When** there are no action profiles, **Then** an appropriate empty state message is shown.

---

### Edge Cases

- What happens when a profile specifies an invalid type value (e.g., `type: workflow`)? The system should preserve the value but may display it differently (or log a warning) since it's not one of the known types.
- How does the system handle case sensitivity (e.g., `type: Action` vs `type: action`)? The system should treat type values as case-insensitive for matching but preserve the original casing for display.
- What happens when a profile is loaded via CLI compose command? The type should be included in the profile metadata but has no effect on composition behavior (types are for organization/classification only).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a new optional `type` field in profile YAML frontmatter.
- **FR-002**: System MUST accept exactly three known type values: "action", "domain", and "step" (case-insensitive matching).
- **FR-003**: System MUST preserve and display the original casing of type values.
- **FR-004**: System MUST treat the type field as optional with no default value.
- **FR-005**: Profiles list in web interface MUST display the type as a visual badge when present.
- **FR-006**: Profile detail view in web interface MUST display the type in the metadata section when present.
- **FR-007**: System MUST include the type field in profile list and show API responses.
- **FR-008**: System MUST accept unknown type values without error (for forward compatibility), but may visually distinguish them from known types.

### Key Entities

- **Profile**: Extended to include an optional Type field. Type values are: "action" (prompts for doing work within steps), "domain" (domain knowledge prompts composed within steps), or "step" (workflow step profiles like "specify", "research", "clarify").
- **ProfileFrontmatter**: Extended to parse the `type` YAML field.
- **ListEntry/ShowResult**: Extended to include type in API/display responses.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can distinguish profile types visually within 1 second of viewing the profiles list.
- **SC-002**: 100% of profiles with type defined in frontmatter correctly display that type in both list and detail views.
- **SC-003**: Profile authors can add the type field to any profile file and see it reflected in the UI on next page load.
- **SC-004**: Existing profiles without a type field continue to function identically to before this feature (no breaking changes).

## Assumptions

- The term "type" is acceptable despite being generic; the three specific values (action, domain, step) provide sufficient context.
- Step profiles correspond to workflow phases like "specify", "research", or "clarify".
- Action and Domain profiles are composed together to execute work within a Step.
- Type classification is purely organizational and does not affect how profiles are composed or executed.
- The web frontend is the primary consumer of this feature; CLI output may also include the type but with lower priority.
