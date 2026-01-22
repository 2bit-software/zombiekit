# Implementation Plan: Recall MCP Conversations

**Feature Branch**: `6971845b-feature-recall-mcp-conversations`
**Created**: 2026-01-21
**Status**: Draft

## Summary

Add two MCP tools (`recall-list-conversations`, `recall-read-conversation`) to expose conversation data for LLM auditing workflows. The storage layer already has `ListConversations` with pagination; we need to add paginated chunk retrieval.

## Dependencies

```
1. Storage Layer Extension (blocking)
   ├── 2. MCP Tool: recall-list-conversations
   └── 3. MCP Tool: recall-read-conversation
4. Integration Tests (depends on 2, 3)
```

## Implementation Steps

### Step 1: Extend Storage Interface

**Goal**: Add `GetConversationChunks(ctx, conversationID, limit, offset)` to support paginated chunk retrieval.

**Files**:
- `internal/recall/storage.go` - Add interface method
- `internal/recall/postgres/storage.go` - Implement PostgreSQL query

**Changes**:
1. Add method signature to `Storage` interface:
   ```go
   GetConversationChunks(ctx context.Context, conversationID string, limit, offset int) ([]Chunk, error)
   ```

2. Implement in PostgreSQL storage with query:
   ```sql
   SELECT id, content, created_at, source, source_id, conversation_id, metadata
   FROM recall_chunks
   WHERE conversation_id = $1
   ORDER BY (metadata->>'timestamp')::timestamptz ASC NULLS LAST, id ASC
   LIMIT $2 OFFSET $3
   ```

**Why `id ASC` as secondary sort**: Spec requires deterministic ordering when timestamps are identical. UUID sorting provides consistent ordering across calls.

**Validation**: Existing `GetByConversation` tests can be adapted; new tests for pagination bounds.

---

### Step 2: Create Recall MCP Tool Package

**Goal**: Create tool structure following the stickymemory pattern.

**Files**:
- `internal/mcp/tools/recall/tool.go` - Tool implementation
- `internal/mcp/tools/recall/types.go` - Response types

**Structure**:
```go
type Tool struct {
    storage recall.Storage
}

func NewTool(storage recall.Storage) *Tool

func (t *Tool) ListConversations(ctx, args) (string, error)
func (t *Tool) ReadConversation(ctx, args) (string, error)
```

**Response Types** (per spec):
```go
type ListResponse struct {
    Page    int                   `json:"page"`
    Limit   int                   `json:"limit"`
    HasMore bool                  `json:"has_more"`
    Items   []ConversationSummary `json:"items"`
}

type ReadResponse struct {
    ConversationID string  `json:"conversation_id"`
    Page           int     `json:"page"`
    Limit          int     `json:"limit"`
    HasMore        bool    `json:"has_more"`
    Items          []Chunk `json:"items"`
}
```

---

### Step 3: Implement recall-list-conversations Tool

**Goal**: List conversations with pagination and optional project filter.

**Input Validation**:
- `page`: Default 1, minimum 1
- `limit`: Default 20, cap at 100
- `project`: Optional string, empty returns all

**has_more Detection**:
```go
items, _ := storage.ListConversations(ctx, limit+1, offset, project)
hasMore := len(items) > limit
if hasMore {
    items = items[:limit]
}
```

**Error Cases**:
- Invalid parameters → return JSON error response (don't return Go error)

---

### Step 4: Implement recall-read-conversation Tool

**Goal**: Read conversation chunks with pagination.

**Input Validation**:
- `conversation_id`: Required, must be valid UUID format
- `page`: Default 1, minimum 1
- `limit`: Default 20, cap at 100

**UUID Validation**:
```go
if _, err := uuid.Parse(conversationID); err != nil {
    return `{"error": "invalid conversation_id format"}`, nil
}
```

**Conversation Existence Check**:
Before fetching chunks, verify conversation exists by checking if any chunks exist for the ID. If none exist, return `{"error": "conversation not found"}`.

**has_more Detection**: Same pattern as list tool.

---

### Step 5: Register Tools in MCP Server

**Goal**: Wire tools into the MCP server with schema definitions.

**Files**:
- `internal/mcp/server.go` - Add tool registration

**Changes**:
1. Add `recallTool *recall.Tool` field to Server struct
2. Create tool in NewServer (requires recall.Storage dependency)
3. Add `registerRecallTools()` method
4. Register two MCP tools with schemas:

```go
// recall-list-conversations
mcp.NewTool("recall-list-conversations",
    mcp.WithDescription("List conversation summaries with pagination"),
    mcp.WithNumber("page", mcp.Description("Page number (1-indexed)")),
    mcp.WithNumber("limit", mcp.Description("Items per page (max 100)")),
    mcp.WithString("project", mcp.Description("Filter by project path prefix")),
)

// recall-read-conversation
mcp.NewTool("recall-read-conversation",
    mcp.WithDescription("Read conversation chunks with pagination"),
    mcp.WithString("conversation_id", mcp.Required(), mcp.Description("Conversation UUID")),
    mcp.WithNumber("page", mcp.Description("Page number (1-indexed)")),
    mcp.WithNumber("limit", mcp.Description("Items per page (max 100)")),
)
```

**Config Gating**: Add `recall-list-conversations` and `recall-read-conversation` to config system.

---

### Step 6: Update Server Dependencies

**Goal**: Provide recall.Storage to MCP server.

**Files**:
- `cmd/brains/serve.go` - Pass recall storage to MCP server

**Changes**:
- The serve command already creates recall storage for the import system
- Pass this storage to the MCP server constructor
- Update `NewServer` signature: `NewServer(memoryStorage, recallStorage, cfg)`

---

### Step 7: Integration Tests

**Goal**: Verify tools work end-to-end with PostgreSQL.

**Files**:
- `internal/mcp/tools/recall/tool_test.go`

**Test Setup**:
- Use existing PostgreSQL test infrastructure
- Seed test data: 3 conversations across 2 projects with varying chunk counts (5, 25, 100+)

**Test Cases**:
| Test | Validates |
|------|-----------|
| ListConversations_DefaultPagination | FR-001, FR-004, FR-010 |
| ListConversations_CustomPage | FR-004, FR-005 |
| ListConversations_ProjectFilter | FR-006 |
| ListConversations_LimitCapped | FR-009 |
| ListConversations_EmptyResult | Edge case |
| ReadConversation_DefaultPagination | FR-002, FR-007 |
| ReadConversation_CustomPage | FR-004, FR-005 |
| ReadConversation_InvalidUUID | Error case |
| ReadConversation_NotFound | Error case |
| ReadConversation_ChronologicalOrder | FR-007, FR-008 |

---

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Storage query performance on large datasets | Low | Existing indexes on conversation_id; LIMIT/OFFSET is standard |
| Breaking existing storage interface | Low | Adding method, not changing existing |
| MCP protocol compatibility | Low | Following established patterns in codebase |

## Open Questions

None - spec is complete and research confirmed all implementation details.

## Traceability

| Spec Requirement | Implementation Step |
|------------------|---------------------|
| FR-001 | Step 3 |
| FR-002 | Step 4 |
| FR-003 | Step 3 (storage already orders by last_message DESC) |
| FR-004 | Steps 3, 4 |
| FR-005 | Steps 3, 4 |
| FR-006 | Step 3 (storage already supports project filter) |
| FR-007 | Steps 1, 4 |
| FR-008 | Step 4 |
| FR-009 | Steps 3, 4 |
| FR-010 | Steps 3, 4 |
