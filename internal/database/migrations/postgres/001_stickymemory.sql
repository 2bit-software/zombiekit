-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Main memories table (mcp-genie compatible schema)
CREATE TABLE IF NOT EXISTS memories (
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (name, version)
);

-- Index for finding latest version efficiently
CREATE INDEX IF NOT EXISTS idx_memories_name_latest
    ON memories (name, version DESC)
    WHERE deleted = FALSE;

-- Index for search (PostgreSQL full-text)
CREATE INDEX IF NOT EXISTS idx_memories_search
    ON memories USING gin(to_tsvector('english', name || ' ' || content))
    WHERE deleted = FALSE;
