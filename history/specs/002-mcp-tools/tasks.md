# Tasks: MCP Tools - Code Reasoning & Sticky Memory

**Input**: Design documents from `/specs/002-mcp-tools/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Approach**: TDD - Write tests first, then implement to make them pass.

**Compatibility**: SQLite implementation MUST match mcp-genie patterns (see research.md section 7).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Project Structure (from plan.md)

```text
internal/
├── mo/                 # Maybe monad (mcp-genie compatible)
├── config/             # Configuration and storage config
├── database/           # Connection pool, migrations
│   └── migrations/     # SQL migration files (both SQLite and PostgreSQL)
├── memory/             # Memory domain (Storage interface + types)
│   ├── postgres/       # PostgreSQL implementation
│   └── sqlite/         # SQLite implementation
├── mcp/
│   ├── server.go       # MCP protocol handler
│   └── tools/
│       ├── stickymemory/
│       └── codereasoning/
├── logging/            # Structured logging setup
└── cli/                # CLI commands

tests/integration/      # Integration tests with testcontainers
```

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and dependencies

- [ ] T001 Add Go dependencies to go.mod: mark3labs/mcp-go, pgx/v5, modernc.org/sqlite, testify, testcontainers-go
- [ ] T002 [P] Create internal/mo/maybe.go with Maybe[T], Just[T], Nothing[T] monad (mcp-genie compatible)
- [ ] T003 [P] Create internal/config/storage.go with StorageConfig, BackendType (sqlite/postgres), env var loading
- [ ] T004 [P] Create internal/logging/logger.go with slog setup (JSON/text handlers, configurable levels)
- [ ] T005 [P] Create internal/database/postgres.go with PostgreSQL connection pool setup (pgxpool)
- [ ] T006 [P] Create internal/database/sqlite.go with SQLite connection setup (modernc.org/sqlite, WAL mode)

---

## Phase 2: Foundational (Database & Migration Infrastructure)

**Purpose**: Core database infrastructure that MUST be complete before ANY user story

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T007 [P] Create internal/database/migrations/postgres/001_stickymemory.sql with PostgreSQL schema from data-model.md
- [ ] T008 [P] Create internal/database/migrations/sqlite/001_stickymemory.sql with SQLite schema (mcp-genie compatible)
- [ ] T009 Create internal/database/migrations.go with embedded migration runner (supports both backends)
- [ ] T010 [P] Create internal/memory/types.go with MemoryItem, MemoryMetadata structs per data-model.md
- [ ] T011 Create internal/memory/storage.go with Storage interface (Set, Get, Delete, List, Clear, Close per mcp-genie)
- [ ] T012 Create internal/memory/sanitize.go with sanitizeName() function per mcp-genie pattern

**Checkpoint**: Foundation ready - database schema and interfaces defined

---

## Phase 3: User Story 1 - AI Assistant Stores and Retrieves Memory Items (Priority: P1) MVP

**Goal**: AI assistant can store, retrieve, list, and delete memories via MCP tool

**Independent Test**: Start MCP server, call `set` to store a memory, restart server, call `get` to verify persistence

### Tests for SQLite Storage (Write FIRST, ensure they FAIL)

- [ ] T013 [US1] Write SQLite storage tests in internal/memory/sqlite/storage_test.go
  - TestGet_ExistingMemory
  - TestGet_NonExistent_ReturnsNothing
  - TestGet_DeletedMemory_ReturnsNothing
  - TestSet_NewMemory_Version1
  - TestSet_UpdateExisting_VersionIncrements
  - TestDelete_SoftDeletesAllVersions
  - TestDelete_NonExistent_NoError
  - TestList_ReturnsLatestVersionPerName
  - TestList_OrderedByUpdatedAt
  - TestList_WithSearch_MatchesName
  - TestList_WithSearch_MatchesContent
  - TestList_WithSearch_CaseInsensitive
  - TestClear_SoftDeletesAll_ReturnsCount

### Implementation for SQLite Storage (Make Tests PASS)

- [ ] T014 [US1] Implement SQLite storage in internal/memory/sqlite/storage.go
  - NewSQLiteStorage(dbPath) implementing Storage interface
  - Set creates new version in transaction (mcp-genie pattern)
  - Get returns latest non-deleted version as mo.Maybe[MemoryItem]
  - Delete soft-deletes ALL versions of a name
  - List returns latest version per name with optional search
  - Clear soft-deletes all and returns count

### Tests for PostgreSQL Storage (Write FIRST, ensure they FAIL)

- [ ] T015 [P] [US1] Write PostgreSQL storage tests in internal/memory/postgres/storage_test.go
  - (Same test cases as SQLite - identical behavior required)

### Implementation for PostgreSQL Storage (Make Tests PASS)

- [ ] T016 [P] [US1] Implement PostgreSQL storage in internal/memory/postgres/storage.go
  - NewPostgresStorage(ctx, pool) implementing Storage interface
  - Same behavior as SQLite (both implement Storage interface)

### Storage Factory

- [ ] T017 [US1] Create internal/memory/factory.go with NewStorage() factory for backend selection

### Tests for MCP Tool (Write FIRST, ensure they FAIL)

- [ ] T018 [US1] Write stickymemory tool tests in internal/mcp/tools/stickymemory/tool_test.go
  - TestTool_Definition_MatchesContract
  - TestTool_Get_ValidRequest
  - TestTool_Get_MissingName_Error
  - TestTool_Get_NotFound_Error
  - TestTool_Set_ValidRequest
  - TestTool_Set_MissingName_Error
  - TestTool_Set_MissingContent_Error
  - TestTool_Set_ContentTooLarge_Error
  - TestTool_List_ValidRequest
  - TestTool_List_WithLimit
  - TestTool_Delete_ValidRequest
  - TestTool_Search_ValidRequest
  - TestTool_Search_WithLimit
  - TestTool_Clear_ValidRequest
  - TestTool_InvalidOperation_Error

### Implementation for MCP Tool (Make Tests PASS)

- [ ] T019 [US1] Create internal/mcp/tools/stickymemory/tool.go with MCP tool handler
  - NewTool(storage Storage) returning mcp.Tool
  - Name sanitization before storage operations
  - Content size validation (1MB max)
  - Response formatting per contracts/stickymemory.json

**Checkpoint**: User Story 1 core functionality complete and tested (both SQLite and PostgreSQL)

---

## Phase 4: User Story 2 - AI Assistant Uses Structured Reasoning (Priority: P1) MVP

**Goal**: AI assistant can record, revise, and branch sequential reasoning thoughts

**Independent Test**: Invoke `code-reasoning` with a sequence of thoughts and verify formatted output

### Tests for Session Logic (Write FIRST, ensure they FAIL)

- [ ] T020 [US2] Write session tests in internal/mcp/tools/codereasoning/session_test.go
  - TestSession_AddThought_Sequential
  - TestSession_AddThought_NonSequential_Error
  - TestSession_AddThought_ExceedsTotal_Error
  - TestSession_AddThought_AfterComplete_Error
  - TestSession_Revision_ValidTarget
  - TestSession_Revision_InvalidTarget_Error
  - TestSession_Branch_CreatesNewBranch
  - TestSession_Branch_AddsToExistingBranch
  - TestSession_Branch_InvalidBranchPoint_Error
  - TestSession_RevisionAndBranch_Conflict_Error
  - TestSession_Complete_SetsFlag
  - TestSession_Format_ShowsChain
  - TestSession_Format_ShowsRevisionMarker
  - TestSession_Format_ShowsBranchMarker

### Implementation for Session Logic (Make Tests PASS)

- [ ] T021 [US2] Create internal/mcp/tools/codereasoning/types.go with Thought struct
- [ ] T022 [US2] Create internal/mcp/tools/codereasoning/session.go with Session struct
  - Thread-safe operations with sync.RWMutex
  - AddThought(thought) error with validation
  - Complete() with final flag set
  - Format() string for display

### Tests for Session Manager (Write FIRST, ensure they FAIL)

- [ ] T023 [US2] Write session manager tests in internal/mcp/tools/codereasoning/manager_test.go
  - TestManager_GetOrCreate_NewSession
  - TestManager_GetOrCreate_ExistingSession
  - TestManager_Cleanup_RemovesOldSessions

### Implementation for Session Manager (Make Tests PASS)

- [ ] T024 [US2] Create internal/mcp/tools/codereasoning/manager.go with SessionManager
  - Per-connection session isolation
  - Thread-safe session map

### Tests for MCP Tool (Write FIRST, ensure they FAIL)

- [ ] T025 [US2] Write MCP tool handler tests in internal/mcp/tools/codereasoning/tool_test.go
  - TestTool_Definition_MatchesContract
  - TestTool_FirstThought_CreatesSession
  - TestTool_SequentialThoughts
  - TestTool_Revision_ValidRequest
  - TestTool_Revision_MissingTarget_Error
  - TestTool_Branch_ValidRequest
  - TestTool_Branch_MissingBranchID_Error
  - TestTool_Complete_FinalThought
  - TestTool_MissingRequiredFields_Error

### Implementation for MCP Tool (Make Tests PASS)

- [ ] T026 [US2] Create internal/mcp/tools/codereasoning/tool.go with MCP tool handler
  - NewTool(manager *SessionManager) returning mcp.Tool
  - Request validation per contracts/code-reasoning.json
  - Response formatting with chain display

**Checkpoint**: User Story 2 core functionality complete and tested

---

## Phase 5: User Story 3 - Developer Manages Memories via CLI (Priority: P2)

**Goal**: Developer can manage memories from command line without MCP server

**Independent Test**: Run CLI commands to list, get, set, and search memories directly

### Tests for CLI Commands (Write FIRST, ensure they FAIL)

- [ ] T027 [US3] Write CLI command tests in internal/cli/memory_test.go
  - TestMemoryList_Success
  - TestMemoryList_JSONFormat
  - TestMemoryGet_Success
  - TestMemoryGet_NotFound
  - TestMemorySet_Success
  - TestMemoryDelete_Success
  - TestMemorySearch_Success
  - TestMemoryClear_Success

### Implementation for CLI Commands (Make Tests PASS)

- [ ] T028 [US3] Create internal/cli/memory.go with `brains memory` subcommand
  - list (with --format json, --limit)
  - get <name> (with --format json)
  - set <name> <content>
  - delete <name>
  - search <query> (with --limit)
  - clear (with --force confirmation)

**Checkpoint**: User Story 3 complete and tested

---

## Phase 6: User Story 4 - MCP Server Starts and Handles Requests (Priority: P2)

**Goal**: MCP server starts and exposes tools via Streamable HTTP, SSE, or stdio

**Independent Test**: Start `brains serve` and send MCP protocol requests via HTTP

### Tests for MCP Server (Write FIRST, ensure they FAIL)

- [ ] T029 [US4] Write MCP server tests in internal/mcp/server_test.go
  - TestServer_ToolsList_ReturnsBothTools
  - TestServer_ToolCall_StickyMemory_Success
  - TestServer_ToolCall_CodeReasoning_Success
  - TestServer_ToolCall_InvalidTool_Error
  - TestServer_ToolCall_InvalidParams_Error

### Implementation for MCP Server (Make Tests PASS)

- [ ] T030 [US4] Create internal/mcp/server.go with MCP server setup
  - NewServer(storage Storage) returning *server.MCPServer
  - Register stickymemory tool
  - Register code-reasoning tool
  - Handle tools/list and tools/call requests

### Tests for CLI Serve Command (Write FIRST, ensure they FAIL)

- [ ] T031 [US4] Write serve command tests in internal/cli/serve_test.go
  - TestServe_DefaultMode_HTTP
  - TestServe_Mode_Stdio
  - TestServe_Mode_SSE
  - TestServe_CustomPort
  - TestServe_LogLevel
  - TestServe_DatabaseConnectionFailure_ClearError

### Implementation for CLI Serve Command (Make Tests PASS)

- [ ] T032 [US4] Create internal/cli/serve.go with `brains serve` command
  - --mode flag (http, sse, stdio) with http default
  - --port flag (default 8080)
  - --log-level flag (debug, info, warn, error)
  - --db-type flag (sqlite default, postgres option)
  - Graceful shutdown on SIGINT/SIGTERM
  - Clear error messages on database connection failure

**Checkpoint**: User Story 4 complete and tested

---

## Phase 7: User Story 5 - Developer Runs Migrations (Priority: P3)

**Goal**: Developer can manage database schema with migration commands

**Independent Test**: Run migrations on fresh database and verify tables exist

### Tests for CLI DB Commands (Write FIRST, ensure they FAIL)

- [ ] T033 [US5] Write migration command tests in internal/cli/db_test.go
  - TestDBMigrate_FreshDatabase_Postgres
  - TestDBMigrate_FreshDatabase_SQLite
  - TestDBMigrate_AlreadyApplied_NoChange
  - TestDBStatus_ShowsApplied
  - TestDBStatus_ShowsPending
  - TestDBStatus_JSONFormat

### Implementation for CLI DB Commands (Make Tests PASS)

- [ ] T034 [US5] Create internal/cli/db.go with `brains db` subcommand
  - migrate (applies pending migrations)
  - status (shows applied/pending migrations)
  - --db-type flag to select backend
  - --format json support

**Checkpoint**: User Story 5 complete and tested

---

## Phase 8: Integration & Acceptance Tests

**Purpose**: End-to-end tests validating full system behavior

### MCP Acceptance Tests

- [ ] T035 Write MCP acceptance tests in tests/integration/mcp_test.go
  - TestMCP_StickyMemory_FullWorkflow (set, get, list, delete, search, clear)
  - TestMCP_StickyMemory_PersistsAcrossRestart
  - TestMCP_StickyMemory_Versioning
  - TestMCP_CodeReasoning_FullChain
  - TestMCP_CodeReasoning_Revision
  - TestMCP_CodeReasoning_Branching
  - TestMCP_ConcurrentConnections (per FR-039)

### Database Integration Tests (PostgreSQL)

- [ ] T036 [P] Write PostgreSQL integration tests in tests/integration/postgres_memory_test.go
  - TestPostgresStorage_WithRealPostgres (testcontainers)
  - TestPostgresStorage_ConcurrentAccess
  - TestPostgresStorage_LargeContent (near 1MB)
  - TestPostgresStorage_VersionHistory

### Database Integration Tests (SQLite)

- [ ] T037 [P] Write SQLite integration tests in tests/integration/sqlite_memory_test.go
  - TestSQLiteStorage_WithTempFile
  - TestSQLiteStorage_ConcurrentAccess (WAL mode)
  - TestSQLiteStorage_LargeContent (near 1MB)
  - TestSQLiteStorage_VersionHistory

### Transport Mode Tests

- [ ] T038 [P] Write transport mode tests in tests/integration/transport_test.go
  - TestTransport_Stdio_WorksWithPipes
  - TestTransport_HTTP_WorksWithHTTPClient
  - TestTransport_SSE_WorksWithSSEClient

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final wiring and improvements

- [ ] T039 Wire all CLI commands into cmd/brains/main.go via internal/cli/root.go
- [ ] T040 [P] Add --db-type global flag for backend selection (sqlite default)
- [ ] T041 [P] Add --log-level global flag per FR-043
- [ ] T042 [P] Create .env.example with all environment variables documented
- [ ] T043 Update quickstart.md with SQLite default workflow
- [ ] T044 Verify performance: <100ms memory operations, <200ms search (SC-002, SC-008)
- [ ] T045 Verify test coverage >80% for all tool packages (SC-006)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - MVP priority
- **User Story 2 (Phase 4)**: Depends on Foundational - MVP priority, can run parallel to US1
- **User Story 3 (Phase 5)**: Depends on US1 (reuses Storage interface)
- **User Story 4 (Phase 6)**: Depends on US1 and US2 (needs both tools implemented)
- **User Story 5 (Phase 7)**: Depends on Foundational only
- **Integration (Phase 8)**: Depends on US1, US2, US4 being complete
- **Polish (Phase 9)**: Depends on all user stories being complete

### TDD Flow Within Each User Story

1. **Write Tests** - Tests should FAIL initially (no implementation yet)
2. **Implement** - Write minimum code to make tests PASS
3. **Refactor** - Clean up while keeping tests GREEN

### Parallel Opportunities

```bash
# Phase 1: Setup tasks can run in parallel
T002: Create mo/maybe.go
T003: Create config/storage.go
T004: Create logging/logger.go
T005: Create database/postgres.go
T006: Create database/sqlite.go

