# Implementation Plan: ZombieKit MCP Tool

**Branch**: `017-zombiekit-mcp` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/017-zombiekit-mcp/spec.md`

## Summary

Add a new "feature" tool to the existing brains MCP server that reads and returns the contents of `~/.brains/templates/step.feature.md`. This tool extends the existing tool registry pattern using mark3labs/mcp-go, following the established patterns in stickymemory and codereasoning tools.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: mark3labs/mcp-go (MCP server framework)
**Storage**: File system read-only (no database)
**Testing**: go test with table-driven tests
**Target Platform**: macOS, Linux
**Project Type**: Single project (existing CLI tool)
**Performance Goals**: <1 second response time (per SC-001)
**Constraints**: Must integrate with existing MCP server; no separate process
**Scale/Scope**: Single tool returning file contents

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains placeholder template content. No specific gates are defined. Proceeding with standard Go best practices:

- [x] Tool is self-contained and testable
- [x] Exposes functionality via MCP protocol (text in/out)
- [x] Tests will be written with implementation
- [x] Simple implementation (single file read operation)

## Project Structure

### Documentation (this feature)

```text
specs/017-zombiekit-mcp/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
internal/mcp/
├── server.go                    # Add zombiekit tool field, registration, handler
└── tools/
    └── zombiekit/
        ├── tool.go              # NEW: Tool struct, Definition(), Execute()
        └── tool_test.go         # NEW: Unit tests

internal/config/
└── tools.go                     # Add "feature" to KnownTools list
```

**Structure Decision**: Following the existing tool pattern (stickymemory, codereasoning), create a new `zombiekit` package under `internal/mcp/tools/`. The tool will be registered in `server.go` following the same pattern as other tools.

## Complexity Tracking

No violations requiring justification. This is a minimal feature following established patterns.
