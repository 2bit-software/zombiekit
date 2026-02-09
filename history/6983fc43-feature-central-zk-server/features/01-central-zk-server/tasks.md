# Tasks: Central ZK Server

## Overview

**Total tasks**: 20
**Parallel opportunities**: 4 groups
**Complexity**: Complex (20 files)
**Critical path**: T001 → T002 → T003 → T004/T005/T006/T007 → T008 → T009-T014 (parallel) → T015 → T016/T017 → T018 → T019 → T020

## Task Groups

### Group 1: Foundation (Sequential)

- [ ] T001 Create entry point `cmd/zk-server/main.go` with serve subcommand
- [ ] T002 Create server config struct and loader `internal/zkserver/config.go`
- [ ] T003 Create HTTP/Connect server setup `internal/zkserver/server.go`

### Group 2: Interceptors (Parallel after T002)

- [ ] T004 [P] Create auth interceptor `internal/zkserver/interceptors/auth.go`
- [ ] T005 [P] Create logging interceptor `internal/zkserver/interceptors/logging.go`
- [ ] T006 [P] Create rate limit interceptor `internal/zkserver/interceptors/ratelimit.go`
- [ ] T007 [P] Create recovery interceptor `internal/zkserver/interceptors/recovery.go`

### Group 3: Health Check (After T003)

- [ ] T008 Implement health check handler `internal/zkserver/health.go`

### Group 4: Storage & Handlers (Parallel after T003)

- [ ] T009 [P] Create profile storage `internal/zkserver/storage/profiles.go`
- [ ] T010 [P] Create initiative storage `internal/zkserver/storage/initiatives.go`
- [ ] T011 [P] Create artifact storage `internal/zkserver/storage/artifacts.go`
- [ ] T012 [P] Create profile handler `internal/zkserver/handlers/profile.go`
- [ ] T013 [P] Create workflow handler `internal/zkserver/handlers/workflow.go`
- [ ] T014 [P] Create artifact handler `internal/zkserver/handlers/artifact.go`

### Group 5: Search Handler (After T003, reuses existing storage)

- [ ] T015 Create search handler `internal/zkserver/handlers/search.go` (reuses `internal/memory/`)

### Group 6: LLM Proxy (Parallel after T002)

- [ ] T016 [P] Create LLM provider interface and Anthropic impl `internal/zkserver/llm/provider.go` + `anthropic.go`
- [ ] T017 [P] Create Ollama provider `internal/zkserver/llm/ollama.go`

### Group 7: Remaining Handlers & Wiring

- [ ] T018 Create LLM handler `internal/zkserver/handlers/llm.go` (after T016/T017)
- [ ] T019 Create config handler `internal/zkserver/handlers/config.go`

### Group 8: Integration

- [ ] T020 Wire all handlers, update main.go, integration tests

---

## Task Details

### T001: Create Entry Point

**File**: `cmd/zk-server/main.go`

**Requirements**:
- Follow pattern from `cmd/brains/main.go`
- Create root command with `serve` subcommand
- Wire version info
- Initialize embedded filesystems in `init()`

**Acceptance**:
- `go build ./cmd/zk-server` succeeds
- `./zk-server --help` shows usage
- `./zk-server serve --help` shows serve flags

---

### T002: Create Server Config

**File**: `internal/zkserver/config.go`

**Requirements**:
- Define `ServerConfig` struct per technical spec
- Load from YAML file (optional) and environment variables
- Validate required fields (TLS cert/key, API key)
- Reuse `config.LoadStorageConfigFromEnv()` for database settings

**Environment Variables**:
| Variable | Purpose | Required |
|----------|---------|----------|
| `ZK_SERVER_API_KEY` | Auth key | Yes |
| `ZK_SERVER_TLS_CERT` | Cert path | Yes |
| `ZK_SERVER_TLS_KEY` | Key path | Yes |
| `ZK_LLM_PROVIDER` | Provider type | No (default: anthropic) |
| `ZK_SHUTDOWN_TIMEOUT` | Shutdown timeout | No (default: 30s) |

**Acceptance**:
- Config loads from env vars
- Missing required fields return clear error
- Defaults applied for optional fields

---

### T003: Create HTTP/Connect Server

**File**: `internal/zkserver/server.go`

**Requirements**:
- Create `Server` struct wrapping `http.Server`
- Configure TLS from config (fail fast if invalid)
- Implement `Start(ctx)` and `Shutdown(ctx)` methods
- Create `http.ServeMux` for mounting handlers

**Acceptance**:
- Server starts with valid TLS config
- Server rejects invalid TLS paths at startup
- Graceful shutdown completes within timeout

---

### T004: Auth Interceptor

**File**: `internal/zkserver/interceptors/auth.go`

**Requirements**:
- Implement `connect.Interceptor` interface
- Check `x-api-key` header on requests
- Use constant-time comparison for key validation
- Skip auth for health check endpoints (`/grpc.health.v1.Health/`)
- Return `connect.CodeUnauthenticated` for missing/invalid key

