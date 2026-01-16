# Implementation Plan: Initiatives Step Framework

**Branch**: `021-initiatives-step-framework` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/021-initiatives-step-framework/spec.md`

## Summary

Implement an MCP tool endpoint (`mcp_zombiekit__step`) that orchestrates initiative-based development workflows. The tool accepts a step name and directory, returning a structured response with: (1) general directive, (2) history folder path, (3) files to read, and (4) composed profile prompt. This extends the existing MCP server infrastructure with a new `initiative` package for state management and a `step` package for step definitions.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: mark3labs/mcp-go v0.43.2 (MCP), urfave/cli/v2 (CLI), adrg/frontmatter (YAML parsing), gopkg.in/yaml.v3
**Storage**: File-based (`.brains/active.json` for state, `./history/` for initiatives, step definitions in `.brains/steps/`)
**Testing**: go test with testify/stretchr for assertions
**Target Platform**: Cross-platform CLI/MCP server
**Project Type**: Single project extending existing CLI
**Performance Goals**: Step execution under 2 seconds (per SC-001)
**Constraints**: Must integrate with existing profile system (`internal/profile`)
**Scale/Scope**: Single-user local development tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution file contains placeholder template - no specific gates defined. Proceeding with standard Go best practices:
- [x] Follow existing code patterns in the codebase
- [x] Use interfaces for testability
- [x] Write unit tests for new packages

## Project Structure

### Documentation (this feature)

```text
specs/021-initiatives-step-framework/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (MCP tool schema)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── initiative/               # NEW: Initiative management
│   ├── service.go            # Service interface and implementation
│   ├── service_test.go
│   ├── types.go              # Initiative, InitiativeState types
│   └── state.go              # State file (.brains/active.json) management
│
├── step/                     # NEW: Step definitions and execution
│   ├── service.go            # Step service interface
│   ├── service_test.go
│   ├── types.go              # Step, StepDefinition, StepResponse types
│   ├── loader.go             # Load step definitions from files
│   ├── loader_test.go
│   ├── defaults.go           # Built-in step definitions
│   └── defaults_test.go
│
├── mcp/
│   ├── server.go             # MODIFY: Add step tool registration
│   └── tools/
│       └── step/             # NEW: MCP step tool
│           ├── tool.go
│           └── tool_test.go
│
└── cli/
    └── initiative.go         # NEW: CLI commands for initiative management (optional)

history/                      # Created per-project by initiative service
└── {timestamp}-{type}-{name}/
    ├── INITIATIVE.md         # Initiative metadata
    └── {artifacts...}

.brains/
├── active.json               # Current initiative state (gitignored)
└── steps/                    # Custom step definitions (optional)
    └── {step-name}.md
```

**Structure Decision**: Extends existing single-project structure. New packages `internal/initiative` and `internal/step` follow the established pattern of domain packages (like `internal/profile`, `internal/memory`). MCP tool follows existing pattern in `internal/mcp/tools/`.

## Complexity Tracking

No constitution violations to justify.

## Implementation Phases

### Phase 0: Research (see research.md)

- Step definition format (reuse profile frontmatter pattern)
- State file format and locking strategy
- Initiative folder naming conventions
- Integration with existing profile composition

### Phase 1: Design (see data-model.md, contracts/)

- Define Initiative, Step, StepResponse types
- Define MCP tool schema
- Design step loader with defaults + custom overrides

### Phase 2: Tasks (see tasks.md)

Task breakdown for implementation.
