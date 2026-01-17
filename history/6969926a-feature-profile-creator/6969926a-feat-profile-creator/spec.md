# Feature Specification: Profile Creator

**Feature Branch**: `6969926a-feature-profile-creator`
**Created**: 2026-01-15
**Status**: Approved
**Input**: User description: "A create-profile workflow that dogfoods the existing research → create → audit → approve cycle. Output goes to local or global profile directories. New profiles become immediately discoverable."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a New Profile via Guided Workflow (Priority: P1)

A developer wants to create a new custom profile for their project. They invoke `/brains.profile.new`, provide a profile name and description, and the system guides them through the research-create-audit cycle to generate a well-structured profile that is immediately usable.

**Why this priority**: This is the core functionality - without it, the feature has no value. It enables the primary use case of creating new profiles.

**Independent Test**: Can be fully tested by invoking the workflow, providing minimal inputs (name, description), and verifying the resulting profile file exists and is discoverable via `profile-list`.

**Acceptance Scenarios**:

1. **Given** no active profile-creation workflow, **When** user invokes `/brains.profile.new` with name "my-domain", **Then** the system prompts for storage location (local/global) and description
2. **Given** storage location selected as "local", **When** workflow completes, **Then** profile is written to `.brains/profiles/my-domain.md`
3. **Given** storage location selected as "global", **When** workflow completes, **Then** profile is written to `~/.brains/profiles/my-domain.md`
4. **Given** profile created successfully, **When** user runs `profile-list`, **Then** the new profile appears in the list with correct name and description

---

### User Story 2 - Profile Content Generation via Research Phase (Priority: P1)

The workflow researches existing profiles and codebase context to generate appropriate profile content. The user provides a description of what the profile should do, and the system researches similar profiles and domain knowledge to create well-structured content.

**Why this priority**: Without intelligent content generation, users would just be creating empty files. The research phase provides the value of AI-assisted authoring.

**Independent Test**: Can be tested by providing a profile purpose (e.g., "help with Go testing patterns") and verifying the generated content includes relevant guidance derived from research.

**Acceptance Scenarios**:

1. **Given** user describes profile purpose as "Go testing best practices", **When** research phase executes, **Then** agent lists existing profiles via `profile-list` and reads 2-3 representative profiles
2. **Given** research complete, **When** create phase executes, **Then** generated profile includes properly formatted frontmatter (name, description, type)
3. **Given** generated content, **When** user reviews, **Then** content is coherent and addresses the stated purpose

---

### User Story 3 - Validation Before Write (Priority: P2)

Before writing the profile to disk, the system validates the profile structure and content. This prevents invalid profiles from entering the system and ensures immediate discoverability.

**Why this priority**: Validation prevents broken profiles that could cause errors in profile composition or listing. Important but not the core functionality.

**Independent Test**: Can be tested by attempting to create a profile with invalid frontmatter or circular includes, and verifying the system rejects it with actionable feedback.

**Acceptance Scenarios**:

1. **Given** generated profile content, **When** validation runs, **Then** system parses YAML frontmatter and reports any syntax errors with line numbers
2. **Given** profile includes non-existent profile "missing-profile", **When** validation runs, **Then** system reports error with similar profile suggestions
3. **Given** all validations pass, **When** user approves, **Then** profile is written to disk via `profile-write` MCP tool
4. **Given** validation fails, **When** user sees feedback, **Then** agent regenerates the profile section with corrections

---

### User Story 4 - Audit and Highlight for Approval (Priority: P2)

After generation, the system runs audit checks on the profile and highlights key decisions for user approval before finalizing.

**Why this priority**: Follows the established workflow pattern and ensures user control over final output. Secondary to core creation functionality.

**Independent Test**: Can be tested by completing generation and verifying audit results are presented, including any issues found and key decisions that need approval.

**Acceptance Scenarios**:

1. **Given** profile generated, **When** audit phase runs, **Then** system checks for: (a) has description, (b) has content body, (c) frontmatter is valid
2. **Given** audit complete with no CRITICAL issues, **When** highlights presented, **Then** agent displays summary showing: profile name, storage location, profile type, and any warnings
3. **Given** user approves highlights (responds "yes" or "approve"), **When** finalization runs, **Then** profile is written and path is displayed

---

### User Story 5 - Storage Location Prompt (Priority: P3)

User is asked where to store the profile (local project vs global user directory) with clear explanation of the implications.

**Why this priority**: Provides flexibility but has sensible defaults. The feature works without this if a default is chosen.

**Independent Test**: Can be tested by starting workflow and verifying the storage prompt appears with clear options and explanations.

**Acceptance Scenarios**:

1. **Given** workflow started, **When** storage prompt appears, **Then** user sees "Local" and "Global" options with descriptions
2. **Given** user selects "Local", **When** proceeding, **Then** target path is set to `.brains/profiles/{name}.md`
3. **Given** user selects "Global", **When** proceeding, **Then** target path is set to `~/.brains/profiles/{name}.md`

---

### Edge Cases

- What happens when a profile with the same name already exists? `profile-write` returns error; agent prompts for overwrite confirmation or suggests alternative name.
- What happens when the target directory doesn't exist? `profile-write` creates `.brains/profiles/` or `~/.brains/profiles/` as needed.
- What happens when the user cancels mid-workflow? Standard conversation cancellation; no files written since write only happens after approval.
- What happens when profile includes create a cycle? Validation via `profile-validate` catches and reports the cycle path (e.g., `[a → b → c → a]`).
- What happens when user provides no description? Agent prompts for description; minimum viable profile requires name only (description derived from purpose).

