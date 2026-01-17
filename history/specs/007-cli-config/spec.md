# Feature Specification: CLI Configuration System

**Feature Branch**: `007-cli-config`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Add global/local configuration files for the brains CLI. When it loads, it should prefer: command line arguments, local config, global config, in order of precedence (first takes precedence). The first thing I want to add configuration for is enabling/disabling MCP tools. This should allow for disabling an entire tool like (profiles), as well as individual sub-tools (like profiles.list)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Disable Specific MCP Tool via Local Config (Priority: P1)

A developer working on a project wants to disable certain MCP tools for that specific project. They create a local configuration file in their project directory to disable the `stickymemory` tool because they don't need persistent memory for this particular project.

**Why this priority**: This is the core use case - allowing per-project customization of which MCP tools are available. Local configuration provides project-specific overrides without affecting other projects.

**Independent Test**: Can be fully tested by creating a local config file that disables a tool, starting the MCP server, and verifying the disabled tool is not exposed.

**Acceptance Scenarios**:

1. **Given** a local config file at `.brains/config.toml` with `[tools.stickymemory]` `enabled = false`, **When** the MCP server starts, **Then** the stickymemory tool is not registered and not visible to clients.
2. **Given** a local config file that disables `profile-list` but not `profile-compose`, **When** the MCP server starts, **Then** only `profile-compose` is available and `profile-list` is not registered.
3. **Given** no configuration files exist, **When** the MCP server starts, **Then** all tools are enabled by default.

---

### User Story 2 - Global Default Configuration (Priority: P2)

A user wants to set default tool preferences across all their projects. They create a global configuration file that disables certain tools they never use, ensuring these defaults apply everywhere unless overridden.

**Why this priority**: Global defaults reduce repetitive configuration and provide a consistent baseline experience across all projects.

**Independent Test**: Can be tested by creating only a global config file, starting the server in a directory with no local config, and verifying global settings are applied.

**Acceptance Scenarios**:

1. **Given** a global config file at `~/.config/brains/config.toml` disabling `code-reasoning`, **When** the MCP server starts in a directory without local config, **Then** the code-reasoning tool is not available.
2. **Given** a global config file exists and XDG_CONFIG_HOME is set to a custom path, **When** the MCP server starts, **Then** it reads from `$XDG_CONFIG_HOME/brains/config.toml`.
3. **Given** XDG_CONFIG_HOME is not set, **When** the MCP server starts on macOS/Linux, **Then** it falls back to `~/.config/brains/config.toml`.

---

### User Story 3 - Override Global with Local Config (Priority: P3)

A developer has global defaults but needs different settings for a specific project. Their local configuration overrides the global settings, with more specific settings taking precedence.

**Why this priority**: Precedence-based configuration is essential for flexible, layered configuration systems. Users expect local settings to override global defaults.

**Independent Test**: Can be tested by creating both global and local configs with conflicting settings and verifying local takes precedence.

**Acceptance Scenarios**:

1. **Given** global config disables `stickymemory` AND local config enables `stickymemory`, **When** the MCP server starts, **Then** `stickymemory` is available (local overrides global).
2. **Given** global config disables entire `profile` category AND local config enables only `profile-list`, **When** the MCP server starts, **Then** only `profile-list` is available within the profile category.
3. **Given** global config sets `[tools.profile]` `enabled = false` AND local config sets `[tools.profile-list]` `enabled = true`, **When** the MCP server starts, **Then** `profile-list` is enabled but `profile-compose` remains disabled.

---

### User Story 4 - Command Line Override (Priority: P4)

A developer wants to temporarily enable or disable a tool for a single session without modifying any configuration files. They use command line flags to override both local and global settings.

**Why this priority**: Command line overrides are essential for debugging, testing, and one-off scenarios. They provide maximum flexibility without permanent changes.

**Independent Test**: Can be tested by running the server with CLI flags that contradict config file settings and verifying CLI takes precedence.

**Acceptance Scenarios**:

