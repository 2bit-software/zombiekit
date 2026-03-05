# Technical Spec: Local Proxy Binary

## Package Architecture

```
cmd/brains-mcp/
  main.go                          # Entry point, config loading, stdio transport

internal/proxy/
  config.go                        # ProxyConfig struct + loader
  connection.go                    # Connect RPC client wrapper + health check
  proxy.go                         # MCP server + tool registration + routing
  router.go                        # Static tool-to-handler routing table
  handlers/
    handler.go                     # Handler type alias
    local/
      codereasoning.go             # Wraps existing codereasoning.Tool
      workflow.go                  # Wraps existing workflowtool.Tool
      initiative.go                # Wraps existing initiativetool.Tool
      profilesave.go               # Wraps existing profiletool.HandleSave
      status.go                    # brains-connection-status (new)
    remote/
      recall.go                    # recall-* -> SearchService RPCs
    hybrid/
      profilecompose.go            # profile-compose: local + remote merge
      profilelist.go               # profile-list: local + remote merge
```

## Key Types

### ProxyConfig

```go
type ProxyConfig struct {
    ServerURL   string        // ZK server base URL (empty = local-only mode)
    TLSCAPath   string        // Path to CA cert for TLS verification
    APIKey      string        // API key for future auth
    CallTimeout time.Duration // Per-RPC timeout (default 10s)
    LogLevel    string        // debug/info/warn/error
}
```

### Connection

```go
type Connection struct {
    baseURL     string
    httpClient  *http.Client
    profiles    profilev1connect.ProfileServiceClient
    search      searchv1connect.SearchServiceClient
    lastHealth  time.Time
    lastHealthOK bool
    lastHealthErr string
    mu          sync.Mutex
}

func NewConnection(cfg *ProxyConfig) (*Connection, error)
func (c *Connection) Profiles() profilev1connect.ProfileServiceClient
func (c *Connection) Search() searchv1connect.SearchServiceClient
func (c *Connection) HealthCheck(ctx context.Context) (bool, error)
func (c *Connection) IsConfigured() bool
```

### Handler

```go
// Handler is the common signature for all tool handlers.
type Handler func(ctx context.Context, args map[string]any) (string, error)
```

### Router

```go
type Router struct {
    handlers map[string]Handler
}

func NewRouter() *Router
func (r *Router) Register(toolName string, handler Handler)
func (r *Router) Dispatch(ctx context.Context, toolName string, args map[string]any) (string, error)
```

### Proxy

```go
type Proxy struct {
    mcpServer  *server.MCPServer
    router     *Router
    connection *Connection  // nil if local-only mode
    logger     *slog.Logger // writes to stderr
}

func NewProxy(cfg *ProxyConfig) (*Proxy, error)
func (p *Proxy) ServeStdio() error
```

## Tool Registration

The proxy registers the same tool definitions (names, schemas) as the monolithic server. Each tool's mcp-go handler follows this pattern:

```go
func (p *Proxy) handleTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args, ok := req.Params.Arguments.(map[string]any)
    if !ok {
        return mcp.NewToolResultError("invalid arguments"), nil
    }
    result, err := p.router.Dispatch(ctx, req.Params.Name, args)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(result), nil
}
```

This is a single generic handler used for all tools -- the router does the dispatch.

## Routing Table (static, built at init)

```go
// Local handlers
router.Register("code-reasoning", localHandlers.CodeReasoning)
router.Register("workflow-compose", localHandlers.WorkflowCompose)
router.Register("initiative", localHandlers.Initiative)
router.Register("profile-save", localHandlers.ProfileSave)
router.Register("brains-connection-status", localHandlers.ConnectionStatus)

// Remote handlers (require server connection)
router.Register("recall-list-conversations", remoteHandlers.RecallListConversations)
router.Register("recall-read-conversation", remoteHandlers.RecallReadConversation)

// Hybrid handlers (local + remote merge)
router.Register("profile-compose", hybridHandlers.ProfileCompose)
router.Register("profile-list", hybridHandlers.ProfileList)
```