**Acceptance**:
- Missing key returns Unauthenticated error
- Invalid key returns Unauthenticated error
- Valid key proceeds to handler
- Health check bypasses auth

---

### T005: Logging Interceptor

**File**: `internal/zkserver/interceptors/logging.go`

**Requirements**:
- Implement `connect.Interceptor` interface
- Log request method, duration, status code
- Add request correlation ID to context
- Use structured logging (slog)

**Acceptance**:
- Requests are logged with method and duration
- Correlation ID present in logs

---

### T006: Rate Limit Interceptor

**File**: `internal/zkserver/interceptors/ratelimit.go`

**Requirements**:
- Implement `connect.Interceptor` interface
- Use `golang.org/x/time/rate` for token bucket
- Default: 100 requests/minute, burst of 20
- Return `connect.CodeResourceExhausted` with `Retry-After` header

**Acceptance**:
- Requests within limit proceed
- Excess requests return ResourceExhausted
- Retry-After header present on rejection

---

### T007: Recovery Interceptor

**File**: `internal/zkserver/interceptors/recovery.go`

**Requirements**:
- Implement `connect.Interceptor` interface
- Recover from panics in handlers
- Log panic with stack trace
- Return `connect.CodeInternal` error

**Acceptance**:
- Panic in handler doesn't crash server
- Panic is logged with stack trace
- Client receives Internal error

---

### T008: Health Check Handler

**File**: `internal/zkserver/health.go`

**Requirements**:
- Implement gRPC health check protocol (`grpc.health.v1`)
- Register on Connect mux without auth interceptor
- Set `NOT_SERVING` before shutdown

**Acceptance**:
- `grpc.health.v1.Health/Check` returns SERVING status
- No authentication required
- Status changes to NOT_SERVING during shutdown

---

### T009: Profile Storage

**File**: `internal/zkserver/storage/profiles.go`

**Requirements**:
- Define `ProfileStore` interface
- Implement PostgreSQL backend for profile CRUD
- Create migration for `profiles` table (see technical spec schema)

**Interface**:
```go
type ProfileStore interface {
    Get(ctx context.Context, name string) (*profilev1.Profile, error)
    List(ctx context.Context) ([]*profilev1.Profile, error)
    Save(ctx context.Context, name, content, location string) error
}
```

**Acceptance**:
- Profile save/get round-trips correctly
- List returns all profiles
- Missing profile returns appropriate error

---

### T010: Initiative Storage

**File**: `internal/zkserver/storage/initiatives.go`

**Requirements**:
- Define `InitiativeStore` interface
- Implement PostgreSQL backend for initiative CRUD
- Create migrations for `initiatives` and `initiative_steps` tables

**Interface**:
```go
type InitiativeStore interface {
    Create(ctx context.Context, name, initiativeType, projectPath string) (*workflowv1.Initiative, error)
    Get(ctx context.Context, id string) (*workflowv1.Initiative, error)
    UpdateStep(ctx context.Context, initiativeID, stepName, status string) error
    Complete(ctx context.Context, id string) error
    List(ctx context.Context, projectPath string, limit int32, cursor string) ([]*workflowv1.Initiative, string, error)
}
```

**Acceptance**:
- Create/Get/Complete lifecycle works
- Step updates persist
- List with pagination works

---

### T011: Artifact Storage

**File**: `internal/zkserver/storage/artifacts.go`

**Requirements**:
- Define `ArtifactStore` interface
- Implement PostgreSQL backend for artifact key-value storage
- Create migration for `artifacts` table

**Interface**:
```go
type ArtifactStore interface {
    Get(ctx context.Context, initiativeID, key string) (*artifactv1.Artifact, error)
    Save(ctx context.Context, initiativeID, key, content string) error
    List(ctx context.Context, initiativeID string) ([]*artifactv1.Artifact, error)
}
```

**Acceptance**:
- Artifact save/get round-trips correctly
- List returns all artifacts for initiative
- Foreign key constraint to initiatives enforced

---

### T012: Profile Handler

**File**: `internal/zkserver/handlers/profile.go`

**Requirements**:
- Implement `profilev1connect.ProfileServiceHandler`
- `ComposeProfile`: compose profiles from database
- `ListProfiles`: list available profiles
- `GetProfile`: retrieve single profile
- `SaveProfile`: store profile (validate YAML syntax)
- `SubscribeProfileUpdates`: return `Unimplemented`

**Acceptance**:
- All RPCs return appropriate responses
- Invalid YAML rejected with InvalidArgument
- SubscribeProfileUpdates returns Unimplemented

---

### T013: Workflow Handler

**File**: `internal/zkserver/handlers/workflow.go`

**Requirements**:
- Implement `workflowv1connect.WorkflowServiceHandler`
- `CreateInitiative`: create new initiative record
- `GetStatus`: get initiative with step status
- `UpdateStep`: update step status
- `CompleteInitiative`: mark initiative complete
- `ListInitiatives`: paginated initiative list

