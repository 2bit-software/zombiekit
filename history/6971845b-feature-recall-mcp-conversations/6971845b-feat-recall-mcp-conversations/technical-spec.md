# Technical Spec: Recall MCP Conversations

**Feature Branch**: `6971845b-feature-recall-mcp-conversations`
**Created**: 2026-01-21

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      MCP Protocol                            │
│  (stdio/SSE transport)                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    internal/mcp/server.go                    │
│  - Registers recall-list-conversations                       │
│  - Registers recall-read-conversation                        │
│  - Routes calls to recall.Tool                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               internal/mcp/tools/recall/tool.go              │
│  - ListConversations(ctx, args) → JSON                       │
│  - ReadConversation(ctx, args) → JSON                        │
│  - Input validation, pagination logic, error handling        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  internal/recall/storage.go                  │
│  Storage interface:                                          │
│  - ListConversations(ctx, limit, offset, project) ✓ exists   │
│  - GetConversationChunks(ctx, convID, limit, offset) NEW     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│             internal/recall/postgres/storage.go              │
│  PostgreSQL implementation with pgvector                     │
└─────────────────────────────────────────────────────────────┘
```

## Interface Definitions

### Storage Interface Extension

```go
// In internal/recall/storage.go

// GetConversationChunks returns chunks for a conversation with pagination.
// Ordered by timestamp ascending (oldest first), then by ID for determinism.
// Returns empty slice if conversation doesn't exist.
GetConversationChunks(ctx context.Context, conversationID string, limit, offset int) ([]Chunk, error)
```

### MCP Tool Interface

```go
// In internal/mcp/tools/recall/tool.go

// Tool provides MCP tools for conversation retrieval.
type Tool struct {
    storage recall.Storage
}

// NewTool creates a recall tool with the given storage backend.
func NewTool(storage recall.Storage) *Tool

// ListConversations handles recall-list-conversations tool calls.
func (t *Tool) ListConversations(ctx context.Context, args map[string]interface{}) (string, error)

// ReadConversation handles recall-read-conversation tool calls.
func (t *Tool) ReadConversation(ctx context.Context, args map[string]interface{}) (string, error)
```

## Data Types

### Response Types

```go
// In internal/mcp/tools/recall/types.go

// ListResponse is the recall-list-conversations response.
type ListResponse struct {
    Page    int                          `json:"page"`
    Limit   int                          `json:"limit"`
    HasMore bool                         `json:"has_more"`
    Items   []recall.ConversationSummary `json:"items"`
}

// ReadResponse is the recall-read-conversation response.
type ReadResponse struct {
    ConversationID string        `json:"conversation_id"`
    Page           int           `json:"page"`
    Limit          int           `json:"limit"`
    HasMore        bool          `json:"has_more"`
    Items          []ChunkOutput `json:"items"`
}

// ChunkOutput is a conversation chunk for MCP output.
// Flattens metadata into top-level fields for easier consumption.
type ChunkOutput struct {
    ID        string `json:"id"`
    Content   string `json:"content"`
    Role      string `json:"role"`
    Timestamp string `json:"timestamp"` // ISO-8601
    Project   string `json:"project"`   // CWD
    GitBranch string `json:"git_branch"`
}

// ErrorResponse is returned for validation and not-found errors.
type ErrorResponse struct {
    Error string `json:"error"`
}
```

### Existing Types (no changes)

```go
// recall.ConversationSummary already matches spec:
type ConversationSummary struct {
    ConversationID string    `json:"conversation_id"`
    Title          string    `json:"title"`
    MessageCount   int       `json:"message_count"`
    FirstMessage   time.Time `json:"first_message"`
    LastMessage    time.Time `json:"last_message"`
    Source         string    `json:"source"`
    Project        string    `json:"project"`
}
```

## SQL Queries

### GetConversationChunks

```sql
SELECT id, content, created_at, source, source_id, conversation_id, metadata
FROM recall_chunks
WHERE conversation_id = $1
ORDER BY (metadata->>'timestamp')::timestamptz ASC NULLS LAST, id ASC
LIMIT $2 OFFSET $3
```

**Index usage**: Existing `idx_recall_chunks_conversation_id` covers the WHERE clause. The ORDER BY uses a function on metadata, which may not be indexed but is acceptable for conversation-sized result sets (typically <1000 chunks).

### ConversationExists (for not-found detection)

```sql
SELECT EXISTS(
    SELECT 1 FROM recall_chunks WHERE conversation_id = $1
)
```

## Pagination Logic

### Constants

```go
const (
    DefaultPageLimit = 20
    MaxPageLimit     = 100
)
```

### Parameter Normalization

```go
func normalizePageParams(args map[string]interface{}) (page, limit int) {
    page = 1
    if p, ok := args["page"].(float64); ok && p >= 1 {
        page = int(p)
    }

    limit = DefaultPageLimit
    if l, ok := args["limit"].(float64); ok {
        if l > 0 {
            limit = int(l)
        }
        if limit > MaxPageLimit {
            limit = MaxPageLimit
        }
    }
    return page, limit
}