1. **Given** local config enables all tools AND the CLI flag `--disable-tool=stickymemory` is passed, **When** the MCP server starts, **Then** `stickymemory` is disabled for that session only.
2. **Given** global config disables `code-reasoning` AND the CLI flag `--enable-tool=code-reasoning` is passed, **When** the MCP server starts, **Then** `code-reasoning` is available for that session.
3. **Given** multiple `--disable-tool` flags are passed, **When** the MCP server starts, **Then** all specified tools are disabled.

---

### Edge Cases

- What happens when a config file has invalid TOML syntax? System logs a warning with the file path and line number, then uses defaults for that config level.
- What happens when a user tries to disable a tool that doesn't exist? System logs a warning but continues startup with valid settings.
- What happens when both global and local configs are missing? All tools are enabled by default.
- What happens when the config directory exists but the file doesn't? System treats this as "no config at this level" and uses defaults.
- What happens when the user disables all tools? MCP server starts with no tools registered (valid but empty tool list).
- What happens when a category is disabled but a sub-tool is explicitly enabled? The explicit sub-tool enable takes precedence (more specific wins).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST load configuration from three sources in order of precedence: command line arguments (highest), local config file, global config file (lowest).
- **FR-002**: System MUST support TOML format for configuration files.
- **FR-003**: System MUST look for local configuration at `.brains/config.toml` relative to the current working directory.
- **FR-004**: System MUST look for global configuration at `$XDG_CONFIG_HOME/brains/config.toml` if XDG_CONFIG_HOME is set.
- **FR-005**: System MUST fall back to `~/.config/brains/config.toml` for global configuration when XDG_CONFIG_HOME is not set (following XDG Base Directory Specification).
- **FR-005a**: System MUST use `%APPDATA%\brains\config.toml` for global configuration on Windows.
- **FR-006**: System MUST support enabling/disabling individual MCP tools by name (e.g., `stickymemory`, `code-reasoning`, `profile-compose`, `profile-list`).
- **FR-007**: System MUST support enabling/disabling tool categories where a single setting affects multiple related tools (e.g., `profile` affects both `profile-compose` and `profile-list`).
- **FR-008**: System MUST treat all tools as enabled by default when no configuration specifies otherwise.
- **FR-009**: System MUST support `--enable-tool=<name>` and `--disable-tool=<name>` CLI flags for temporary overrides.
- **FR-010**: System MUST allow multiple `--enable-tool` and `--disable-tool` flags in a single invocation.
- **FR-011**: System MUST log warnings for invalid configuration syntax or unknown tool names without failing startup.
- **FR-012**: System MUST apply more specific settings over less specific ones (sub-tool settings override category settings).
- **FR-013**: System MUST log loaded configuration file paths at debug level on successful startup.

### Key Entities

- **Configuration**: Represents merged settings from all three sources with precedence applied. Contains tool enable/disable states.
- **Tool Registry**: Maps tool names to their enabled/disabled state. Supports both individual tools and category-based groupings.
- **Config Source**: Represents a single configuration file (global or local) with its parsed TOML content and origin path.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can disable any MCP tool using a config file and verify it's not exposed when listing available tools.
- **SC-002**: Configuration changes take effect on server restart without requiring code changes or rebuilds.
- **SC-003**: The precedence chain (CLI > local > global > defaults) works correctly in all combinations tested.
- **SC-004**: Invalid configuration files produce clear warning messages that identify the file and error location.
- **SC-005**: Users can configure tool availability for a project in under 30 seconds by creating a simple TOML file.

## Clarifications

### Session 2025-12-22

- Q: What is the global configuration path on Windows? → A: Use `%APPDATA%\brains\config.toml` (standard Windows convention)
- Q: What should successful configuration loading log? → A: Log loaded config paths at debug level (visible only when debug enabled)

## Assumptions

- The TOML file format is acceptable for configuration (widely supported, human-readable, Go has good libraries).
- The XDG Base Directory Specification is the appropriate standard for global config location on Unix systems.
- Tool categories are determined by naming convention (hyphenated names like `profile-list` belong to the `profile` category).
- The configuration system will be extensible for future settings beyond tool enable/disable.
