# Implementation Plan: Remove profile-show and profile-validate MCP Tools

**Branch**: `006-remove-mcp-tools` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-remove-mcp-tools/spec.md`

## Summary

Remove the `profile-show` and `profile-validate` tools from the MCP interface while retaining `profile-compose` and `profile-list`. This is a code removal task that simplifies the MCP tool surface by eliminating redundant tools (functionality available via CLI).

## Technical Context

**Language/Version**: Go 1.24.0
**Primary Dependencies**: mark3labs/mcp-go v0.43.2 (MCP server)
**Storage**: N/A (no storage changes)
**Testing**: go test with stretchr/testify
**Target Platform**: CLI/MCP server (macOS, Linux)
**Project Type**: Single project (Go CLI with embedded MCP server)
**Performance Goals**: N/A (removal task)
**Constraints**: Must not break existing profile-compose and profile-list functionality
**Scale/Scope**: Affects 2 files: `internal/mcp/server.go`, `internal/mcp/tools/profile/tool.go`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution is not configured (template placeholder). Proceeding with standard Go best practices:

- [x] Changes are minimal and focused (removal only)
- [x] Existing tests must continue to pass
- [x] No new dependencies introduced
- [x] Code follows existing patterns

**Gate Status**: PASS

## Project Structure

### Documentation (this feature)

```text
specs/006-remove-mcp-tools/
├── plan.md              # This file
├── spec.md              # Feature specification
└── checklists/
    └── requirements.md  # Quality checklist
```

### Source Code (repository root)

```text
internal/
├── mcp/
│   ├── server.go           # Remove profile-show and profile-validate registration
│   ├── server_test.go      # Update/remove related tests
│   └── tools/
│       └── profile/
│           └── tool.go     # Keep HandleShow/HandleValidate for potential CLI use
└── ...

cmd/
└── brains/
    └── main.go             # No changes needed
```

**Structure Decision**: Existing single-project Go structure. Changes confined to `internal/mcp/` package.

## Complexity Tracking

No violations - this is a simple removal task.

---

## Phase 0: Research

**Status**: Complete - No research needed

This is a straightforward code removal task. The codebase has been examined:

1. **MCP Server Registration** (`internal/mcp/server.go:174-222`):
   - `profile-show` tool registered at lines 199-213
   - `profile-validate` tool registered at lines 215-222
   - Handler functions `handleProfileShow` (lines 255-268) and `handleProfileValidate` (lines 270-283)

2. **Profile Tool Handlers** (`internal/mcp/tools/profile/tool.go`):
   - `HandleShow` (lines 88-117) - Keep for potential CLI use
   - `HandleValidate` (lines 119-155) - Keep for potential CLI use

3. **Retained Tools**:
   - `profile-compose` - Must remain functional
   - `profile-list` - Must remain functional

**Decision**: Remove only the MCP tool registration and handler wrappers in `server.go`. Keep the underlying `HandleShow` and `HandleValidate` methods in `tool.go` for potential future CLI integration.

---

## Phase 1: Design

### Changes Required

#### File: `internal/mcp/server.go`

**Remove from `registerProfileTools()` function:**
1. Remove `profile-show` tool registration (lines 199-213)
2. Remove `profile-validate` tool registration (lines 215-222)

**Remove handler functions:**
1. Remove `handleProfileShow` function (lines 255-268)
2. Remove `handleProfileValidate` function (lines 270-283)

#### File: `internal/mcp/server_test.go`

**Update tests:**
1. Remove any tests for `profile-show` tool
2. Remove any tests for `profile-validate` tool
3. Ensure tests for `profile-compose` and `profile-list` still pass

#### File: `internal/mcp/tools/profile/tool.go`

**No changes** - Keep `HandleShow` and `HandleValidate` methods for potential CLI use.

### Data Model

N/A - No data model changes.

### API Contracts

**MCP Tool List (After Change)**:

| Tool Name | Status |
|-----------|--------|
| stickymemory | Retained |
| code-reasoning | Retained |
| profile-compose | Retained |
| profile-list | Retained |
| profile-show | **REMOVED** |
| profile-validate | **REMOVED** |

### Verification

1. Run `go build ./...` - Must compile without errors
2. Run `go test ./internal/mcp/...` - All tests must pass
3. Start MCP server and verify tool list contains exactly 4 tools (stickymemory, code-reasoning, profile-compose, profile-list)

---

## Next Steps

Run `/speckit.tasks` to generate the implementation task list.
