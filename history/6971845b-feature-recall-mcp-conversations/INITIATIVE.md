# Initiative: recall-mcp-conversations

**Type**: feature
**Status**: completed
**Created**: 2026-01-21T17:58:51-08:00
**ID**: 6971845b-feature-recall-mcp-conversations

## Description

Add MCP tools to allow LLM agents to browse and read conversation history stored in the recall system. This enables agents to access past conversations for context and continuity.

## Goals

- Expose conversation history via MCP protocol
- Support pagination for large conversation lists and long conversations
- Provide proper error handling for invalid inputs

## Implementation

### Files Created
- `internal/mcp/tools/recall/tool.go` - Core tool implementation
- `internal/mcp/tools/recall/types.go` - Response types
- `internal/mcp/tools/recall/tool_test.go` - Unit tests (17 test cases)

### Files Modified
- `internal/recall/storage.go` - Added GetConversationChunks and ConversationExists to interface
- `internal/recall/postgres/storage.go` - PostgreSQL implementation of new methods
- `internal/mcp/server.go` - Registered recall tools
- `internal/cli/serve.go` - Wired recall storage initialization
- `internal/config/tools.go` - Added tools to KnownTools list
- `internal/webplugins/recall/plugin_test.go` - Updated mock for interface compliance

### MCP Tools Added
- `recall-list-conversations` - List conversation summaries with pagination
- `recall-read-conversation` - Read conversation chunks with pagination

## Completion

**Completed**: 2026-01-21T19:20:00-08:00
**Duration**: ~1.5 hours

### Outcomes
- T001: Add GetConversationChunks to storage interface - Complete
- T002: Create response types for MCP tools - Complete
- T003: Implement recall-list-conversations tool - Complete
- T004: Implement recall-read-conversation tool - Complete
- T005: Register recall tools in MCP server - Complete
- T006: Update serve command to pass recall storage - Complete
- T007: Add recall tools to default enabled tools list - Complete
- T008-T010: Integration tests for recall tools - Complete
- T011: Verify project builds and lints cleanly - Complete
- T012: Manual smoke test via MCP - Complete

### Notes
All tests pass. Smoke test verified:
- Listing conversations returns paginated results with has_more flag
- Reading conversations returns chronologically ordered chunks
- Error handling works for invalid UUIDs and missing conversations
- Recall tools are automatically registered when PostgreSQL backend is configured
