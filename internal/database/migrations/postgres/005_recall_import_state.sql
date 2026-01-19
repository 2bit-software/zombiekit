-- Import state tracking for incremental imports
-- Tracks last successfully imported entry per JSONL file to enable fast skipping

CREATE TABLE IF NOT EXISTS recall_import_state (
    file_path TEXT PRIMARY KEY,
    last_entry_uuid TEXT NOT NULL,
    file_mtime BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE recall_import_state IS 'Tracks per-file import sync state for incremental imports';
COMMENT ON COLUMN recall_import_state.file_path IS 'Absolute path to the JSONL file';
COMMENT ON COLUMN recall_import_state.last_entry_uuid IS 'UUID of last successfully imported entry';
COMMENT ON COLUMN recall_import_state.file_mtime IS 'Unix nanosecond timestamp of file modification time at last import';

-- History gap column for divergence tracking
-- When true, indicates this chunk was imported after detecting sync divergence
ALTER TABLE recall_chunks
    ADD COLUMN IF NOT EXISTS history_gap BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN recall_chunks.history_gap IS 'True if this chunk was imported after detecting sync divergence; history before this point may be missing or inconsistent';

-- Partial index for efficient querying of chunks with history gaps
CREATE INDEX IF NOT EXISTS idx_recall_chunks_history_gap
    ON recall_chunks(history_gap) WHERE history_gap = TRUE;
