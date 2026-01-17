# Research: ZombieKit MCP Tool

**Feature**: 017-zombiekit-mcp
**Date**: 2025-12-23

## Tool Registration Pattern

**Decision**: Follow existing tool registration pattern in `internal/mcp/server.go`

**Rationale**: The codebase has an established pattern for MCP tools:
1. Create tool package under `internal/mcp/tools/`
2. Implement `Tool` struct with `Definition()` and `Execute()` methods
3. Register in `server.go` with enablement check via `config.IsToolEnabled()`
4. Add handler method that delegates to `Tool.Execute()`

**Alternatives considered**:
- Separate MCP server binary: Rejected (adds operational complexity, user clarified integration)
- Plugin architecture: Rejected (over-engineering for single tool)

## Home Directory Expansion

**Decision**: Use `os.UserHomeDir()` for cross-platform home directory resolution

**Rationale**:
- Standard Go library function
- Works on macOS, Linux, Windows
- Handles edge cases (missing HOME env var) with appropriate errors

**Alternatives considered**:
- Manual `$HOME` env var lookup: Less portable, Windows incompatible
- Third-party library: Unnecessary for this simple case

## Error Handling Strategy

**Decision**: Return descriptive MCP tool errors with file path and reason

**Rationale**:
- Per SC-003: "Error messages provide actionable information"
- Existing tools (stickymemory) return JSON with error details
- MCP protocol supports error responses via `mcp.NewToolResultError()`

**Error scenarios**:
1. File not found → Return path attempted + "file not found"
2. Permission denied → Return path + "permission denied"
3. Empty file → Return empty string (valid response)
4. Large file → Read entire file (no arbitrary limit for MVP)

## Tool Naming Convention

**Decision**: Tool name will be `feature` (simple, matches user request)

**Rationale**:
- User explicitly requested "feature" as the tool name
- Follows existing simple naming: `stickymemory`, `code-reasoning`
- Future ZombieKit tools can follow pattern: `zombiekit-*` or just add more tools

**Alternatives considered**:
- `zombiekit-feature`: More explicit but verbose for single tool
- `get-feature-template`: Too specific, limits future flexibility

## Configuration Integration

**Decision**: Add `feature` to `KnownTools` in `internal/config/tools.go`

**Rationale**:
- Enables `--enable-tool feature` and `--disable-tool feature` CLI flags
- Follows existing pattern for all registered tools
- Category would be `feature` (no hyphen prefix)

## File Reading Approach

**Decision**: Use `os.ReadFile()` for simple synchronous file read

**Rationale**:
- Template files are small (< 100KB typically)
- No streaming needed for MVP
- Simplest implementation

**Alternatives considered**:
- Buffered streaming: Over-engineering for template files
- Memory-mapped files: Unnecessary complexity
