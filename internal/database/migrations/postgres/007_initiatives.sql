-- Initiatives table for workflow state
CREATE TABLE IF NOT EXISTS initiatives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'feature',
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    description TEXT,
    branch_name VARCHAR(255),
    steps JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_initiatives_status ON initiatives(status);
CREATE INDEX IF NOT EXISTS idx_initiatives_created_at ON initiatives(created_at DESC);
