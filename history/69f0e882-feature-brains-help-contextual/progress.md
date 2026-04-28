# Progress Log

## T001 - Add fields to StatusResponse struct
- Status: Complete
- Files: `internal/mcp/tools/initiative/types.go`
- Notes: Added `StepStatus`, `StepsCompleted`, `StepsTotal` to StatusResponse

## T002 - Add field mappings in handleStatus()
- Status: Complete
- Files: `internal/mcp/tools/initiative/tool.go`
- Notes: Added 3 field mappings from StatusResult to StatusResponse

## T003 - Replace findAvailableDocs() with directory scan
- Status: Complete
- Files: `internal/initiative/service.go`
- Notes: Replaced hardcoded knownDocs list with os.ReadDir scan of all .md files (excluding INITIATIVE.md). Follows pattern from `internal/step/loader.go:loadAllFromDir()`.

## T004 - Verify Go changes with tests
- Status: Complete
- Notes: All `TestService_Status` subtests pass. MCP tool package builds clean. Pre-existing `internal/server` protobuf test failure unrelated to changes.

## T005 - Rewrite help.md with state-aware instructions
- Status: Complete
- Files: `embed/commands/help.md`
- Notes: Full rewrite. Two modes (no-initiative / active-initiative). Calls `initiative status` for state, reads INITIATIVE.md for step table and Source section. Includes step description lookup table, command filtering, edge case handling.

## T006+T007 - Manual validation
- Status: Complete (build-verified)
- Notes: Binary rebuilt with `task install`. Verified via `strings` that new JSON fields (`step_status`, `steps_completed`) and help.md content (step description table) are embedded. Live test requires MCP server restart (next Claude Code session).
