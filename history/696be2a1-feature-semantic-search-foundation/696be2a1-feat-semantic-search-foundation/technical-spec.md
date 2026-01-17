# Technical Specification: Semantic Search Foundation

**Feature**: semantic-search-foundation
**Linear Ticket**: DEV-72
**Created**: 2026-01-17

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                           CLI Layer                             │
│  brains recall ingest | brains recall list | brains recall search│
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Service Layer                            │
│                     /internal/recall/                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   types.go  │  │ embedder.go │  │      storage.go         │  │
│  │   Chunk     │  │  Embedder   │  │ Storage interface       │  │
│  │ SearchResult│  │ (interface) │  └───────────┬─────────────┘  │
│  └─────────────┘  └──────┬──────┘              │                │
│                          │                     │                │
│                          ▼                     ▼                │
│              ┌───────────────────┐  ┌─────────────────────────┐ │
│              │  OllamaEmbedder   │  │ postgres/storage.go     │ │
│              │  (implementation) │  │ (implementation)        │ │
│              └─────────┬─────────┘  └───────────┬─────────────┘ │
└────────────────────────┼────────────────────────┼───────────────┘
                         │                        │
                         ▼                        ▼
              ┌──────────────────┐     ┌──────────────────────────┐
              │     Ollama       │     │   PostgreSQL + pgvector  │
              │ localhost:11434  │     │   localhost:9432         │
              └──────────────────┘     └──────────────────────────┘
```

---

## Database Schema

### Table: `recall_chunks`

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE recall_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    embedding vector(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Duplicate detection (exact content match)
CREATE UNIQUE INDEX idx_recall_chunks_content_hash ON recall_chunks(content_hash);

-- Approximate nearest neighbor search
CREATE INDEX idx_recall_chunks_embedding ON recall_chunks USING hnsw (embedding vector_cosine_ops);
```

### Column Details

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PK, auto-generated | `gen_random_uuid()` |
| content | TEXT | NOT NULL | Original text, no size limit |
| content_hash | TEXT | NOT NULL, UNIQUE | SHA-256 hex of content |
| embedding | vector(768) | nullable | nomic-embed-text dimensions |
| created_at | TIMESTAMPTZ | NOT NULL, default NOW() | BR-007 timestamp requirement |

---

## Package Structure

```
/internal/recall/
├── types.go           # Chunk, SearchResult types
├── embedder.go        # Embedder interface + OllamaEmbedder
├── storage.go         # Storage interface
├── hash.go            # ContentHash function
└── postgres/
    ├── storage.go     # PostgreSQL Storage implementation
    └── storage_test.go
```

---

## Type Definitions

### types.go

```go
package recall

import "time"

// Chunk represents a stored piece of content
type Chunk struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}

// SearchResult wraps a chunk with its similarity score
type SearchResult struct {
    Chunk      Chunk   `json:"chunk"`
    Similarity float64 `json:"similarity"` // 0.0 to 1.0
}
```

---

## Interfaces

### storage.go

```go
package recall

import "context"

// Storage defines the contract for recall chunk persistence
type Storage interface {
    // Ingest stores content with its embedding.
    // Returns (id, created, error) where created=false indicates duplicate.
    Ingest(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)

    // List returns all chunks ordered by created_at DESC.
    List(ctx context.Context, limit int) ([]Chunk, error)

    // Search finds chunks by cosine similarity to the query embedding.
    Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)

    // Close releases any resources held by the storage.
    Close() error
}
```

### embedder.go

```go
package recall

import "context"

// Embedder generates vector embeddings for text
type Embedder interface {
    // Embed returns the embedding vector for the given text.
    // Implementations should handle any necessary prefixes.
    Embed(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error)
}

// EmbedPurpose indicates how the text will be used
type EmbedPurpose int

const (
    PurposeDocument EmbedPurpose = iota // For content being stored
    PurposeQuery                         // For search queries
)
```

---

## OllamaEmbedder Implementation

