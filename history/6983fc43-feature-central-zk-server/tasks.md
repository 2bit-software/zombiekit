# Tasks: Central ZK Server

**Initiative**: DEV-111 - Central ZK Server: Core Infrastructure
**Branch**: `feat/central-zk-server`

## Overview

Build the central ZK server that hosts stateful services: database, LLM proxy, and gRPC endpoints. This implements the server-side counterpart to the local proxy.

## Acceptance Criteria (from DEV-111)

- [ ] Server starts, accepts gRPC connections with TLS
- [ ] Existing RAG search works through gRPC instead of direct DB access
- [ ] LLM proxy completes requests and streams responses
- [ ] Initiative CRUD operations work through gRPC
- [ ] Existing conversation importer targets the server's database

---

## Phase 1: Server Skeleton

### T001 - Create cmd/zk-server entry point
**Status**: complete
**Dependencies**: none

Create the main entry point for the central server:
- `cmd/zk-server/main.go` with CLI flags for config
- Config loading (listen address, TLS cert paths, database DSN)
- Signal handling and graceful shutdown

**Verification**: `go build ./cmd/zk-server` compiles ✓

### T002 - Implement gRPC/Connect server with TLS
**Status**: complete
**Dependencies**: T001

Set up the Connect-RPC server:
- TLS termination with configurable certs
- Health check endpoint (simple HTTP `/healthz`)
- All 6 service stubs registered (return unimplemented initially)
- Graceful shutdown with connection draining

**Verification**: Server starts, `curl -k https://localhost:8443/healthz` returns 200 ✓

### T003 - Add logging interceptor
**Status**: complete
**Dependencies**: T002

Implement Connect interceptor for structured logging:
- Log request method, duration, status
- Request ID propagation
- Use existing `internal/logging` package

**Verification**: Requests show in logs with timing ✓

---

## Phase 2: Database Layer

### T004 - Wire existing PostgreSQL storage
**Status**: complete
**Dependencies**: T002

Connect existing database infrastructure:
- Use `internal/database/postgres.go` for connection pool
- Run migrations on startup (optional, configurable)
- Expose pool to service handlers via dependency injection

**Verification**: Server starts with valid DSN, migrations run ✓

### T005 - Implement Profile storage
**Status**: complete
**Dependencies**: T004

Add database storage for profiles (server-authoritative copies):
- Create migration: `profiles` table (name, content, domains, dependencies, location, timestamps)
- Storage interface in `internal/server/storage/profile.go`
- PostgreSQL implementation

**Verification**: Profile CRUD via direct DB calls works ✓

---

## Phase 3: Service Handlers (MVP)

### T006 - Implement ProfileService handlers
**Status**: complete
**Dependencies**: T005

Implement the 4 MVP RPCs from `profile.proto`:
- `ComposeProfile` - compose multiple profiles into merged content ✓
- `ListProfiles` - return all profiles ✓
- `GetProfile` - return single profile by name ✓
- `SaveProfile` - create/update profile ✓

**Verification**: `curl` calls return valid responses ✓

### T007 - Implement WorkflowService handlers
**Status**: complete
**Dependencies**: T004

Implement all 5 RPCs from `workflow.proto`:
- `CreateInitiative` - create new initiative ✓
- `GetStatus` - get initiative status (by ID or active) ✓
- `UpdateStep` - update step status ✓
- `CompleteInitiative` - mark initiative complete ✓
- `ListInitiatives` - list with pagination and filters ✓

**Verification**: Create/Get/List initiatives work via gRPC ✓

### T008 - Implement SearchService handlers
**Status**: complete
**Dependencies**: T004

Implement all 3 RPCs from `search.proto`:
- `Search` - vector search with pgvector ✓
- `GetConversation` - get single conversation with chunks ✓
- `ListConversations` - list with pagination ✓

Wire to existing `internal/recall` storage.

**Verification**: Search returns results from recall data ✓

### T009 - Implement ConfigService handlers (MVP)
**Status**: complete
**Dependencies**: T004

Implement 2 MVP RPCs from `config.proto`:
- `GetConfig` - get config entries (all or by keys) ✓
- `UpdateConfig` - update config entries ✓

**Verification**: Get/Update config works via gRPC ✓

### T010 - Implement ArtifactService handlers
**Status**: complete
**Dependencies**: T004

Implement all 3 RPCs from `artifact.proto`:
- `GetArtifact` - get artifact by initiative + path ✓
- `SaveArtifact` - save artifact content ✓
- `ListArtifacts` - list artifacts for initiative ✓

**Verification**: Save/Get/List artifacts works via gRPC ✓

---

## Phase 4: LLM Proxy

### T011 - Implement LLMService handlers
**Status**: deferred
**Dependencies**: T002

Implement both RPCs from `llm.proto`:
- `Complete` - unary request/response
- `CompleteStream` - server streaming response

Proto note: "Contract defined, implementation deferred to followup ticket"

**Note**: LLMService stub returns CodeUnimplemented. Full implementation deferred per proto comment.

---

## Phase 5: Integration Tests

### T012 - Server integration tests
**Status**: pending (future work)
**Dependencies**: T006, T007, T008, T009, T010

Create integration test suite:
- Use testcontainers for PostgreSQL
- Test each service's happy path
- Test error cases (not found, invalid input)

**Note**: Manual testing completed. Formal integration test suite can be added later.

---

## Deferred (Out of Scope)

These are explicitly NOT included in DEV-111:

- **Web UI** (DEV-113)
- **Profile streaming/caching** (DEV-112)
- **Auth beyond basic API key** (DEV-114)
- **SubscribeProfileUpdates** streaming RPC
- **SubscribeConfigUpdates** streaming RPC

---

## Task Dependencies

```
T001 (entry point)
  └── T002 (server + TLS)
        ├── T003 (logging)
        └── T004 (database)
              ├── T005 (profile storage)
              │     └── T006 (ProfileService)
              ├── T007 (WorkflowService)
              ├── T008 (SearchService)
              ├── T009 (ConfigService)
              └── T010 (ArtifactService)
        └── T011 (LLMService)
              └── T012 (integration tests)
```
