# Feature Specification: Skill Installer

**Feature Branch**: `69cedeed-feature-skill-installer`
**Created**: 2026-04-02
**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Install skill globally via CLI (Priority: P1)

A developer wants to make a zombiekit skill available in all their Claude projects. They run a single brains CLI command to install it into their global Claude skills directory.

**Why this priority**: The global install is the primary use case — most skills are cross-project utilities. Without this, the feature has no value.

**Independent Test**: Run `brains skill install <name> --global` and verify a valid SKILL.md appears at `~/.claude/skills/<name>/SKILL.md` with correct frontmatter and a body that delegates to profile-compose.

**Acceptance Scenarios**:

1. **Given** a valid profile exists named `create-pr`, **When** user runs `brains skill install create-pr --global`, **Then** `~/.claude/skills/create-pr/SKILL.md` is created with correct frontmatter and a body instructing Claude to call `mcp__zombiekit__profile-compose` with `profiles: ["create-pr"]`.
2. **Given** the skill directory already exists, **When** user runs the install command again, **Then** the file is overwritten (reinstall) with no error.
3. **Given** an unknown profile name is given, **When** user runs the install command, **Then** an error is returned listing available profile names.

---

### User Story 2 - Install skill locally via CLI (Priority: P2)

A developer wants a skill scoped to a specific project — not polluting their global skills. They install it into the project's local `.claude/skills/` directory.

**Why this priority**: Local scoping is essential for project-specific workflows. Requires same core logic as P1, just a different target directory.

**Independent Test**: Run `brains skill install <name>` (no flag, defaults to local) from a project root and verify `.claude/skills/<name>/SKILL.md` is created in the working directory.

**Acceptance Scenarios**:

1. **Given** the user is in a project directory, **When** they run `brains skill install create-pr` (no scope flag), **Then** `.claude/skills/create-pr/SKILL.md` is created in the current working directory.
2. **Given** `.claude/skills/` does not exist, **When** the command runs, **Then** the directory is created automatically.

---

### User Story 3 - Install skill via MCP tool (Priority: P3)

Claude (or another MCP client) installs a skill on behalf of the user during a conversation, without the user needing to open a terminal.

**Why this priority**: Enables agentic workflows where Claude can self-configure. Depends on the same core logic as P1/P2.

**Independent Test**: Call the `skill-install` MCP tool with a skill name and scope, then verify the SKILL.md file exists at the expected location.

**Acceptance Scenarios**:

1. **Given** an MCP client calls `skill-install` with `name: "create-pr"` and `scope: "global"`, **Then** `~/.claude/skills/create-pr/SKILL.md` is created with correct content.
2. **Given** an MCP client calls `skill-install` with `name: "create-pr"` and `scope: "local"` and a `working_directory`, **Then** `{working_directory}/.claude/skills/create-pr/SKILL.md` is created.
3. **Given** an unknown profile name, **When** the tool is called, **Then** the tool returns a non-nil error value with the message `"Profile '{name}' not found. Available profiles:\n  - profile1\n  - profile2"` — not a panic, not a formatted success string.

---

### Edge Cases

- What happens when the skill name contains path traversal characters (e.g., `../evil`)?
- What if the user lacks write permission to `~/.claude/skills/`?
- What if profile-compose is not available (MCP server not running) — should the installer still work since it only generates a static reference to it?
- What if a skill with the same name exists as a different type (e.g., flat `.md` file instead of directory)?

## Requirements

### Functional Requirements

- **FR-001**: System MUST create `{target_dir}/{skill-name}/SKILL.md` when installing a skill.
- **FR-002**: Generated SKILL.md MUST be exactly this template (no deviations in structure or ordering):
  ```
  ---
  name: {skill-name}
  description: >
    {description}
  allowed-tools: mcp__zombiekit__profile-compose
  ---

  Call `mcp__zombiekit__profile-compose` with `profiles: ["{skill-name}"]` and follow the returned instructions exactly.
  ```
  Where `{description}` is the `description` field from the profile's frontmatter if present, otherwise `"Delegates to the {skill-name} profile via profile-compose."`. The `allowed-tools` value is always hardcoded to `mcp__zombiekit__profile-compose` (MVP; no dynamic tool list).
