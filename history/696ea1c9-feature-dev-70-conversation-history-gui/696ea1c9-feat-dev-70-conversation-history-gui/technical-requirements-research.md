# Technical Requirements: Conversation History Search GUI

**Feature**: DEV-70
**Updated**: 2026-01-19

## Implementation Preferences (from Linear ticket & research)

These are technical constraints and preferences extracted from the requirements and codebase analysis.

### Architecture

- **Plugin Pattern**: Implement as `internal/webplugins/recall/` following memory plugin structure
- **HTMX**: Server-rendered partials, no JavaScript framework
- **Routing**: Chi router scoped under `/recall`

### Storage Layer Changes

**Required**: Add `ListConversations()` method to `recall.Storage` interface:

```go
// ConversationSummary contains aggregated metadata for a conversation.
type ConversationSummary struct {
    ConversationID string
    Title          string    // Derived from first user message or CWD
    MessageCount   int
    FirstMessage   time.Time
    LastMessage    time.Time
    Source         string    // e.g., "claude"
    Project        string    // CWD/project path
}

// ListConversations returns conversations ordered by last activity (descending - most recent first).
// The limit parameter controls how many conversations to return per page.
// If limit is 0, the implementation uses a default maximum (e.g., 100).
// The offset parameter supports pagination (skip first N results).
ListConversations(ctx context.Context, limit, offset int) ([]ConversationSummary, error)
```

**Behavior**:
- Returns conversations ordered by `LastMessage` timestamp descending (most recent first)
- `limit=0` → implementation default (PostgreSQL: 100)
- `limit>0` → return at most `limit` conversations
- `offset` → skip first N conversations (for pagination)

**SQL Approach** (PostgreSQL):
```sql
SELECT
    conversation_id,
    COUNT(*) as message_count,
    MIN((metadata->>'timestamp')::timestamptz) as first_message,
    MAX((metadata->>'timestamp')::timestamptz) as last_message,
    source,
    (SELECT content FROM recall_chunks c2
     WHERE c2.conversation_id = recall_chunks.conversation_id
     AND c2.metadata->>'role' = 'user'
     ORDER BY (c2.metadata->>'timestamp')::timestamptz
     LIMIT 1) as first_user_message,
    MAX(metadata->>'cwd') as project
FROM recall_chunks
WHERE conversation_id IS NOT NULL
GROUP BY conversation_id, source
ORDER BY last_message DESC
LIMIT $1 OFFSET $2  -- limit defaults to 100 if input is 0
```

**Implementation constant**:
```go
const DefaultConversationLimit = 100
```

### Embedding Service Dependency

- Semantic search requires Ollama running with configured embedding model
- Browse/list does NOT require Ollama (pure SQL)
- Search endpoint should check embedder availability and return graceful error

### Template Files

```
internal/webplugins/recall/templates/
├── list.html          # Conversation list with pagination + search input
├── view.html          # Conversation detail with all messages
└── search-results.html # Semantic search results (separate from list)
```

### Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/recall` | `list` | Paginated conversation list |
| GET | `/recall/{id}` | `view` | Conversation detail |
| GET | `/recall/search` | `search` | Semantic search results |

### Dependencies

- `internal/recall` - Storage interface and types
- `internal/recall/postgres` - PostgreSQL implementation
- `internal/web` - Plugin registration, renderer
- `internal/search` - Searchable interface for global search
- `internal/config` - Ollama/PostgreSQL configuration

### Not Required (Out of Scope)

Per Linear ticket:
- Search logic implementation (use existing `recall.Storage.Search`)
- Data ingestion (handled by DEV-69)
- Keyword search (semantic only for P1)
- Export functionality
- Conversation editing/deletion

### Performance Considerations

- Conversation list pagination: Fetch only summary data, not full messages
- Detail view: Load all messages at once (typical conversation < 100 messages)
- Search: Embedding generation ~100-500ms, DB search ~50ms
- Consider adding index: `CREATE INDEX idx_recall_conversation_id ON recall_chunks(conversation_id)`

### Configuration

No new configuration needed. Uses existing:
- `BRAINS_POSTGRES_URL` - Database connection
- `BRAINS_OLLAMA_URL` - Embedding service
- `BRAINS_EMBEDDING_MODEL` - Embedding model name
