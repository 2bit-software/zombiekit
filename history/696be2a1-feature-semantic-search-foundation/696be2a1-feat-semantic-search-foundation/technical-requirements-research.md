# Technical Requirements Research

**Feature**: Semantic Search Foundation
**Linear Ticket**: DEV-72
**Created**: 2026-01-17

---

## Technical Stack (from ticket)

- PostgreSQL + pgvector
- Ollama (local embedding model)
- Cobra CLI framework (existing in codebase)

## Implementation Hints

### 1. Database Setup

- PostgreSQL with pgvector extension
- Schema for storing text chunks + embeddings
- Docker Compose configuration already exists with `pgvector/pgvector:pg16` on port 9432

### 2. Embedding Generation

- Ollama integration for vector generation
- Configurable model selection
- **Assumption**: Ollama is running locally, managed by user

### 3. CLI Commands (`brains recall`)

- `brains recall ingest` - Accept text input, generate embeddings, store
- `brains recall list` - Show stored entries
- `brains recall search <query>` - Find entries via semantic similarity

### 4. Validation

- Round-trip test: ingest text → search with similar/exact text → find it

## Current State (from audit)

### EXISTS

| Requirement | Status | Location |
|-------------|--------|----------|
| PostgreSQL with pgvector | ✅ | `docker-compose.yml` - using `pgvector/pgvector:pg16` on port 9432 |
| Docker Compose configuration | ✅ | Root `docker-compose.yml` |
| Migration infrastructure | ✅ | `/internal/database/migrations.go` + embedded SQL |
| CLI framework | ✅ | Cobra-based CLI in `/internal/cli/` |
| Full-text search | ✅ | `brains memory search` command exists |

### MISSING (Core Deliverables)

| Requirement | Status | Notes |
|-------------|--------|-------|
| **Embeddings table schema** | ❌ | No pgvector column in any migration |
| **pgvector extension creation** | ❌ | Not in SQL migrations |
| **Ollama integration** | ❌ | No Go client, no docker-compose service |
| **Configurable model selection** | ❌ | Nothing embedding-related |
| **`brains recall ingest`** | ❌ | No `recall` subcommand exists |
| **`brains recall list`** | ❌ | No vector-stored entries listing |
| **`brains recall search`** | ❌ | No semantic similarity search |
| **Round-trip validation test** | ❌ | No embedding/search integration tests |

## Suggested Schema

```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE chunks (
    id UUID PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(384),  -- or model-appropriate dimension
    metadata JSONB,
    created_at TIMESTAMPTZ
);
CREATE INDEX ON chunks USING ivfflat (embedding vector_cosine_ops);
```

## Implementation Risks

- **Embedding model selection**: Different Ollama models produce different quality results; no guidance in spec on which to use or how to evaluate
- **Duplicate detection granularity**: "Same content" could mean exact match, near-match, or semantic equivalence—implementation will need to pick an approach
- **pgvector configuration**: Vector dimension must match embedding model; mismatch breaks everything silently
- **Ollama availability**: If Ollama service isn't running, all operations fail—need clear error messaging

## Suggested Tests

### Acceptance
- Round-trip: ingest text → search with exact text → find it
- Round-trip: ingest text → search with semantically similar query → find it
- Duplicate handling: ingest same text twice → only one entry exists
- Empty state: search with no content → appropriate message
- List with timestamps: ingest content → list shows entry with correct timestamp

### Unit
- Embedding generation produces consistent vectors for identical input
- Similarity search returns results in descending relevance order
- Duplicate detection correctly identifies matching content

### Manual
- Verify search quality with varied semantic queries (subjective evaluation)
- Confirm Ollama integration works with at least one embedding model