```go
package recall

import (
    "context"
    "fmt"

    "github.com/ollama/ollama/api"
)

type OllamaEmbedder struct {
    client *api.Client
    model  string
}

func NewOllamaEmbedder(ollamaURL, model string) (*OllamaEmbedder, error) {
    // Parse URL and create client
    client := api.NewClient(...)
    return &OllamaEmbedder{client: client, model: model}, nil
}

func (e *OllamaEmbedder) Embed(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error) {
    // Apply task prefix per nomic-embed-text best practices
    var prefixed string
    switch purpose {
    case PurposeDocument:
        prefixed = "search_document: " + text
    case PurposeQuery:
        prefixed = "search_query: " + text
    }

    resp, err := e.client.Embed(ctx, &api.EmbedRequest{
        Model: e.model,
        Input: prefixed,
    })
    if err != nil {
        return nil, fmt.Errorf("ollama embed: %w", err)
    }

    if len(resp.Embeddings) == 0 {
        return nil, fmt.Errorf("ollama returned no embeddings")
    }

    return resp.Embeddings[0], nil
}
```

---

## PostgreSQL Storage Implementation

### Key Methods

**Ingest**:
```go
func (s *Storage) Ingest(ctx context.Context, content string, embedding []float32) (string, bool, error) {
    hash := ContentHash(content)
    id := uuid.New().String()

    result, err := s.pool.Exec(ctx, `
        INSERT INTO recall_chunks (id, content, content_hash, embedding)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (content_hash) DO NOTHING
    `, id, content, hash, pgvector.NewVector(embedding))
    if err != nil {
        return "", false, err
    }

    // RowsAffected() == 0 means duplicate
    created := result.RowsAffected() > 0
    if !created {
        // Return existing ID for duplicate
        var existingID string
        s.pool.QueryRow(ctx,
            "SELECT id FROM recall_chunks WHERE content_hash = $1", hash,
        ).Scan(&existingID)
        return existingID, false, nil
    }
    return id, true, nil
}
```

**List**:
```go
func (s *Storage) List(ctx context.Context, limit int) ([]Chunk, error) {
    rows, err := s.pool.Query(ctx, `
        SELECT id, content, created_at
        FROM recall_chunks
        ORDER BY created_at DESC
        LIMIT $1
    `, limit)
    // ... scan rows into []Chunk
}
```

**Search**:
```go
func (s *Storage) Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error) {
    rows, err := s.pool.Query(ctx, `
        SELECT id, content, created_at, 1 - (embedding <=> $1) AS similarity
        FROM recall_chunks
        WHERE embedding IS NOT NULL
        ORDER BY embedding <=> $1
        LIMIT $2
    `, pgvector.NewVector(embedding), limit)
    // ... scan rows into []SearchResult
}
```

---

## CLI Commands

### recall.go

```go
func newRecallCommand() *cli.Command {
    return &cli.Command{
        Name:  "recall",
        Usage: "Semantic memory storage and retrieval",
        Subcommands: []*cli.Command{
            {
                Name:      "save",
                Usage:     "Store text content for semantic search",
                ArgsUsage: "<text> or - for stdin",
                Action:    recallSaveAction,
            },
            {
                Name:   "list",
                Usage:  "List all stored content",
                Flags:  []cli.Flag{&cli.IntFlag{Name: "limit", Value: 20}},
                Action: recallListAction,
            },
            {
                Name:      "search",
                Usage:     "Search stored content by meaning",
                ArgsUsage: "<query>",
                Flags:     []cli.Flag{&cli.IntFlag{Name: "limit", Value: 5}},
                Action:    recallSearchAction,
            },
        },
    }
}
```

### Command Behavior

**save**:
```
$ brains recall save "The deployment failed due to memory limits"
Stored: The deployment failed due to memory limits (a1b2c3d4)

$ brains recall save "The deployment failed due to memory limits"
(no output - silent duplicate)
```

**list**:
```
$ brains recall list
ID        CREATED              CONTENT
a1b2c3d4  2026-01-17 10:30:00  The deployment failed due to memory...
b2c3d4e5  2026-01-17 10:29:00  Fixed the login page CSS issue...

$ brains recall list
No content stored yet.
```

