# Implementation Plan: Plugin-Style Web GUI Architecture

**Branch**: `008-plugin-web-gui` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/008-plugin-web-gui/spec.md`

## Summary

Implement a plugin-style web GUI architecture where tools self-register to participate in a unified web interface. The system provides a shell layout with sidebar navigation and content area, using Chi router for routing, html/template for server-side rendering, HTMX for partial page updates, and Tailwind CSS for styling. A profiles plugin demonstrates the architecture as a reference implementation.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**:
- Chi router (`github.com/go-chi/chi/v5`) - NEW
- html/template (stdlib)
- HTMX 1.9.x (CDN)
- Tailwind CSS (CDN)
- embed (stdlib for static assets)
**Storage**: N/A (uses existing profile.Service from internal/profile)
**Testing**: `go test` with table-driven tests
**Target Platform**: Linux/macOS/Windows server
**Project Type**: Single Go binary with embedded web assets
**Performance Goals**: <500ms perceived navigation, partial updates <50% data transfer vs full page
**Constraints**: Single binary deployment, no external asset build process
**Scale/Scope**: Single-user local development tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution template is not yet configured for this project. Proceeding with standard Go best practices:

| Principle | Status | Notes |
|-----------|--------|-------|
| Testability | PASS | Plugin interface enables mock testing |
| Simplicity | PASS | Minimal dependencies, stdlib html/template |
| Single Binary | PASS | All assets embedded via go:embed |
| CLI Integration | PASS | Web server started via existing `serve` command |

## Project Structure

### Documentation (this feature)

```text
specs/008-plugin-web-gui/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (internal Go interfaces)
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── web/
│   ├── plugin.go         # WebPlugin interface, SidebarItem, PluginRegistry
│   ├── server.go         # Chi router setup, plugin mounting, graceful shutdown
│   ├── render.go         # Renderer with HTMX detection
│   ├── middleware.go     # Logger, Recovery middleware
│   ├── templates/
│   │   ├── shell.html    # Main layout with sidebar
│   │   ├── home.html     # Dashboard overview
│   │   ├── 404.html      # Not found page
│   │   └── error.html    # Error page
│   └── static/
│       ├── css/
│       │   └── app.css   # Minimal custom styles (Tailwind via CDN)
│       └── js/
│           └── app.js    # Minimal JS for sidebar active state
│
├── webplugins/
│   └── profiles/
│       ├── plugin.go     # WebPlugin implementation
│       ├── handlers.go   # List and view handlers
│       └── templates/
│           ├── list.html
│           └── view.html
│
└── cli/
    └── serve.go          # Updated to wire web plugins

cmd/
└── brains/
    └── main.go           # Unchanged (CLI entry point)

tests/
└── integration/
    └── web_test.go       # Integration tests for web GUI
```

**Structure Decision**: Extends existing `internal/web` package with plugin architecture. Web plugins go in new `internal/webplugins/` directory to keep them separate from core web infrastructure.

## Complexity Tracking

No violations requiring justification. Architecture follows Go idioms with minimal abstractions.
