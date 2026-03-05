# Research Summary: Local Proxy Binary

## Current MCP Server Architecture

### Entry Point & Transport
- CLI: `brains serve` with modes: stdio, SSE, HTTP
- Framework: `mcp-go v0.43.2` (github.com/mark3labs/mcp-go)
- Transport configured via `--mode` flag
- Stdio mode is primary for Claude Code integration

### Existing Tools (9 total)

| Tool | Category | Storage | Filesystem |
|------|----------|---------|------------|
| stickymemory | database | memory.Storage (SQLite/PG) | none |
| code-reasoning | in-memory | SessionManager | none |
| profile-compose | filesystem | none | READ .brains/profiles/, ~/.brains/profiles/ |
| profile-list | filesystem | none | READ .brains/profiles/, ~/.brains/profiles/ |
| profile-save | filesystem | none | WRITE .brains/profiles/, ~/.brains/profiles/ |
| workflow-compose | filesystem | none | READ .brains/workflows/ |
| initiative | filesystem | none | READ/WRITE .brains/initiatives/ |
| recall-list-conversations | database | recall.Storage (PG only) | none |
| recall-read-conversation | database | recall.Storage (PG only) | none |

### Tool Routing Classification

**Local (filesystem):** profile-compose, profile-list, profile-save, workflow-compose, initiative
**Server (database):** stickymemory, recall-list-conversations, recall-read-conversation
**Local (in-memory):** code-reasoning

### Configuration Hierarchy
1. CLI flags (highest)
2. Environment variables
3. Local config (.brains/config.toml)
4. Global config (~/.config/brains/config.toml)
5. Defaults (lowest)

## Central ZK Server (DEV-111)

### Available gRPC Services (via Connect RPC)

| Service | Methods | Status |
|---------|---------|--------|
| ProfileService | ComposeProfile, ListProfiles, GetProfile, SaveProfile, SubscribeProfileUpdates | Implemented (streaming deferred) |
| WorkflowService | CreateInitiative, GetStatus, UpdateStep, CompleteInitiative, ListInitiatives | Implemented |
| ArtifactService | GetArtifact, SaveArtifact, ListArtifacts | Implemented |
| ConfigService | GetConfig, UpdateConfig, SubscribeConfigUpdates | Implemented (streaming deferred) |
| SearchService | Search, GetConversation, ListConversations | Implemented |
| LLMService | Complete, CompleteStream | Deferred (returns Unimplemented) |

### Connect Client Pattern
```go
client := profilev1connect.NewProfileServiceClient(http.DefaultClient, baseURL)
resp, err := client.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{...}))
```

### Server Config
- Default listen: `:8443`
- TLS optional (cert/key paths)
- Env vars: ZK_LISTEN, ZK_TLS_CERT, ZK_TLS_KEY, ZK_POSTGRES_URL

## Key Design Observations

1. **Clean separation exists**: Tools already divide into "filesystem" and "database" categories
2. **No existing client code**: Only generated Connect clients exist; no wrappers yet
3. **MCP and GUI are separate**: `brains serve` (MCP) and `brains gui` (web) are independent processes
4. **Tool response pattern**: All tools return JSON strings via toJSON() helper
5. **Session management**: code-reasoning uses per-connection sessions via SessionManager
6. **Storage interfaces**: memory.Storage and recall.Storage are well-defined interfaces that can be backed by either local or remote implementations
