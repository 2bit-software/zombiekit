# Implementation Plan: Semantic Search Foundation

**Feature**: semantic-search-foundation
**Linear Ticket**: DEV-72
**Created**: 2026-01-17

---

## Overview

Build the foundational RAG infrastructure enabling ZombieKit operators to store arbitrary text and retrieve it via semantic similarity. This unblocks DEV-69 (Claude Conversation Importer).

## Dependencies

```
github.com/ollama/ollama/api     - Official Ollama Go client
github.com/pgvector/pgvector-go  - pgvector types for Go
```

Both are well-maintained, widely used, and appropriate for this use case.

---

## Phase 1: Database Schema & Infrastructure

### Step 1.1: Create pgvector Migration

**File**: `/internal/database/migrations/postgres/002_recall_chunks.sql`

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE recall_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    embedding vector(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_recall_chunks_content_hash ON recall_chunks(content_hash);
CREATE INDEX idx_recall_chunks_embedding ON recall_chunks USING hnsw (embedding vector_cosine_ops);
```

**Rationale**:
- `vector(768)` matches nomic-embed-text dimensions
- `content_hash` enables O(1) duplicate detection (SHA-256 of content)
- HNSW index for fast approximate nearest neighbor search
- Cosine distance (`<=>`) is standard for semantic similarity

**Note**: No SQLite migration. This feature is PostgreSQL-only due to pgvector dependency.

### Step 1.2: Update go.mod

Add dependencies:
```
github.com/ollama/ollama/api
github.com/pgvector/pgvector-go
```

---

## Phase 2: Core Package Structure

### Step 2.1: Create recall Package

**Directory**: `/internal/recall/`

**Files**:
- `types.go` - Chunk type, search result type
- `storage.go` - Storage interface
- `postgres/storage.go` - PostgreSQL implementation
- `embedder.go` - Ollama embedding client wrapper

### Step 2.2: Types Definition

**File**: `/internal/recall/types.go`

```go
type Chunk struct {
    ID        string
    Content   string
    CreatedAt time.Time
}

type SearchResult struct {
    Chunk      Chunk
    Similarity float64  // 0.0 to 1.0, higher = more similar
}
```

### Step 2.3: Storage Interface

**File**: `/internal/recall/storage.go`

```go
type Storage interface {
    // Ingest stores content if not duplicate, returns (id, created, error)
    // created=false means duplicate was detected
    Ingest(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)

    // List returns all chunks ordered by created_at desc
    List(ctx context.Context, limit int) ([]Chunk, error)

    // Search returns chunks ranked by cosine similarity to query embedding
    Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)

    Close() error
}
```

### Step 2.4: Embedder Interface

**File**: `/internal/recall/embedder.go`

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

type OllamaEmbedder struct {
    client *api.Client
    model  string
}
```

Uses `search_document:` prefix for ingestion, `search_query:` prefix for search.

---

## Phase 3: PostgreSQL Storage Implementation

### Step 3.1: Implement Storage

**File**: `/internal/recall/postgres/storage.go`

Key implementation details:
- Use pgxpool for connection pooling
- Register pgvector types via `pgxvec.RegisterTypes`
- Content hash: `SHA-256(content)` hex-encoded
- Duplicate detection: INSERT with ON CONFLICT DO NOTHING, check rows affected

### Step 3.2: Vector Registration

Must call `pgxvec.RegisterTypes` on pool AfterConnect to enable vector type marshaling.

---

## Phase 4: CLI Commands

### Step 4.1: Create recall Command

**File**: `/internal/cli/recall.go`

```go
func newRecallCommand() *cli.Command {
    return &cli.Command{
        Name:  "recall",
        Usage: "Semantic memory storage and retrieval",
        Subcommands: []*cli.Command{
            newRecallSaveCommand(),
            newRecallListCommand(),
            newRecallSearchCommand(),
        },
    }
}
```

### Step 4.2: Save Command

**Usage**: `brains recall save <text>` or `brains recall save -` (stdin)

**Behavior**:
1. Read content from arg or stdin
2. Generate embedding via Ollama
3. Store in database with content hash
4. Print confirmation: "Stored: <truncated content> (<id>)"
5. Duplicates: silent no-op (no output, no error)

### Step 4.3: List Command

**Usage**: `brains recall list [--limit N]`

**Behavior**:
1. Query all chunks, ordered by created_at DESC
2. Print table: ID | Created | Content (truncated)
3. If empty: "No content stored yet"

