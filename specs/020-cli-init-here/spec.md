# Feature Specification: CLI Init Here

**Feature Branch**: `020-cli-init-here`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "Update the init function with --here option to set up ZombieKit in the current folder"

## Clarifications

### Session 2025-12-23

- Q: Should `--here` be required for full setup, or should default (no flags) do full setup? → A: Default (no flags) does full setup; `--here` is redundant and removed
- Q: What output verbosity level should be the default behavior? → A: Verbose by default - list every file as it's copied/skipped

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Initialize ZombieKit in Current Repository (Priority: P1)

As a developer, I want to run `brains init` in my project repository to set up ZombieKit with all necessary configuration, Claude Code integration, and templates so I can start using ZombieKit workflows immediately.

**Why this priority**: This is the core functionality of the feature - without this, users cannot set up ZombieKit in their projects at all.

**Independent Test**: Can be fully tested by running `brains init` in a fresh git repository and verifying all expected directories and files are created, then confirming Claude Code skills/commands are accessible.

**Acceptance Scenarios**:

1. **Given** a directory without ZombieKit setup, **When** user runs `brains init`, **Then** the following structure is created:
   - `.claude/` directory
   - `.claude/commands/` directory with all ZombieKit commands copied from embedded filesystem
   - `.brains/` directory
   - `.brains/templates/` directory with all templates copied from embedded filesystem

2. **Given** a directory with an existing `.claude/` folder, **When** user runs `brains init`, **Then** the command creates `.claude/commands/` if missing and copies ZombieKit commands without overwriting existing files

3. **Given** a directory with existing `.brains/` folder, **When** user runs `brains init`, **Then** the command creates `.brains/templates/` if missing and copies templates without overwriting existing files

---

### User Story 2 - View Init Help and Options (Priority: P2)

As a developer, I want to see clear help documentation for the init command so I understand all available options and their effects.

**Why this priority**: Good documentation helps users understand the tool, but the core functionality must work first.

**Independent Test**: Can be tested by running `brains init --help` and verifying all options are documented.

**Acceptance Scenarios**:

1. **Given** user runs `brains init --help`, **When** command executes, **Then** help text displays:
   - Description of the `--global` flag (existing)
   - Description of the `--force` flag
   - Examples of usage

---

### User Story 3 - Force Overwrite Existing Files (Priority: P3)

As a developer, I want a `--force` option to overwrite existing command and template files when I need to update to newer versions of ZombieKit assets.

**Why this priority**: This is an advanced use case for updating existing setups; most users will only run init once.

**Independent Test**: Can be tested by creating a project with existing ZombieKit files, modifying one, then running `brains init --force` and verifying the modified file is replaced with the embedded version.

**Acceptance Scenarios**:

1. **Given** a directory with existing ZombieKit commands and templates, **When** user runs `brains init --force`, **Then** all command and template files are overwritten with the embedded versions

2. **Given** a directory with existing custom files in `.claude/commands/` that are not ZombieKit commands, **When** user runs `brains init --force`, **Then** only ZombieKit command files are overwritten; custom files remain untouched

---

### Edge Cases

- What happens when the current directory is not writable? System displays a clear permission error message
- What happens when embedded filesystems are empty or corrupted? System displays an error indicating the binary may be corrupted and suggests reinstalling
- What happens when run outside a git repository? The command proceeds normally; git is not required
- What happens when a file copy fails mid-operation? System reports the specific file that failed and continues with remaining files, providing a summary at the end
- What happens when `.claude/commands/` already contains a file with the same name as a ZombieKit command? Without `--force`, skip the file and warn user; with `--force`, overwrite it

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: CLI MUST perform full ZombieKit setup when `brains init` is run with no flags (default behavior)
- **FR-002**: CLI MUST embed the `./integrations/claude/commands/` folder contents in the compiled binary at build time
- **FR-003**: CLI MUST embed the `./templates/templates/` folder contents in the compiled binary at build time
- **FR-004**: CLI MUST create `.claude/` directory if it does not exist
- **FR-005**: CLI MUST create `.claude/commands/` directory if it does not exist
- **FR-006**: CLI MUST copy all embedded command files from `integrations/claude/commands/` to `.claude/commands/`
- **FR-007**: CLI MUST create `.brains/` directory if it does not exist
- **FR-008**: CLI MUST create `.brains/templates/` directory if it does not exist
- **FR-009**: CLI MUST copy all embedded template files from `templates/templates/` to `.brains/templates/`
- **FR-010**: CLI MUST NOT overwrite existing files unless `--force` flag is provided
- **FR-011**: CLI MUST display each file as it is copied, skipped, or overwritten (verbose output by default)
- **FR-012**: CLI MUST support `--force` flag to overwrite existing files
- **FR-013**: CLI MUST register the new `.brains` directory in the profile registry after successful initialization
- **FR-014**: Init command MUST be CLI-only (no MCP equivalent required)
- **FR-015**: CLI MUST preserve file permissions from the embedded filesystem when copying files

### Key Entities

- **Embedded Filesystem (Commands)**: Contains all `.md` files from `integrations/claude/commands/` that are Claude Code skills/commands for ZombieKit workflows
- **Embedded Filesystem (Templates)**: Contains all `.md` files from `templates/templates/` that are specification and planning templates
- **Target Directory**: The current working directory where ZombieKit is being initialized

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete ZombieKit setup in under 30 seconds using a single command
- **SC-002**: All files from both embedded filesystems are correctly copied to target directories
- **SC-003**: Running `brains init` twice (without `--force`) results in no data loss or file corruption
- **SC-004**: After initialization, all ZombieKit Claude Code commands are immediately available via `/brains.*` in Claude Code
- **SC-005**: After initialization, all templates are accessible for spec/plan/task generation workflows

## Assumptions

- The `go:embed` directive will be used for embedding files at compile time
- The embedded filesystems will include all files recursively from the source directories
- File permissions will default to 0644 for files and 0755 for directories if not preserved from source
- The existing `--global` flag behavior will remain unchanged; default (no flags) now performs full local setup
- No MCP-based initialization is required as per user specification