**search**:
```
$ brains recall search "out of memory errors"
SIMILARITY  CREATED              ID        CONTENT
0.8234      2026-01-17 10:30:00  a1b2c3d4  The deployment failed due to memory...
0.4521      2026-01-17 10:28:00  c3d4e5f6  Memory allocation patterns in Go...

$ brains recall search "quantum physics"
No matching content found.
```

---

## Configuration

### config.toml

```toml
[storage]
backend = "postgres"
postgres_url = "postgres://brains:brains_dev@localhost:9432/brains"

[recall]
ollama_url = "http://localhost:11434"
embedding_model = "nomic-embed-text"
```

### Environment Variables

| Variable | Default | Notes |
|----------|---------|-------|
| BRAINS_OLLAMA_URL | http://localhost:11434 | Ollama API endpoint |
| BRAINS_EMBEDDING_MODEL | nomic-embed-text | Must produce 768-dim vectors |

---

## Error Handling

### Ollama Errors

```go
func checkOllamaAvailable(ctx context.Context, client *api.Client) error {
    // Try a simple list models call
    _, err := client.List(ctx)
    if err != nil {
        return fmt.Errorf("cannot connect to Ollama at %s\nMake sure Ollama is running: ollama serve", url)
    }
    return nil
}
```

### Database Errors

```go
func checkDatabaseAvailable(ctx context.Context, pool *pgxpool.Pool) error {
    if err := pool.Ping(ctx); err != nil {
        return fmt.Errorf("cannot connect to PostgreSQL database\nCheck your configuration and ensure PostgreSQL is running")
    }
    return nil
}
```

---

## Content Hashing

### hash.go

```go
package recall

import (
    "crypto/sha256"
    "encoding/hex"
)

// ContentHash returns a SHA-256 hash of the content for duplicate detection
func ContentHash(content string) string {
    hash := sha256.Sum256([]byte(content))
    return hex.EncodeToString(hash[:])
}
```

---

## Requirements Traceability

| Requirement | Implementation |
|-------------|----------------|
| BR-001: Add arbitrary text | `brains recall save` command |
| BR-002: Natural language search | `brains recall search` with cosine similarity |
| BR-003: View all content | `brains recall list` command |
| BR-004: Confirm storage | "Stored: ..." output message |
| BR-005: Show relevance | Similarity score in search results |
| BR-006: Local operation | Ollama + PostgreSQL, both local |
| BR-007: Timestamps | `created_at` column, shown in list/search |
| BR-008: Silent duplicate handling | ON CONFLICT DO NOTHING + "Already exists" message |

---

## Testing Strategy

### Unit Tests

1. **ContentHash**: Same input → same hash, different input → different hash
2. **OllamaEmbedder**: Correct prefixes applied for document vs query
3. **Storage.Ingest**: Duplicate detection works (same content → created=false)
4. **Storage.Search**: Results ordered by similarity DESC

### Integration Tests

```go
func TestRoundTrip(t *testing.T) {
    // Requires running PostgreSQL and Ollama
    storage := setupTestStorage(t)
    embedder := setupTestEmbedder(t)

    content := "The deployment failed because of memory limits"

    // Ingest
    emb, _ := embedder.Embed(ctx, content, PurposeDocument)
    id, created, _ := storage.Ingest(ctx, content, emb)
    assert.True(t, created)

    // Search with semantically similar query
    queryEmb, _ := embedder.Embed(ctx, "out of memory errors", PurposeQuery)
    results, _ := storage.Search(ctx, queryEmb, 5)

    assert.Len(t, results, 1)
    assert.Equal(t, id, results[0].Chunk.ID)
    assert.Greater(t, results[0].Similarity, 0.5)
}
```

---

## Future Considerations (Out of Scope)

- **Chunking**: DEV-69 will need to split large conversations. Consider `parent_id` or `source_id` column later.
- **Metadata**: JSONB column could store source info, tags. Omitted for now per YAGNI.
- **Batch ingestion**: Ollama Embed API supports batching. Optimize when needed.
- **MCP tool**: Expose recall as MCP tool for Claude integration.
- **Web UI**: DEV-71 will add browser interface.
