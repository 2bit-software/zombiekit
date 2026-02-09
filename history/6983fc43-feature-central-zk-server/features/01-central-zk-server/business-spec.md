# Business Specification: Central ZK Server

## Overview

A central server that hosts stateful services for the ZombieKit client-server architecture. The server handles database operations, LLM proxying, and gRPC endpoints that local proxy clients connect to.

## Actors

- **Local Proxy Client**: MCP binary running locally in stdio mode, proxies requests to this server
- **Server Operator**: Deploys and configures the server
- **LLM Provider**: External service (Anthropic, Ollama, OpenAI) that processes LLM requests

## Core Capabilities

### 1. gRPC Server Infrastructure

**What it does:**
- Accepts encrypted gRPC/Connect connections from local proxy clients
- Terminates TLS at the server
- Routes requests to appropriate service handlers
- Reports health status for orchestration/monitoring

**Observable behaviors:**
- Server starts and listens on configured address (default: `:50051`)
- Connections without valid TLS are rejected
- Health endpoint returns current serving status
- Server shuts down gracefully when signaled (completes in-flight requests)

### 2. Authentication

**What it does:**
- Validates API key on each request
- Rejects unauthorized requests with clear error

**Configuration:**
- API key stored in environment variable `ZK_SERVER_API_KEY`
- Single shared key per deployment (multi-key deferred to DEV-114)

**Client transmission:**
- API key sent in gRPC metadata with key `x-api-key`
- Example: `x-api-key: sk_live_abc123...`

**Observable behaviors:**
- Missing API key returns `Unauthenticated` error
- Invalid API key returns `Unauthenticated` error
- Valid API key allows request to proceed
- Health check endpoint does NOT require authentication

### 3. TLS Configuration

**What it does:**
- Encrypts all traffic between client and server
- Validates server identity to clients

**Configuration:**
- Certificate file path: `ZK_SERVER_TLS_CERT` env var or config file
- Key file path: `ZK_SERVER_TLS_KEY` env var or config file
- Certificate format: PEM
- mTLS: Not required (server TLS only)

**Observable behaviors:**
- Server fails to start if cert/key files missing or invalid
- Clients must trust server certificate (or CA that signed it)
- Self-signed certificates acceptable for development

### 4. Database Operations

**What it does:**
- Stores and retrieves conversations (for RAG search)
- Stores and retrieves embedding vectors (pgvector)
- Manages profile storage (database-backed, not filesystem)
- Manages initiative and artifact metadata

**Profile storage clarification:**
- Profiles stored in database, not filesystem
- `working_directory` parameter in proto is client-side only
- Server ignores `working_directory`; uses database storage

**Observable behaviors:**
- Search queries return relevant conversation chunks
- Profile reads return stored profile content
- Initiative CRUD operations persist across restarts
- Database connection failures are reported clearly (fail-fast)

### 5. LLM Proxy

**What it does:**
- Receives LLM requests from local proxy clients
- Forwards requests to configured provider
- Streams responses back to the requesting client
- Applies rate limiting to prevent overload

**Provider configuration:**
- Single provider per deployment (multi-provider deferred)
- Provider type set via `ZK_LLM_PROVIDER` (anthropic, ollama, openai)
- Provider credentials via provider-specific env vars (e.g., `ANTHROPIC_API_KEY`)

**Rate limiting configuration:**
- Global rate limit (not per-client in v1)
- Default: 100 requests/minute, burst of 20
- Configurable via `ZK_LLM_RATE_LIMIT` and `ZK_LLM_RATE_BURST`

**Observable behaviors:**
- Non-streaming requests return complete responses
- Streaming requests deliver incremental tokens
- Rate-exceeded requests receive `ResourceExhausted` error with `Retry-After` header
- Provider errors are surfaced to the client with context
- Client disconnect cancels upstream provider request (no orphan requests)

### 6. Configuration Management

**What it does:**
- Reads server configuration from file and environment
- Provides runtime configuration to connected clients
- Validates configuration at startup