- **FR-003**: Generated SKILL.md body MUST contain exactly one instruction line delegating to `mcp__zombiekit__profile-compose` with the matching profile name — no inline instructions.
- **FR-004**: System MUST support `--global` flag to target `~/.claude/skills/`. When `--global` is absent, default to local `.claude/skills/` relative to the working directory. No `--local` flag is required.
- **FR-005**: System MUST create intermediate directories if they don't exist.
- **FR-006**: System MUST return an error when the requested profile name does not exist. Error format: `"Profile '{name}' not found. Available profiles:\n  - profile1\n  - profile2\n..."`. CLI MUST exit with non-zero code. MCP tool MUST return this as the error value (not panic).
- **FR-007**: Skill name MUST match `^[a-z0-9][a-z0-9-]*[a-z0-9]$` or be a single character `^[a-z0-9]$`. Any name containing `/`, `\`, `.`, `..`, spaces, uppercase letters, underscores, or other special characters MUST be rejected before any file operation. Error: `"Invalid skill name '{name}'. Use lowercase letters, digits, and hyphens (e.g. 'my-skill')."`.
- **FR-008**: MCP tool MUST accept `name` (string, required), `scope` (`"local"` | `"global"`, required), and `working_directory` (string, optional — defaults to process CWD when scope is local) parameters.
- **FR-009**: Install MUST be idempotent — re-running overwrites SKILL.md only; other files in the skill directory are left untouched; exit code is 0 both times; file content is identical. If `{skill-name}` exists as a regular file (not a directory), return error: `"'{skill-name}' exists as a file at {path}. Remove it manually or choose a different name."`.
- **FR-010**: Local install MUST target `{cwd}/.claude/skills/{name}/SKILL.md`. No project markers (.git, .brains) are required — it works from any directory.
- **FR-011**: When profile is valid, CLI MUST print `"Installed skill '{name}' to {full-path}"` on success.

### Key Entities

- **Skill**: A named directory under a Claude skills folder containing a `SKILL.md` file. Identified by `name` (alphanumeric + hyphens).
- **Scope**: Either `global` (`~/.claude/skills/`) or `local` (`.claude/skills/` relative to working directory).
- **Profile**: An existing zombiekit profile that the installed skill delegates to. The profile name is what gets passed to `profile-compose`.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A freshly installed skill is immediately usable by Claude Code without any additional configuration.
- **SC-002**: Running `brains skill install <name>` twice produces identical output (idempotent).
- **SC-003**: An installed skill's body calls `profile-compose` — changing the profile file updates behavior without reinstalling the skill.
- **SC-004**: Invalid or malicious skill names are rejected before any file is written.
- **SC-005**: When an invalid profile name is provided, the error message includes the formatted list of available profile names so the user can correct and retry without additional commands.

## Implementation Architecture

- **CLI command**: `brains skill install <name> [--global]`
  - File: `internal/cli/skill.go` — `newSkillCommand()` returning parent `*cli.Command` with `install` subcommand
  - Register in `internal/cli/root.go` via `newSkillCommand()` in the `Commands` slice
  - Follow pattern from `internal/cli/profile.go`
- **MCP tool**: `skill-install`
  - File: `internal/mcp/tools/skillinstall/tool.go`
  - Add `"skill-install"` to `KnownTools` slice in `internal/config/tools.go`
  - Instantiate and register in `internal/mcp/server.go`
- **Core logic**: Extract into a shared `internal/skill/installer.go` package (pure function, no I/O coupling) used by both CLI and MCP tool
- **Profile service reuse**: Use `profile.NewService(workingDir).List()` (or equivalent) to enumerate available profiles for validation error messages

## Testing Requirements

### Test Strategy

Integration tests at the CLI and MCP tool boundaries. The core install logic (generate SKILL.md content, write to correct path) is testable as a unit with a temp directory. No mocking of the filesystem — use real temp dirs.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Install a known profile and assert SKILL.md exists at expected path |
| FR-002 | Integration | Assert generated file content matches exact template; profile with description uses it; profile without description uses fallback |
| FR-003 | Integration | Assert SKILL.md body is exactly the one-liner delegating to profile-compose |
| FR-004 | Integration | Install with `--global` → `~/.claude/skills/`; without → `.claude/skills/` in CWD |
| FR-005 | Integration | Install where `.claude/skills/` does not exist; assert directories created |
| FR-006 | Integration | Install unknown profile; assert exit code non-zero and error message lists available profiles by name |
| FR-007 | Unit | Valid: `my-skill`, `a`, `abc-def-123`. Invalid: `../evil`, `./bad`, `Bad`, `my_skill`, `-lead`, `trail-`, empty string |
| FR-008 | Integration | MCP tool: scope=global, scope=local with working_directory, scope=local without working_directory |
| FR-009 | Integration | Run install twice; assert exit code 0 both times; file content identical; other files in dir untouched |
| FR-010 | Integration | Install local from non-project directory; assert works without .git or .brains present |
| FR-011 | Integration | Assert stdout contains expected success message with correct path |

### Edge Case Coverage

- Path traversal in skill name (`../evil`, `./bad`) → rejected by FR-007 validation before any write
- Permission denied on target directory → error surfaced to caller with OS error message
- Profile not found → FR-006 error with list of available profile names
- Existing skill directory with non-SKILL.md files → SKILL.md overwritten, other files untouched (FR-009)
- `{skill-name}` exists as a flat file → error with manual removal instructions (FR-009)
- `profile-compose` MCP server not running → installer succeeds (static file write); skill fails at runtime when Claude invokes it — expected, not an installer concern
- Profile has no `description` field → fallback description used (FR-002)
