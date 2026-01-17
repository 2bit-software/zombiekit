# Feature Specification: MCP Tools - Code Reasoning & Sticky Memory

**Feature Branch**: `002-mcp-tools`
**Created**: 2025-12-21
**Status**: Draft
**Input**: User description: "Implement code-reasoning and sticky memory features backed by PostgreSQL with full CLI and MCP interfaces, extensive unit and database tests"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - AI Assistant Stores and Retrieves Memory Items (Priority: P1) MVP

An AI assistant (Claude Code, Cursor, etc.) uses the MCP server to persist information across sessions. The assistant saves notes, decisions, or context that should survive conversation restarts. Later, the assistant retrieves this information to maintain continuity.

**Why this priority**: Core functionality - without persistent storage, the tool provides no value. This is the foundation all other features build upon.

**Independent Test**: Can be fully tested by starting the MCP server, calling `set` to store a memory, restarting the server, and calling `get` to verify the memory persists.

**Acceptance Scenarios**:

1. **Given** the MCP server is running with database connection, **When** an AI assistant calls `stickymemory` with operation `set`, name `project-context`, and content `Working on authentication feature`, **Then** the memory is stored and a success response is returned with version number.

2. **Given** a memory named `project-context` exists, **When** an AI assistant calls `stickymemory` with operation `get` and name `project-context`, **Then** the stored content is returned along with metadata (version, timestamps).

3. **Given** multiple memories exist, **When** an AI assistant calls `stickymemory` with operation `list`, **Then** all memory names are returned with their metadata, sorted by most recently updated.

4. **Given** a memory exists, **When** an AI assistant calls `stickymemory` with operation `delete` and the memory name, **Then** the memory is soft-deleted (preserved for history) and no longer appears in list results.

---

### User Story 2 - AI Assistant Uses Structured Reasoning (Priority: P1) MVP

An AI assistant uses the code-reasoning tool to work through complex problems step-by-step. The tool tracks the reasoning chain, allows revisions to earlier thoughts, and supports branching to explore alternative approaches.

**Why this priority**: Core functionality - enables AI assistants to demonstrate transparent reasoning for complex tasks like debugging, architecture decisions, or code reviews.

**Independent Test**: Can be tested by invoking the `code-reasoning` tool with a sequence of thoughts and verifying the formatted output shows the reasoning chain.

**Acceptance Scenarios**:

1. **Given** a new reasoning session, **When** an AI assistant calls `code-reasoning` with thought 1 of 3, **Then** the thought is recorded and a formatted response shows the thought with its position in the chain.

2. **Given** thoughts 1-3 exist, **When** an AI assistant calls `code-reasoning` with `is_revision: true` and `revises_thought: 2`, **Then** thought 2 is updated and marked with a revision indicator in the chain.

3. **Given** thoughts 1-3 exist, **When** an AI assistant calls `code-reasoning` with `branch_from_thought: 2` and `branch_id: alternative-approach`, **Then** a new branch is created and tracked separately from the main chain.

4. **Given** a reasoning chain in progress, **When** the AI assistant sets `next_thought_needed: false`, **Then** the reasoning session is marked complete and final summary is returned.

---

### User Story 3 - Developer Manages Memories via CLI (Priority: P2)

A developer uses the brains CLI to view, search, and manage stored memories without needing the MCP server. This supports debugging, data inspection, and manual data management.

**Why this priority**: Secondary to MCP functionality but essential for operations, debugging, and manual data management.

**Independent Test**: Can be tested by running CLI commands to list, get, set, and search memories directly against the database.

**Acceptance Scenarios**:

1. **Given** the database is configured, **When** a developer runs `brains memory list`, **Then** all stored memories are displayed with names, sizes, and timestamps.

2. **Given** memories exist with various content, **When** a developer runs `brains memory search "authentication"`, **Then** memories containing that term (in name or content) are returned.

3. **Given** a memory named `api-notes` exists, **When** a developer runs `brains memory get api-notes`, **Then** the full content and metadata are displayed.

4. **Given** the developer wants to add a memory, **When** they run `brains memory set my-note "Content here"`, **Then** the memory is created and confirmation is shown.

5. **Given** a memory exists, **When** a developer runs `brains memory delete old-note`, **Then** the memory is soft-deleted and confirmation is shown.

---

### User Story 4 - MCP Server Starts and Handles Requests (Priority: P2)

A developer or AI system starts the MCP server which exposes the sticky-memory and code-reasoning tools via the Model Context Protocol. The server supports multiple transport modes: Streamable HTTP (default), SSE (legacy), and stdio (for CLI integration).

**Why this priority**: Required for AI assistant integration but depends on the tools being implemented first.

**Independent Test**: Can be tested by starting `brains serve` and sending MCP protocol requests via HTTP, verifying correct responses.

**Acceptance Scenarios**:

1. **Given** a valid database configuration, **When** a developer runs `brains serve`, **Then** the MCP server starts on Streamable HTTP (default port 8080) and logs that it's ready to accept connections.

