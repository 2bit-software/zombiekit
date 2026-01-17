# Implementation Plan: Simplified Plugin Registration API

**Branch**: `012-plugin-registration-api` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/012-plugin-registration-api/spec.md`

## Summary

Simplify plugin registration from `registry.Register(plugin)` (where plugin provides its own ID via `ID() string` method) to `webgui.Register("name", plugin)` where the name is provided at registration time. This enables plugins to be portable (not hardcoding their name), provides automatic URL prefixing for relative paths, and simplifies sidebar configuration.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: go-chi/chi/v5 (routing), mark3labs/mcp-go (MCP tools)
**Storage**: N/A (no storage changes - this is an API/interface change)
**Testing**: go test with testify/stretchr
**Target Platform**: Linux/macOS server (CLI tool with web interface)
**Project Type**: single
**Performance Goals**: N/A (registration happens once at startup)
**Constraints**: Breaking change to WebPlugin interface; must migrate existing plugins
**Scale/Scope**: 2 existing plugins (memory, profiles) to migrate

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file is a template with placeholder principles. No specific gates defined.

**Assessment**: PASS - no constitution violations. Proceeding with standard Go best practices.

## Project Structure

### Documentation (this feature)

```text
specs/012-plugin-registration-api/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── web/
│   ├── plugin.go        # MODIFY: WebPlugin interface, PluginRegistry
│   ├── server.go        # MODIFY: Plugin mounting, URL prefixing
│   ├── render.go        # REVIEW: May need sidebar path prefixing
│   └── middleware.go    # No changes expected
├── webplugins/
│   ├── memory/
│   │   ├── plugin.go    # MODIFY: Remove ID(), update URLs to relative
│   │   └── handlers.go  # REVIEW: Redirect URLs
│   └── profiles/
│       ├── plugin.go    # MODIFY: Remove ID(), update URLs to relative
│       └── handlers.go  # REVIEW: Redirect URLs
└── search/
    └── search.go        # REVIEW: SearchResult.URL handling
```

**Structure Decision**: Single project structure. Changes are limited to `internal/web/` (core changes) and `internal/webplugins/` (plugin migrations).

## Complexity Tracking

No constitution violations to justify.
