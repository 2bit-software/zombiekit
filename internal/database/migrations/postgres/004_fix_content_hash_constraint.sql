-- Migration 004: Fix content_hash constraint for sourced imports
--
-- Problem: idx_recall_chunks_content_hash is globally unique, preventing
-- different conversations from having identical content.
--
-- Solution: Make content_hash unique only for non-sourced content.
-- Sourced content (with source_id) can have duplicate content across
-- different conversations.

-- Drop the global unique constraint
DROP INDEX IF EXISTS idx_recall_chunks_content_hash;

-- Recreate as partial index: unique only when source_id IS NULL
-- This preserves deduplication for Save() calls while allowing
-- SaveWithSource() to have duplicate content across different sources
CREATE UNIQUE INDEX idx_recall_chunks_content_hash
    ON recall_chunks(content_hash)
    WHERE source_id IS NULL;