## Workflow Sequence

```
1. User invokes /brains.profile.new [name] [description]
   ↓
2. Agent prompts for missing inputs (name, description, storage location)
   ↓
3. Research Phase:
   - Call profile-list to see existing profiles
   - Read 2-3 representative profiles to understand structure
   - Output: Mental model of profile patterns (not written to file)
   ↓
4. Create Phase:
   - Generate profile content with frontmatter + body
   - Output: Profile content as markdown string
   ↓
5. Audit Phase:
   - Parse frontmatter for validity
   - Check includes exist (if any)
   - Check for circular dependencies (if includes present)
   - Output: Audit findings classified as CRITICAL/MAJOR/MINOR
   ↓
6. Highlight Phase:
   - Present summary: name, location, type, any issues
   - Wait for user approval ("yes"/"no")
   ↓
7. Write Phase (on approval):
   - Call profile-write MCP tool with content and path
   - Display confirmation with path
```

## Requirements *(mandatory)*

### Functional Requirements

#### Workflow Entry

- **FR-001**: System MUST provide a `/brains.profile.new` command as entry point for the workflow
- **FR-002**: System MUST prompt for profile name if not provided in arguments
- **FR-002a**: System MUST normalize profile names (lowercase, alphanumeric + hyphens only)
- **FR-003**: System MUST prompt for storage location (local vs global) after collecting name and description

#### Research Phase

- **FR-010**: Agent MUST call `profile-list` to enumerate existing profiles
- **FR-011**: Agent MUST read 2-3 representative profiles to understand structure patterns
- **FR-012**: Agent MUST gather context about the profile's intended purpose from user input

#### Create Phase

- **FR-020**: Agent MUST generate profile with valid YAML frontmatter including: `name`, `description`
- **FR-021**: Agent MUST generate profile body content based on research and user description
- **FR-022**: Agent MAY include optional frontmatter fields: `includes`, `inherits`, `type`
- **FR-023**: Generated frontmatter MUST conform to schema: `name` (string), `description` (string), `includes` (string[]), `inherits` (bool, default true), `type` (enum: skill|action|domain|step, default: domain)

#### Validation (Audit Phase)

- **FR-030**: Agent MUST parse frontmatter as YAML and report syntax errors with context
- **FR-031**: Agent MUST verify included profiles exist via `profile-list`
- **FR-032**: Agent MUST detect circular dependencies by analyzing include chains
- **FR-033**: Agent MUST classify issues as CRITICAL (blocks write), MAJOR (should fix), MINOR (acceptable)

#### Write Phase

- **FR-040**: System MUST provide `profile-write` MCP tool with parameters: `name` (string), `content` (string), `location` ("local"|"global"), `overwrite` (bool, default false)
- **FR-041**: `profile-write` MUST create target directory if it doesn't exist
- **FR-042**: `profile-write` MUST return error if profile exists and `overwrite` is false
- **FR-043**: `profile-write` MUST write atomically (temp file + rename)
- **FR-044**: `profile-write` MUST return the absolute path of the written file

#### Discoverability

- **FR-050**: New profiles MUST be immediately discoverable via `profile-list` after write (no restart required)
- **FR-051**: New profiles MUST be immediately usable via `profile-compose` after write

### MCP Tool: profile-write

**Purpose**: Write a validated profile to disk at the specified location.

**Parameters**:
```json
{
  "name": "string (required) - Profile name (will be used as filename)",
  "content": "string (required) - Full profile content including frontmatter",
  "location": "string (required) - 'local' or 'global'",
  "overwrite": "boolean (optional, default false) - Allow overwriting existing profile"
}
```

**Response** (success):
```json
{
  "success": true,
  "path": "/absolute/path/to/.brains/profiles/name.md"
}
```

**Response** (error - exists):
```json
{
  "success": false,
  "error": "PROFILE_EXISTS",
  "message": "Profile 'name' already exists at /path/to/name.md",
  "hint": "Use overwrite: true to replace, or choose a different name"
}
```

### Key Entities

- **Profile**: A markdown file with YAML frontmatter containing name, description, optional includes/inherits/type, and markdown body content
- **Storage Location**: Either "local" (`.brains/profiles/`) or "global" (`~/.brains/profiles/`)
- **Profile Frontmatter Schema**:
  ```yaml
  name: string        # Required, must match filename (without .md)
  description: string # Required, human-readable purpose
  includes: string[]  # Optional, profiles to compose before this one
  inherits: bool      # Optional, default true, prepend parent versions
  type: enum          # Optional, one of: skill, action, domain, step
  ```

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Workflow completes with ≤5 user interactions (name, description, location, approval, optional overwrite)
- **SC-002**: Generated profiles pass validation 100% of the time before write attempt
- **SC-003**: New profiles appear in `profile-list` output immediately after creation
- **SC-004**: New profiles can be composed via `profile-compose` immediately after creation
- **SC-005**: Workflow follows the research → create → audit → highlight pattern
- **SC-006**: Invalid profile attempts are caught with error messages that include: what's wrong, where it's wrong, how to fix it

## Clarifications

### C1: Working Directory Context (2026-01-15)
**Question**: What should happen when `location: "local"` is used but there's no clear project root?
**Answer**: Use MCP server's working directory, which is already passed via the `dir` parameter to all tools. This follows the existing pattern.

### C2: Profile Type Default (2026-01-15)
**Question**: When the user doesn't specify a profile type, what should the default be?
**Answer**: Default to `domain` type. This is the most general-purpose type for user-created profiles that provide context/knowledge.
