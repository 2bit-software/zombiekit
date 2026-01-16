# Implementation Plan: Initiative Feature Workflow

**Branch**: `022-initiative-feature-workflow` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/022-initiative-feature-workflow/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement the "feature" step for the ZombieKit initiative framework. This step creates initiative folders with cycles, copies templates, manages git branches, and returns multi-phase directives that guide LLMs through a research→create→audit→highlight workflow.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2 (CLI), mark3labs/mcp-go v0.43.2 (MCP), adrg/frontmatter (YAML parsing), gopkg.in/yaml.v3
**Storage**: File-based (`.brains/active.json` for state, `./history/` for initiatives, embedded filesystem for templates)
**Testing**: go test with testify/stretchr, testcontainers for integration
**Target Platform**: CLI tool, cross-platform (Darwin, Linux, Windows)
**Project Type**: Single project with internal/ packages
**Performance Goals**: <2 seconds for initiative creation (per SC-001)
**Constraints**: Stateless CLI (no LLM calls), synchronous file operations, atomic state updates
**Scale/Scope**: Single-user CLI tool, local file operations only

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: The project constitution (`.specify/memory/constitution.md`) contains template placeholders. Applying common Go project best practices as gates:

| Gate | Status | Notes |
|------|--------|-------|
| Library-first design | ✅ PASS | Feature extends internal/initiative and internal/step packages |
| Interface boundaries | ✅ PASS | Uses existing service interfaces, MCP tool patterns |
| Test coverage | ✅ PASS | Will include unit tests for new cycle logic |
| Error handling | ✅ PASS | Uses structured error types (InitiativeError, StepError) |
| No breaking changes | ✅ PASS | Extends existing types, adds new functionality |
| Simplicity (YAGNI) | ✅ PASS | Builds on existing patterns, minimal new abstractions |

## Project Structure

### Documentation (this feature)

```text
specs/022-initiative-feature-workflow/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── initiative/
│   ├── types.go         # Add Cycle type, extend InitiativeState
│   ├── service.go       # Add CreateCycle, GetActiveCycle methods
│   ├── state.go         # Extend for cycle tracking
│   └── cycle.go         # NEW: Cycle management logic
├── step/
│   ├── types.go         # Extend StepResponse with cycle info
│   ├── service.go       # Add feature step handling
│   ├── feature.go       # NEW: Feature step implementation
│   └── loader.go        # Unchanged
└── mcp/tools/step/
    └── tool.go          # Extend for feature step parameters

templates/
├── steps/
│   └── feature.md       # NEW: Feature step definition
└── templates/
    ├── spec-template.md     # Existing (used by feature step)
    ├── research-template.md # NEW: Research output template
    └── audit/               # NEW: Audit directory structure

cmd/brains/
└── main.go              # Register new embedded templates

.claude/commands/
└── brains.feature.md    # NEW: Claude Code skill for feature workflow
```

**Structure Decision**: Extends existing Go package structure in `internal/`. New functionality added to existing packages (initiative, step) with minimal new files. Templates added to embedded filesystem.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. All gates pass.

## Post-Design Constitution Re-Check

| Gate | Status | Notes |
|------|--------|-------|
| Library-first design | ✅ PASS | Cycle management in initiative package, git service in step package |
| Interface boundaries | ✅ PASS | StepResponse extended, CycleType/CycleStatus enums defined |
| Test coverage | ✅ PASS | Unit tests for cycle.go, feature.go, git.go planned in quickstart |
| Error handling | ✅ PASS | Uses structured errors (InitiativeError, StepError, CycleError) |
| No breaking changes | ✅ PASS | history_folder preserved for backward compatibility |
| Simplicity (YAGNI) | ✅ PASS | Minimal new types (Cycle, CycleType, Phase), reuses copyEmbeddedFiles |

## Generated Artifacts

| Artifact | Path | Purpose |
|----------|------|---------|
| research.md | `specs/022-initiative-feature-workflow/research.md` | Research findings and decisions |
| data-model.md | `specs/022-initiative-feature-workflow/data-model.md` | Entity definitions and relationships |
| mcp-step-tool.md | `specs/022-initiative-feature-workflow/contracts/mcp-step-tool.md` | MCP tool API contract |
| step-directive.md | `specs/022-initiative-feature-workflow/contracts/step-directive.md` | Directive format specification |
| quickstart.md | `specs/022-initiative-feature-workflow/quickstart.md` | Implementation guide |

## Next Steps

Run `/speckit.tasks` to generate implementation tasks from this plan.
