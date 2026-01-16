# Data Model: WebGUI Status Page

**Feature**: 014-webgui-status
**Date**: 2025-12-22

## Overview

This feature introduces read-only status aggregation types. No persistent storage changes are required - all data is computed at request time from existing system state.

## Entities

### StatusInfo

Aggregate container for all system status information.

| Field | Type | Description |
|-------|------|-------------|
| Version | VersionInfo | Application version details |
| Database | DatabaseStatus | Database backend status |
| Runtime | RuntimeInfo | Runtime environment details |
| Plugins | []PluginStatus | Registered plugin information |
| Config | ConfigInfo | Configuration summary |

**Source**: Computed on each home page request
**Persistence**: None (transient)

### VersionInfo

Application build information.

| Field | Type | Description |
|-------|------|-------------|
| Version | string | Semantic version or "dev" |
| Commit | string | Git commit hash (short) |
| BuildDate | string | Build timestamp |
| GoVersion | string | Go compiler version |

**Source**: `internal/version.Get()` - set via ldflags at build time
**Persistence**: None (compile-time constants)

### DatabaseStatus

Database backend status and connection health.

| Field | Type | Description |
|-------|------|-------------|
| Backend | string | "sqlite" or "postgres" |
| Location | string | File path (SQLite) or host/database (PostgreSQL, sanitized) |
| Connected | bool | True if database is reachable |
| Error | string | Error message if not connected, empty otherwise |

**Source**: `config.StorageConfig` + optional ping
**Persistence**: None (computed)

**Validation rules**:
- Location MUST NOT contain credentials (passwords, API keys)
- For PostgreSQL: show only host and database name

### RuntimeInfo

Runtime environment information.

| Field | Type | Description |
|-------|------|-------------|
| OS | string | Operating system (e.g., "darwin", "linux") |
| Arch | string | Architecture (e.g., "amd64", "arm64") |
| Platform | string | Combined OS/Arch (e.g., "darwin/arm64") |
| Uptime | time.Duration | Time since server started |
| UptimeHuman | string | Human-readable uptime (e.g., "2h 15m") |
| NumCPU | int | Number of logical CPUs |
| NumGoroutines | int | Current goroutine count |

**Source**: `runtime` stdlib package + server start time
**Persistence**: None (computed)

### PluginStatus

Status of a registered web plugin.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Plugin identifier |
| Path | string | URL mount path |
| Healthy | bool | True if plugin is operational |

**Source**: `web.PluginRegistry.All()`
**Persistence**: None (computed)

### ConfigInfo

Key configuration values.

| Field | Type | Description |
|-------|------|-------------|
| Port | int | HTTP server port |
| LogLevel | string | Current log level |
| ProfilePaths | []string | Profile search directories (if available) |

**Source**: `web.ServerConfig` + CLI flags/env vars
**Persistence**: None (configuration)

## Relationships

```text
StatusInfo
├── VersionInfo (1:1)
├── DatabaseStatus (1:1)
├── RuntimeInfo (1:1)
├── PluginStatus (1:N)
└── ConfigInfo (1:1)
```

All relationships are composition (embedded values), not references.

## State Transitions

This feature has no state transitions - all entities are read-only snapshots computed at request time.

## Data Volume Assumptions

- Single StatusInfo computed per home page request
- Typically 2-5 plugins registered
- No caching required (data is cheap to compute)
- No persistence or migrations

## Schema Changes

None. This feature does not modify any database schema.

## New Interfaces

### Pinger (Optional Interface)

For storage implementations that support health checks:

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

Existing implementations (`sqlite.Storage`, `postgres.Storage`) can optionally implement this interface. Type assertion is used at runtime to check availability.

## Type Definitions (Go)

```go
package web

import "time"

// StatusInfo aggregates all system status information for display.
type StatusInfo struct {
    Version  VersionInfo
    Database DatabaseStatus
    Runtime  RuntimeInfo
    Plugins  []PluginStatus
    Config   ConfigInfo
}

// VersionInfo contains application build information.
type VersionInfo struct {
    Version   string
    Commit    string
    BuildDate string
    GoVersion string
}

// DatabaseStatus contains database backend status.
type DatabaseStatus struct {
    Backend   string // "sqlite" or "postgres"
    Location  string // Sanitized path or host/db
    Connected bool
    Error     string // Empty if connected
}

// RuntimeInfo contains runtime environment information.
type RuntimeInfo struct {
    OS            string
    Arch          string
    Platform      string
    Uptime        time.Duration
    UptimeHuman   string
    NumCPU        int
    NumGoroutines int
}

// PluginStatus contains plugin registration status.
type PluginStatus struct {
    Name    string
    Path    string
    Healthy bool
}

// ConfigInfo contains key configuration values.
type ConfigInfo struct {
    Port         int
    LogLevel     string
    ProfilePaths []string
}
```
