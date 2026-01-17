# Feature Specification: Profile Composition System

**Feature Branch**: `003-profiles`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Implement profiles functionality with MCP interface and CLI methods for managing composable prompts with hierarchical inheritance between local and global .brains/profiles/ directories."

## Clarifications

### Session 2025-12-22

- Q: What fields should be required vs optional in profile frontmatter, and what are defaults? → A: All fields optional with sensible defaults (name derived from filename, inherits=true, includes=[]).
- Q: How should recursion depth be limited during includes resolution? → A: Pre-read all profile files, resolve the full DAG upfront, error if same profile appears twice in any root-to-leaf path (cycle detection, no arbitrary depth limit).
- Q: Where should the registry of known `.brains/` directories be stored? → A: JSON file at `~/.brains/registry.json`, protected by OS-level file lock (flock/syscall.Flock) for concurrent process safety.
- Q: How should composed profiles be formatted in output? → A: Raw concatenation with no separators - just content joined together.
- Q: How many levels of `.brains/` directories should inheritance consider? → A: All `.brains/` directories found walking from CWD up to git root, plus global (`~/.brains/`).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Compose Profiles from Multiple Sources (Priority: P1)

A developer working in a project wants to compose a specialized prompt from multiple profiles. They run a CLI command that merges profiles from their local project and global settings into a single coherent prompt context.

**Why this priority**: This is the core value proposition of the profile system - composing prompts from hierarchical sources with proper resolution and inheritance.

**Independent Test**: Can be fully tested by creating sample profiles in local and global directories, running the compose command, and verifying the output contains correctly merged content with proper precedence.

**Acceptance Scenarios**:

1. **Given** profiles exist in local `.brains/profiles/` and global `~/.brains/profiles/`, **When** user runs `brains profile compose research,database`, **Then** the system outputs merged content with local profiles taking precedence over global ones.
2. **Given** a profile has `inherits: true` in its frontmatter, **When** composing that profile, **Then** parent directory versions of the same-named profile are prepended to the content.
3. **Given** a profile includes other profiles via `includes: [a, b]`, **When** composing that profile, **Then** included profiles appear before the including profile in the output.
4. **Given** the same profile would be included multiple times, **When** composing, **Then** it appears only once at its first occurrence.

---

### User Story 2 - List Available Profiles (Priority: P1)

A developer wants to discover what profiles are available to them from all sources in their current context.

**Why this priority**: Users need visibility into available profiles to effectively compose them - this enables discoverability.

**Independent Test**: Can be tested by creating profiles in various locations and verifying the list command shows all of them with correct source attribution.

**Acceptance Scenarios**:

1. **Given** profiles exist in local `.brains/profiles/` and global `~/.brains/profiles/`, **When** user runs `brains profile list`, **Then** all unique profiles are listed with their source locations.
2. **Given** duplicate profile names exist in local and global directories, **When** listing, **Then** the list shows which source would be used based on precedence (local wins).
3. **Given** `--format json` flag is used, **When** listing, **Then** output is structured JSON with profile metadata.

---

### User Story 3 - Show Individual Profile Content (Priority: P1)

A developer wants to inspect what a specific profile contains, either with inheritance resolved or as raw local content.

**Why this priority**: Understanding what a profile contains before composing is essential for debugging and learning the profile system.

**Independent Test**: Can be tested by creating a profile with inheritance, then verifying show displays resolved content by default and raw content with `--raw` flag.

**Acceptance Scenarios**:

1. **Given** a profile with inheritance enabled, **When** user runs `brains profile show database`, **Then** the fully resolved content (with inherited content) is displayed.
2. **Given** the `--raw` flag is used, **When** showing a profile, **Then** only the local file content is displayed without inheritance.
3. **Given** a profile does not exist, **When** showing it, **Then** an appropriate error message is displayed with suggestions for similar profile names.

---

### User Story 4 - Create New Profile (Priority: P2)

A developer wants to create a new profile with a template structure that includes proper frontmatter.

**Why this priority**: Creating profiles should be easy and result in properly formatted files.

**Independent Test**: Can be tested by running the create command and verifying the file is created with correct frontmatter structure.

**Acceptance Scenarios**:

1. **Given** user runs `brains profile create my-new-profile`, **When** the command completes, **Then** a new file exists at `.brains/profiles/my-new-profile.md` with valid frontmatter template.
2. **Given** a profile with that name already exists locally, **When** creating, **Then** the command fails with an error rather than overwriting.
3. **Given** `--global` flag is used, **When** creating, **Then** the profile is created in `~/.brains/profiles/` instead.

---

### User Story 5 - Validate Profile Configuration (Priority: P2)

A developer wants to check their profile configuration for errors like circular dependencies or missing references.

**Why this priority**: Catching configuration errors early prevents runtime failures during composition.

**Independent Test**: Can be tested by creating profiles with intentional errors and verifying the validate command detects them.

**Acceptance Scenarios**:

1. **Given** profiles with circular includes (A includes B, B includes A), **When** user runs `brains profile validate`, **Then** the circular dependency is reported with the cycle path.
2. **Given** a profile includes a non-existent profile, **When** validating, **Then** the missing reference is reported.
3. **Given** all profiles are valid, **When** validating, **Then** success is reported with no errors.

---

### User Story 6 - MCP Tool Interface for Profile Composition (Priority: P2)

An AI agent needs to compose profiles programmatically via the MCP server interface rather than CLI.