# Phase 2: Migrations and types can run in parallel
T007: PostgreSQL migration SQL
T008: SQLite migration SQL
T010: Create memory/types.go

# US1: PostgreSQL and SQLite implementations can run in parallel
T013-T014: SQLite storage (tests + impl)
T015-T016: PostgreSQL storage (tests + impl)

# US1 + US2 can run in parallel after Foundational (both are P1 MVP)

# Phase 8: Integration tests can run in parallel
T036: PostgreSQL integration tests
T037: SQLite integration tests
T038: Transport mode tests
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (stickymemory)
4. Complete Phase 4: User Story 2 (code-reasoning)
5. **STOP and VALIDATE**: Both core tools work via unit tests
6. Add Phase 6 (MCP server) for integration

### Incremental Delivery

1. Setup + Foundational = Database and interfaces ready
2. US1 + US2 = Core tools implemented and unit tested
3. US4 = MCP server exposes tools
4. Integration tests validate full system
5. US3 + US5 = CLI polish for developer experience

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story is independently testable
- TDD: Write tests FIRST, implement SECOND
- Commit after each test+implementation pair
- testcontainers-go provides real PostgreSQL for integration tests
- SQLite tests use temp files (no containers needed)
- **Both backends implement identical Storage interface** - implementations are swappable
- **SQLite is default backend** - zero configuration required
- **mcp-genie compatibility is MANDATORY** for SQLite implementation
- mo.Maybe monad used for optional return values (Get operation)
