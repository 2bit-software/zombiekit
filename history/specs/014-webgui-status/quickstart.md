# Quickstart: WebGUI Status Page

**Feature**: 014-webgui-status
**Date**: 2025-12-22

## Overview

This document provides a quick reference for implementing the WebGUI Status Page feature.

## Files to Create

| File | Purpose |
|------|---------|
| `internal/web/status.go` | StatusInfo struct and gathering functions |
| `internal/web/status_test.go` | Unit tests for status gathering |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/web/server.go` | Add StatusConfig to Server, update homeHandler |
| `internal/web/templates/home.html` | Add status display sections |
| `internal/cli/gui.go` | Pass StatusConfig to Server |

## Key Implementation Steps

### 1. Create status.go

```go
package web

import (
    "context"
    "net/url"
    "runtime"
    "strings"
    "time"

    "github.com/zombiekit/brains/internal/config"
    "github.com/zombiekit/brains/internal/version"
)

// StatusConfig holds dependencies for status gathering.
type StatusConfig struct {
    ServerPort    int
    LogLevel      string
    StorageConfig config.StorageConfig
    StartTime     time.Time
    // Optional: Pinger for health checks
}

// GatherStatus collects all status information.
func GatherStatus(ctx context.Context, cfg StatusConfig, registry *PluginRegistry) StatusInfo {
    // Implementation
}
```

### 2. Update Server

Add `StatusConfig` to Server and call `GatherStatus` in homeHandler.

### 3. Update home.html Template

Add sections for:
- Version info (version, commit, build date, go version)
- Database status (backend, location, connection indicator)
- Runtime (OS/arch, uptime, goroutines)
- Plugins (count, list)
- Config (port, log level)

## Testing Checklist

- [ ] StatusInfo populated correctly with mock data
- [ ] PostgreSQL URL sanitization removes credentials
- [ ] Uptime calculation is accurate
- [ ] Template renders all status sections
- [ ] HTMX partial updates work correctly
- [ ] Graceful degradation when data unavailable

## Verification

```bash
# Run tests
go test ./internal/web/...

# Start GUI and verify home page
go run ./cmd/brains gui --port 8080
# Visit http://localhost:8080 and verify status display
```

## Dependencies

- No new external dependencies
- Uses stdlib: `runtime`, `time`, `net/url`
- Uses existing: `internal/version`, `internal/config`
