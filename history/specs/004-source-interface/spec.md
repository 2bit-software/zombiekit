# Feature Specification: Profile Source Abstraction

**Feature Branch**: `004-source-interface`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Add --source argument to profiles mode supporting 'brains' and 'claude' sources with interface abstraction for read/write operations"

## Clarifications

### Session 2025-12-22

- Q: Should the claude source traverse parent directories like brains does, or only check local and global? → A: Local + global only (no parent traversal).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use Brains Profiles (Default Behavior) (Priority: P1)

A developer working on a project wants to use the existing brains profile system without changes. They run profile commands without specifying a source and expect the current behavior to continue working.

**Why this priority**: This preserves backward compatibility for all existing users and workflows. The brains source is the default and most common use case.

**Independent Test**: Can be fully tested by running `brains profile list` and `brains profile compose research` without any `--source` flag and verifying behavior matches the current implementation.

**Acceptance Scenarios**:

1. **Given** a user has profiles in `.brains/profiles/` and `~/.brains/profiles/`, **When** they run `brains profile list` (no --source), **Then** profiles are listed from brains directories as before.
2. **Given** a user runs `brains profile compose research,database`, **When** no --source is specified, **Then** composition works from brains directories with current precedence rules.
3. **Given** a user runs `brains profile create my-profile`, **When** no --source is specified, **Then** the profile is created in `.brains/profiles/` as before.

---

### User Story 2 - List Claude Agents as Profiles (Priority: P1)

A developer wants to see what Claude agents are available in their project. They use the `--source claude` flag to list agents from Claude's `.claude/agents/` directory.

**Why this priority**: Listing is the foundational read operation that enables discoverability of Claude agents. Users need to know what's available before they can work with it.

**Independent Test**: Can be tested by creating sample agents in `.claude/agents/` and `~/.claude/agents/`, running `brains profile list --source claude`, and verifying all agents are shown with correct metadata.

**Acceptance Scenarios**:

1. **Given** agents exist in `.claude/agents/`, **When** user runs `brains profile list --source claude`, **Then** all agents are listed with their name, description, and source location.
2. **Given** agents exist in both local `.claude/agents/` and global `~/.claude/agents/`, **When** listing with `--source claude`, **Then** both are shown with local taking precedence when names conflict.
3. **Given** `--format json` is used, **When** listing Claude agents, **Then** output includes agent-specific fields (model, color) in structured format.

---

### User Story 3 - Show Claude Agent Details (Priority: P1)

A developer wants to inspect the content of a specific Claude agent to understand what it does before using or editing it.

**Why this priority**: Understanding agent content is essential for users to effectively work with and customize Claude agents.

**Independent Test**: Can be tested by creating an agent with known content, running `brains profile show agent-name --source claude`, and verifying the full content is displayed correctly.

**Acceptance Scenarios**:

1. **Given** an agent named `systems-architect` exists in `.claude/agents/`, **When** user runs `brains profile show systems-architect --source claude`, **Then** the agent's full content is displayed including frontmatter and body.
2. **Given** the `--raw` flag is used, **When** showing a Claude agent, **Then** raw file content is displayed without any processing.
3. **Given** an agent doesn't exist, **When** showing it, **Then** an error is displayed with suggestions for similar agent names.

---

### User Story 4 - Compose Claude Agents (Priority: P2)

A developer wants to compose multiple Claude agents together, similar to how brains profiles can be composed.

**Why this priority**: Composition enables combining agent capabilities, though this is less common for Claude agents since they tend to be more self-contained.

**Independent Test**: Can be tested by creating multiple agents, running `brains profile compose agent1,agent2 --source claude`, and verifying the combined output.

**Acceptance Scenarios**:

1. **Given** multiple Claude agents exist, **When** user runs `brains profile compose architect,reviewer --source claude`, **Then** the agents' content is concatenated in order.
2. **Given** an agent includes other agents via `includes` field, **When** composing, **Then** included agents are resolved and prepended.
3. **Given** the same agent would be included multiple times, **When** composing, **Then** it appears only once at its first occurrence.

---

### User Story 5 - Create New Claude Agent (Priority: P1)

A developer wants to create a new Claude agent with proper structure and frontmatter template.

**Why this priority**: Creating agents is a core write operation that enables users to build their own Claude agents with correct format.

**Independent Test**: Can be tested by running `brains profile create my-agent --source claude` and verifying the file is created at `.claude/agents/my-agent.md` with valid Claude agent frontmatter.

**Acceptance Scenarios**:

1. **Given** user runs `brains profile create reviewer --source claude`, **When** the command completes, **Then** a new file exists at `.claude/agents/reviewer.md` with valid Claude agent frontmatter (name, description, model, color).
2. **Given** an agent with that name already exists, **When** creating, **Then** the command fails with an error rather than overwriting.
3. **Given** `--global` flag is used, **When** creating a Claude agent, **Then** the agent is created in `~/.claude/agents/` instead.
4. **Given** `.claude/agents/` directory doesn't exist, **When** creating, **Then** an appropriate error suggests running an init command.

---

### User Story 6 - Validate Claude Agents (Priority: P2)

A developer wants to check their Claude agent configurations for errors like invalid frontmatter or broken includes.

