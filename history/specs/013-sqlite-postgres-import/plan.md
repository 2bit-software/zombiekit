# Implementation Plan: SQLite to PostgreSQL Migration Tool

**Branch**: `013-sqlite-postgres-import` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/013-sqlite-postgres-import/spec.md`

## Summary

This feature implements a one-way incremental migration tool for transferring memory data from SQLite to PostgreSQL. The tool tracks import history to support repeated migrations where only new/updated items since the last import are transferred. Key capabilities include preview/dry-run mode, progress reporting, conflict resolution via version comparison, and exclusive SQLite locking during import.

## Technical Context

**Language/Version**: Go 1.24.0
**Primary Dependencies**: urfave/cli/v2 (CLI), modernc.org/sqlite (SQLite), jackc/pgx/v5 (PostgreSQL)
**Storage**: SQLite (source, read-only), PostgreSQL (target, read-write with new import_metadata table)
**Testing**: go test with testcontainers-go for PostgreSQL integration tests
**Target Platform**: Linux/macOS/Windows (CLI tool)
**Project Type**: Single project (extends existing CLI)
**Performance Goals**: 1000 items imported in under 30 seconds (per SC-001)
**Constraints**: Exclusive lock on SQLite during import; read-only source; zero data loss on failure
**Scale/Scope**: Supports incremental imports over time; typical dataset 1-10k memory items

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution template is not yet populated with project-specific principles. Default engineering principles apply:
- [x] Code must be independently testable
- [x] Clear purpose for new package/commands
- [x] Test-first approach for core logic
- [x] Structured logging for observability

No violations identified.

## Project Structure

### Documentation (this feature)

```text
specs/013-sqlite-postgres-import/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI interface spec)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── memory/
│   ├── sqlite/storage.go      # Existing - read operations
│   ├── postgres/storage.go    # Existing - write operations
│   └── importer/              # NEW - import logic
│       ├── importer.go        # Core import service
│       ├── importer_test.go   # Unit tests
│       ├── metadata.go        # ImportMetadata operations
│       └── types.go           # ImportResult, ImportOptions
├── cli/
│   └── import.go              # NEW - CLI command handler
│   └── import_test.go         # CLI integration tests

tests/
└── integration/
    └── import_test.go         # End-to-end import tests with testcontainers
```

**Structure Decision**: Single project extending existing CLI with new `brains db import` subcommand. Import logic encapsulated in new `internal/memory/importer` package following existing patterns.

## Complexity Tracking

No violations requiring justification.
