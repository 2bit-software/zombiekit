# Progress Log

## T001 - Create entry point
- Status: Complete
- Files: cmd/brains-mcp/main.go
- Notes: CLI flags for server-url, tls-ca, api-key, log-level, env-file. Registers embedded FS.

## T002 - Create proxy config
- Status: Complete
- Files: internal/proxy/config.go
- Notes: ProxyConfig with Validate() that defaults CallTimeout to 10s

## T003 - Create server connection
- Status: Complete
- Files: internal/proxy/connection.go
- Notes: Connect RPC client wrapper with health check cache (5s TTL), TLS support

## T004 - Create router + proxy shell + handler type
- Status: Complete
- Files: internal/proxy/router.go, internal/proxy/proxy.go, internal/proxy/handlers/handler.go
- Notes: Generic handleTool dispatches all 9 tools through router. Panics on duplicate registration.

## T005 - Code-reasoning handler
- Status: Complete
- Files: internal/proxy/handlers/local/codereasoning.go
- Notes: Wraps existing SessionManager + Tool

## T006 - Workflow handler
- Status: Complete
- Files: internal/proxy/handlers/local/workflow.go
- Notes: Delegates to workflowtool.HandleCompose

## T007 - Initiative handler
- Status: Complete
- Files: internal/proxy/handlers/local/initiative.go
- Notes: Delegates to initiativetool.Execute

## T008 - Profile-save handler
- Status: Complete
- Files: internal/proxy/handlers/local/profilesave.go
- Notes: Delegates to profiletool.HandleSave

## T009 - Connection status handler
- Status: Complete
- Files: internal/proxy/handlers/local/status.go
- Notes: Interface-based design for testability. Returns JSON with connected/server_url/last_check/error.

## T010 - Recall list conversations handler
- Status: Complete
- Files: internal/proxy/handlers/remote/recall.go
- Notes: Translates MCP args to Connect RPC ListConversations call. Guards with server not configured.

## T011 - Recall read conversation handler
- Status: Complete
- Files: internal/proxy/handlers/remote/recall.go (same file)
- Notes: Same pattern as T010 for GetConversation RPC.

## T012 - Profile compose hybrid handler
- Status: Complete
- Files: internal/proxy/handlers/hybrid/profilecompose.go
- Notes: Local-first with server fallback via ComposeProfile RPC. Full merge deferred.

## T013 - Profile list hybrid handler
- Status: Complete
- Files: internal/proxy/handlers/hybrid/profilelist.go
- Notes: Merges local + remote profile lists. Local overrides remote by name.

## T014 - Wire all handlers
- Status: Complete
- Files: internal/proxy/proxy.go (updated)
- Notes: All 9 handlers registered in registerHandlers(). No stickymemory.

## T015 - Integration tests
- Status: Complete
- Files: internal/proxy/router_test.go, proxy_test.go, handlers/local/status_test.go, handlers/remote/recall_test.go
- Notes: 13 tests passing. Router dispatch, proxy wiring, connection status, recall guards, arg parsing.

## T016 - Build target
- Status: Complete
- Files: Taskfile.dev.yml (updated)
- Notes: Added build:mcp task with source tracking. Binary builds and shows help.
