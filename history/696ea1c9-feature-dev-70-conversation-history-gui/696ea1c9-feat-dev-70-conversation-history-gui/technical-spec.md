# Technical Specification: Conversation History Search GUI

**Feature**: DEV-70
**Created**: 2026-01-19
**Status**: Draft

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Web Layer                            │
├─────────────────────────────────────────────────────────────┤
│  internal/webplugins/recall/                                │
│  ├── plugin.go       (WebPlugin + Searchable interfaces)    │
│  ├── handlers.go     (HTTP handlers)                        │
│  └── templates/      (HTMX templates)                       │
├─────────────────────────────────────────────────────────────┤
│                      Storage Layer                          │
├─────────────────────────────────────────────────────────────┤
│  internal/recall/                                           │
│  ├── storage.go      (Storage interface)                    │
│  ├── types.go        (ConversationSummary, Chunk, etc.)     │
│  └── postgres/       (PostgreSQL implementation)            │
└─────────────────────────────────────────────────────────────┘
```

## Type Definitions

### New Type: ConversationSummary

```go
// internal/recall/types.go

// ConversationSummary contains aggregated metadata for a conversation.
type ConversationSummary struct {
    ConversationID string    `json:"conversation_id"`
    Title          string    `json:"title"`         // First user message (truncated) or "[No title]"
    MessageCount   int       `json:"message_count"`
    FirstMessage   time.Time `json:"first_message"`
    LastMessage    time.Time `json:"last_message"`
    Source         string    `json:"source"`        // e.g., "claude"
    Project        string    `json:"project"`       // CWD from metadata
}
```

### Extended Storage Interface

```go
// internal/recall/storage.go

type Storage interface {
    // ... existing methods ...

    // ListConversations returns conversations ordered by last activity (most recent first).
    // limit=0 uses implementation default (100), offset supports pagination.
    // project="" returns all conversations; non-empty filters by CWD prefix.
    ListConversations(ctx context.Context, limit, offset int, project string) ([]ConversationSummary, error)

    // ListDistinctProjects returns all unique project paths (CWD) from stored conversations.
    // Used to populate the project filter dropdown.
    ListDistinctProjects(ctx context.Context) ([]string, error)
}
```

## Database Queries

### ListConversations SQL

```sql
WITH first_user_msg AS (
    SELECT DISTINCT ON (conversation_id)
        conversation_id,
        SUBSTRING(content, 1, 100) as title
    FROM recall_chunks
    WHERE conversation_id IS NOT NULL
      AND metadata->>'role' = 'user'
    ORDER BY conversation_id, COALESCE((metadata->>'timestamp')::timestamptz, created_at) ASC
)
SELECT
    rc.conversation_id,
    COALESCE(fum.title, '[No title]') as title,
    COUNT(*) as message_count,
    MIN(COALESCE((rc.metadata->>'timestamp')::timestamptz, rc.created_at)) as first_message,
    MAX(COALESCE((rc.metadata->>'timestamp')::timestamptz, rc.created_at)) as last_message,
    rc.source,
    MAX(rc.metadata->>'cwd') as project
FROM recall_chunks rc
LEFT JOIN first_user_msg fum ON rc.conversation_id = fum.conversation_id
WHERE rc.conversation_id IS NOT NULL
  AND ($3 = '' OR rc.metadata->>'cwd' LIKE $3 || '%')  -- project filter
GROUP BY rc.conversation_id, rc.source, fum.title
HAVING COUNT(*) > 0  -- exclude empty conversations
ORDER BY last_message DESC
LIMIT $1 OFFSET $2;
```

**Parameters**:
- `$1`: limit (default 100 if 0)
- `$2`: offset (for pagination)
- `$3`: project prefix filter (empty string = no filter)

### ListDistinctProjects SQL

```sql
SELECT DISTINCT metadata->>'cwd' as project
FROM recall_chunks
WHERE metadata->>'cwd' IS NOT NULL
ORDER BY project;
```

### Performance Index

```sql
-- Already exists from DEV-67
CREATE INDEX idx_recall_chunks_conversation ON recall_chunks(conversation_id) WHERE conversation_id IS NOT NULL;