### Step 4.4: Search Command

**Usage**: `brains recall search <query> [--limit N]`

**Behavior**:
1. Generate embedding for query
2. Search by cosine similarity
3. Print results with similarity scores AND timestamps (per BR-007)
4. If no results: "No matching content found"

**Output format**: `SIMILARITY | CREATED | ID | CONTENT`

### Step 4.5: Register Command

**File**: `/internal/cli/root.go`

Add `newRecallCommand()` to the Commands slice.

---

## Phase 5: Configuration

### Step 5.1: Add Config Fields

**File**: `/internal/config/storage.go`

Add to `FileStorageConfig`:
```go
OllamaURL       string `toml:"ollama_url"`
EmbeddingModel  string `toml:"embedding_model"`
```

Add to `StorageConfig`:
```go
OllamaURL       string
EmbeddingModel  string
```

### Step 5.2: Environment Variables

**File**: `/internal/config/loader.go`

Support:
- `BRAINS_OLLAMA_URL` (default: `http://localhost:11434`)
- `BRAINS_EMBEDDING_MODEL` (default: `nomic-embed-text`)

---

## Phase 6: Error Handling

### Step 6.1: Ollama Connectivity

All commands must check Ollama availability before proceeding. If unavailable:
```
Error: Cannot connect to Ollama at http://localhost:11434
Make sure Ollama is running: ollama serve
```

### Step 6.2: Database Connectivity

If PostgreSQL unavailable:
```
Error: Cannot connect to PostgreSQL database
Check your database configuration and ensure PostgreSQL is running
```

---

## Phase 7: Testing

### Step 7.1: Unit Tests

**File**: `/internal/recall/postgres/storage_test.go`

- Test duplicate detection (same content → no new row)
- Test list ordering (newest first)
- Test search ranking (more similar → higher score)

### Step 7.2: Integration Test

**File**: `/internal/recall/integration_test.go`

Round-trip test:
1. Ingest "The deployment failed because of memory limits"
2. Search "out of memory errors"
3. Assert result contains ingested content

Requires running PostgreSQL and Ollama.

---

## Implementation Order

| Step | Description | Blocking | Est. Complexity |
|------|-------------|----------|-----------------|
| 1.1 | Database migration | Nothing | Low |
| 1.2 | Add dependencies | Nothing | Low |
| 2.1-2.4 | Core types and interfaces | Step 1.2 | Low |
| 3.1-3.2 | PostgreSQL storage | Step 2.x | Medium |
| 5.1-5.2 | Configuration | Nothing | Low |
| 4.1-4.5 | CLI commands | Steps 3.x, 5.x | Medium |
| 6.1-6.2 | Error handling | Step 4.x | Low |
| 7.1-7.2 | Tests | Steps 4.x | Medium |

**Critical path**: Migration → Dependencies → Types → Storage → Config → CLI → Tests

---

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Embedding model | nomic-embed-text | 768 dims, 8K context, best general-purpose |
| Vector dimension | 768 (fixed) | Match nomic-embed-text, simplifies schema |
| Duplicate detection | Content hash (SHA-256) | O(1) lookup, exact match semantics |
| Index type | HNSW | Faster queries than IVFFlat, good for small-medium datasets |
| Distance metric | Cosine | Standard for semantic similarity |
| PostgreSQL only | Yes | pgvector required, no SQLite equivalent |
| Task prefixes | search_document/search_query | nomic-embed-text best practice |

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Ollama not running | Clear error message with fix instructions |
| Wrong embedding model dimensions | Startup validation: test embed call, verify vector length = 768 |
| Large content | No limit per spec, but consider chunking in future (DEV-69 will need it) |
| Slow embedding generation | Batch support exists in Ollama API, use for bulk imports later |

### Step 6.3: Embedding Dimension Validation

On first Ollama operation, validate the configured model produces 768-dimensional vectors:

```go
testEmb, err := embedder.Embed(ctx, "test", PurposeDocument)
if len(testEmb) != 768 {
    return fmt.Errorf("embedding model %q produces %d dimensions, expected 768 (nomic-embed-text)", model, len(testEmb))
}
```

---

## Out of Scope (Confirmed)

- SQLite support (no pgvector equivalent)
- Content chunking (DEV-69 concern)
- Delete/update operations (per spec)
- MCP tool integration (future work)
- Web UI (DEV-71)
