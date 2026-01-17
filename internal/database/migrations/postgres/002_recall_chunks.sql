-- pgvector extension for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Recall chunks table for semantic search
CREATE TABLE IF NOT EXISTS recall_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    embedding vector(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique index for duplicate detection via content hash
CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_content_hash
    ON recall_chunks(content_hash);

-- HNSW index for fast approximate nearest neighbor search using cosine distance
CREATE INDEX IF NOT EXISTS idx_recall_chunks_embedding
    ON recall_chunks USING hnsw (embedding vector_cosine_ops);
