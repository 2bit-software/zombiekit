# Implementation Plan: Central ZK Server

## Overview

Build the central ZK server with gRPC/Connect endpoints, database access, and LLM proxy capabilities. The implementation is structured in phases to allow incremental testing.

## Phase 1: Foundation (Infrastructure)

### 1.1 Create Entry Point
**Files:**
- `cmd/zk-server/main.go`

**Tasks:**
1. Create minimal main.go following existing pattern from `cmd/brains/main.go`
2. Initialize embedded filesystems in `init()`
3. Create root CLI command with `serve` subcommand
4. Wire up version info

**Dependencies:** None

### 1.2 Server Configuration
**Files:**
- `internal/zkserver/config.go`

**Tasks:**
1. Define `ServerConfig` struct with fields:
   - ListenAddress (default `:50051`)
   - TLS cert/key paths
   - Shutdown timeout
   - LLM provider settings
   - Rate limit settings
2. Load from YAML file and environment variables
3. Validate configuration at startup
4. Reuse `LoadStorageConfigFromEnv()` for database config

**Dependencies:** None

### 1.3 HTTP/Connect Server Setup
**Files:**
- `internal/zkserver/server.go`

**Tasks:**
1. Create `Server` struct wrapping `http.Server`
2. Configure TLS from config
3. Mount Connect handlers on paths
4. Implement `Start(ctx)` and `Shutdown(ctx)` methods
5. Use existing `shutdown.Manager` pattern

**Dependencies:** 1.2

### 1.4 Interceptor Chain
**Files:**
- `internal/zkserver/interceptors/auth.go`
- `internal/zkserver/interceptors/logging.go`
- `internal/zkserver/interceptors/ratelimit.go`
- `internal/zkserver/interceptors/recovery.go`

**Tasks:**
1. Create Connect interceptor for API key auth (check `x-api-key` metadata)
2. Create logging interceptor with request correlation
3. Create rate limiting interceptor using `golang.org/x/time/rate`
4. Create recovery interceptor for panic handling
5. Chain interceptors in correct order: rate limit → logging → auth → recovery

**Dependencies:** 1.2

### 1.5 Health Check
**Files:**
- `internal/zkserver/health.go`

**Tasks:**
1. Implement gRPC health check protocol (`grpc.health.v1`)
2. Register health server with Connect mux
3. Exempt from auth interceptor
4. Set NOT_SERVING before shutdown

**Dependencies:** 1.3

---

## Phase 2: Database Services

### 2.1 Profile Storage
**Files:**
- `internal/zkserver/storage/profiles.go`
- `internal/zkserver/handlers/profile.go`

**Tasks:**
1. Create `ProfileStore` interface for database operations
2. Implement PostgreSQL backend for profile CRUD
3. Implement `ProfileServiceHandler`:
   - `ComposeProfile` - compose profiles from database
   - `ListProfiles` - list available profiles
   - `GetProfile` - retrieve single profile
   - `SaveProfile` - store profile (with YAML validation)
   - `SubscribeProfileUpdates` - return `Unimplemented` (deferred)

**Dependencies:** 1.3, database connection

### 2.2 Search/RAG Storage
**Files:**
- `internal/zkserver/storage/conversations.go`
- `internal/zkserver/handlers/search.go`

**Tasks:**
1. Reuse existing conversation storage from `internal/memory/`
2. Implement `SearchServiceHandler`:
   - `Search` - vector similarity search with pgvector
   - `GetConversation` - retrieve conversation by ID
   - `ListConversations` - paginated conversation list

**Dependencies:** 1.3, database connection

### 2.3 Workflow/Initiative Storage
**Files:**
- `internal/zkserver/storage/initiatives.go`
- `internal/zkserver/handlers/workflow.go`

**Tasks:**
1. Create database tables for initiatives and steps
2. Implement `WorkflowServiceHandler`:
   - `CreateInitiative` - create new initiative record
   - `GetStatus` - get initiative with step status
   - `UpdateStep` - update step status
   - `CompleteInitiative` - mark initiative complete
   - `ListInitiatives` - paginated initiative list

