# Tasks: Local Proxy Binary

## Overview

**Total tasks**: 16
**Parallel opportunities**: 3 groups
**Complexity**: Medium (15 files)
**Critical path**: T001 → T002 → T003 → T004 → T005-T009 (parallel) → T010-T011 (parallel) → T012-T013 (parallel) → T014 → T015 → T016

## Task Groups

### Group 1: Foundation (Sequential)

- [ ] T001 Create entry point `cmd/brains-mcp/main.go`
- [ ] T002 Create proxy config `internal/proxy/config.go`
- [ ] T003 Create server connection `internal/proxy/connection.go`
- [ ] T004 Create router and proxy shell `internal/proxy/router.go` + `internal/proxy/proxy.go`

### Group 2: Local Handlers (Parallel after T004)

- [ ] T005 [P] Create code-reasoning handler `internal/proxy/handlers/local/codereasoning.go`
- [ ] T006 [P] Create workflow-compose handler `internal/proxy/handlers/local/workflow.go`
- [ ] T007 [P] Create initiative handler `internal/proxy/handlers/local/initiative.go`
- [ ] T008 [P] Create profile-save handler `internal/proxy/handlers/local/profilesave.go`
- [ ] T009 [P] Create brains-connection-status handler `internal/proxy/handlers/local/status.go`

### Group 3: Remote Handlers (Parallel after T003)

- [ ] T010 [P] Create recall-list-conversations handler `internal/proxy/handlers/remote/recall.go`
- [ ] T011 [P] Create recall-read-conversation handler (same file as T010)

### Group 4: Hybrid Handlers (Parallel after T003)

- [ ] T012 [P] Create profile-compose hybrid handler `internal/proxy/handlers/hybrid/profilecompose.go`
- [ ] T013 [P] Create profile-list hybrid handler `internal/proxy/handlers/hybrid/profilelist.go`

### Group 5: Integration & Testing (Sequential)

- [ ] T014 Wire all handlers into proxy, register tool schemas `internal/proxy/proxy.go`
- [ ] T015 Write integration tests `internal/proxy/proxy_test.go` + handler tests
- [ ] T016 Add build target to Taskfile `Taskfile.dev.yml`

---

## Task Details

### T001: Create Entry Point

**File**: `cmd/brains-mcp/main.go`
**Plan Phase**: 1.1

**Requirements**:
- Follow pattern from `cmd/brains/main.go` and `cmd/zk-server/main.go`
- CLI flags: `--server-url`, `--tls-ca`, `--api-key`, `--log-level`, `--env-file`
- Initialize slog logger writing to **stderr** (stdout is MCP JSON-RPC)
- Load env file if `--env-file` provided
- Create ProxyConfig, create Proxy, call ServeStdio

**Acceptance**:
- `go build ./cmd/brains-mcp` succeeds
- `./brains-mcp --help` shows usage and flags

---

### T002: Create Proxy Config

**File**: `internal/proxy/config.go`
**Plan Phase**: 1.2

**Requirements**:
- `ProxyConfig` struct: ServerURL, TLSCAPath, APIKey, CallTimeout, LogLevel
- `Validate()` method: CallTimeout defaults to 10s if zero
- ServerURL is optional (empty = local-only mode)
- No config file loading (env vars + CLI flags only for now)

**Acceptance**:
- Config with empty ServerURL is valid (local-only mode)
- Config with ServerURL validates correctly

---

### T003: Create Server Connection

**File**: `internal/proxy/connection.go`
**Plan Phase**: 1.3

**Requirements**:
- `Connection` struct wrapping Connect RPC clients
- `NewConnection(cfg *ProxyConfig) (*Connection, error)` -- creates httpClient with TLS config and timeout
- Exposes typed clients: `Profiles() profilev1connect.ProfileServiceClient`, `Search() searchv1connect.SearchServiceClient`
- `HealthCheck(ctx) (bool, error)` -- GET /healthz with 5s cache
- `IsConfigured() bool` -- returns false when ServerURL empty
- Thread-safe health check cache (sync.Mutex)

**Acceptance**:
- `IsConfigured()` returns false when no ServerURL
- `HealthCheck()` returns cached result within 5s TTL
- Clients are created with correct TLS and timeout settings

---

### T004: Create Router and Proxy Shell

**Files**: `internal/proxy/router.go`, `internal/proxy/proxy.go`, `internal/proxy/handlers/handler.go`
**Plan Phase**: 2.1, 2.2

