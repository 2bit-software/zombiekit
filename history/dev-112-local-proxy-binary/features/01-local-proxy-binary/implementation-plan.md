# Implementation Plan: Local Proxy Binary

## Overview

Build `brains-mcp`, a new MCP stdio server binary that routes tool calls to local handlers or the central ZK server via Connect RPC.

## Phase 1: Foundation (entry point + config + connection)

### 1.1 Entry Point
**Files:** `cmd/brains-mcp/main.go`

- Minimal main.go following existing `cmd/brains/main.go` pattern
- Single subcommand or direct `serve` behavior (stdio-only)
- CLI flags: `--server-url`, `--tls-ca`, `--api-key`, `--log-level`, `--env-file`
- Initialize logger writing to **stderr** (stdout reserved for MCP JSON-RPC)
- Load config, create proxy, run stdio transport

### 1.2 Proxy Config
**Files:** `internal/proxy/config.go`

- `ProxyConfig` struct: ServerURL, TLSCAPath, APIKey, CallTimeout, LogLevel
- Load from env vars (ZK_SERVER_URL, ZK_TLS_CA, ZK_API_KEY) and CLI flags
- ServerURL is optional -- nil means local-only mode
- CallTimeout defaults to 10s

### 1.3 Server Connection
**Files:** `internal/proxy/connection.go`

- `Connection` struct wrapping Connect RPC clients
- Creates all service clients (ProfileService, SearchService) from single httpClient + baseURL
- `HealthCheck() (bool, error)` -- GET /healthz
- Cached health state with 5s TTL
- `IsConfigured() bool` -- returns false when ServerURL is empty

**Dependencies:** Phase 1.2

## Phase 2: Router + MCP Server Shell

### 2.1 Router
**Files:** `internal/proxy/router.go`

- `Router` struct with static routing table
- Three handler types: `LocalHandler`, `RemoteHandler`, `HybridHandler`
- All implement: `func(ctx, map[string]any) (string, error)`
- Routing table built at startup, not evaluated per-request
- Registration: `router.Register(toolName, handler)` -- panics on duplicate

### 2.2 MCP Server
**Files:** `internal/proxy/proxy.go`

- `Proxy` struct wrapping `mcp-go` MCPServer + Router + Connection
- `NewProxy(cfg *ProxyConfig) (*Proxy, error)`
- Registers all tool definitions with mcp-go (same schemas as monolithic server)
- Each tool's mcp-go handler extracts args, delegates to router
- `ServeStdio() error` -- starts stdio transport

**Dependencies:** Phase 2.1, Phase 1.3

## Phase 3: Local Handlers

### 3.1 Reuse Existing Tools
**Files:** `internal/proxy/handlers/local/`

All local handlers wrap existing tool implementations directly:

| File | Tool | Reuses |
|------|------|--------|
| `codereasoning.go` | code-reasoning | `codereasoning.NewTool(sessionManager)` |
| `workflow.go` | workflow-compose | `workflowtool.NewTool()` |
| `initiative.go` | initiative | `initiativetool.NewTool()` |
| `profilesave.go` | profile-save | `profiletool.NewTool().HandleSave()` |

Each file is a thin adapter: `func(ctx, args) (string, error)` calling the existing tool's method.

### 3.2 New Tool: brains-connection-status
**Files:** `internal/proxy/handlers/local/status.go`

- Input: none
- Calls `connection.HealthCheck()` (uses cached result if <5s old)
- Returns JSON: `{connected, server_url, last_check, error}`

**Dependencies:** Phase 1.3

## Phase 4: Remote Handlers

### 4.1 Recall Handlers
**Files:** `internal/proxy/handlers/remote/recall.go`

Two handlers translating MCP args to Connect RPC:

**recall-list-conversations:**
```
args {page, limit, project}
-> SearchService.ListConversations({
     pagination: {page_size: limit},
     project_filter: project
   })
-> JSON response matching current format
```

**recall-read-conversation:**
```
args {conversation_id, page, limit}
-> SearchService.GetConversation({
     conversation_id: conversation_id,
     pagination: {page_size: limit}
   })
-> JSON response matching current format
```

Both return "server not configured" if connection is nil.
Both return "server unreachable: {error}" on RPC failure.

**Dependencies:** Phase 1.3, Phase 2.1

## Phase 5: Hybrid Handlers

### 5.1 Profile Compose (Hybrid)
**Files:** `internal/proxy/handlers/hybrid/profilecompose.go`

1. Read local profiles from filesystem (reuse `profile.Resolver`)
2. If server configured: call `ProfileService.ListProfiles({})` to get remote profiles
3. Build merged map: remote as base, local overrides by name
4. Resolve requested profile names from merged map
5. Compose content (respecting dependencies from frontmatter)
6. If server unreachable: fall back to local-only with warning

### 5.2 Profile List (Hybrid)
**Files:** `internal/proxy/handlers/hybrid/profilelist.go`

1. Read local profiles from filesystem
2. If server configured: call `ProfileService.ListProfiles({})` for remote profiles
3. Merge: local overrides remote by name
4. Annotate each profile with source: "local" or "remote"
5. If server unreachable: fall back to local-only with warning

**Dependencies:** Phase 1.3, Phase 2.1, existing `internal/profile/` package

## Phase 6: Integration & Testing

### 6.1 Wire Everything
**Files:** Update `cmd/brains-mcp/main.go`, `internal/proxy/proxy.go`

- Create all handlers with dependencies
- Register all tools with router
- Verify tool schemas match monolithic server exactly

### 6.2 Tests
**Files:** `internal/proxy/proxy_test.go`, `internal/proxy/handlers/remote/recall_test.go`, `internal/proxy/handlers/hybrid/profile_test.go`

- **Unit tests:** Router dispatches to correct handler
- **Integration tests (testcontainers):** Remote handlers communicate with real ZK server
- **Hybrid tests:** Profile merge with mock server responses
- **Local-only mode:** Server-dependent tools return clear errors when unconfigured

### 6.3 Build & Install
**Files:** Update `Taskfile.dev.yml`

- Add `build:mcp` target for `cmd/brains-mcp/`
- Add to `install` task

## Implementation Order

```
Phase 1: Foundation
  1.1 Entry Point
  1.2 Config
  1.3 Connection

Phase 2: Router + Server Shell
  2.1 Router
  2.2 MCP Server (registers tools, delegates to router)

Phase 3: Local Handlers (parallel)
  3.1 Reuse existing tools (4 handlers)
  3.2 brains-connection-status

Phase 4: Remote Handlers
  4.1 Recall handlers (2 handlers)

Phase 5: Hybrid Handlers
  5.1 Profile compose
  5.2 Profile list

Phase 6: Integration
  6.1 Wire up
  6.2 Tests
  6.3 Build
```

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| MCP tool schema drift between proxy and monolithic | Extract shared tool definitions to a common package |
| Profile merge logic complexity | Reuse existing `profile.Resolver` for local resolution |
| Connect client TLS setup | Follow pattern from server integration tests |
| Stdio logging corruption | DI logger on stderr, never use singleton logger |