**Acceptance**:
- Full initiative lifecycle works
- Step status updates correctly
- Pagination works correctly

---

### T014: Artifact Handler

**File**: `internal/zkserver/handlers/artifact.go`

**Requirements**:
- Implement `artifactv1connect.ArtifactServiceHandler`
- `GetArtifact`: retrieve artifact by key
- `SaveArtifact`: store artifact
- `ListArtifacts`: list artifacts for initiative

**Acceptance**:
- Artifact CRUD works
- Non-existent artifacts return NotFound
- List scoped to initiative

---

### T015: Search Handler

**File**: `internal/zkserver/handlers/search.go`

**Requirements**:
- Implement `searchv1connect.SearchServiceHandler`
- `Search`: vector similarity search (reuse `internal/memory/` or `internal/recall/`)
- `GetConversation`: retrieve conversation by ID
- `ListConversations`: paginated conversation list

**Acceptance**:
- Search returns relevant results
- Project filter works
- Pagination works

---

### T016: LLM Provider Interface + Anthropic

**Files**: `internal/zkserver/llm/provider.go`, `internal/zkserver/llm/anthropic.go`

**Requirements**:
- Define `Provider` interface with `Complete` and `CompleteStream`
- Implement Anthropic provider using SDK
- Support context cancellation for client disconnect

**Interface**:
```go
type Provider interface {
    Complete(ctx context.Context, req *llmv1.CompleteRequest) (*llmv1.CompleteResponse, error)
    CompleteStream(ctx context.Context, req *llmv1.CompleteStreamRequest) (<-chan *llmv1.CompleteStreamResponse, <-chan error)
}
```

**Acceptance**:
- Non-streaming requests complete
- Streaming requests deliver tokens incrementally
- Context cancellation stops upstream request

---

### T017: Ollama Provider

**File**: `internal/zkserver/llm/ollama.go`

**Requirements**:
- Implement Ollama provider using HTTP client
- Support both streaming and non-streaming
- Use configured Ollama URL

**Acceptance**:
- Requests to Ollama succeed
- Streaming works correctly
- Context cancellation works

---

### T018: LLM Handler

**File**: `internal/zkserver/handlers/llm.go`

**Requirements**:
- Implement `llmv1connect.LLMServiceHandler`
- `Complete`: forward to provider, return full response
- `CompleteStream`: forward to provider, stream tokens back
- Cancel upstream request on client disconnect

**Acceptance**:
- Non-streaming requests work
- Streaming requests work
- Client disconnect cancels upstream

---

### T019: Config Handler

**File**: `internal/zkserver/handlers/config.go`

**Requirements**:
- Implement `configv1connect.ConfigServiceHandler`
- `GetConfig`: return current runtime config
- `UpdateConfig`: return `Unimplemented`
- `SubscribeConfigUpdates`: return `Unimplemented`

**Acceptance**:
- GetConfig returns runtime config
- UpdateConfig returns Unimplemented
- SubscribeConfigUpdates returns Unimplemented

---

### T020: Integration & Tests

**Files**:
- `cmd/zk-server/main.go` (update to wire handlers)
- `internal/zkserver/server.go` (update to mount handlers)
- `tests/integration/zkserver_test.go`

**Requirements**:
- Wire all handlers with dependencies
- Mount handlers on Connect mux with interceptor chain
- Integration tests for:
  - Server startup with valid config
  - Auth rejection without API key
  - Profile round-trip
  - Search functionality
  - LLM proxy (mock provider)
  - Graceful shutdown

**Acceptance**:
- All integration tests pass
- Server starts and handles requests
- Graceful shutdown works

---

## Traceability Matrix

| Task | Spec Requirement | Plan Phase |
|------|------------------|------------|
| T001 | Entry point | 1.1 |
| T002 | Server config | 1.2 |
| T003 | HTTP/Connect server | 1.3 |
| T004-T007 | Auth, logging, rate limit, recovery | 1.4 |
| T008 | Health check | 1.5 |
| T009, T012 | Profile storage + handler | 2.1 |
| T010, T013 | Initiative storage + handler | 2.3 |
| T011, T014 | Artifact storage + handler | 2.4 |
| T015 | Search handler | 2.2 |
| T016-T018 | LLM proxy | 3.1-3.2 |
| T019 | Config handler | 4.1 |
| T020 | Integration | 5.1-5.2 |

## Execution Order

```
Sequential:
  T001 → T002 → T003 → T008

Parallel Group A (after T002):
  T004, T005, T006, T007

Parallel Group B (after T003):
  T009, T010, T011, T012, T013, T014, T015

Parallel Group C (after T002):
  T016, T017

Sequential (after Group B + C):
  T018 → T019 → T020
```

## Suggested Implementation Command

After approval, run:

```
/brains.next implement
```