-- Consider adding for project filtering
CREATE INDEX idx_recall_chunks_cwd ON recall_chunks((metadata->>'cwd')) WHERE metadata->>'cwd' IS NOT NULL;
```

## Web Plugin Structure

### Directory Layout

```
internal/webplugins/recall/
├── plugin.go
├── handlers.go
├── templates/
│   ├── list.html
│   ├── view.html
│   └── search-results.html
└── plugin_test.go
```

### Plugin Definition

```go
// internal/webplugins/recall/plugin.go

package recall

import (
    "embed"
    "io/fs"

    "github.com/go-chi/chi/v5"
    "github.com/zombiekit/brains/internal/recall"
    "github.com/zombiekit/brains/internal/search"
    "github.com/zombiekit/brains/internal/web"
)

//go:embed templates
var templatesFS embed.FS

var (
    _ web.TemplatePlugin = (*Plugin)(nil)
    _ search.Searchable  = (*Plugin)(nil)
)

type Plugin struct {
    storage  recall.Storage
    embedder recall.Embedder
}

func NewPlugin(storage recall.Storage, embedder recall.Embedder) *Plugin {
    return &Plugin{storage: storage, embedder: embedder}
}

func (p *Plugin) SidebarItems() []web.SidebarItem {
    return []web.SidebarItem{
        {
            ID:    "conversations",
            Label: "Conversations",
            Path:  "/",
            Order: 30,  // After memory (20)
        },
    }
}

func (p *Plugin) MountRoutes(r chi.Router) {
    h := newHandlers(p.storage, p.embedder)

    r.Get("/", h.list)
    r.Get("/search", h.search)
    r.Get("/{id}", h.view)
}

func (p *Plugin) Templates() fs.FS {
    sub, _ := fs.Sub(templatesFS, "templates")
    return sub
}

func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
    // Implementation details in handlers section
}
```

### Handlers

```go
// internal/webplugins/recall/handlers.go

package recall

import (
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/zombiekit/brains/internal/recall"
    "github.com/zombiekit/brains/internal/web"
)

const (
    DefaultPageLimit = 20
    MaxPageLimit     = 100
)

var PageLimitOptions = []int{10, 20, 50, 100}

type handlers struct {
    storage  recall.Storage
    embedder recall.Embedder
}

func newHandlers(storage recall.Storage, embedder recall.Embedder) *handlers {
    return &handlers{storage: storage, embedder: embedder}
}

// Data types for templates

type ListData struct {
    Conversations []recall.ConversationSummary
    Pagination    PaginationData
    Project       string  // Current filter
    Projects      []string // Available projects for dropdown
    Error         string
}

type ViewData struct {
    ConversationID string
    Title          string
    Messages       []recall.Chunk
    MessageCount   int
    DateRange      string
    Error          string
}

type SearchData struct {
    Query   string
    Results []SearchResultGroup
    Error   string
}

type SearchResultGroup struct {
    ConversationID string
    Title          string
    Snippets       []string
    Similarity     float64   // Highest similarity score among snippets (0.0-1.0)
    LastMessage    time.Time
}

type PaginationData struct {
    CurrentPage  int
    TotalPages   int
    TotalItems   int
    Limit        int
    HasPrev      bool
    HasNext      bool
    PrevPage     int
    NextPage     int
    LimitOptions []int
}

// Handler implementations follow memory plugin patterns
```

### Route Configuration

| Method | Path | Handler | HTMX Target | Description |
|--------|------|---------|-------------|-------------|
| GET | `/recall` | `list` | `#content` | Paginated conversation list |
| GET | `/recall/search` | `search` | `#content` | Semantic search results |
| GET | `/recall/{id}` | `view` | `#content` | Conversation detail |

## Template Specifications

### list.html

