# Research: WebGUI Status Page

**Feature**: 014-webgui-status
**Date**: 2025-12-22

## Research Summary

This feature has no unresolved NEEDS CLARIFICATION items. The research below documents key implementation decisions and patterns discovered during codebase analysis.

## Decision 1: Status Information Aggregation

**Decision**: Create a `StatusInfo` struct in `internal/web/status.go` that aggregates all status data.

**Rationale**:
- Centralizes status gathering in one place
- Easy to test independently
- Follows existing pattern (e.g., `PageData` in render.go)
- Allows lazy/on-demand gathering of status info

**Alternatives considered**:
- Gather status directly in homeHandler: Rejected - mixes concerns, harder to test
- Create separate StatusService: Rejected - over-engineering for read-only display

## Decision 2: Database Health Check

**Decision**: Add a `Ping(context.Context) error` method to the Storage interface, call it at status display time.

**Rationale**:
- Storage implementations (SQLite, PostgreSQL) can provide health checks
- Context allows timeout control
- Returns error for unhealthy state

**Alternatives considered**:
- Track last operation success: Rejected - doesn't reflect current state
- Background health polling: Rejected - adds complexity for infrequent access
- Interface already exists: Need to verify if Storage interface can be extended

**Note**: The current `Storage` interface in `internal/memory/storage.go` does not have a Ping method. Options:
1. Add Ping to Storage interface (breaking change if external implementations exist)
2. Use type assertion to optional interface (Pinger)
3. Skip health check for initial implementation

**Recommendation**: Use optional `Pinger` interface to avoid breaking changes:
```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

## Decision 3: Database Backend Detection

**Decision**: Use `config.BackendType` from the config package to determine backend type.

**Rationale**:
- Already exists in `internal/config/storage.go`
- `BackendSQLite` and `BackendPostgres` constants defined
- Config is already passed to storage factory

**Implementation path**:
- Pass `config.StorageConfig` to web.Server or ServerConfig
- Extract backend type and path/URL for display
- Sanitize PostgreSQL URL to remove credentials

## Decision 4: Credential Sanitization for PostgreSQL

**Decision**: Parse PostgreSQL connection URL and redact password before display.

**Rationale**:
- FR-011 requires no credential exposure
- Standard URL parsing can extract host/database safely
- Show host and database name only

**Implementation**:
```go
import "net/url"

func sanitizePostgresURL(connURL string) string {
    u, err := url.Parse(connURL)
    if err != nil {
        return "(invalid connection string)"
    }
    return fmt.Sprintf("%s/%s", u.Host, strings.TrimPrefix(u.Path, "/"))
}
```

## Decision 5: Process Uptime

**Decision**: Store server start time at initialization, calculate uptime on each request.

**Rationale**:
- Simple, no external dependencies
- Accurate to second precision
- `time.Since(startTime)` is efficient

**Implementation**:
- Store `startTime time.Time` in Server struct
- Set in NewServer or Start
- Display as human-readable duration (e.g., "2h 15m 30s")

## Decision 6: Plugin Health Status

**Decision**: Plugins are considered "healthy" if registered, track initialization errors separately.

**Rationale**:
- Current PluginRegistry tracks successful registrations only
- Initialization failures are logged but not tracked
- For V1, show registered plugins with count
- Plugin-level health checks can be added later via optional interface

**Implementation path**:
- Display count of registered plugins
- List plugin names
- Future: Add optional `HealthChecker` interface for plugins

## Decision 7: Template Layout

**Decision**: Add status sections to existing home.html template, grouped by category.

**Rationale**:
- FR-012 requires maintaining visual consistency
- Existing template uses Tailwind CSS classes
- Grid layout for responsive design already in use

**Layout structure**:
1. Version section (top, prominent)
2. Database section (connection status with indicator)
3. Runtime section (OS, architecture, uptime)
4. Plugins section (existing, enhanced with count)
5. Configuration section (port, log level)

## Key Findings from Codebase Analysis

### Existing Version Package
- `internal/version/version.go` provides `Get() *BuildInfo`
- BuildInfo contains: Version, Commit, BuildDate, GoVersion
- Set via ldflags at build time
- Has `Short()` and `PrettyPrint()` helpers

### Existing Config Package
- `internal/config/storage.go` has `StorageConfig` struct
- `Backend` field is `BackendType` (sqlite or postgres)
- `SQLitePath` and `PostgresURL` fields available
- `LoadStorageConfigFromEnv()` for environment-based config

### Existing Web Package
- `Server` struct in `server.go` has `config ServerConfig`
- `homeHandler` passes data via `PageData.Content`
- Templates receive data as `.Content.FieldName`
- `Renderer` provides `Render(w, r, template, data)` method

### Runtime Package (stdlib)
- `runtime.GOOS` - operating system
- `runtime.GOARCH` - architecture
- `runtime.NumCPU()` - CPU count (useful debug info)
- `runtime.NumGoroutine()` - goroutine count (useful debug info)

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Storage interface change breaks external implementations | Low | Medium | Use optional Pinger interface |
| PostgreSQL URL parsing fails | Low | Low | Return safe fallback string |
| Config not available in web package | Low | Medium | Pass via ServerConfig |
| Template changes break HTMX | Low | Low | Test with both full page and HTMX partial loads |

## Dependencies Verification

All required functionality exists in stdlib or existing packages:
- `runtime` (stdlib) - OS, architecture, goroutines
- `time` (stdlib) - uptime calculation
- `net/url` (stdlib) - URL parsing for credential removal
- `internal/version` - build info
- `internal/config` - storage config
- `internal/web` - existing server structure

No new external dependencies required.
