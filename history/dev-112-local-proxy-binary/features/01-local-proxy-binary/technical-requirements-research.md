# Technical Requirements Research: Local Proxy Binary

## Architecture

### Proxy Pattern
The proxy is a **routing MCP server** -- it implements the same mcp-go tool interface but dispatches each tool call to either a local handler, a remote Connect RPC client, or a hybrid of both.

### Key Constraint
"The routing logic should be a simple lookup table, not conditional branching that grows over time." (from DEV-112 ticket)

### Proposed Structure
```
cmd/brains-mcp/main.go
internal/proxy/
  config.go                     # Proxy config (server URL, TLS, timeouts)
  proxy.go                      # MCP server with routing table
  router.go                     # Tool -> handler routing (static lookup)
  connection.go                 # Connect RPC client lifecycle + health check
  handlers/
    local/
      workflow.go               # workflow-compose (reuse existing handler)
      codereasoning.go          # code-reasoning (reuse existing handler)
      initiative.go             # initiative (reuse existing handler)
      profilesave.go            # profile-save (reuse existing handler)
      status.go                 # brains-connection-status (new)
    remote/
      recall.go                 # recall-* -> SearchService
    hybrid/
      profilecompose.go         # profile-compose: fetch remote + merge local
      profilelist.go            # profile-list: merge local + remote listings
```

## Tool-to-Handler Mapping

### Local Handlers (reuse existing code)

| Tool | Existing Package | Handler Signature |
|------|-----------------|-------------------|
| workflow-compose | `internal/mcp/tools/workflow/` | `HandleCompose(ctx, args) -> JSON` |
| code-reasoning | `internal/mcp/tools/codereasoning/` | `Execute(ctx, args) -> JSON` |
| initiative | `internal/mcp/tools/initiative/` | `Execute(ctx, args) -> JSON` |
| profile-save | `internal/mcp/tools/profile/` | `HandleSave(ctx, args) -> JSON` |

These handlers use `mcp.CallToolRequest` -> `*mcp.CallToolResult`. Reuse directly by wrapping with the same handler signature.

### Remote Handlers (arg -> RPC mapping)

**recall-list-conversations:**
```
MCP args: {page: 1, limit: 20, project: "/path/to/project"}
-> SearchService.ListConversations({
     pagination: {page_size: 20},
     project_filter: "/path/to/project"
   })
-> return {conversations: [...], pagination: {total_count: N}}
```

**recall-read-conversation:**
```
MCP args: {conversation_id: "abc-123", page: 1, limit: 20}
-> SearchService.GetConversation({
     conversation_id: "abc-123",
     pagination: {page_size: 20}
   })
-> return {conversation: {...}, chunks: [...]}
```

### Hybrid Handlers (local + remote merge)

**profile-compose:**
```
MCP args: {profiles: ["base", "go-expert"], working_directory: "/path"}
1. Call ProfileService.ListProfiles({}) -> remote profiles
2. Read local profiles from .brains/profiles/ and ~/.brains/profiles/
3. Build merged map: remote profiles as base, local profiles override by name
4. Resolve requested profile names from merged map
5. Compose content (respecting dependencies)
6. Return composed content
```

**profile-list:**
```
MCP args: {working_directory: "/path"}
1. Call ProfileService.ListProfiles({}) -> remote profiles
2. Read local profiles from .brains/profiles/ and ~/.brains/profiles/
3. Merge: local overrides remote by name
4. Return merged list with source annotation (local/remote)
```

**If server unreachable for hybrid tools:**
- Fall back to local-only profiles (same as current behavior)
- Include warning in response: "Server unreachable, showing local profiles only"

### New Tool: brains-connection-status

```
MCP args: {} (none)
1. If last health check < 5s ago, return cached result
2. GET {server_url}/healthz with call timeout
3. Return {connected: bool, server_url: string, last_check: timestamp, error: string|null}
```

## Dependencies

### Existing (already in go.mod)
- `github.com/mark3labs/mcp-go v0.43.2` -- MCP server framework
- `connectrpc.com/connect v1.19.1` -- Connect RPC client
- `github.com/urfave/cli/v2` -- CLI framework
- Generated Connect clients in `gen/zombiekit/brains/*/v1/*connect/`

### Removal
- `stickymemory` tool and its storage dependencies can be removed from the proxy binary (memory.Storage not needed)

## Connect Client Setup

```go
httpClient := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
    },
}
profiles := profilev1connect.NewProfileServiceClient(httpClient, serverURL)
search := searchv1connect.NewSearchServiceClient(httpClient, serverURL)
```

## MCP Protocol Constraints (stdio mode)

- stdout is reserved for MCP JSON-RPC -- no logging to stdout
- stderr available for logging (use DI logger writing to stderr)
- Process lifecycle tied to Claude Code session
- No HTTP server needed in stdio mode

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Server URL not configured | Start normally. Server-dependent tools return: "server not configured" |
| Server unreachable (remote tool) | Return error immediately: "server unreachable: {details}" |
| Server unreachable (hybrid tool) | Fall back to local-only, include warning |
| Server returns error | Forward error message to MCP response |
| Invalid tool args | Return validation error (same as current) |

## Testing Strategy

- **Unit tests:** Router dispatches to correct handler type (local/remote/hybrid)
- **Integration tests:** Remote + hybrid handlers communicate with real ZK server (testcontainers)
- **Contract tests:** MCP tool call with known args produces same response as monolithic server