**Why this priority**: Validation catches configuration errors early, though Claude agents have simpler structure so fewer validation concerns.

**Independent Test**: Can be tested by creating agents with intentional errors (invalid YAML, missing fields) and verifying validate detects them.

**Acceptance Scenarios**:

1. **Given** agents with circular includes exist, **When** user runs `brains profile validate --source claude`, **Then** the circular dependency is reported.
2. **Given** an agent has invalid YAML frontmatter, **When** validating, **Then** the parse error is reported with the file path.
3. **Given** all agents are valid, **When** validating, **Then** success is reported.

---

### User Story 7 - Initialize Claude Agents Directory (Priority: P2)

A developer wants to set up the Claude agents directory structure in their project or globally.

**Why this priority**: Users need a way to bootstrap the Claude agents directory before creating agents.

**Independent Test**: Can be tested by running init in a new directory and verifying `.claude/agents/` is created.

**Acceptance Scenarios**:

1. **Given** no `.claude/agents/` directory exists, **When** user runs `brains init --source claude`, **Then** a `.claude/agents/` directory is created.
2. **Given** `--global` flag is used, **When** running init with --source claude, **Then** `~/.claude/agents/` is created.
3. **Given** `.claude/agents/` already exists, **When** running init, **Then** the command succeeds without modifying existing content.

---

### Edge Cases

- What happens when Claude agent frontmatter is missing required fields (name, description)? System uses defaults: name from filename, empty description.
- How are Claude-specific fields (model, color) handled during composition? These are metadata-only and do not appear in composed output.
- What happens when switching sources mid-session (e.g., composing brains profile that includes a Claude agent)? Each source is independent; cross-source includes are not supported.
- How does the system behave when global Claude directory doesn't exist but local does? System proceeds with local agents only.
- What happens when a source is specified but no agents/profiles exist for that source? System returns an empty list for list operations, or an error for operations requiring specific items.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a `--source` argument on all profile subcommands (`list`, `show`, `compose`, `create`, `validate`).
- **FR-002**: System MUST support two source types: `brains` (default) and `claude`.
- **FR-003**: System MUST default to `brains` source when `--source` is not specified, preserving backward compatibility.
- **FR-004**: System MUST abstract profile source operations behind an interface to enable future source types.
- **FR-005**: The brains source MUST continue using `.brains/profiles/` directories with existing resolution rules (local > parent > global).
- **FR-006**: The claude source MUST read profiles from `.claude/agents/` directories using two-level resolution: local (CWD) and global (`~/.claude/agents/`) only, with no parent directory traversal.
- **FR-007**: The claude source MUST parse Claude agent frontmatter format (name, description, model, color).
- **FR-008**: The claude source MUST write new agents with proper Claude frontmatter structure.
- **FR-009**: Both sources MUST support the same operations: list, show, compose, create, validate.
- **FR-010**: The source interface MUST support both read and write operations.
- **FR-011**: System MUST NOT read or process Claude skills (only agents).
- **FR-012**: System MUST provide consistent error types across sources (ProfileNotFoundError, NotInitializedError, etc.).
- **FR-013**: System MUST support `--format json` output for all read operations regardless of source.
- **FR-014**: The claude source MUST default `inherits` to `false` (Claude agents are typically standalone), while brains defaults to `true`.

### Key Entities

- **ProfileSource Interface**: Abstraction for reading and writing profiles from different backends. Methods include: FindProfileDirs(), LoadProfiles(), LoadAllProfiles(), CreateProfile(), etc.
- **BrainsSource**: Implementation of ProfileSource for `.brains/profiles/` directories. Current behavior, refactored behind the interface.
- **ClaudeSource**: Implementation of ProfileSource for `.claude/agents/` directories. Handles Claude-specific frontmatter (model, color fields).
- **SourceType**: Enum-like type representing available sources ("brains", "claude").
- **AgentFrontmatter**: Claude-specific frontmatter structure extending base ProfileFrontmatter with model and color fields.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can run all existing profile commands without `--source` and experience identical behavior to before (100% backward compatible).
- **SC-002**: Users can list, show, compose, create, and validate Claude agents using `--source claude` with same UX as brains profiles.
- **SC-003**: Profile operations complete within same performance bounds regardless of source (under 1 second for typical operations).
- **SC-004**: New sources can be added by implementing the ProfileSource interface without modifying existing source implementations.
- **SC-005**: JSON output includes source-specific fields (model, color for claude) without breaking existing JSON consumers.
- **SC-006**: Error messages clearly indicate which source produced the error.

## Assumptions

- Claude agents use `.claude/agents/` as their standard directory (matching Claude Code conventions).
- Claude agent files use `.md` extension with YAML frontmatter.
- The `model` and `color` fields in Claude agent frontmatter are optional metadata.
- Cross-source includes (brains profile including a Claude agent) are not supported in this iteration.
- The `inherits` behavior for Claude agents defaults to false, matching how Claude agents are typically standalone.
- Global Claude agents directory is `~/.claude/agents/`.
- Skills (located in `.claude/skills/` or similar) are explicitly out of scope.
- Claude source uses simple two-level resolution (local + global) without parent directory traversal, matching Claude Code's actual behavior.
