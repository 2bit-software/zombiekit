# Quickstart: PostgreSQL Configuration with SQLite Fallback

**Feature**: 015-postgres-config
**Date**: 2025-12-22

## Overview

This feature enables PostgreSQL connection configuration via TOML config files with automatic SQLite fallback when PostgreSQL is unavailable.

## Configuration

### 1. Create Config File

Create `.brains/config.toml` in your project directory (or `~/.config/brains/config.toml` for global config):

```toml
[storage]
backend = "postgres"
postgres_url = "postgres://username:password@localhost:5432/brains"
connection_timeout = 5
max_connections = 10
min_connections = 2
```

### 2. Start PostgreSQL (Optional)

If PostgreSQL is running and accessible, the application will connect to it. If not, it automatically falls back to SQLite.

### 3. Run the Application

```bash
brains serve
```

Expected output when PostgreSQL connects:
```
INFO Storage initialized backend=postgres host=localhost:5432/brains
INFO Starting MCP server mode=http port=8080
```

Expected output when fallback occurs:
```
WARN PostgreSQL connection failed, falling back to SQLite error="connection refused"
INFO Storage initialized backend=sqlite path=~/.brains/memories.db
INFO Starting MCP server mode=http port=8080
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| backend | sqlite | Storage backend: "sqlite" or "postgres" |
| postgres_url | (empty) | PostgreSQL connection URL |
| sqlite_path | ~/.brains/memories.db | SQLite database path |
| connection_timeout | 5 | Timeout in seconds for PostgreSQL connection |
| max_connections | 10 | Maximum PostgreSQL pool connections |
| min_connections | 2 | Minimum PostgreSQL pool connections |

## Precedence Order

Configuration is merged in this order (later overrides earlier):

1. **Defaults** (sqlite, ~/.brains/memories.db)
2. **Global config** (~/.config/brains/config.toml)
3. **Local config** (.brains/config.toml)
4. **Environment variables** (BRAINS_BACKEND, BRAINS_POSTGRES_URL, etc.)
5. **CLI flags** (--db-type)

## Environment Variables

Environment variables override config file settings:

| Variable | Maps to |
|----------|---------|
| BRAINS_BACKEND | backend |
| BRAINS_POSTGRES_URL | postgres_url |
| BRAINS_SQLITE_PATH | sqlite_path |
| BRAINS_POSTGRES_MAX_CONNS | max_connections |
| BRAINS_POSTGRES_MIN_CONNS | min_connections |

## Checking Status

View which backend is active via the web GUI:

```bash
brains gui
# Open http://localhost:8080/status
```

The status page displays:
- **Backend**: sqlite or postgres
- **Location**: Database path or host/database name

## Common Scenarios

### Scenario 1: PostgreSQL for Shared Team Database

```toml
[storage]
backend = "postgres"
postgres_url = "postgres://team:secret@db.internal:5432/brains"
```

### Scenario 2: PostgreSQL with SQLite Fallback for Offline Work

Same config as above - when VPN disconnects, work continues on local SQLite.

### Scenario 3: Force SQLite Despite Config

```bash
BRAINS_BACKEND=sqlite brains serve
```

### Scenario 4: Test Against Different Database

```bash
BRAINS_POSTGRES_URL="postgres://test:test@localhost:5433/test_db" brains serve
```
