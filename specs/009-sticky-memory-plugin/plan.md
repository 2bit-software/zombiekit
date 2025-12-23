# Implementation Plan: Sticky Memory Web Plugin

**Branch**: `009-sticky-memory-plugin` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-sticky-memory-plugin/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a frontend web plugin for the sticky memory tool providing list, view, create, edit, delete, and search capabilities. The plugin integrates with the existing web GUI architecture (feature 008), reuses the `internal/memory` storage backend, and includes markdown rendering toggle using a CDN-loaded library (marked.js).

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: go-chi/chi/v5 (routing), html/template (rendering), mark3labs/mcp-go (MCP), marked.js (CDN - client-side markdown)
**Storage**: Reuses existing `internal/memory` package (SQLite default, PostgreSQL optional)
**Testing**: go test with table-driven tests
**Target Platform**: Local web server (localhost), modern browsers
**Project Type**: Web plugin (extends existing web application)
**Performance Goals**: Page load <1 second (per SC-001), CRUD operations <30 seconds (per SC-002)
**Constraints**: Single-user local tool, memory content max 1MB, memory names max 255 chars
**Scale/Scope**: 100+ memories searchable (per SC-004), 7 templates (list, view, create, edit, delete confirmation, partials)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: PASS (constitution not yet configured for this project)

The project constitution file (`.specify/memory/constitution.md`) contains placeholder template content without specific principles defined. No gates to evaluate.

**Implicit Best Practices Applied**:
- Reuse existing patterns from profiles plugin (feature 008)
- Follow established project structure in `internal/webplugins/`
- Use existing `internal/memory` storage interface
- Maintain consistency with HTMX/Tailwind patterns already in codebase

## Project Structure

### Documentation (this feature)

```text
specs/009-sticky-memory-plugin/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/webplugins/memory/
├── plugin.go            # Plugin definition (WebPlugin + TemplatePlugin interfaces)
├── handlers.go          # HTTP handlers (list, view, create, edit, delete, search)
├── handlers_test.go     # Handler unit tests
└── templates/
    ├── list.html        # Memory list with search, pagination
    ├── view.html        # Memory detail with markdown toggle
    ├── form.html        # Create/edit form (shared)
    └── delete.html      # Delete confirmation modal/page

internal/cli/gui.go      # Modified to register memory plugin
internal/web/static/js/
└── markdown.js          # Optional: markdown toggle logic (or inline in templates)
```

**Structure Decision**: Follows existing `webplugins/profiles/` pattern. Plugin implements `WebPlugin` and `TemplatePlugin` interfaces, mounts at `/memory/`, and uses embedded template filesystem. Handler structure mirrors profiles plugin with service dependency injection.

## Complexity Tracking

> No constitution violations to justify. Design follows established patterns.

## Post-Design Constitution Re-Check

**Status**: PASS

**Design Validation**:
- ✅ Reuses existing `internal/memory` storage (no new storage layer)
- ✅ Follows `webplugins/profiles/` plugin pattern exactly
- ✅ Uses established HTMX/Tailwind patterns from feature 008
- ✅ No new external dependencies (marked.js is CDN-loaded, consistent with existing Tailwind/HTMX CDN approach)
- ✅ 4 templates total (list, view, form, delete) - minimal complexity
- ✅ Handler tests follow existing testing patterns

**Risk Assessment**: LOW
- All integration points are well-documented
- Existing patterns reduce implementation risk
- No architectural changes required
