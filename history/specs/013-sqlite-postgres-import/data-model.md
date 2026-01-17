# Data Model: SQLite to PostgreSQL Migration Tool

**Feature**: 013-sqlite-postgres-import
**Date**: 2025-12-22

## Entities

### ImportMetadata (NEW)

Tracks import history to enable incremental imports (FR-002, SC-006).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | integer | PK, auto-increment | Unique identifier |
| source_path_hash | text | UNIQUE, NOT NULL | SHA256 hash of absolute SQLite path |
| source_path | text | NOT NULL | Original path for display/logging |
| last_import_at | timestamp with timezone | NOT NULL | When last import completed |
| last_imported_updated_at | timestamp with timezone | | Max updated_at from source at import time |
| items_imported | integer | NOT NULL, DEFAULT 0 | Total items imported in last run |
| created_at | timestamp with timezone | NOT NULL, DEFAULT NOW() | Record creation time |
| updated_at | timestamp with timezone | NOT NULL, DEFAULT NOW() | Record modification time |

**Relationships**: None (standalone metadata table)

**Location**: PostgreSQL target database (same database as memories table)

### MemoryItem (EXISTING - Reference)

Existing entity in both SQLite and PostgreSQL (no changes).

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| name | text | PK (with version) | Memory name identifier |
| version | integer | PK (with name) | Version number |
| content | text | NOT NULL | Memory content |
| deleted | boolean | NOT NULL, DEFAULT FALSE | Soft-delete flag |
| created_at | timestamp | NOT NULL | Record creation time |
| updated_at | timestamp | NOT NULL | Record modification time |

**Notes**:
- Primary key is composite: (name, version)
- SQLite uses TIMESTAMP (text), PostgreSQL uses TIMESTAMPTZ
- Timestamps normalized to UTC during import

## State Transitions

### ImportMetadata States

```
[Not Exists] --> [Created] --> [Updated]
                     |              |
                     v              v
              first import    subsequent imports
              creates record   updates last_import_at
```

### Import Operation Flow

```
1. Check ImportMetadata for source_path_hash
   - Not found: Full import (all items)
   - Found: Incremental import (items where updated_at > last_imported_updated_at)

2. For each item from SQLite:
   - Check PostgreSQL for existing name/version
   - Apply conflict resolution (FR-012)
   - Insert or skip

3. On completion:
   - Create/Update ImportMetadata with max(updated_at) from imported items
```

## Data Volume Assumptions

- Typical dataset: 100-10,000 memory items
- Average item content size: 1-10 KB
- Import batch size: 100 items per transaction
- Performance target: 1000 items in 30 seconds (SC-001)

## Validation Rules

### ImportMetadata

- `source_path_hash` must be valid SHA256 hex string (64 characters)
- `source_path` must be non-empty
- `last_import_at` must be set on every import
- `items_imported` must be >= 0

### Import Conflict Resolution (FR-012)

1. If source version > target version: Import source, soft-delete target versions
2. If source version == target version: Skip (idempotent)
3. If source version < target version: Skip (target is ahead)

## Migration

New table `import_metadata` created via Go code (not schema migration) following the pattern in `internal/memory/postgres/storage.go`:

```go
func (s *Importer) ensureSchema(ctx context.Context) error {
    _, err := s.pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS import_metadata (
            id SERIAL PRIMARY KEY,
            source_path_hash TEXT NOT NULL UNIQUE,
            source_path TEXT NOT NULL,
            last_import_at TIMESTAMPTZ NOT NULL,
            last_imported_updated_at TIMESTAMPTZ,
            items_imported INTEGER NOT NULL DEFAULT 0,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `)
    return err
}
```
