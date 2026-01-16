# CLI Contract: brains db import

**Feature**: 013-sqlite-postgres-import
**Date**: 2025-12-22

## Command Synopsis

```
brains db import --from <sqlite-path> [--to <postgres-url>] [options]
```

## Description

Imports memory data from a SQLite database to PostgreSQL. Supports incremental imports where only items created or updated since the last import are transferred.

## Flags

| Flag | Short | Type | Required | Default | Description |
|------|-------|------|----------|---------|-------------|
| `--from` | `-f` | string | Yes | - | Path to source SQLite database file |
| `--to` | `-t` | string | No | `$BRAINS_POSTGRES_URL` | PostgreSQL connection URL |
| `--dry-run` | `-n` | bool | No | false | Preview import without making changes |
| `--batch-size` | - | int | No | 100 | Items per batch transaction |
| `--verbose` | `-v` | bool | No | false | Show detailed progress |
| `--format` | - | string | No | "text" | Output format: text, json |

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `BRAINS_POSTGRES_URL` | PostgreSQL connection URL | `postgres://user:pass@host:5432/db` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (invalid arguments, connection failure) |
| 2 | Partial failure (some items failed to import) |

## Output Format

### Text Mode (default)

```
Importing from /path/to/memories.db to PostgreSQL...
Progress: 100/100 items imported
Skipped: 5 (already exist)
Errors: 0

Import completed in 2.3s
Summary:
  Imported: 100
  Skipped: 5
  Errors: 0
```

### Dry-Run Mode

```
Dry run - no changes will be made

Would import from /path/to/memories.db:
  Total items in source: 105
  New items to import: 100
  Already imported: 5

Items to import:
  - my-memory (version 3)
  - another-memory (version 1)
  ...
```

### JSON Mode

```json
{
  "source": "/path/to/memories.db",
  "target": "postgres://...",
  "dry_run": false,
  "result": {
    "imported": 100,
    "skipped": 5,
    "errors": 0,
    "error_details": [],
    "duration_ms": 2300
  }
}
```

### Error Output

Errors written to stderr:

```
Error: SQLite database not found: /path/to/memories.db
Error: PostgreSQL connection failed: connection refused
Error: Import failed: item "my-memory" version 2: constraint violation
```

## Examples

### Basic Import

```bash
brains db import --from ~/.brains/memories.db --to "postgres://localhost:5432/brains"
```

### Preview Import

```bash
brains db import --from ~/.brains/memories.db --dry-run
```

### Import with Progress

```bash
brains db import --from ~/.brains/memories.db --verbose
```

### CI/CD Usage (JSON output)

```bash
brains db import --from backup.db --format json | jq '.result.imported'
```

## Behavior Notes

1. **Exclusive Lock**: SQLite database is exclusively locked during import (blocks other processes)
2. **Incremental**: Only items updated after the last successful import are processed
3. **Idempotent**: Running the same import twice produces the same result (skips already-imported items)
4. **Atomic Batches**: Each batch is committed atomically; failures don't affect previous batches
5. **Conflict Resolution**: Higher version in source replaces lower version in target; lower version in source is skipped
