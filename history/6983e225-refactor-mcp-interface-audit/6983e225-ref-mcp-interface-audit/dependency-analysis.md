# Dependency Analysis

## Files To Be Modified

### 1. `internal/config/tools.go`
- Remove `"feature"`, `"profile-show"`, `"profile-validate"` from `KnownTools` slice

### 2. `internal/mcp/server.go`
- Remove `zombiekitTool` field from `Server` struct (line 35)
- Remove `zombiekit.NewTool()` instantiation (line 61)
- Remove `zombiekitTool` assignment in server struct (line 79)
- Remove `feature` tool registration block (lines 161-168)
- Remove `handleFeature` method (lines 180-193)
- Remove `zombiekit` import (line 20)

### 3. `internal/mcp/tools/zombiekit/` (entire directory)
- Delete `tool.go` file
- Delete any test files

## Files NOT Affected

- `internal/mcp/tools/step/` - remains unchanged, provides `step: "feature"` functionality
- `internal/mcp/tools/profile/` - remains unchanged
- All other tool directories

## Test Impact

- Any tests specifically for the `feature` tool will be removed
- Integration tests should still pass (they should use `step` not `feature`)
