# Feature Specification: Remove profile-show and profile-validate MCP Tools

**Feature Branch**: `006-remove-mcp-tools`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "remove the profile-show, profile-validate from the mcp interface"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Streamlined MCP Tool Surface (Priority: P1)

As a developer using the MCP interface, I want a simpler tool surface that only exposes the essential profile operations (compose and list), so that the interface is less cluttered and easier to understand.

**Why this priority**: This is the core request - removing unnecessary tools to simplify the MCP interface. The profile-show and profile-validate functionality can still be accessed via the CLI commands, making the MCP tools redundant.

**Independent Test**: Can be fully tested by starting the MCP server and verifying that only profile-compose and profile-list tools are advertised, and that requests to profile-show or profile-validate return appropriate errors or are simply not available.

**Acceptance Scenarios**:

1. **Given** an MCP server is running, **When** a client lists available tools, **Then** profile-show and profile-validate are NOT in the list
2. **Given** an MCP server is running, **When** a client attempts to call profile-show, **Then** the request fails with "unknown tool" or equivalent error
3. **Given** an MCP server is running, **When** a client attempts to call profile-validate, **Then** the request fails with "unknown tool" or equivalent error

---

### User Story 2 - Retained Essential Tools (Priority: P1)

As a developer using the MCP interface, I want profile-compose and profile-list to remain available, so that I can still discover and use profiles programmatically.

**Why this priority**: Equal priority to story 1 because these tools must remain functional after the removal.

**Independent Test**: Can be fully tested by starting the MCP server and successfully calling profile-compose and profile-list tools.

**Acceptance Scenarios**:

1. **Given** an MCP server is running, **When** a client calls profile-list, **Then** the available profiles are returned
2. **Given** an MCP server is running, **When** a client calls profile-compose with valid profile names, **Then** the composed content is returned

---

### Edge Cases

- What happens when existing clients have cached the old tool list?
  - They will receive "unknown tool" errors when trying to use removed tools; this is expected behavior
- What happens to the underlying Go handler functions (HandleShow, HandleValidate)?
  - They remain in the profile tool package for potential CLI usage but are no longer exposed via MCP

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST NOT expose profile-show as an MCP tool
- **FR-002**: System MUST NOT expose profile-validate as an MCP tool
- **FR-003**: System MUST continue to expose profile-compose as an MCP tool with full functionality
- **FR-004**: System MUST continue to expose profile-list as an MCP tool with full functionality
- **FR-005**: System MUST return the standard MCP "unknown tool" error when removed tools are called

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: MCP server advertises exactly 2 profile tools (profile-compose, profile-list) instead of 4
- **SC-002**: All existing tests for profile-compose and profile-list continue to pass
- **SC-003**: No runtime errors occur when starting the MCP server after removal
- **SC-004**: Code cleanup reduces the MCP server registration code by removing 2 tool definitions
