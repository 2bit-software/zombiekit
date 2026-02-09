# Technical Requirements: Central ZK Server

These are implementation preferences extracted from user input and ticket context. They inform HOW to build, not WHAT to build.

## Entry Point Structure

- Create `/cmd/zk-server/main.go` as unique server entry point
- Follow existing minimal main pattern from `/cmd/brains/main.go`
- Initialize embedded filesystems in `init()` for profiles/workflows

## Code Organization

- All shared logic in `./internal/` for reuse in client (built later)
- Server-specific code can live in `internal/zkserver/` or similar
- Service handlers implement generated interfaces from `/gen/`

## Technology Stack

- **Protocol**: ConnectRPC (already in use, not raw gRPC)
- **Database**: PostgreSQL with pgvector (existing setup)
- **TLS**: Server-side TLS at minimum, configurable paths
- **Config**: YAML file similar to existing `startup.go` pattern

## Configuration File Structure (Proposed)

```yaml
server:
  listen_address: ":50051"
  tls:
    cert_file: "/path/to/server.crt"
    key_file: "/path/to/server.key"
    # Optional: ca_file for mTLS

database:
  # Reuse existing BRAINS_POSTGRES_URL pattern
  url: "postgres://..."
  max_conns: 10
  min_conns: 2

llm:
  provider: "anthropic"  # or "ollama", "openai"
  api_key_env: "ANTHROPIC_API_KEY"
  # Provider-specific settings
  ollama_url: "http://localhost:11434"

rate_limiting:
  enabled: true
  requests_per_second: 100
  burst: 50
```

## Reusable Components

From existing codebase:
- `internal/config/storage.go` - Storage configuration loading
- `internal/database/postgres.go` - PostgreSQL pool management
- `internal/shutdown/manager.go` - Graceful shutdown coordination
- `internal/logging/` - Logger initialization

## New Components Needed

- `internal/zkserver/config.go` - Server-specific configuration
- `internal/zkserver/server.go` - gRPC/Connect server setup
- `internal/zkserver/handlers/` - Service handler implementations
- `cmd/zk-server/main.go` - Entry point

## Service Handlers to Implement

From generated proto code:
1. `ProfileServiceHandler` - profile composition, listing, caching
2. `WorkflowServiceHandler` - workflow execution
3. `ConfigServiceHandler` - runtime config push
4. `SearchServiceHandler` - RAG search over conversations
5. `LLMServiceHandler` - LLM proxy with streaming
6. `ArtifactServiceHandler` - initiative/artifact metadata

## Not In Scope (per DEV-111)

- Web UI (DEV-113)
- Profile streaming/caching logic (DEV-112)
- Auth beyond basic API key (DEV-114)
