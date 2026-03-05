# Completeness Audit: Local Proxy Binary

## CRITICAL

### C1: Initiative tool routing contradicts codebase
The spec routes `initiative` to Server (WorkflowService.*), but the actual tool performs extensive local filesystem operations: creating directories, copying templates, reading/writing state files, creating git branches. WorkflowService has no concept of local paths or git.

### C2: Profile tool routing contradicts codebase
The spec routes all profile tools to Server (ProfileService.*), but the current implementation is entirely filesystem-based -- reading from `.brains/profiles/` and `~/.brains/profiles/`. The ProfileService exists on the server but doesn't handle local filesystem semantics (working_directory, local vs global location).

### C3: Linear ticket mentions "LLM calls" and "Complexity analysis" as proxied operations -- spec is silent
Neither appears in the routing table. No MCP tools exist for these today, but the ticket scopes them in. Must be explicitly deferred or included.

## MAJOR

### M1: No MemoryService proto -- stickymemory cannot route to server
Stickymemory is database-backed but no server-side gRPC service exists for it. Cannot satisfy "all non-filesystem operations route through gRPC" without a new service.

### M2: "Clear error when server is unreachable" is untestable
No concrete error format, codes, or diagnostic content defined.

### M3: brains_connection_status tool has no behavior spec
Listed in acceptance criteria but no schema, output format, or behavior defined.

### M4: Protocol terminology inconsistency
Spec says "Connect RPC," Linear ticket says "gRPC." Need consistent terminology.

### M5: Missing edge case: in-flight calls during connection drop
No timeout/retry semantics for individual tool calls defined.

### M6: Arg-to-RPC mappings missing
Only one example (profile-compose). Initiative tool demultiplexes one MCP tool to 4+ RPCs -- dispatch logic undefined.

## MINOR
- Routing table includes new tool without noting it's new (m1)
- Binary naming unresolved (m2)
- Server URL "required" contradicts graceful degradation (m3)
- ArtifactService/ConfigService not mentioned (m4)
- Retry parameters underspecified (m5)
