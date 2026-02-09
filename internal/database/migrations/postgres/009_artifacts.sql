-- Artifacts table for initiative-scoped file storage
CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiative_id UUID NOT NULL,
    path VARCHAR(1024) NOT NULL,
    content BYTEA NOT NULL,
    content_type VARCHAR(255) NOT NULL DEFAULT 'text/plain',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(initiative_id, path)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_initiative_id ON artifacts(initiative_id);
