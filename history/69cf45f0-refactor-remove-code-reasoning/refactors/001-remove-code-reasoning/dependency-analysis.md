# Dependency Analysis

## Files to delete

- `internal/mcp/tools/codereasoning/types.go`
- `internal/mcp/tools/codereasoning/tool.go`
- `internal/mcp/tools/codereasoning/session.go`
- `internal/mcp/tools/codereasoning/manager.go`
- `internal/mcp/tools/codereasoning/tool_test.go`
- `internal/mcp/tools/codereasoning/session_test.go`
- `internal/mcp/tools/codereasoning/manager_test.go`

## Files to edit

1. **`internal/mcp/server.go`**
   - Remove import of `codereasoning` package
   - Remove `codeReasoning` and `sessionManager` fields from `Server` struct
   - Remove instantiation in `NewServer` (lines 60, 62)
   - Remove assignment in struct literal (lines 89-90)
   - Remove registration block in `registerTools` (lines 132-167)
   - Remove `handleCodeReasoning` method (lines 235-251)
   - Update package doc comment (line 2)

2. **`internal/mcp/server_test.go`**
   - Remove `TestServer_CodeReasoning_Execute` test function

3. **`internal/config/tools.go`**
   - Remove `"code-reasoning"` from `KnownTools` slice
   - Remove `"code-reasoning" -> "code"` from `ToolCategory` doc comment

## Callers / dependents

The `codereasoning` package is only imported by `internal/mcp/server.go`. No other packages depend on it. Removal is fully contained.
