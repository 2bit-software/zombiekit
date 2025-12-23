# Implementation Plan: WebGUI Status Page

**Branch**: `014-webgui-status` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/014-webgui-status/spec.md`

## Summary

Add comprehensive system status information to the WebGUI home page, including version details (version, commit, build date, Go version), database backend status (type, connection health, sanitized path/host), runtime environment (OS, architecture, uptime), plugin status, and configuration summary. The implementation extends the existing home page handler and template without introducing new dependencies.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: go-chi/chi/v5 (routing), html/template (rendering), internal/version (build info), internal/config (storage config)
**Storage**: SQLite (default) or PostgreSQL - read-only status display
**Testing**: go test with testify/stretchr (existing pattern)
**Target Platform**: Cross-platform (Linux, macOS, Windows)
**Project Type**: Single Go project with embedded web server
**Performance Goals**: Home page loads within 500ms (per SC-002)
**Constraints**: No credential exposure (FR-011), graceful degradation (FR-013)
**Scale/Scope**: Single instance, developer-focused debug interface

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution is a template without project-specific gates defined. This feature:
- Uses existing patterns (WebPlugin, PluginRegistry, Renderer)
- Follows existing code structure (internal/web, internal/version)
- Requires no new dependencies
- Is a read-only display feature with no storage changes

**Status**: PASS - No constitution violations identified.

## Project Structure

### Documentation (this feature)

```text
specs/014-webgui-status/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API contracts)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── version/
│   └── version.go       # Existing - build info (version, commit, date, go version)
├── config/
│   └── storage.go       # Existing - StorageConfig with Backend type
├── web/
│   ├── server.go        # MODIFY - add StatusInfo to homeHandler
│   ├── status.go        # NEW - StatusInfo aggregation
│   ├── status_test.go   # NEW - status info tests
│   └── templates/
│       └── home.html    # MODIFY - add status display sections
└── cli/
    └── gui.go           # MODIFY - pass config to server for status

cmd/
└── brains/
    └── main.go          # Existing - entry point
```

**Structure Decision**: Single project structure. Changes are localized to `internal/web` package with minor updates to `internal/cli/gui.go` to pass configuration context.

## Complexity Tracking

No violations to justify. This feature:
- Adds no new packages or dependencies
- Uses existing patterns for web rendering
- Is purely a display feature (read-only)
