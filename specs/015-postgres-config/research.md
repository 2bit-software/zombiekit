# Research: PostgreSQL Configuration with SQLite Fallback

**Feature**: 015-postgres-config
**Date**: 2025-12-22

## Summary

No NEEDS CLARIFICATION items in technical context. All required technologies already exist in the codebase. This document captures design decisions based on existing patterns.

## Research Findings

### 1. TOML Config File Structure for Storage

**Decision**: Add `[storage]` section to existing TOML config structure

**Rationale**:
- The project already uses BurntSushi/toml for config parsing (see `internal/config/loader.go`)
- Existing `Config` struct uses TOML tags (see `internal/config/config.go`)
- The pattern of global → local → env precedence is already established

**Alternatives considered**:
- Separate storage config file: Rejected - adds complexity, user must manage multiple files
- YAML instead of TOML: Rejected - inconsistent with existing config format
- JSON: Rejected - less readable, no comments

### 2. Connection Timeout with Context

**Decision**: Use `context.WithTimeout` for connection attempts

**Rationale**:
- pgx/v5 supports context-based cancellation natively
- Allows clean timeout handling without goroutine management
- Existing `NewPostgresPool` already accepts context (see `internal/database/postgres.go:21`)

**Alternatives considered**:
- pgxpool connection timeout config: Less explicit, harder to communicate via logs
- Manual goroutine + timer: More complex, context is cleaner

### 3. Fallback Implementation Pattern

**Decision**: Try-catch pattern at startup in `serve.go`

**Rationale**:
- Single point of decision making
- Fallback only happens once at startup (per clarification: session-authoritative)
- Logging of fallback reason is straightforward

**Alternatives considered**:
- Factory pattern with automatic fallback: Over-engineered for session-scoped decision
- Retry with backoff: Rejected - spec requires immediate fallback, not retry

### 4. StorageConfig Field Addition

**Decision**: Add `ConnectionTimeout` field to existing `StorageConfig` struct

**Rationale**:
- Keeps all storage-related config in one struct
- Consistent with existing `MaxConns`/`MinConns` pattern
- Easy to merge from TOML, env, and CLI

**Alternatives considered**:
- Separate timeout config: Fragmented, harder to manage
- Global timeout constant: Not configurable as spec requires

### 5. Config Loading Order

**Decision**: Maintain existing precedence: default → global file → local file → env → CLI

**Rationale**:
- Consistent with existing tool config behavior
- Users expect local to override global
- Environment variables allow container/CI override without file changes

**Implementation**:
```
1. StorageConfig with defaults (sqlite, ~/.brains/memories.db, 5s timeout)
2. Merge global config.toml [storage] if present
3. Merge local .brains/config.toml [storage] if present
4. Override with BRAINS_* environment variables if set
5. Override with CLI flags if provided
```

### 6. Actual Backend Tracking for Status Display

**Decision**: Store actual backend in use (vs. configured backend) in a new field

**Rationale**:
- Status display needs to show what's actually connected (per FR-009)
- After fallback, configured=postgres but actual=sqlite
- Existing `web/status.go` reads from `StorageConfig.Backend`

**Implementation**:
- Add `ActualBackend` field to track post-fallback state
- Or: Update `Backend` field after fallback decision
- Chose: Update `Backend` field - simpler, no additional field needed

## Dependencies Check

| Dependency | Version | Purpose | Already in go.mod |
|------------|---------|---------|-------------------|
| BurntSushi/toml | v1.6.0 | TOML parsing | Yes |
| jackc/pgx/v5 | v5.7.6 | PostgreSQL driver | Yes |
| modernc.org/sqlite | v1.41.0 | SQLite driver | Yes |
| urfave/cli/v2 | v2.27.7 | CLI framework | Yes |

No new dependencies required.

## Test Strategy

1. **Unit tests** (`internal/config/storage_test.go`):
   - Parse TOML with [storage] section
   - Merge priority: global < local < env
   - Default values when not specified
   - Invalid TOML handling

2. **Integration tests** (`tests/integration/config_fallback_test.go`):
   - PostgreSQL available: connects successfully
   - PostgreSQL unavailable: falls back to SQLite with warning
   - Connection timeout: falls back within timeout period
   - Use testcontainers for real PostgreSQL
