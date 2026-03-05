# AI-Friendliness Audit: Local Proxy Binary

## CRITICAL

### C1: Initiative routing creates impossible implementation
An AI agent cannot implement the initiative proxy handler. The spec says "route to server" but the tool requires local filesystem access (directory creation, template copying, git branches) that the server doesn't support. An AI would either build a broken proxy or get stuck.

### C2: Profile routing omits local filesystem semantics
The `profile-save` tool accepts `working_directory` and `location: "local"|"global"` -- concepts that don't exist in ProfileService RPCs. An AI has no guidance on whether to drop these args, translate them, or keep local behavior for some operations.

### C3: Stickymemory handler cannot be implemented
The tech requirements place `stickymemory.go` in `handlers/remote/` but no MemoryService proto exists. An AI would hit a compile error trying to create a client for a non-existent service.

## MAJOR

### M1: No field-level MCP-to-RPC argument mapping
Only one example provided. The initiative tool dispatches one MCP tool to 4+ RPCs based on an `action` arg -- the demux logic isn't described. An AI would need to reverse-engineer the current tool and the proto to figure out the mapping.

### M2: Open questions are blocking decisions
Binary naming (#1) affects package layout. Stickymemory routing (#2) affects handler placement. Offline mode (#3) affects startup validation. An AI cannot proceed without answers.

### M3: Per-call timeout vs. reconnection behavior unclear
Is backoff per-call or background? For a stdio MCP server, long retries block the LLM. The spec should say "fail immediately with error, no per-call retry."

### M4: "Reuse existing implementations" lacks specifics
Existing tools use `mcp.CallToolRequest` -> `*mcp.CallToolResult` signatures. The proxy needs to either reuse these handlers directly or extract and re-wrap the logic. Not specified which approach.

### M5: Config startup behavior unresolved
Is ZK_SERVER_URL required at startup? If yes, local-only mode is impossible. If no, how does the proxy know which tools are available?

## MINOR
- Tool naming inconsistency (hyphens vs underscores)
- Version pinning in tech requirements may drift
- Directory structure assumes stickymemory routing decision
- Acceptance criterion #2 not testable without Claude Code in the loop
- Proposed structure premature given unresolved routing
