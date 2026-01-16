# Feature Specification: ZombieKit Echo Tool

**Feature Branch**: `018-zombiekit-echo`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "add an echo endpoint/tool to the ZombieKit tool in the brains mcp server"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Echo String Input (Priority: P1)

A developer or AI agent uses the ZombieKit MCP server and needs a simple diagnostic tool to verify the MCP connection is working correctly. They call the echo tool with a string message and receive that message back unchanged.

**Why this priority**: The echo tool's primary purpose is as a connectivity test and debugging aid. Without basic string echo functionality, the tool has no value.

**Independent Test**: Can be fully tested by calling the echo tool with a string parameter and verifying the exact same string is returned.

**Acceptance Scenarios**:

1. **Given** the MCP server is running with the echo tool enabled, **When** a user calls the echo tool with `{"message": "Hello, World!"}`, **Then** the tool returns `"Hello, World!"` as the response.
2. **Given** the MCP server is running with the echo tool enabled, **When** a user calls the echo tool with `{"message": ""}` (empty string), **Then** the tool returns an empty string as the response.
3. **Given** the MCP server is running with the echo tool enabled, **When** a user calls the echo tool with `{"message": "Unicode: \u4e2d\u6587 \u{1F600}"}`, **Then** the tool returns the same Unicode string including Chinese characters and emojis.

---

### User Story 2 - Missing Message Parameter Handling (Priority: P2)

A developer calls the echo tool without providing the required message parameter and receives a clear error message explaining what's needed.

**Why this priority**: Proper error handling improves developer experience and helps with debugging integration issues.

**Independent Test**: Can be tested by calling the echo tool with an empty arguments object and verifying a descriptive error is returned.

**Acceptance Scenarios**:

1. **Given** the MCP server is running with the echo tool enabled, **When** a user calls the echo tool with `{}` (no message parameter), **Then** the tool returns an error indicating the message parameter is required.
2. **Given** the MCP server is running with the echo tool enabled, **When** a user calls the echo tool with `{"message": null}`, **Then** the tool returns an error indicating the message must be a non-null string.

---

### User Story 3 - Tool Configuration (Priority: P3)

An administrator can enable or disable the echo tool via the brains configuration system, following the same pattern as other tools.

**Why this priority**: Follows existing tool patterns and allows operators to control which tools are exposed.

**Independent Test**: Can be tested by modifying the configuration to disable the echo tool and verifying it no longer appears in the tool list.

**Acceptance Scenarios**:

1. **Given** the echo tool is enabled in configuration (default), **When** a client lists available tools, **Then** the echo tool appears in the list with correct name and description.
2. **Given** the echo tool is disabled in configuration, **When** a client lists available tools, **Then** the echo tool does not appear in the list.

---

### Edge Cases

- What happens when the message is extremely long (>1MB)? *Assumption: The tool will accept any length string that can fit in memory; no artificial limit imposed.*
- How does the system handle non-string types passed as message? *The MCP schema enforces string type; non-string values should be rejected by the framework before reaching the handler.*

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose an `echo` MCP tool within the ZombieKit tool namespace
- **FR-002**: The echo tool MUST accept a single required parameter named `message` of type string
- **FR-003**: The echo tool MUST return the exact input message string unchanged as the tool result
- **FR-004**: The echo tool MUST return a descriptive error when the message parameter is missing or null
- **FR-005**: The echo tool MUST be configurable via the existing tool enable/disable configuration pattern
- **FR-006**: The echo tool MUST preserve Unicode characters, including multi-byte sequences and emoji, without modification
- **FR-007**: The echo tool MUST have a clear description suitable for MCP tool discovery

### Key Entities

- **Echo Tool**: A new MCP tool that takes a string message and returns it unchanged. Implemented as part of the ZombieKit tool package.
- **Tool Definition**: MCP schema defining the tool's name (`echo`), description, and input schema with required `message` string parameter.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The echo tool successfully returns any valid UTF-8 string input unchanged 100% of the time
- **SC-002**: The echo tool appears in MCP tool listings when enabled in configuration
- **SC-003**: The echo tool is hidden from MCP tool listings when disabled in configuration
- **SC-004**: Error messages for missing/invalid parameters are clear enough that a developer can correct the issue without additional documentation
- **SC-005**: All unit tests pass for the new echo functionality
- **SC-006**: The implementation follows existing code patterns in the ZombieKit tool package (consistent style with `tool.go`)

## Assumptions

- The echo tool will be added to the existing `internal/mcp/tools/zombiekit` package rather than creating a new package
- The tool will follow the same registration pattern used by the existing `feature` tool
- The tool name for configuration purposes will be `echo` (enabling/disabling via config)
- No rate limiting or size restrictions on the echo message beyond system memory constraints
