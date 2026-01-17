# Implementation Plan: PostgreSQL Configuration with SQLite Fallback

**Branch**: `015-postgres-config` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/015-postgres-config/spec.md`

## Summary

Enable PostgreSQL connection configuration via TOML config files (`.brains/config.toml`) with automatic fallback to SQLite when PostgreSQL is unavailable. Environment variables continue to override config file values. The feature extends the existing `internal/config` package to read storage settings from TOML files and adds connection validation with timeout to the startup sequence.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: BurntSushi/toml (config parsing), jackc/pgx/v5 (PostgreSQL), modernc.org/sqlite (SQLite), urfave/cli/v2 (CLI)
**Storage**: PostgreSQL (primary when configured) or SQLite (default/fallback)
**Testing**: go test, testcontainers-go (integration tests with real PostgreSQL)
**Target Platform**: Linux/macOS/Windows CLI application
**Project Type**: Single project (CLI tool + MCP server)
**Performance Goals**: Connection timeout configurable (default 5 seconds), fallback should not block user
**Constraints**: Must not break existing environment variable configuration
**Scale/Scope**: Local development tool, single-user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution template has not been customized for this project, so no specific gates apply. Proceeding with standard Go best practices:

- [x] **Testability**: Feature can be unit tested with mock config and integration tested with testcontainers
- [x] **Simplicity**: Extends existing config system rather than creating new abstraction
- [x] **Observability**: Logging via slog for all configuration loading and fallback events

## Project Structure

### Documentation (this feature)

```text
specs/015-postgres-config/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API changes)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
internal/
├── config/
│   ├── config.go        # Extend Config struct with storage section
│   ├── storage.go       # MODIFY: Add config file loading, timeout field
│   ├── loader.go        # MODIFY: Add storage config loading from TOML
│   └── merger.go        # MODIFY: Add storage config merging
├── database/
│   └── postgres.go      # MODIFY: Add connection timeout support
├── cli/
│   └── serve.go         # MODIFY: Add fallback logic on PostgreSQL failure
└── web/
    └── status.go        # Already supports backend display (no changes needed)

tests/
├── integration/
│   └── config_fallback_test.go  # NEW: Test PostgreSQL unavailable fallback
└── unit/
    └── config/
        └── storage_test.go      # NEW: Test config file parsing
```

**Structure Decision**: Extends existing single-project structure. All changes within `internal/config`, `internal/database`, and `internal/cli` packages.

## Complexity Tracking

No constitution violations. Feature is a straightforward extension of existing patterns:
- Uses existing TOML parsing (BurntSushi/toml)
- Uses existing PostgreSQL connection (pgx/v5)
- Uses existing config merging pattern (global → local → env → CLI)
