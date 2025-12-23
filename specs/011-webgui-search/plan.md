# Implementation Plan: Web GUI Search Bar

**Branch**: `011-webgui-search` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/011-webgui-search/spec.md`

## Summary

Add a debounced search bar to the web GUI header that queries all plugins implementing the Searchable interface, displays up to 3 results per plugin in a dropdown, and navigates to selected results using HTMX partial page updates while preserving the shell layout.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: go-chi/chi/v5 (routing), html/template (rendering), HTMX 1.9.10 (client-side), Tailwind CSS (styling via CDN)
**Storage**: N/A (uses existing memory plugin storage; search is read-only)
**Testing**: Go testing package with testify assertions, table-driven tests
**Target Platform**: Web browser (desktop/mobile), local development tool
**Project Type**: Single Go binary with embedded web UI
**Performance Goals**: Search results within 500ms of last keystroke (300ms debounce + 200ms query time)
**Constraints**: Results limited to 3 per plugin; client-side debouncing to reduce server load
**Scale/Scope**: Single user, local deployment; memory plugin currently only searchable plugin

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution template contains placeholder values, so no specific gates are enforced.

**Applicable principles from codebase review**:
- Uses existing Searchable interface (no new abstractions needed)
- Follows existing plugin pattern (integration with PluginRegistry)
- Uses existing HTMX patterns from shell.html
- Tests will be written for new handler and JavaScript logic

**Status**: PASS (no violations)

## Project Structure

### Documentation (this feature)

```text
specs/011-webgui-search/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── search/
│   ├── search.go        # Existing Searchable interface and SearchResult type
│   └── search_test.go   # Existing tests
├── web/
│   ├── server.go        # ADD: search endpoint handler
│   ├── search.go        # NEW: search aggregation logic
│   ├── search_test.go   # NEW: search handler tests
│   ├── render.go        # MODIFY: may need search bar in PageData
│   ├── templates/
│   │   └── shell.html   # MODIFY: add search bar to header
│   └── static/
│       └── js/
│           └── app.js   # MODIFY: add debounce/search JS
└── webplugins/
    └── memory/
        └── plugin.go    # Existing (already implements Searchable)
```

**Structure Decision**: Single project structure. The search feature adds a new endpoint to the existing web server and modifies the shell template. No new packages required beyond a search.go file in internal/web.

## Complexity Tracking

No constitution violations to justify.
