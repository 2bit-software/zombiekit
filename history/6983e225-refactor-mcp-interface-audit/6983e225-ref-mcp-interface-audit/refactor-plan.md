# Refactor Plan

## Step 1: Update KnownTools List

**File**: `internal/config/tools.go`

Remove from `KnownTools` slice:
- `"feature"` (deprecated - use `step` with `step: "feature"`)
- `"profile-show"` (orphaned - never registered)
- `"profile-validate"` (orphaned - never registered)

**Rollback**: Restore the removed entries

---

## Step 2: Remove feature tool registration from server

**File**: `internal/mcp/server.go`

1. Remove import: `"github.com/2bit-software/zombiekit/internal/mcp/tools/zombiekit"`
2. Remove field: `zombiekitTool *zombiekit.Tool` from Server struct
3. Remove instantiation: `zombiekitTool := zombiekit.NewTool()`
4. Remove assignment: `zombiekitTool: zombiekitTool,` in struct literal
5. Remove registration block (lines 161-168):
   ```go
   if s.config.IsToolEnabled("feature") {
       featureDef := s.zombiekitTool.Definition()
       // ...
   }
   ```
6. Remove handler method: `handleFeature`

**Rollback**: Restore the removed code

---

## Step 3: Delete zombiekit tool package

**Directory**: `internal/mcp/tools/zombiekit/`

Delete entire directory.

**Rollback**: Restore from git

---

## Step 4: Verify

1. Run `go build ./...` - should compile
2. Run `go test ./...` - tests should pass
3. Manually verify MCP server starts and lists tools correctly

---

## Summary

| Item | Action | Risk |
|------|--------|------|
| `feature` tool | Remove | Low - superseded by `step` |
| `profile-show` | Remove from KnownTools | None - never registered |
| `profile-validate` | Remove from KnownTools | None - never registered |
| `zombiekit/` package | Delete | Low - only used by `feature` tool |
