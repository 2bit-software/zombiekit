# Implementation Plan: Profile Type Classification

**Branch**: `018-profile-type` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/018-profile-type/spec.md`

## Summary

Add a new optional `type` field to profile YAML frontmatter supporting three values: "action", "domain", and "step". Extend the Go data structures (`Profile`, `ProfileFrontmatter`, `ListEntry`, `ShowResult`) to include this field, update the frontmatter parser, and expose the type in the web UI (profiles list badges and detail view metadata section). P3 filtering feature is deferred to future iteration.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**:
- adrg/frontmatter (YAML parsing)
- go-chi/chi/v5 (web routing)
- html/template (Go templates for web UI)
- urfave/cli/v2 (CLI)
**Storage**: N/A (file-based profiles, no database changes)
**Testing**: go test with testify/assert
**Target Platform**: Linux/macOS server, web browser
**Project Type**: Single CLI/web application with embedded web server
**Performance Goals**: N/A (metadata display only, no performance-critical paths)
**Constraints**: None (simple field addition)
**Scale/Scope**: Small feature - 4 files to modify, 2 templates to update

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains only template placeholders (no actual principles defined). Proceeding without constitution gates. Standard Go best practices apply:
- [x] Changes are minimal and focused on the feature
- [x] No new dependencies required
- [x] Existing patterns followed (matches Model/Color field precedent)
- [x] Tests will be added for new functionality
- [x] Backwards compatible (type field is optional)

## Project Structure

### Documentation (this feature)

```text
specs/018-profile-type/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API changes)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── profile/
│   ├── types.go           # Add Type field to Profile, ProfileFrontmatter, ListEntry, ShowResult
│   ├── frontmatter.go     # Update ParseProfile to populate Type field
│   └── frontmatter_test.go # Add tests for Type parsing
└── webplugins/
    └── profiles/
        └── templates/
            ├── list.html   # Add type badge display
            └── view.html   # Add type in metadata section
```

**Structure Decision**: Single project structure. All changes are within the existing `internal/` package hierarchy. The profile package handles data structures and parsing; webplugins/profiles handles web UI templates.

## Complexity Tracking

> No violations. Feature is minimal:
> - Adds one optional string field to existing structs
> - Follows established pattern (identical to Model/Color fields)
> - No new abstractions, no new packages
