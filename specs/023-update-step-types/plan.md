# Implementation Plan: Update Step Types

**Branch**: `023-update-step-types` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/023-update-step-types/spec.md`

## Summary

Update the available workflow steps from the current set (init, specify, plan, implement, etc.) to a streamlined set of nine steps: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete. The feature step will use the existing specify workflow (research-create-audit-highlight phases). Legacy steps (init, specify, implement) will be removed entirely with no backwards compatibility.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2, mark3labs/mcp-go, gopkg.in/yaml.v3, adrg/frontmatter
**Storage**: File-based (YAML frontmatter in markdown, JSON for state)
**Testing**: testify/assert, testify/suite
**Target Platform**: CLI tool, cross-platform (macOS, Linux, Windows)
**Project Type**: Single CLI application
**Performance Goals**: N/A (interactive CLI tool)
**Constraints**: Must maintain compatibility with existing initiative folder structure
**Scale/Scope**: Single-user CLI tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. General Best Practices | PASS | Code follows existing patterns, meaningful names, small functions |
| II. Go Development Standards | PASS | Using urfave/cli, error context, no panic in non-test code |
| III. Testing Discipline | PASS | Tests exist for step package, will add tests for new steps |
| IV. Database Standards | N/A | No database changes in this feature |
| Technology Stack | PASS | Using approved stack (Go, urfave/cli, testify) |
| Development Workflow | PASS | Constitution check done, tests before implementation |

**Gate Result**: PASS - No violations, proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/023-update-step-types/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── step/
│   ├── types.go         # Step, StepSource, StepResponse, Phase types
│   ├── service.go       # Step execution logic (modify for new steps)
│   ├── loader.go        # Step definition loading from templates
│   ├── feature.go       # Feature step handler (already exists)
│   ├── bug.go           # NEW: Bug step handler
│   ├── refactor.go      # NEW: Refactor step handler
│   └── *_test.go        # Tests for each handler
├── initiative/
│   ├── types.go         # Initiative, Cycle, State types
│   ├── service.go       # Initiative management
│   └── cycle.go         # Cycle management
└── mcp/tools/step/
    └── tool.go          # MCP tool for step execution

templates/steps/
├── feature.md           # Feature step directive (rename from current)
├── bug.md               # Bug step directive (NEW)
├── refactor.md          # Refactor step directive (NEW)
├── plan.md              # Plan step directive (exists)
├── tasks.md             # Tasks step directive (exists)
├── eat.md               # Eat step directive (rename implement.md)
├── audit.md             # Audit step directive (exists)
├── clarify.md           # Clarify step directive (exists)
└── complete.md          # Complete step directive (exists)
```

**Structure Decision**: Single project layout. Modifications primarily in `internal/step/` package with corresponding template updates in `templates/steps/`.

## Complexity Tracking

No constitution violations to justify.
