# Implementation Plan: Searchable Interface

**Branch**: `010-searchable-interface` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-searchable-interface/spec.md`

## Summary

Define a standalone `Searchable` interface with `SearchResult` type that plugins can optionally implement to enable search functionality. The interface is independent of `WebPlugin` for future reuse in CLI/MCP contexts. Includes support for query parameters (max_results, sort_order) and searching across names and content.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: None new required - interface-only feature
**Storage**: N/A (interface contract only; implementations provide storage)
**Testing**: go test with testify/assert (existing pattern)
**Target Platform**: Any (interface definition is platform-agnostic)
**Project Type**: Single Go module with internal packages
**Performance Goals**: Implementations should return results within 100ms for <10k items
**Constraints**: Interface must have zero dependencies on `internal/web` package
**Scale/Scope**: Interface definition + example implementation on existing plugin

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution is a template without project-specific rules. Applying standard Go best practices:

| Principle | Status | Notes |
|-----------|--------|-------|
| Interface segregation | PASS | Searchable is separate from WebPlugin |
| Package dependencies | PASS | New search package has no web dependencies |
| Testability | PASS | Interface enables mock implementations |
| Simplicity | PASS | Minimal interface with single method |

## Project Structure

### Documentation (this feature)

```text
specs/010-searchable-interface/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (Go interface definitions)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── search/              # NEW: Searchable interface package
│   ├── search.go        # SearchResult, SortOrder, Searchable interface
│   └── search_test.go   # Interface contract tests
├── web/
│   └── plugin.go        # Existing WebPlugin (unchanged)
└── webplugins/
    ├── profiles/
    │   └── plugin.go    # Add Searchable implementation
    └── memory/
        └── plugin.go    # Add Searchable implementation
```

**Structure Decision**: New `internal/search` package at the same level as `internal/web` to ensure zero coupling. The search package is imported by webplugins that want to implement Searchable.

## Complexity Tracking

No violations to justify. Design is minimal:
- One new package (`internal/search`)
- One interface (`Searchable`)
- One result type (`SearchResult`)
- One enum type (`SortOrder`)
