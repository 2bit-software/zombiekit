# Safety Net Assessment

## Existing Test Coverage

### Tests to be Removed
- `internal/mcp/tools/zombiekit/tool_test.go` - tests for the deprecated `feature` tool

### Tests That Remain (should continue passing)
- `internal/mcp/tools/step/` tests - cover `step` tool including `step: "feature"`
- `internal/mcp/tools/profile/` tests
- `internal/mcp/tools/initiative/` tests
- `internal/mcp/server.go` tests (if any)
- All other tool tests

## Verification Plan

1. **Before changes**: Run `go test ./...` to establish baseline
2. **After changes**: Run `go test ./...` - should pass (minus removed tests)
3. **Manual verification**: Start MCP server, verify tool list no longer includes `feature`

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Something depends on `feature` tool | Low | Medium | Grep shows no runtime dependencies |
| KnownTools change breaks config validation | Low | Low | Users may get warnings for "feature" in config |
| `step` tool doesn't cover all `feature` use cases | Low | Medium | `step` with `step: "feature"` is more capable |

## Gaps Identified

None - the `step` tool fully supersedes the `feature` tool with additional functionality (initiative context, state tracking, etc.)

## Recommended Additional Tests

None required - existing `step` tool tests cover the functionality.