**Requirements**:
- `Handler` type: `func(ctx context.Context, args map[string]any) (string, error)`
- `Router` struct with `Register(toolName, handler)` and `Dispatch(ctx, toolName, args) (string, error)`
- `Proxy` struct wrapping mcp-go MCPServer + Router + Connection + logger
- `NewProxy(cfg *ProxyConfig) (*Proxy, error)` -- creates connection (if configured), router, mcp server
- Single generic `handleTool` method that extracts args and delegates to router.Dispatch
- `ServeStdio() error` -- starts stdio transport
- Register all 9 tool definitions with same schemas as monolithic server (tool schemas only, handlers wired in T014)

**Acceptance**:
- Router panics on duplicate registration
- Router returns error for unknown tool name
- Proxy creates mcp-go server with all tool schemas registered

---

### T005: Code-Reasoning Local Handler

**File**: `internal/proxy/handlers/local/codereasoning.go`
**Plan Phase**: 3.1

**Requirements**:
- Create `NewCodeReasoningHandler() Handler`
- Instantiate `codereasoning.NewSessionManager()` and `codereasoning.NewTool(sessionManager)`
- Delegate to `tool.Execute(ctx, "default", args)`

**Acceptance**:
- Returns valid JSON response for a thought request
- Session state persists across calls within same process

---

### T006: Workflow-Compose Local Handler

**File**: `internal/proxy/handlers/local/workflow.go`
**Plan Phase**: 3.1

**Requirements**:
- Create `NewWorkflowHandler() Handler`
- Instantiate `workflowtool.NewTool()`
- Delegate to `tool.HandleCompose(ctx, args)`

**Acceptance**:
- Returns composed workflow content for valid workflow name

---

### T007: Initiative Local Handler

**File**: `internal/proxy/handlers/local/initiative.go`
**Plan Phase**: 3.1

**Requirements**:
- Create `NewInitiativeHandler() Handler`
- Instantiate `initiativetool.NewTool()`
- Delegate to `tool.Execute(ctx, args)`

**Acceptance**:
- Create/status/complete/list actions work correctly

---

### T008: Profile-Save Local Handler

**File**: `internal/proxy/handlers/local/profilesave.go`
**Plan Phase**: 3.1

**Requirements**:
- Create `NewProfileSaveHandler() Handler`
- Instantiate `profiletool.NewTool()`
- Delegate to `tool.HandleSave(ctx, args)`

**Acceptance**:
- Saves profile to local filesystem at specified location

---

### T009: Connection Status Handler

**File**: `internal/proxy/handlers/local/status.go`
**Plan Phase**: 3.2

**Requirements**:
- Create `NewConnectionStatusHandler(conn *Connection) Handler`
- If connection nil/not configured: return `{connected: false, server_url: "", error: "server not configured"}`
- Otherwise: call `conn.HealthCheck(ctx)`, return JSON result
- Include `last_check` timestamp in response

**Acceptance**:
- Returns `connected: false` when no server configured
- Returns health check result when server configured
- Response is valid JSON matching spec format

---

### T010: Recall List Conversations Remote Handler

**File**: `internal/proxy/handlers/remote/recall.go`
**Plan Phase**: 4.1

**Requirements**:
- Create `NewRecallListHandler(conn *Connection) Handler`
- Guard: if `!conn.IsConfigured()` return error "server not configured"
- Extract args: page (default 1), limit (default 20, max 100), project (default "")
- Call `conn.Search().ListConversations(ctx, request)`
- Format response JSON matching monolithic server output format

**Acceptance**:
- Returns "server not configured" when no connection
- Returns formatted conversation list from server
- Pagination args are correctly mapped

---

### T011: Recall Read Conversation Remote Handler

**File**: `internal/proxy/handlers/remote/recall.go` (same file as T010)
**Plan Phase**: 4.1

**Requirements**:
- Create `NewRecallReadHandler(conn *Connection) Handler`
- Guard: if `!conn.IsConfigured()` return error "server not configured"
- Extract args: conversation_id (required), page (default 1), limit (default 20, max 100)
- Call `conn.Search().GetConversation(ctx, request)`
- Format response JSON matching monolithic server output format

**Acceptance**:
- Returns "server not configured" when no connection
- Returns error for missing conversation_id
- Returns formatted conversation chunks from server

---

### T012: Profile Compose Hybrid Handler

**File**: `internal/proxy/handlers/hybrid/profilecompose.go`
**Plan Phase**: 5.1

