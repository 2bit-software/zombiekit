# Business Spec: Local Proxy Binary

## Problem

ZombieKit currently runs as a monolithic MCP server that directly accesses both the local filesystem and a PostgreSQL database. The client-server architecture (DEV-109) splits this into a central ZK server (DEV-111, complete) and a local proxy. The proxy is the missing piece -- it must present the same MCP tool interface to Claude Code while routing operations to the appropriate backend.

## Solution

A new binary (`brains-mcp`) that runs as an MCP stdio server for Claude Code. It handles local filesystem operations directly and forwards database/service operations to the central ZK server via Connect RPC.

## User Experience

### For Claude Code Users
- **Zero behavior change**: All remaining MCP tool names and argument schemas remain identical
- **New config required**: Server connection settings (address, TLS cert)
- **Graceful degradation**: Local-only tools work without server connection; server-dependent tools return clear errors
- **Tool removal**: `stickymemory` is removed entirely (not available in proxy)

### Tool Routing

Tools are classified at startup. The routing table is static:

| Tool | Route | Rationale |
|------|-------|-----------|
| profile-compose | Hybrid | Fetches remote profiles from server, merges with local profiles, composes from combined set |
| profile-list | Hybrid | Lists both local and remote profiles |
| profile-save | Local | Writes to local `.brains/profiles/` or `~/.brains/profiles/` (unchanged) |
| workflow-compose | Local | Reads embedded/local workflow definitions |
| initiative | Local | All filesystem ops (create dirs, templates, git branches). May sync metadata to WorkflowService if needed |
| code-reasoning | Local | In-memory, per-session, no persistence needed |
| recall-list-conversations | Server (SearchService.ListConversations) | Database-backed |
| recall-read-conversation | Server (SearchService.GetConversation) | Database-backed |
| brains-connection-status | Local (new tool) | Reports server connectivity |

**Removed tools:**
- `stickymemory` -- removed entirely, not available in proxy

### Hybrid Profile Flow

For `profile-compose` and `profile-list`:
1. Request current profiles from server (ProfileService.ListProfiles)
2. Read local profiles from `.brains/profiles/` and `~/.brains/profiles/`
3. Merge: local profiles override remote profiles with the same name
4. Compose/list from the merged set

For `profile-save`:
- Writes to local filesystem only (unchanged behavior)
- Profiles saved locally will override server profiles of the same name on next compose/list

### Connection Management
- Connect RPC over HTTP/2 to central server
- TLS when server is configured with certs
- Per-call timeout: 10s, no retries -- fail immediately with error to LLM
- No background reconnection loop (stdio process lifecycle tied to Claude Code session)
- Clear, actionable error messages when server is unreachable

### Configuration

New connection settings added to existing config hierarchy (env > config file > default):

| Setting | Source | Default |
|---------|--------|---------|
| Server URL | `ZK_SERVER_URL` / config | (optional -- local-only mode if unset) |
| API Key | `ZK_API_KEY` / config | (optional, for future auth) |
| TLS CA Cert | `ZK_TLS_CA` / config | (optional, system default) |
| Call timeout | config | 10s |

When `ZK_SERVER_URL` is not set, the proxy starts in local-only mode. Server-dependent tools (recall, hybrid profiles) return a clear error: "server not configured."

### New Tool: brains-connection-status

**Input:** none
**Output:**
```json
{
  "connected": true|false,
  "server_url": "http://...",
  "last_check": "2026-03-04T...",
  "error": "connection refused" | null
}
```
Performs an active health check (GET /healthz) on the server. Returns cached result if called within 5s of last check.

## Acceptance Criteria

1. `brains-mcp` binary runs as MCP server in stdio mode
2. MCP tool calls produce identical responses to the monolithic server for all retained tools
3. Local-only tools (workflow-compose, code-reasoning, initiative, profile-save) work without server connection
4. Server-dependent tools (recall-*) route through Connect RPC to ZK server
5. Hybrid tools (profile-compose, profile-list) merge local + remote profiles
6. Clear error returned when server is unreachable for a server-dependent tool
7. `brains-connection-status` tool reports server connectivity
8. `stickymemory` tool is removed
9. Integration tests verify tool routing and server communication

## Out of Scope

- Profile caching/streaming push (DEV-113)
- Auth beyond basic API key (DEV-115)
- Web UI migration (DEV-114)
- Migration tooling from monolithic setup (DEV-116)
- LLM proxy (deferred in DEV-111)
- Complexity analysis (no MCP tool exists today)

## Resolved Decisions

1. **Binary naming**: `brains-mcp`
2. **Stickymemory**: Removed entirely -- not on server, not local
3. **Offline mode**: No explicit mode. If `ZK_SERVER_URL` unset, starts in local-only. Server-dependent tools fail with clear error.
4. **Initiative**: Local with filesystem ops. WorkflowService sync is optional/deferred.
5. **Profiles**: Hybrid -- fetch remote, merge with local, local overrides remote.