**Observable behaviors:**
- Invalid configuration prevents server start with clear error
- Clients can request current runtime config via `GetConfig` RPC
- `UpdateConfig` RPC returns `Unimplemented` in v1 (exists for forward compatibility)
- Config changes require server restart (no hot-reload in v1)

## Service Contracts

The server implements handlers for these pre-defined service contracts (from proto definitions):

| Service | Purpose |
|---------|---------|
| HealthService | Standard gRPC health checking (grpc.health.v1) |
| ProfileService | Compose, list, get, save profiles (database-backed) |
| WorkflowService | Initiative lifecycle (create, complete) and step management |
| ConfigService | Serve runtime configuration to clients |
| SearchService | RAG search over stored conversations |
| LLMService | Proxy LLM requests with streaming |
| ArtifactService | Store and retrieve initiative artifacts |

**Clarification:** WorkflowService handles initiative CRUD. ArtifactService handles artifact storage within initiatives.

## Acceptance Criteria

### Infrastructure
- [ ] Server starts and accepts gRPC connections with TLS on port 50051
- [ ] Health check endpoint (`grpc.health.v1.Health/Check`) reports SERVING status
- [ ] Graceful shutdown completes within 30 seconds (configurable)
- [ ] Invalid configuration prevents startup with descriptive error message

### Authentication
- [ ] Request without `x-api-key` metadata returns `Unauthenticated`
- [ ] Request with invalid API key returns `Unauthenticated`
- [ ] Request with valid API key proceeds to handler
- [ ] Health check works without authentication

### Database
- [ ] Search for "authentication" returns chunks containing auth-related content
- [ ] Search with `project_filter` returns only matching project's chunks
- [ ] Profile save/get round-trips correctly
- [ ] Initiative create/get/complete lifecycle works through WorkflowService

### LLM Proxy
- [ ] LLM proxy completes non-streaming requests (returns full response)
- [ ] LLM proxy streams responses incrementally for streaming requests
- [ ] 101st request within 1 minute receives `ResourceExhausted` error
- [ ] Client disconnect mid-stream cancels upstream LLM request

### Configuration
- [ ] `GetConfig` returns runtime configuration
- [ ] `UpdateConfig` returns `Unimplemented`

### Migration
- [ ] Existing conversation importer (`tests/integration/`) targets the server's database

## Out of Scope

These are explicitly excluded from this specification:

- **Web UI** - Separate frontend serving (DEV-113)
- **Profile streaming/caching** - Real-time profile push to clients (DEV-112)
- **Advanced auth** - Multi-key, JWT, OAuth (DEV-114)
- **Offline/degraded mode** - Server unavailable = client fails clearly
- **Multi-tenant support** - Single tenant only in v1
- **Multiple LLM providers** - Single configured provider per deployment
- **Per-client rate limiting** - Global limits only in v1
- **Hot configuration reload** - Restart required for config changes

## Error Handling

| Scenario | Expected Behavior |
|----------|-------------------|
| Database unreachable | Server fails to start with connection error |
| TLS cert missing/invalid | Server fails to start with cert error |
| LLM provider unavailable | Request fails with upstream error, server stays healthy |
| Client disconnects mid-stream | Server cleans up resources, cancels upstream, logs disconnect |
| Rate limit exceeded | Client receives `ResourceExhausted` with `Retry-After` |
| Invalid API key | Client receives `Unauthenticated` error |
| Database storage full | Write operations fail with `ResourceExhausted` |

## Non-Functional Requirements

- **Startup time**: Server ready to accept connections within 5 seconds (excluding DB migrations)
- **Graceful shutdown**: Default 30 seconds, configurable via `ZK_SHUTDOWN_TIMEOUT`
- **Connection handling**: Support 100+ concurrent client connections (configurable)
- **Logging**: Structured logging with request correlation (via interceptor)
- **Observability**: Prometheus metrics endpoint at `/metrics` (deferred to v1.1)

## Resolved Questions

| Question | Resolution |
|----------|------------|
| Profile validation | Server validates YAML/TOML syntax only, trusts content semantics |
| Rate limit strategy | Global rate limiting in v1, per-client deferred |
| Multiple LLM providers | No, single provider per deployment |