```html
<div class="container">
    <!-- Search form -->
    <form hx-get="/recall/search" hx-target="#content" hx-push-url="true">
        <input type="text" name="q" placeholder="Search conversations...">
        <button type="submit">Search</button>
    </form>

    <!-- Project filter -->
    <select name="project" hx-get="/recall" hx-target="#content" hx-push-url="true">
        <option value="">All Projects</option>
        {{range .Content.Projects}}
        <option value="{{.}}" {{if eq . $.Content.Project}}selected{{end}}>{{.}}</option>
        {{end}}
    </select>

    {{if .Content.Error}}
    <div class="alert error">{{.Content.Error}}</div>
    {{else if not .Content.Conversations}}
    <div class="empty-state">No conversations yet.</div>
    {{else}}
    <ul class="conversation-list">
        {{range .Content.Conversations}}
        <li>
            <a href="/recall/{{.ConversationID}}" hx-get="/recall/{{.ConversationID}}"
               hx-target="#content" hx-push-url="true">
                <h3>{{.Title}}</h3>
                <span class="badge">{{.Source}}</span>
                <span class="meta">{{.MessageCount}} messages</span>
                <span class="meta">{{formatTime .LastMessage}}</span>
            </a>
        </li>
        {{end}}
    </ul>

    <!-- Pagination -->
    {{template "pagination" .Content.Pagination}}
    {{end}}
</div>
```

### view.html

```html
<div class="container">
    <a href="/recall" hx-get="/recall" hx-target="#content" hx-push-url="true">
        &larr; Back to list
    </a>

    {{if .Content.Error}}
    <div class="alert error">{{.Content.Error}}</div>
    {{else}}
    <header>
        <h1>{{.Content.Title}}</h1>
        <span class="meta">{{.Content.MessageCount}} messages | {{.Content.DateRange}}</span>
    </header>

    <div class="messages">
        {{range .Content.Messages}}
        <div class="message {{.Metadata.Role}}">
            <div class="message-header">
                <span class="role">{{.Metadata.Role}}</span>
                <span class="timestamp">{{formatTime .Metadata.Timestamp}}</span>
            </div>
            <div class="message-content">
                <pre>{{.Content}}</pre>
            </div>
        </div>
        {{end}}
    </div>
    {{end}}
</div>
```

### search-results.html

```html
<div class="container">
    <a href="/recall" hx-get="/recall" hx-target="#content" hx-push-url="true">
        &larr; Back to list
    </a>

    <h2>Search: "{{.Content.Query}}"</h2>

    {{if .Content.Error}}
    <div class="alert error">{{.Content.Error}}</div>
    {{else if not .Content.Results}}
    <div class="empty-state">No matching conversations found.</div>
    {{else}}
    <ul class="search-results">
        {{range .Content.Results}}
        <li>
            <a href="/recall/{{.ConversationID}}" hx-get="/recall/{{.ConversationID}}"
               hx-target="#content" hx-push-url="true">
                <h3>{{.Title}}</h3>
                <div class="snippets">
                    {{range .Snippets}}
                    <p class="snippet">...{{.}}...</p>
                    {{end}}
                </div>
                <span class="meta">{{formatTime .LastMessage}}</span>
            </a>
        </li>
        {{end}}
    </ul>
    {{end}}
</div>
```

## Plugin Registration

```go
// internal/cli/gui.go (additions)

// After existing plugin registrations:

// Register recall plugin if storage and embedder available
if recallStorage != nil {
    embedderConfig := config.EmbedderConfig{
        OllamaURL:      cfg.OllamaURL,
        EmbeddingModel: cfg.EmbeddingModel,
    }
    embedder, err := ollama.NewEmbedder(embedderConfig)
    if err != nil {
        logger.Warn("recall embedder unavailable, search disabled", "error", err)
        embedder = nil // Plugin will handle nil embedder gracefully
    }

    recallPlugin := recallweb.NewPlugin(recallStorage, embedder)
    registry.Register("recall", recallPlugin)
}
```

## Error Handling

### Embedder Unavailable (FR-007)

```go
func (h *handlers) search(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")

    // Check embedder availability
    if h.embedder == nil {
        data := SearchData{
            Query: query,
            Error: "Search unavailable - embedding service offline. Browse conversations instead.",
        }
        renderer.Render(w, r, "recall/search-results.html", data)
        return
    }

    // ... proceed with search
}
```