**Why this priority**: MCP integration enables AI agents to leverage the profile system without shell access.

**Independent Test**: Can be tested by starting the MCP server and invoking the profile composition tool with appropriate parameters.

**Acceptance Scenarios**:

1. **Given** the MCP server is running, **When** a client calls the `profile-compose` tool with profile names, **Then** the composed content is returned as tool output.
2. **Given** the `profile-list` tool is called, **When** processing, **Then** available profiles are returned as structured data.
3. **Given** an error occurs during composition, **When** the MCP tool is called, **Then** an appropriate error response is returned with error code and message.

---

### User Story 7 - Initialize Brains Directory (Priority: P2)

A developer wants to set up the brains profile system in their project or globally.

**Why this priority**: Users need a simple way to bootstrap the profile system before they can create and use profiles.

**Independent Test**: Can be tested by running init in a new directory and verifying the expected directory structure is created.

**Acceptance Scenarios**:

1. **Given** no `.brains/` directory exists, **When** user runs `brains init`, **Then** a `.brains/profiles/` directory is created in the current directory.
2. **Given** `--global` flag is used, **When** running init, **Then** `~/.brains/profiles/` is created if it doesn't exist.
3. **Given** `.brains/` already exists, **When** running init, **Then** the command succeeds without modifying existing content.

---

### Edge Cases

- What happens when a profile file is unreadable due to permissions? System returns a permission error with the specific file path.
- How does the system handle corrupted YAML frontmatter in profile files? System reports a parse error with line number and skips the corrupted profile.
- What happens when the home directory (`~`) is not accessible? System proceeds with local profiles only and logs a warning.
- How are profiles handled when called from a directory with no `.brains/` ancestors up to git root? System uses only global profiles if available.
- What happens when a profile name contains special characters or spaces? System normalizes names (lowercase, hyphens) and validates against allowed character set.
- How does the system behave when the same profile is specified multiple times in a compose command? System deduplicates, using first occurrence only.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST resolve profiles from two source locations: local `.brains/profiles/` (project-level) and global `~/.brains/profiles/` (user-level).
- **FR-002**: System MUST give local profiles precedence over global profiles when the same profile name exists in both locations.
- **FR-003**: System MUST parse YAML frontmatter from profile files to extract `name`, `description`, `includes`, and `inherits` fields.
- **FR-004**: System MUST default `inherits` to `true`, `includes` to empty array, and `name` to filename (without `.md` extension) when not specified in a profile's frontmatter.
- **FR-005**: System MUST pre-read all referenced profile files and build a complete dependency DAG before composition begins.
- **FR-006**: System MUST resolve `includes` with depth-first ordering and deduplicate profiles that appear multiple times across different branches.
- **FR-007**: System MUST detect cycles by checking if any profile appears twice in a single root-to-leaf path and error immediately with the cycle path.
- **FR-008**: System MUST walk up the directory tree from CWD to find all `.brains/` directories, stopping at git root, then include global `~/.brains/`. For inheritance, content from parent directories is prepended (global first, then git root level down to CWD).
- **FR-009**: System MUST support JSON output format for all read operations via `--format json` flag.
- **FR-010**: System MUST return structured error responses with error codes, messages, and suggestions for recovery.
- **FR-011**: System MUST expose profile composition functionality via MCP tools interface.
- **FR-012**: System MUST register new `.brains/` directories discovered during composition for future reference by the registry.
- **FR-013**: System MUST use OS-level file locking (flock) when reading/writing the registry to ensure concurrent process safety.
- **FR-014**: System MUST output composed profiles as raw concatenation with no separators between profile contents.

### Key Entities

- **Profile**: A composable unit of prompt content with optional YAML frontmatter and markdown body content. Stored as `.md` files in `.brains/profiles/` directories. Frontmatter fields (all optional): `name` (defaults to filename without extension), `description` (defaults to empty), `includes` (defaults to `[]`), `inherits` (defaults to `true`).
- **ProfileSource**: The origin of a profile - either `local` (project `.brains/profiles/`) or `global` (`~/.brains/profiles/`).
- **CompositionResult**: The merged output of multiple profiles including the final content, profiles used, resolution details, and metadata (character count, estimated tokens, warnings).
- **Registry**: A persistent list of known `.brains/` directories across projects for cross-project profile discovery. Stored as JSON at `~/.brains/registry.json` with OS-level file locking (flock) for concurrent process safety.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can compose 3+ profiles from different sources in a single command and receive merged output within 1 second for typical profile sizes (under 50KB total).
- **SC-002**: Profile resolution correctly applies local-over-global precedence rules in 100% of test cases.
- **SC-003**: Circular dependency detection via DAG pre-resolution identifies all cycles before composition begins and reports the exact cycle path.
- **SC-004**: JSON output mode provides structured data that can be parsed by standard JSON tooling without errors.
- **SC-005**: MCP tools provide equivalent functionality to CLI commands with consistent response formats.
- **SC-006**: Users can discover all available profiles from any project directory without manual configuration.
- **SC-007**: New users can initialize and create their first profile within 2 minutes using CLI commands.

## Assumptions

- Users have write access to their home directory for global profile storage.
- The `~/.brains/` directory can be created if it doesn't exist.
- Profile files use `.md` extension and contain valid UTF-8 text.
- The current working directory can be determined at runtime.
- Git repository root can be detected by finding a `.git` directory.
- Profile names are case-insensitive and normalized to lowercase with hyphens.
