# Data Model: MCP Tools

**Date**: 2025-12-21
**Branch**: `002-mcp-tools`
**Compatibility**: mcp-genie (`telegraph/ai/tools/mcp-genie`)

## Overview

This document defines the data model for the sticky memory and code reasoning tools. Only the Memory entity is persisted to PostgreSQL/SQLite; the Thought/Session entities are in-memory only.

**IMPORTANT**: The schema uses an **append-only versioning model** compatible with mcp-genie. Each `Set` operation creates a new row with an incremented version number. The `(name, version)` pair forms the composite primary key.

---

## Entities

### Memory (Persisted)

Represents a named piece of text content with versioning support.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `name` | TEXT | NOT NULL, PK (composite) | User-facing identifier (sanitized) |
| `version` | INTEGER | NOT NULL, PK (composite) | Version number (1, 2, 3...) |
| `content` | TEXT | NOT NULL | Memory content (max 1MB enforced in app) |
| `deleted` | BOOLEAN | NOT NULL, DEFAULT FALSE | Soft-delete flag |
| `created_at` | TIMESTAMP | NOT NULL | Creation timestamp |
| `updated_at` | TIMESTAMP | NOT NULL | Last update timestamp |

**Validation Rules**:
- `name`: Must match pattern `^[a-zA-Z0-9._-]+$` (sanitized before storage)
- `name`: Max length 255 characters
- `content`: Max size 1MB (1,048,576 bytes)

**Version Semantics** (mcp-genie compatible):
- Each `Set` creates a NEW row with version = max(existing versions) + 1
- `Get` returns the latest non-deleted version
- `Delete` soft-deletes ALL versions of a name
- `List` returns latest non-deleted version for each unique name

**State Transitions**:
- Created → Active (via `set` operation, version=1)
- Active → Active (via `set` operation, new row with version=N+1)
- Active → Deleted (via `delete` operation, `deleted=true` on ALL versions)

**Indexes**:
- PRIMARY KEY on `(name, version)`
- INDEX on `(name, version DESC) WHERE deleted = FALSE` for latest lookup
- INDEX for search performance

### Go Types (mcp-genie compatible)

```go
// memory/types.go

// MemoryItem represents a single memory entry
type MemoryItem struct {
    Name      string    `json:"name" db:"name"`
    Content   string    `json:"content" db:"content"`
    Version   int       `json:"version" db:"version"`
    Deleted   bool      `json:"deleted" db:"deleted"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MemoryMetadata contains metadata about a memory item (for list operations)
