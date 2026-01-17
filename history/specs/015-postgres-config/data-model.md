# Data Model: PostgreSQL Configuration with SQLite Fallback

**Feature**: 015-postgres-config
**Date**: 2025-12-22

## Entities

### StorageConfig (Extended)

The existing `StorageConfig` struct in `internal/config/storage.go` is extended with new fields.

**Current fields** (unchanged):
| Field | Type | Description |
|-------|------|-------------|
| Backend | BackendType | Storage backend type ("sqlite" or "postgres") |
| SQLitePath | string | Path to SQLite database file |
| PostgresURL | string | PostgreSQL connection string |
| MaxConns | int32 | Maximum connections in pool |
| MinConns | int32 | Minimum connections in pool |

**New fields**:
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| ConnectionTimeout | time.Duration | 5s | Timeout for PostgreSQL connection attempts |
| FallbackEnabled | bool | true | Whether to fall back to SQLite on failure |

### FileStorageConfig (New - TOML representation)

Internal struct for parsing TOML config. Maps to StorageConfig after parsing.

| Field | TOML Key | Type | Description |
|-------|----------|------|-------------|
| Backend | backend | string | "sqlite" or "postgres" |
| PostgresURL | postgres_url | string | Connection string |
| SQLitePath | sqlite_path | string | Path to SQLite file |
| ConnectionTimeout | connection_timeout | int | Seconds (converts to Duration) |
| MaxConns | max_connections | int | Max pool connections |
| MinConns | min_connections | int | Min pool connections |

### Config (Extended)

The existing `Config` struct in `internal/config/config.go` is extended.

**Current fields** (unchanged):
| Field | Type | Description |
|-------|------|-------------|
| Tools | map[string]ToolConfig | Tool enablement configuration |

**New fields**:
| Field | Type | Description |
|-------|------|-------------|
| Storage | *FileStorageConfig | Storage section from TOML file |

## TOML Schema

```toml
# .brains/config.toml or ~/.config/brains/config.toml

[tools]
# Existing tool configuration...

[storage]
# Backend type: "sqlite" (default) or "postgres"
backend = "postgres"

# PostgreSQL connection URL (only used when backend = "postgres")
postgres_url = "postgres://user:password@localhost:5432/brains"

# SQLite database path (only used when backend = "sqlite")
# Supports ~ expansion
sqlite_path = "~/.brains/memories.db"

# Connection timeout in seconds (default: 5)
connection_timeout = 5

# PostgreSQL connection pool settings
max_connections = 10
min_connections = 2
```

## State Transitions

### Connection State Machine

```
                 ┌─────────────────┐
                 │   Configured    │
                 │   (postgres)    │
                 └────────┬────────┘
                          │
                          ▼
              ┌──────────────────────┐
              │ Attempt Connection   │
              │ (with timeout)       │
              └──────────┬───────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
    ┌──────────┐  ┌───────────┐  ┌───────────┐
    │ Success  │  │ Timeout   │  │  Error    │
    │          │  │           │  │           │
    └────┬─────┘  └─────┬─────┘  └─────┬─────┘
         │              │              │
         ▼              └──────┬───────┘
    ┌──────────┐               │
    │ postgres │               ▼
    │ (active) │        ┌───────────┐
    └──────────┘        │ Fallback  │
                        │ to SQLite │
                        └─────┬─────┘
                              │
                              ▼
                        ┌───────────┐
                        │  sqlite   │
                        │ (active)  │
                        └───────────┘
```

**Invariants**:
- Once fallback occurs, session remains on SQLite (no auto-reconnect)
- Backend field reflects actual connected backend, not configured backend
- Fallback always logs a warning with the failure reason

## Validation Rules

1. **PostgresURL format**: Must be valid URL when backend=postgres
   - Scheme: `postgres://` or `postgresql://`
   - Required: host, port, database name
   - Optional: user, password (in URL or env)

2. **SQLitePath**: Must be writable path
   - Supports `~` expansion for home directory
   - Creates parent directories if needed

3. **ConnectionTimeout**: Must be positive integer
   - Default: 5 seconds
   - Range: 1-300 seconds (enforce upper limit to prevent hangs)

4. **MaxConns/MinConns**: Positive integers
   - MaxConns >= MinConns
   - Defaults: MaxConns=10, MinConns=2

## Relationships

```
Config 1 ──────┬──────> * ToolConfig
              │
              └──────> 0..1 FileStorageConfig
                              │
                              │ merges into
                              ▼
                        StorageConfig
                              │
                              │ configures
                              ▼
                 ┌────────────┴────────────┐
                 │                         │
                 ▼                         ▼
           PostgresPool              SQLiteStorage
```
