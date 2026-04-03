# Constraints

## Behavior that MUST NOT change

- All other MCP tools continue to register and function identically
- Server startup, stdio, and SSE transports unaffected
- Tool enable/disable config continues to work for remaining tools
- No public API changes to `NewServer` or `Server` methods