**Total: 9 tools** (8 existing + 1 new, minus stickymemory which is removed)

## Local Handler Pattern

Local handlers wrap existing tool implementations:

```go
func NewInitiativeHandler() Handler {
    tool := initiativetool.NewTool()
    return func(ctx context.Context, args map[string]any) (string, error) {
        return tool.Execute(ctx, args)
    }
}
```

## Remote Handler Pattern

Remote handlers translate args to Connect RPC calls:

```go
func NewRecallListHandler(conn *Connection) Handler {
    return func(ctx context.Context, args map[string]any) (string, error) {
        if !conn.IsConfigured() {
            return "", errors.New("server not configured")
        }
        // Extract args
        limit := intArg(args, "limit", 20)
        project := stringArg(args, "project", "")
        // Call RPC
        resp, err := conn.Search().ListConversations(ctx,
            connect.NewRequest(&searchv1.ListConversationsRequest{
                Pagination:    &commonv1.PageRequest{PageSize: int32(limit)},
                ProjectFilter: project,
            }))
        if err != nil {
            return "", fmt.Errorf("server unreachable: %w", err)
        }
        // Format response matching monolithic server output
        return formatConversationList(resp.Msg), nil
    }
}
```

## Hybrid Handler Pattern

Hybrid handlers merge local + remote data:

```go
func NewProfileComposeHandler(conn *Connection) Handler {
    localTool := profiletool.NewTool()
    return func(ctx context.Context, args map[string]any) (string, error) {
        profileNames := stringSliceArg(args, "profiles")
        workDir := stringArg(args, "working_directory", "")

        // 1. Get local profiles
        localProfiles := localTool.ResolveProfiles(workDir)

        // 2. Get remote profiles (if server available)
        var remoteProfiles []*profilev1.Profile
        if conn.IsConfigured() {
            resp, err := conn.Profiles().ListProfiles(ctx,
                connect.NewRequest(&profilev1.ListProfilesRequest{}))
            if err != nil {
                // Warn but continue with local-only
                // (include warning in result)
            } else {
                remoteProfiles = resp.Msg.Profiles
            }
        }

        // 3. Merge: local overrides remote by name
        merged := mergeProfiles(localProfiles, remoteProfiles)

        // 4. Compose requested profiles from merged set
        return composeFromMerged(merged, profileNames)
    }
}
```

## Logging

- Logger initialized writing to **stderr** (not stdout)
- Passed via dependency injection to handlers that need it
- **Never** use `logging.Logger()` singleton (would need init at startup and risks stdout contamination)
- MCP tools must not write to stdout

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
```

## Error Response Format

All error responses follow this pattern for consistency:

| Scenario | Error Message |
|----------|---------------|
| Server not configured | `"server not configured: {tool} requires a ZK server connection. Set ZK_SERVER_URL to enable."` |
| Server unreachable | `"server unreachable: {underlying error}"` |
| Invalid args | `"invalid arguments: {details}"` |
| Server-side error | Forward server error message directly |
| Hybrid fallback | Success response with warning: `"[warning: server unreachable, showing local profiles only]\n{result}"` |

## Configuration Loading

```go
// Priority: CLI flags > env vars > defaults
cfg := &ProxyConfig{
    ServerURL:   cliFlag("server-url", envOr("ZK_SERVER_URL", "")),
    TLSCAPath:   cliFlag("tls-ca", envOr("ZK_TLS_CA", "")),
    APIKey:      cliFlag("api-key", envOr("ZK_API_KEY", "")),
    CallTimeout: 10 * time.Second,
    LogLevel:    cliFlag("log-level", envOr("ZK_LOG_LEVEL", "info")),
}
```

## Build Target

```yaml
# Taskfile.dev.yml
build:mcp:
  desc: Build brains-mcp binary
  cmds:
    - go build -o bin/brains-mcp ./cmd/brains-mcp/
  sources:
    - cmd/brains-mcp/**/*.go
    - internal/proxy/**/*.go
    - internal/mcp/tools/**/*.go
```