2. **Given** the server is running on HTTP, **When** a developer runs `brains serve --port 3000`, **Then** the server starts on the specified port.

3. **Given** a developer needs stdio mode, **When** they run `brains serve --mode stdio`, **Then** the server communicates via stdin/stdout instead of HTTP.

4. **Given** a developer needs SSE compatibility, **When** they run `brains serve --mode sse`, **Then** the server uses legacy SSE transport for backward compatibility.

5. **Given** the server is running, **When** an MCP client sends a `tools/list` request, **Then** both `stickymemory` and `code-reasoning` tools are returned with their schemas.

6. **Given** the server is running, **When** multiple concurrent tool calls arrive, **Then** all are processed correctly without data corruption.

7. **Given** database connection fails, **When** server starts, **Then** a clear error message is shown with troubleshooting steps.

---

### User Story 5 - Developer Runs Migrations (Priority: P3)

A developer sets up or updates the database schema using migration commands. The system tracks which migrations have been applied.

**Why this priority**: Infrastructure task - needed before first use but not frequently after.

**Independent Test**: Can be tested by running migrations on a fresh database and verifying tables exist.

**Acceptance Scenarios**:

1. **Given** a fresh database, **When** a developer runs `brains db migrate`, **Then** all pending migrations are applied and success is reported.

2. **Given** migrations have been run, **When** a developer runs `brains db migrate` again, **Then** the system reports no pending migrations.

3. **Given** migrations exist, **When** a developer runs `brains db status`, **Then** a list of applied and pending migrations is shown.

---

### Edge Cases

- What happens when a memory name contains special characters? System sanitizes by replacing invalid characters with underscores.
- What happens when memory content exceeds size limits? System returns an error before attempting storage, specifying the limit.
- What happens when database is unavailable during MCP request? Tool returns appropriate error code and message without crashing.
- What happens when code-reasoning receives invalid thought numbers? System validates and returns clear error with correct format example.
- What happens when searching with empty query? System returns all items up to the limit.
- What happens when getting a deleted memory? System returns "not found" as if it never existed.
- What happens when two clients update same memory simultaneously? Versioning ensures both updates are recorded; latest version wins for reads.

## Requirements *(mandatory)*

### Functional Requirements

**Sticky Memory - Core Operations**

- **FR-001**: System MUST support storing key-value pairs where key is a string name and value is text content up to 1MB.
- **FR-002**: System MUST support retrieving stored content by exact name match.
- **FR-003**: System MUST support listing all stored memory items with metadata (name, size, version, timestamps).
- **FR-004**: System MUST support soft-deleting memories (logical delete preserving history).
- **FR-005**: System MUST support clearing all memories with confirmation of count deleted.
- **FR-006**: System MUST sanitize memory names by replacing characters outside the valid set (`a-z`, `A-Z`, `0-9`, `-`, `_`, `.`) with underscores.

**Sticky Memory - Versioning**

- **FR-007**: System MUST automatically increment version number on each update to a memory.
- **FR-008**: System MUST preserve all historical versions in the database.
- **FR-009**: System MUST return the latest non-deleted version when retrieving by name.

**Sticky Memory - Search**

- **FR-010**: System MUST support case-insensitive search across memory names and content.
- **FR-011**: System MUST support limiting search results with configurable limit (default 10).

**Code Reasoning - Core Operations**

- **FR-012**: System MUST accept sequential thoughts numbered 1 through N where N is specified upfront.
- **FR-013**: System MUST validate that thought numbers are sequential and within bounds.
- **FR-014**: System MUST track whether more thoughts are needed via explicit flag.
- **FR-015**: System MUST return formatted output showing the thought chain with numbers and markers.

**Code Reasoning - Advanced Features**

- **FR-016**: System MUST support revising a previous thought by number, marking it with revision indicator.
- **FR-017**: System MUST support branching from any thought to explore alternatives.
- **FR-018**: System MUST track branch IDs and allow multiple concurrent branches.
- **FR-019**: System MUST validate that revision and branching are not combined in same request.

**MCP Server - Transport Modes**

- **FR-020**: System MUST implement MCP protocol with Streamable HTTP transport as the default mode.
- **FR-021**: System MUST support SSE transport mode for backward compatibility with older MCP clients.
- **FR-022**: System MUST support stdio transport mode for CLI integration and local development.
- **FR-023**: System MUST allow transport mode selection via `--mode` flag (http, sse, stdio).
- **FR-024**: System MUST allow port configuration via `--port` flag for HTTP-based transports (default 8080).

**MCP Server - Protocol**

- **FR-025**: System MUST expose both tools via `tools/list` endpoint with JSON schemas.
- **FR-026**: System MUST handle `tools/call` requests and route to appropriate tool.
- **FR-027**: System MUST return proper MCP error responses for invalid requests.
- **FR-028**: System MUST support health checks reporting database connectivity status.

