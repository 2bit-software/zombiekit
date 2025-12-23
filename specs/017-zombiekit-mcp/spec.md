# Feature Specification: ZombieKit MCP Tool

**Feature Branch**: `017-zombiekit-mcp`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "Add a new tool called 'ZombieKit' with a single tool 'feature' that returns ~/.brains/templates/step.feature.md as a test bed for further tests."

## Clarifications

### Session 2025-12-23

- Q: Should ZombieKit be a separate MCP server or integrated into existing brains MCP server? → A: Add as new tool to existing brains MCP server (extends current tool registry)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Invoke Feature Tool via MCP (Priority: P1)

As an AI assistant using MCP tools, I want to invoke the "feature" tool from the brains MCP server so that I can retrieve the contents of a step feature template for further operations.

**Why this priority**: This is the core and only functionality of the initial ZombieKit MCP tool. Without this capability, the tool serves no purpose.

**Independent Test**: Can be fully tested by invoking the MCP tool and verifying it returns the expected file contents. Delivers the foundation for all future ZombieKit functionality.

**Acceptance Scenarios**:

1. **Given** the brains MCP server is running, **When** a client calls the "feature" tool with no parameters, **Then** the tool returns the contents of `~/.brains/templates/step.feature.md`
2. **Given** the brains MCP server is running, **When** the "feature" tool is called but the template file does not exist, **Then** the tool returns a clear error message indicating the file was not found
3. **Given** the brains MCP server is running, **When** a user lists available tools, **Then** the "feature" tool appears with appropriate description

---

### Edge Cases

- What happens when `~/.brains/templates/step.feature.md` does not exist?
- What happens when the file exists but is not readable (permission denied)?
- What happens when the file is empty?
- How does the system handle very large template files?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST register ZombieKit tools within the existing brains MCP server tool registry
- **FR-002**: System MUST provide a "feature" tool accessible via the MCP protocol
- **FR-003**: The "feature" tool MUST return the contents of `~/.brains/templates/step.feature.md` when invoked
- **FR-004**: The "feature" tool MUST expand the `~` to the user's home directory path
- **FR-005**: System MUST return a descriptive error when the template file cannot be read
- **FR-006**: The "feature" tool MUST be listed in MCP tool discovery responses with a clear description

### Key Entities

- **ZombieKit Tool Group**: A logical grouping of tools registered within the existing brains MCP server
- **Feature Tool**: An MCP tool that reads and returns template file contents
- **Template File**: The markdown file at `~/.brains/templates/step.feature.md` containing step feature template content

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can invoke the "feature" tool and receive file contents within 1 second
- **SC-002**: Tool correctly expands home directory path on all supported platforms (macOS, Linux)
- **SC-003**: Error messages provide actionable information (file path attempted, reason for failure)
- **SC-004**: Tool integrates with existing brains MCP infrastructure without requiring separate configuration

## Assumptions

- The user has the brains CLI installed and configured
- The `~/.brains/templates/` directory structure may or may not exist; errors should be handled gracefully
- This tool will eventually support additional parameters and operations, but for now returns a fixed file path
- ZombieKit tools are registered within the existing brains MCP server tool registry (no separate process)