type MemoryMetadata struct {
    Name      string    `json:"name" db:"name"`
    Size      int       `json:"size" db:"-"`
    Version   int       `json:"version" db:"version"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
```

---

### Thought (In-Memory Only)

Represents a single step in a reasoning chain. Not persisted.

| Field | Type | Description |
|-------|------|-------------|
| `number` | int | Position in chain (1-indexed) |
| `content` | string | Thought content |
| `isRevision` | bool | Whether this revises a previous thought |
| `revisesNumber` | int | Which thought this revises (if isRevision) |
| `branchID` | string | Branch identifier (if branching) |
| `branchFromNumber` | int | Which thought this branches from |
| `createdAt` | time.Time | When the thought was recorded |

**Validation Rules**:
- `number`: Must be sequential within chain (1, 2, 3...)
- `number`: Cannot exceed `totalThoughts` declared at session start
- `isRevision` and `branchID` are mutually exclusive
- `revisesNumber`: Must reference existing thought (1 ≤ n ≤ current max)
- `branchFromNumber`: Must reference existing thought

### Session (In-Memory Only)

Manages a reasoning chain's state for a single connection.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique session identifier |
| `thoughts` | []Thought | Main reasoning chain |
| `branches` | map[string][]Thought | Named branches |
| `totalThoughts` | int | Expected total (can adjust) |
| `completed` | bool | Whether chain is finished |
| `createdAt` | time.Time | Session start time |

**Lifecycle**:
- Created on first thought submission
- Updated on each thought addition
- Completed when `next_thought_needed: false`
- Destroyed on connection close or server restart

---

## Database Schema

Both PostgreSQL and SQLite implementations share the same logical schema with `(name, version)` as composite primary key.

### PostgreSQL Migration (postgres/001_stickymemory.sql)

```sql
-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Main memories table (mcp-genie compatible schema)
CREATE TABLE memories (
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
```

### SQLite Migration (sqlite/001_stickymemory.sql)

```sql
-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL
);

-- Main memories table (mcp-genie compatible schema)
CREATE TABLE IF NOT EXISTS memories (
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (name, version)
);
```

---

## Storage Interface (mcp-genie compatible)

```go
// memory/storage.go

import "github.com/your-org/zombiekit/internal/mo"

// Storage defines the interface for storing and retrieving memory items
type Storage interface {
    // Set stores a memory item (creates new version)
    Set(ctx context.Context, name, content string) error

    // Get retrieves the latest non-deleted version of a memory item
    Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error)

    // Delete soft-deletes all versions of a memory item
    Delete(ctx context.Context, name string) error

    // List returns all items, optionally filtered by search query
    List(ctx context.Context, search string) ([]MemoryMetadata, error)

    // Clear removes all items
    Clear(ctx context.Context) (int, error)

    // Close closes any resources
    Close() error
}
```

**Maybe Monad** (from mcp-genie):
```go
// mo/maybe.go

type Maybe[T any] struct {
    value T
    valid bool
}

func Just[T any](v T) Maybe[T]    // Returns Maybe with value
func Nothing[T any]() Maybe[T]    // Returns empty Maybe
func (m Maybe[T]) HasValue() bool // Check if value exists
func (m Maybe[T]) Value() T       // Get value (caller must check HasValue first)
```

---

## Relationships

```
┌─────────────────────────────┐
│         memories            │
├─────────────────────────────┤
│ name (PK)                   │
│ version (PK)                │
│ content                     │
│ deleted                     │
│ created_at                  │
│ updated_at                  │
└─────────────────────────────┘

Each name can have multiple versions (1, 2, 3...).
Latest non-deleted version is returned by Get.
Delete marks ALL versions as deleted.

In-Memory Only (not persisted):

┌─────────────────┐       ┌─────────────────┐
│    Session      │       │    Thought      │
├─────────────────┤       ├─────────────────┤
│ id              │──1:N──│ number          │
│ totalThoughts   │       │ content         │
│ completed       │       │ isRevision      │
│ branches        │       │ revisesNumber   │
│ createdAt       │       │ branchID        │
└─────────────────┘       └─────────────────┘
```

---

## Query Patterns

### Set Memory (Create New Version)

**PostgreSQL**:
```sql
-- Get next version in a transaction
SELECT COALESCE(MAX(version), 0) + 1 FROM memories WHERE name = $1;

-- Insert new version
INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
VALUES ($1, $2, $3, FALSE, NOW(), NOW());
```

**SQLite**:
```sql
-- Get next version in a transaction
SELECT COALESCE(MAX(version), 0) + 1 FROM memories WHERE name = ?;

-- Insert new version
INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
VALUES (?, ?, ?, FALSE, ?, ?);
```

### Get Memory (Latest Non-Deleted Version)

**PostgreSQL**:
```sql
SELECT name, version, content, deleted, created_at, updated_at
FROM memories
WHERE name = $1 AND deleted = FALSE
ORDER BY version DESC
LIMIT 1;
```

**SQLite**:
```sql
SELECT name, version, content, deleted, created_at, updated_at
FROM memories
WHERE name = ? AND deleted = FALSE
ORDER BY version DESC
LIMIT 1;
```

### List Memories (Latest Version per Name)

**PostgreSQL**:
```sql
SELECT DISTINCT ON (name) name, version, LENGTH(content) as size, created_at, updated_at
FROM memories
WHERE deleted = FALSE
ORDER BY name, version DESC
LIMIT 100;
```

**SQLite**:
```sql
SELECT name, version, length(content) as size, created_at, updated_at
FROM memories m1
WHERE deleted = FALSE
AND version = (
    SELECT MAX(version)
    FROM memories m2
    WHERE m2.name = m1.name AND m2.deleted = FALSE
)
ORDER BY updated_at DESC;
```

### Search Memories

**PostgreSQL**:
```sql
SELECT DISTINCT ON (name) name, version, LENGTH(content) as size, created_at, updated_at
FROM memories
WHERE deleted = FALSE
  AND (name ILIKE '%' || $1 || '%' OR content ILIKE '%' || $1 || '%')
ORDER BY name, version DESC
LIMIT $2;
```

**SQLite**:
```sql
SELECT name, version, length(content) as size, created_at, updated_at
FROM memories m1
WHERE deleted = FALSE
AND version = (
    SELECT MAX(version)
    FROM memories m2
    WHERE m2.name = m1.name AND m2.deleted = FALSE
)
AND (LOWER(name) LIKE LOWER('%' || ? || '%') OR LOWER(content) LIKE LOWER('%' || ? || '%'))
ORDER BY updated_at DESC;
```

### Soft Delete (All Versions)

**PostgreSQL**:
```sql
UPDATE memories
SET deleted = TRUE, updated_at = NOW()
WHERE name = $1 AND deleted = FALSE;
```

**SQLite**:
```sql
UPDATE memories
SET deleted = TRUE, updated_at = ?
WHERE name = ? AND deleted = FALSE;
```

### Clear All

**PostgreSQL**:
```sql
-- Count first
SELECT COUNT(DISTINCT name) FROM memories WHERE deleted = FALSE;

-- Then soft delete all
UPDATE memories
SET deleted = TRUE, updated_at = NOW()
WHERE deleted = FALSE;
```

**SQLite**:
```sql
-- Count first
SELECT COUNT(DISTINCT name) FROM memories WHERE deleted = FALSE;

-- Then soft delete all
UPDATE memories
SET deleted = TRUE, updated_at = ?
WHERE deleted = FALSE;
```

---

## Configuration (mcp-genie compatible)

Environment variables for backend selection:

| Variable | Description | Default |
|----------|-------------|---------|
| `BRAINS_BACKEND` | Backend type: `sqlite` or `postgres` | `sqlite` |
| `BRAINS_SQLITE_PATH` | Path to SQLite database file | `~/.brains/memories.db` |
| `BRAINS_POSTGRES_URL` | PostgreSQL connection string | (none) |
| `BRAINS_POSTGRES_MAX_CONNS` | Max PostgreSQL connections | `10` |
| `BRAINS_POSTGRES_MIN_CONNS` | Min PostgreSQL connections | `2` |

```go
// config/storage.go

type StorageConfig struct {
    Backend     BackendType // sqlite or postgres
    SQLitePath  string      // Path to SQLite database
    PostgresURL string      // PostgreSQL connection string
    MaxConns    int32       // Max connections (postgres only)
    MinConns    int32       // Min connections (postgres only)
}
```
