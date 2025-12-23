# Feature Specification: Profile Import Subcommand

**Feature Branch**: `005-profile-import`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Add an import subcommand for the profiles subtool, which allows us to import profiles from a non-brains source into the brains format. For example, `./brains profiles import claude` and it would import the claude agents into the brains profile store. For now, if there are collisions, we will just overwrite the target. It would be good to keep global agents as global brain profiles, and repository agents (if any) as repository agents. Since claude agents have no concept of inherits, during this process, let's assume that they do not."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Import Claude Agents to Brains Profiles (Priority: P1)

A developer wants to migrate their existing Claude agents into the brains profile system to take advantage of brains-specific features like inheritance and cross-project composition. They run `brains profiles import claude` and all their Claude agents are converted to brains profiles.

**Why this priority**: This is the core functionality of the feature - converting Claude agents to brains format enables users to leverage the full brains profile system with their existing agents.

**Independent Test**: Can be fully tested by creating sample Claude agents in `.claude/agents/` and `~/.claude/agents/`, running `brains profiles import claude`, and verifying corresponding profiles exist in `.brains/profiles/` and `~/.brains/profiles/` with correct content conversion.

**Acceptance Scenarios**:

1. **Given** Claude agents exist in `.claude/agents/`, **When** user runs `brains profiles import claude`, **Then** corresponding profiles are created in `.brains/profiles/` with Claude content converted to brains frontmatter format.
2. **Given** Claude agents exist in `~/.claude/agents/`, **When** user runs `brains profiles import claude`, **Then** corresponding profiles are created in `~/.brains/profiles/` (global remains global).
3. **Given** both local and global Claude agents exist, **When** importing, **Then** local agents become local brains profiles and global agents become global brains profiles.

---

### User Story 2 - Overwrite Existing Profiles on Collision (Priority: P1)

A developer has already imported some Claude agents but has updated the original agents and wants to re-import them, overwriting the previous brains versions.

**Why this priority**: The user specified that collisions should overwrite the target. This is essential for the re-import/update workflow.

**Independent Test**: Can be tested by creating a brains profile with a name that matches a Claude agent, running import, and verifying the brains profile content is replaced with the imported Claude agent content.

**Acceptance Scenarios**:

1. **Given** a brains profile named `reviewer` exists and a Claude agent named `reviewer` exists, **When** user runs `brains profiles import claude`, **Then** the brains profile is overwritten with the Claude agent content converted to brains format.
2. **Given** multiple profiles would be overwritten, **When** importing, **Then** all matching profiles are overwritten without prompting (per user specification).
3. **Given** the `--dry-run` flag is used, **When** importing with collisions, **Then** the command reports what would be overwritten without making changes.

---

### User Story 3 - Preview Import with Dry Run (Priority: P2)

A developer wants to see what profiles would be created or overwritten before committing to the import operation.

**Why this priority**: Dry run provides safety and visibility into the import operation, helping users understand impact before execution.

**Independent Test**: Can be tested by running `brains profiles import claude --dry-run` and verifying the output shows planned operations without creating/modifying any files.

**Acceptance Scenarios**:

1. **Given** Claude agents exist that would create new profiles, **When** user runs `brains profiles import claude --dry-run`, **Then** the command lists all profiles that would be created with their target paths.
2. **Given** Claude agents would overwrite existing profiles, **When** running with `--dry-run`, **Then** the command clearly indicates which profiles would be overwritten.
3. **Given** no Claude agents exist, **When** running dry run, **Then** the command reports that no agents were found to import.

---

### User Story 4 - Import Summary Report (Priority: P2)

A developer wants feedback about what was imported after the operation completes.

**Why this priority**: Summary reporting confirms the operation completed successfully and provides accountability.

**Independent Test**: Can be tested by running import with multiple agents and verifying the output includes counts and details of imported profiles.

**Acceptance Scenarios**:

1. **Given** import completes successfully, **When** the command finishes, **Then** a summary is displayed showing: number of profiles created, number overwritten, and their locations.
2. **Given** `--format json` is used, **When** import completes, **Then** the summary is output as structured JSON with lists of created and overwritten profile paths.
3. **Given** some agents fail to import (e.g., parse errors), **When** import completes, **Then** failures are reported with specific error messages while successful imports proceed.

---

### Edge Cases

- What happens when Claude agents directory doesn't exist? System reports that no agents were found and exits gracefully with informational message.
- What happens when brains profiles directory doesn't exist? System creates the necessary `.brains/profiles/` directory structure before importing.
- How are Claude-specific fields (model, color) handled during conversion? These fields are discarded as they have no equivalent in brains profile frontmatter.
- What happens when a Claude agent has invalid frontmatter? System reports the parse error for that agent and continues importing valid agents.
- How is the `includes` field handled for Claude agents that reference other agents? Include references are preserved in brains format, allowing the composed profile to work if referenced profiles are also imported.
- What happens when the user doesn't have write permissions to the target directory? System reports a permission error and fails gracefully.
- What if a Claude agent has an empty body (only frontmatter)? System creates the brains profile with frontmatter and empty body - this is valid.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an `import` subcommand under `brains profiles` that accepts a source type argument.
- **FR-002**: System MUST support `claude` as an import source type, reading from `.claude/agents/` directories.
- **FR-003**: System MUST convert Claude agent frontmatter to brains profile frontmatter format during import.
- **FR-004**: System MUST set `inherits: false` for all imported profiles (Claude agents have no inherits concept).
- **FR-005**: System MUST preserve profile scope: local Claude agents (`.claude/agents/`) become local brains profiles (`.brains/profiles/`), global Claude agents (`~/.claude/agents/`) become global brains profiles (`~/.brains/profiles/`).
- **FR-006**: System MUST overwrite existing brains profiles when a name collision occurs (as specified by user).
- **FR-007**: System MUST support a `--dry-run` flag that shows what would be imported without making changes.
- **FR-008**: System MUST provide a summary report after import showing counts and paths of created/overwritten profiles.
- **FR-009**: System MUST support `--format json` flag for structured output of import results.
- **FR-010**: System MUST create target brains profile directories if they don't exist.
- **FR-011**: System MUST handle partial failures gracefully, continuing to import valid agents when some fail.
- **FR-012**: System MUST preserve the body content of Claude agents unchanged during import.
- **FR-013**: System MUST discard Claude-specific frontmatter fields (model, color) that have no brains equivalent.
- **FR-014**: System MUST preserve `name`, `description`, and `includes` fields from Claude agent frontmatter.

### Key Entities

- **ImportSource**: The external system to import from (currently only `claude`). Extensible to support other sources in the future.
- **ImportResult**: Summary of an import operation including: profiles created, profiles overwritten, profiles failed, error details.
- **ProfileConversion**: The transformation of a source profile format to brains profile format, mapping relevant fields and discarding incompatible ones.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can import all Claude agents from both local and global directories in a single command execution.
- **SC-002**: 100% of valid Claude agents are successfully converted to brains profiles with correct frontmatter mapping.
- **SC-003**: Import operation completes within 5 seconds for typical agent collections (under 50 agents total).
- **SC-004**: Dry run output accurately reflects what the actual import would do (same counts and paths).
- **SC-005**: Users can re-run import multiple times without errors (idempotent overwrite behavior).
- **SC-006**: Import failures for individual agents do not prevent other agents from being imported.

## Assumptions

- The source interface from 004-source-interface is implemented and provides access to Claude agents.
- Claude agents use `.md` extension with YAML frontmatter matching the format defined in 004-source-interface.
- Users accept that Claude-specific fields (model, color) will be lost during import.
- Users accept that overwrite is the default collision behavior (no interactive prompts).
- The brains profile directory structure can be created if it doesn't exist.
- Import is a one-way operation; there is no export from brains back to Claude format.
