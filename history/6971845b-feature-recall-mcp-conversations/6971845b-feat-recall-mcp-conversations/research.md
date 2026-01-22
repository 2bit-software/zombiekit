---
status: complete
updated: 2026-01-21
---

# Research: Recall MCP Conversations

## Executive Summary

The recall plugin already has a PostgreSQL storage layer with `ListConversations` and chunk retrieval methods. The MCP server follows a consistent tool registration pattern. New MCP tools for listing and reading conversations can reuse existing storage interfaces with minimal new code.

## Findings

### Codebase Context

**Recall Storage Interface** (`internal/recall/storage.go`):
- `ListConversations(ctx, limit, offset, project)` - Already supports pagination
- `GetConversationChunks(ctx, conversationID, limit, offset)` - Returns chunks with metadata
- `ConversationSummary` type: ID, Title, MessageCount, FirstMessage, LastMessage, Source, Project

**Data Structures** (`internal/recall/types.go`):
- `Chunk`: ID, Content, CreatedAt, Source, SourceID, ConversationID, Metadata
- `Metadata`: Role (user/assistant), Timestamp, GitBranch, CWD, ParentID
- Conversations grouped by `conversation_id` field

**MCP Tool Pattern** (`internal/mcp/server.go`):
- Tools registered via `mcp.NewTool()` with fluent schema builders
- Handler functions receive `map[string]interface{}` args
- Return JSON string or error
- Config-gated via `IsToolEnabled()`

**Pagination Pattern** (web handlers):
- Offset-based: `limit` and `page` (1-indexed)
- Uses `limit+1` fetch to detect "has more"
- Constants: DefaultPageLimit=20, MaxPageLimit=100

### Domain Knowledge

**MCP Tool Best Practices**:
- Keep tools focused (single responsibility)
- Return structured JSON for LLM parsing
- Include cursor/pagination info in responses for iterative access
- Error messages should be actionable

**Conversation Auditing Requirements**:
- Chronological ordering essential for review
- Pagination enables memory-efficient processing
- Metadata (role, timestamp, project) provides context

## Decision Points

- [x] **D1**: Single tool vs two tools
  - **Chosen**: Two tools (`recall-list-conversations`, `recall-read-conversation`)
  - **Rationale**: Separation of concerns; listing doesn't need full content, reading doesn't need the list

- [x] **D2**: Pagination style
  - **Chosen**: Offset-based with `page` and `limit` parameters
  - **Rationale**: Matches existing storage interface; simple for LLM consumption

- [x] **D3**: Response format for chunks
  - **Chosen**: JSON with role, content, timestamp, and metadata
  - **Rationale**: Provides all context needed for auditing; easily parseable

## Recommendations

1. **Create two MCP tools**:
   - `recall-list-conversations`: Returns conversation summaries with pagination
   - `recall-read-conversation`: Returns chunks for a specific conversation with pagination

2. **Reuse existing storage methods** rather than creating new queries

3. **Include pagination metadata** in responses: `{page, limit, has_more, items}`

4. **Filter by project** optional in list tool to scope conversations to specific codebases

5. **Order conversations by `last_message` descending** (latest first per user requirement)

## Sources

- `internal/recall/storage.go` - Storage interface definition
- `internal/recall/postgres/storage.go` - PostgreSQL implementation
- `internal/recall/types.go` - Data structures
- `internal/mcp/server.go` - Tool registration patterns
- `internal/mcp/tools/stickymemory/tool.go` - List/read tool example
- `internal/webplugins/recall/handlers.go` - Pagination implementation
