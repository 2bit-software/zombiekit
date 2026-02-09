-- Profiles table for server-authoritative profile storage
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    domains TEXT[] DEFAULT '{}',
    dependencies TEXT[] DEFAULT '{}',
    location VARCHAR(50) NOT NULL DEFAULT 'global',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles(name);
CREATE INDEX IF NOT EXISTS idx_profiles_location ON profiles(location);