**Observability**

- **FR-042**: System MUST emit structured JSON logs to stdout/stderr.
- **FR-043**: System MUST support configurable log levels via `--log-level` flag (debug, info, warn, error).
- **FR-044**: System MUST log all tool invocations with request metadata (tool name, duration, success/error).

**CLI Interface**

- **FR-029**: System MUST provide `brains memory` subcommand with list, get, set, delete, search, clear operations.
- **FR-030**: System MUST provide `brains serve` command to start MCP server with transport mode options.
- **FR-031**: System MUST provide `brains db migrate` and `brains db status` commands.
- **FR-032**: All CLI commands MUST support `--format json` for machine-readable output.

**Database**

- **FR-033**: System MUST support both PostgreSQL and SQLite as storage backends.
- **FR-033a**: System MUST use PostgreSQL 16+ as the primary production backend with connection pooling.
- **FR-033b**: System MUST use SQLite as a lightweight alternative for local development and single-user deployments.
- **FR-033c**: System MUST allow backend selection via `--db-type` flag (postgres, sqlite) with sqlite as default.
- **FR-034**: System MUST create necessary tables via migrations, with separate migration files for each backend.
- **FR-035**: System MUST use connection pooling for PostgreSQL access; SQLite uses single connection with WAL mode.
- **FR-036**: System MUST handle database connection failures gracefully by failing fast with clear error messages (no retries).

**Testing**

- **FR-037**: System MUST have unit tests for all tool operations with mocked storage.
- **FR-038**: System MUST have integration tests that run against real databases (PostgreSQL via testcontainers, SQLite via temp files).
- **FR-039**: System MUST have tests for concurrent access scenarios.
- **FR-040**: System MUST have tests for error conditions and edge cases.
- **FR-041**: System MUST have tests for each transport mode (HTTP, SSE, stdio).

### Key Entities

- **Memory**: A named piece of text content with versioning. Key attributes: name (identifier), content (text up to 1MB), version (auto-incremented), deleted flag, created_at, updated_at.

- **Thought**: A single step in a reasoning chain. Key attributes: thought_number, total_thoughts, content, is_revision flag, revises_thought reference, branch_id, next_thought_needed flag.

- **Migration**: A database schema change. Key attributes: version number, name, applied_at timestamp.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: AI assistants can store and retrieve memories with 100% data integrity across server restarts.
- **SC-002**: Memory operations (set, get, list) complete in under 100 milliseconds for typical payloads.
- **SC-003**: Code reasoning supports chains of up to 20 thoughts without performance degradation.
- **SC-004**: System handles 10 concurrent MCP connections without errors or data corruption.
- **SC-005**: All CLI commands provide helpful error messages that guide users to resolution.
- **SC-006**: Unit test coverage exceeds 80% for all tool packages.
- **SC-007**: Integration tests cover all database operations and verify data persistence.
- **SC-008**: Search returns relevant results within 200 milliseconds for databases with up to 10,000 memories.

## Assumptions

- PostgreSQL 16 with pgvector extension is available for production deployments.
- SQLite is the default backend for local development and single-user scenarios (zero configuration).
- Database connection string is provided via environment variable or configuration file (PostgreSQL) or defaults to local file (SQLite).
- MCP clients follow the standard MCP protocol specification (2025-03-26 or compatible).
- Code reasoning state is session-scoped (in-memory) and does not persist across server restarts.
- Memory content is plain text; binary content is not supported in this version.
- The `brains serve` command runs in foreground; background/daemon mode is out of scope.
- Streamable HTTP is the current MCP transport standard; SSE is provided for backward compatibility.
- Both backends implement the same Repository interface for consistent behavior.

## Out of Scope

- Web UI for memory management (future feature).
- Memory encryption at rest.
- Memory sharing between different database instances.
- Automatic memory expiration/TTL.
- Vector embeddings for semantic search (uses text search only).
- WebSocket transport (Streamable HTTP covers real-time needs).
- Authentication/authorization for MCP server (local-only service trusts network/client).
- Rate limiting for MCP server (local-only, trusted clients).

## Clarifications

### Session 2025-12-21

- Q: Does the MCP server require authentication for tool access? → A: None - trust the local network/client (local-only service)
- Q: What level of observability should the MCP server provide? → A: Structured logging (JSON) with configurable log levels
- Q: Should the MCP server implement rate limiting? → A: No rate limiting (local-only, trusted clients)
- Q: What retry strategy should be used for database connection failures? → A: Fail immediately on first error (no retries)
- Q: What characters are valid in memory names? → A: Identifier-safe only (a-z, A-Z, 0-9, -, _, .)
- Q: Which database backends should be supported? → A: Both PostgreSQL (production) and SQLite (development/single-user). SQLite is the default for zero-configuration startup.
- Q: How should SQLite handle concurrent access? → A: Use WAL mode with single connection; adequate for single-user local scenarios.
