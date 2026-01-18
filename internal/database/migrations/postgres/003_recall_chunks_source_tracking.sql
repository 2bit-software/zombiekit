-- Add source tracking columns for conversation import
ALTER TABLE recall_chunks
    ADD COLUMN IF NOT EXISTS source TEXT,
    ADD COLUMN IF NOT EXISTS source_id TEXT,
    ADD COLUMN IF NOT EXISTS conversation_id TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Unique index for duplicate detection: (source, source_id)
-- Allows same source_id from different sources
CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_source_id
    ON recall_chunks(source, source_id)
    WHERE source_id IS NOT NULL;

-- Index for conversation retrieval
CREATE INDEX IF NOT EXISTS idx_recall_chunks_conversation
    ON recall_chunks(conversation_id)
    WHERE conversation_id IS NOT NULL;

-- Index for source filtering
CREATE INDEX IF NOT EXISTS idx_recall_chunks_source
    ON recall_chunks(source)
    WHERE source IS NOT NULL;