**Requirements**:
- Create `NewProfileComposeHandler(conn *Connection) Handler`
- Read local profiles using existing `profile.Resolver` or `profiletool.NewTool()`
- If server configured: call `conn.Profiles().ListProfiles(ctx, request)` for remote profiles
- Merge: remote profiles as base, local profiles override by name
- Compose requested profile names from merged set (respecting dependencies)
- If server unreachable: fall back to local-only, prepend warning to result
- If server not configured: use local-only silently (same as current behavior)

**Acceptance**:
- Local-only mode returns same result as monolithic server
- With server: remote profiles are included in composition
- Local profiles override remote profiles with same name
- Server failure gracefully degrades to local-only with warning

---

### T013: Profile List Hybrid Handler

**File**: `internal/proxy/handlers/hybrid/profilelist.go`
**Plan Phase**: 5.2

**Requirements**:
- Create `NewProfileListHandler(conn *Connection) Handler`
- Read local profiles using existing profile resolution
- If server configured: call `conn.Profiles().ListProfiles(ctx, request)` for remote profiles
- Merge: local overrides remote by name
- Annotate each profile with source: "local" or "remote"
- If server unreachable: fall back to local-only with warning

**Acceptance**:
- Local-only mode returns same result as monolithic server
- With server: both local and remote profiles listed
- Source annotation present in output

---

### T014: Wire All Handlers

**File**: `internal/proxy/proxy.go` (update)
**Plan Phase**: 6.1

**Requirements**:
- In `NewProxy()`: create all handlers with appropriate dependencies
- Register all handlers with router:
  - 5 local handlers (code-reasoning, workflow-compose, initiative, profile-save, brains-connection-status)
  - 2 remote handlers (recall-list-conversations, recall-read-conversation)
  - 2 hybrid handlers (profile-compose, profile-list)
- Verify all 9 tool schemas registered with mcp-go match monolithic server schemas

**Acceptance**:
- `go build ./cmd/brains-mcp` succeeds
- All 9 tools registered and routable
- No stickymemory tool registered

---

### T015: Integration Tests

**Files**: `internal/proxy/proxy_test.go`, `internal/proxy/handlers/remote/recall_test.go`, `internal/proxy/handlers/hybrid/profile_test.go`
**Plan Phase**: 6.2

**Requirements**:
- **Router test**: Dispatch routes to correct handler, unknown tool returns error
- **Local handler tests**: Each local handler returns valid JSON for happy path
- **Remote handler tests** (testcontainers): recall handlers communicate with real ZK server
- **Hybrid handler tests**: Profile merge logic with mock server responses
- **Local-only mode test**: Server-dependent tools return clear error when unconfigured
- **Connection status test**: Returns correct state for configured/unconfigured

**Acceptance**:
- All tests pass
- Remote tests use testcontainers (skip in short mode)
- No stickymemory tests

---

### T016: Build Target

**File**: `Taskfile.dev.yml`
**Plan Phase**: 6.3

**Requirements**:
- Add `build:mcp` task: `go build -o bin/brains-mcp ./cmd/brains-mcp/`
- Add source tracking for incremental builds
- Update `install` task in root Taskfile.yml to include brains-mcp

**Acceptance**:
- `task dev -- build:mcp` produces `bin/brains-mcp`
- Binary starts and shows help

---

## Traceability Matrix

| Task | Spec Acceptance Criteria | Plan Phase |
|------|------------------------|------------|
| T001 | AC1 (stdio binary) | 1.1 |
| T002 | AC6 (error when unconfigured) | 1.2 |
| T003 | AC4, AC6 (server connection) | 1.3 |
| T004 | AC2 (identical tool schemas) | 2.1, 2.2 |
| T005-T008 | AC3 (local tools work offline) | 3.1 |
| T009 | AC7 (connection status) | 3.2 |
| T010-T011 | AC4 (server-dependent route) | 4.1 |
| T012-T013 | AC5 (hybrid profile merge) | 5.1, 5.2 |
| T014 | AC2, AC8 (all tools, no stickymemory) | 6.1 |
| T015 | AC9 (integration tests) | 6.2 |
| T016 | AC1 (buildable binary) | 6.3 |

## Execution Order

```
Sequential:
  T001 → T002 → T003 → T004

Parallel Group A (after T004):
  T005, T006, T007, T008, T009

Parallel Group B (after T003):
  T010, T011

Parallel Group C (after T003):
  T012, T013

Sequential (after all groups):
  T014 → T015 → T016
```