### Invalid Conversation ID

```go
func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    chunks, err := h.storage.GetByConversation(r.Context(), id)
    if err != nil {
        data := ViewData{Error: "Failed to load conversation"}
        renderer.Render(w, r, "recall/view.html", data)
        return
    }

    if len(chunks) == 0 {
        data := ViewData{Error: "Conversation not found"}
        w.WriteHeader(http.StatusNotFound)
        renderer.Render(w, r, "recall/view.html", data)
        return
    }

    // ... render conversation
}
```

## Searchable Interface Implementation

```go
func (p *Plugin) Search(query string, maxResults int, sortOrder search.SortOrder) ([]search.SearchResult, error) {
    if query == "" {
        return []search.SearchResult{}, nil
    }

    // Check embedder availability
    if p.embedder == nil {
        return []search.SearchResult{}, nil // Silent failure for global search
    }

    ctx := context.Background()

    // Generate embedding
    embedding, err := p.embedder.Embed(ctx, query)
    if err != nil {
        return []search.SearchResult{}, nil // Silent failure
    }

    // Search
    limit := maxResults
    if limit <= 0 {
        limit = 10
    }
    results, err := p.storage.Search(ctx, embedding, limit)
    if err != nil {
        return []search.SearchResult{}, nil
    }

    // Convert to SearchResult, grouping by conversation
    seen := make(map[string]bool)
    var searchResults []search.SearchResult

    for _, r := range results {
        if r.Chunk.ConversationID == "" || seen[r.Chunk.ConversationID] {
            continue
        }
        seen[r.Chunk.ConversationID] = true

        // Truncate content for title
        title := r.Chunk.Content
        if len(title) > 100 {
            title = title[:100] + "..."
        }

        // URL is relative to plugin root - framework auto-prefixes with "/recall"
        searchResults = append(searchResults, search.SearchResult{
            Title: title,
            URL:   "/" + r.Chunk.ConversationID,
        })
    }

    return searchResults, nil
}
```

## Testing Strategy

### Storage Layer Tests

```go
// internal/recall/postgres/storage_test.go

func TestListConversations(t *testing.T) {
    // Setup test database with multiple conversations

    tests := []struct {
        name     string
        limit    int
        offset   int
        project  string
        wantLen  int
    }{
        {"default limit", 0, 0, "", 100},  // or actual count
        {"custom limit", 5, 0, "", 5},
        {"with offset", 5, 5, "", 5},
        {"filter by project", 10, 0, "/home/user/project", 2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            results, err := storage.ListConversations(ctx, tt.limit, tt.offset, tt.project)
            require.NoError(t, err)
            assert.Len(t, results, tt.wantLen)
        })
    }
}
```

### Handler Tests

```go
// internal/webplugins/recall/handlers_test.go

func TestListHandler(t *testing.T) {
    storage := &mockStorage{...}
    plugin := NewPlugin(storage, nil)

    req := httptest.NewRequest("GET", "/recall", nil)
    w := httptest.NewRecorder()

    // ... setup router and call handler

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "Conversations")
}
```

## Configuration

No new configuration required. Uses existing:

| Env Var | Usage |
|---------|-------|
| `BRAINS_POSTGRES_URL` | Database connection |
| `BRAINS_OLLAMA_URL` | Embedding service |
| `BRAINS_EMBEDDING_MODEL` | Model for embeddings |

## Performance Considerations

1. **Conversation list**: O(n) where n = conversations, but database does aggregation
2. **Detail view**: Single query, typically < 100 messages
3. **Search**: ~100-500ms for embedding + ~50ms for DB search
4. **Index on conversation_id**: Already exists

## Open Questions

1. **Conversation title derivation**: First user message vs CWD vs custom? → Decision: First user message (spec says derived from first user message)

2. **Search result grouping**: Show individual messages or group by conversation? → Decision: Group by conversation with snippets

3. **Project filter UI**: Dropdown vs autocomplete vs text input? → Decision: Dropdown (simpler, works for reasonable project counts)
