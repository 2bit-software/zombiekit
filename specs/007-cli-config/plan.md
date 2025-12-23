# Implementation Plan: CLI Configuration System

**Branch**: `007-cli-config` | **Date**: 2025-12-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-cli-config/spec.md`

## Summary

Add a layered configuration system for the brains CLI that loads settings from global config, local config, and CLI flags with proper precedence (CLI > local > global > defaults). Initial configuration supports enabling/disabling MCP tools individually or by category.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI), BurntSushi/toml (TOML parsing - already indirect dep), slog (logging)
**Storage**: TOML files at `.brains/config.toml` (local), `~/.config/brains/config.toml` (global Unix), `%APPDATA%\brains\config.toml` (Windows)
**Testing**: Go standard testing + testify assertions (already in use)
**Target Platform**: Cross-platform (Linux, macOS, Windows)
**Project Type**: Single CLI application
**Performance Goals**: Config loading < 10ms (file I/O only, no network)
**Constraints**: No external network calls, graceful degradation on missing/invalid configs
**Scale/Scope**: Single-user CLI tool, ~4 MCP tools to manage

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains template placeholders only. Applying standard Go CLI best practices:

| Gate | Status | Notes |
|------|--------|-------|
| Library-First | PASS | Config loading as standalone `internal/config` package |
| CLI Interface | PASS | Exposes `--enable-tool` and `--disable-tool` flags |
| Test-First | PASS | Unit tests for config parsing, integration tests for precedence |
| Integration Testing | PASS | Test config file loading with real filesystem |
| Observability | PASS | Debug logging for loaded config paths (FR-013) |
| Simplicity | PASS | TOML parsing with standard library patterns, no over-engineering |

No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/007-cli-config/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A for this feature - no APIs)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── config/
│   ├── config.go        # Existing - extend with TOML loading
│   ├── loader.go        # NEW: Config file discovery and loading
│   ├── loader_test.go   # NEW: Loader unit tests
│   ├── merger.go        # NEW: Precedence-based config merging
│   ├── merger_test.go   # NEW: Merger unit tests
│   ├── storage.go       # Existing storage config
│   └── tools.go         # NEW: Tool enable/disable logic
│   └── tools_test.go    # NEW: Tool config tests
├── cli/
│   └── serve.go         # Modify: Add --enable-tool/--disable-tool flags
├── mcp/
│   └── server.go        # Modify: Use config to filter registered tools
```

**Structure Decision**: Extends existing `internal/config` package with new files for loader, merger, and tool configuration. Minimal changes to existing CLI and MCP server code.

## Complexity Tracking

No violations to justify - implementation follows existing patterns.