func calculateOffset(page, limit int) int {
    return (page - 1) * limit
}
```

### has_more Detection

```go
// Fetch one extra item to detect more pages
items, err := storage.ListConversations(ctx, limit+1, offset, project)
if err != nil {
    return "", fmt.Errorf("list conversations: %w", err)
}

hasMore := len(items) > limit
if hasMore {
    items = items[:limit]
}
```

## Error Handling

### Error Response Pattern

Return JSON error objects, not Go errors, for user-facing validation issues:

```go
func (t *Tool) ReadConversation(ctx context.Context, args map[string]interface{}) (string, error) {
    convID, ok := args["conversation_id"].(string)
    if !ok || convID == "" {
        return `{"error": "conversation_id is required"}`, nil
    }

    if _, err := uuid.Parse(convID); err != nil {
        return `{"error": "invalid conversation_id format"}`, nil
    }

    // Check existence
    exists, err := t.conversationExists(ctx, convID)
    if err != nil {
        return "", fmt.Errorf("check conversation exists: %w", err) // Internal error
    }
    if !exists {
        return `{"error": "conversation not found"}`, nil
    }

    // ... proceed with fetch
}
```

### Error Categories

| Category | Response Type | Example |
|----------|---------------|---------|
| Validation error | JSON error object | Missing required param, invalid UUID |
| Not found | JSON error object | Conversation doesn't exist |
| Internal error | Go error | Database failure |

## MCP Schema Definitions

### recall-list-conversations

```go
mcp.NewTool("recall-list-conversations",
    mcp.WithDescription("List conversation summaries with pagination. Returns conversations ordered by last activity (most recent first)."),
    mcp.WithNumber("page",
        mcp.Description("Page number (1-indexed). Defaults to 1."),
    ),
    mcp.WithNumber("limit",
        mcp.Description("Items per page. Defaults to 20, maximum 100."),
    ),
    mcp.WithString("project",
        mcp.Description("Filter by project path prefix (e.g., '/Users/me/project'). Empty returns all."),
    ),
)
```

### recall-read-conversation

```go
mcp.NewTool("recall-read-conversation",
    mcp.WithDescription("Read conversation chunks with pagination. Returns chunks in chronological order (oldest first)."),
    mcp.WithString("conversation_id",
        mcp.Required(),
        mcp.Description("Conversation UUID to read"),
    ),
    mcp.WithNumber("page",
        mcp.Description("Page number (1-indexed). Defaults to 1."),
    ),
    mcp.WithNumber("limit",
        mcp.Description("Items per page. Defaults to 20, maximum 100."),
    ),
)
```

## Configuration

Add tool enablement to config:

```go
// In internal/config/config.go, add to default enabled tools
var defaultEnabledTools = []string{
    "stickymemory",
    "code-reasoning",
    // ... existing tools
    "recall-list-conversations",
    "recall-read-conversation",
}
```

## Server Integration

### NewServer Signature Change

```go
// Current
func NewServer(storage memory.Storage, cfg *config.Config) *Server

// Updated
func NewServer(memoryStorage memory.Storage, recallStorage recall.Storage, cfg *config.Config) *Server
```

### Registration

```go
func (s *Server) registerRecallTools() {
    if s.config.IsToolEnabled("recall-list-conversations") {
        listTool := mcp.NewTool("recall-list-conversations", /* ... */)
        s.mcpServer.AddTool(listTool, s.handleRecallListConversations)
    }

    if s.config.IsToolEnabled("recall-read-conversation") {
        readTool := mcp.NewTool("recall-read-conversation", /* ... */)
        s.mcpServer.AddTool(readTool, s.handleRecallReadConversation)
    }
}
```

## Test Data Setup

For integration tests, seed with known data:

```go
func seedTestConversations(t *testing.T, storage recall.Storage) {
    // Conversation 1: 5 chunks, project /foo
    // Conversation 2: 25 chunks, project /foo/bar
    // Conversation 3: 105 chunks, project /baz

    // Use fixed timestamps for deterministic ordering tests
    // Use fixed UUIDs for reproducibility
}
```

## File Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/recall/storage.go` | Modify | Add GetConversationChunks method |
| `internal/recall/postgres/storage.go` | Modify | Implement GetConversationChunks |
| `internal/mcp/tools/recall/tool.go` | Create | Tool implementation |
| `internal/mcp/tools/recall/types.go` | Create | Response types |
| `internal/mcp/tools/recall/tool_test.go` | Create | Integration tests |
| `internal/mcp/server.go` | Modify | Register tools, add recall storage field |
| `cmd/brains/serve.go` | Modify | Pass recall storage to MCP server |
| `internal/config/config.go` | Modify | Add tools to defaults |