**Dependencies:** 1.3, database connection

### 2.4 Artifact Storage
**Files:**
- `internal/zkserver/storage/artifacts.go`
- `internal/zkserver/handlers/artifact.go`

**Tasks:**
1. Create database table for artifacts (key-value with initiative FK)
2. Implement `ArtifactServiceHandler`:
   - `GetArtifact` - retrieve artifact by key
   - `SaveArtifact` - store artifact
   - `ListArtifacts` - list artifacts for initiative

**Dependencies:** 2.3

---

## Phase 3: LLM Proxy

### 3.1 Provider Interface
**Files:**
- `internal/zkserver/llm/provider.go`
- `internal/zkserver/llm/anthropic.go`
- `internal/zkserver/llm/ollama.go`

**Tasks:**
1. Define `Provider` interface:
   - `Complete(ctx, request) (response, error)`
   - `CompleteStream(ctx, request) (stream, error)`
2. Implement Anthropic provider using SDK
3. Implement Ollama provider using HTTP client
4. Factory function to create provider from config

**Dependencies:** 1.2

### 3.2 LLM Handler
**Files:**
- `internal/zkserver/handlers/llm.go`

**Tasks:**
1. Implement `LLMServiceHandler`:
   - `Complete` - forward to provider, return full response
   - `CompleteStream` - forward to provider, stream tokens back
2. Cancel upstream request on client disconnect
3. Rate limiting applied via interceptor

**Dependencies:** 3.1, 1.4

---

## Phase 4: Configuration Service

### 4.1 Config Handler
**Files:**
- `internal/zkserver/handlers/config.go`

**Tasks:**
1. Implement `ConfigServiceHandler`:
   - `GetConfig` - return current runtime config
   - `UpdateConfig` - return `Unimplemented`
   - `SubscribeConfigUpdates` - return `Unimplemented` (deferred)

**Dependencies:** 1.2

---

## Phase 5: Integration & Testing

### 5.1 Wire Everything Together
**Files:**
- `cmd/zk-server/main.go` (update)
- `internal/zkserver/server.go` (update)

**Tasks:**
1. Create all handlers with dependencies
2. Mount all service handlers on Connect mux
3. Start server with signal handling

**Dependencies:** All previous phases

### 5.2 Integration Tests
**Files:**
- `tests/integration/zkserver_test.go`

**Tasks:**
1. Test server startup with valid config
2. Test auth rejection without API key
3. Test profile round-trip
4. Test search functionality
5. Test LLM proxy (mock provider)
6. Test graceful shutdown

**Dependencies:** 5.1

---

## Implementation Order

```
Phase 1: Foundation
  1.1 Entry Point ─┐
  1.2 Config ──────┼─→ 1.3 Server ─→ 1.4 Interceptors ─→ 1.5 Health
                   │
Phase 2: Database  │
  2.1 Profiles ────┤
  2.2 Search ──────┼─→ (can be parallel)
  2.3 Workflow ────┤
       └─→ 2.4 Artifacts

Phase 3: LLM
  3.1 Providers ─→ 3.2 Handler

Phase 4: Config
  4.1 Config Handler

Phase 5: Integration
  5.1 Wire Up ─→ 5.2 Tests
```

## Estimated Complexity

| Phase | Complexity | Notes |
|-------|-----------|-------|
| 1.1-1.2 | Low | Follows existing patterns |
| 1.3 | Medium | TLS + Connect setup |
| 1.4 | Medium | Interceptor chain |
| 1.5 | Low | Standard health check |
| 2.1-2.4 | Medium | Database schema + handlers |
| 3.1-3.2 | High | LLM streaming integration |
| 4.1 | Low | Simple config read |
| 5.1-5.2 | Medium | Integration testing |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Connect interceptors differ from gRPC | Research showed Connect uses `connect.WithInterceptors()` - validated in research |
| LLM streaming complexity | Use context cancellation for cleanup, test thoroughly |
| Database schema changes | Run migrations separately from server startup |
| TLS configuration errors | Fail fast with clear error messages |
