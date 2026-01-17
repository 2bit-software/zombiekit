# Implementation Plan: Profile-MCP Integration

**Branch**: `024-profile-mcp-integration` | **Date**: 2025-12-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/024-profile-mcp-integration/spec.md`

## Summary

Update embedded step profiles (`templates/steps/*.md`) to work correctly with the `initiative` and `step` MCP tools. The profiles must handle the structured JSON response from the Go backend, provide clear phase-by-phase directives for multi-phase workflows, and ensure agents can execute complete workflows using only MCP responses.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2, mark3labs/mcp-go, gopkg.in/yaml.v3, adrg/frontmatter
**Storage**: File-based (YAML frontmatter in markdown, JSON for state in `.brains/active.json`)
**Testing**: go test with testify/assert, testify/suite
**Target Platform**: CLI tool (macOS, Linux)
**Project Type**: Single project with embedded templates
**Performance Goals**: N/A (developer CLI tool, single-user)
**Constraints**: Profiles must be loadable from embedded FS, global (~/.brains/steps/), and local (.brains/steps/)
**Scale/Scope**: 8 step profiles, ~20 files modified

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. General Best Practices** | | |
| Meaningful variable names | ✅ Pass | Existing code follows convention |
| Comments explain WHY | ✅ Pass | Update profiles with clear rationale |
| Single responsibility | ✅ Pass | Each step profile handles one workflow |
| Prioritize readability | ✅ Pass | Markdown profiles are human-readable |
| **II. Go Development Standards** | | |
| Use `any` not `interface{}` | ✅ Pass | Existing code compliant |
| Error handling with context | ✅ Pass | StepError/ToolError types used |
| Use context for cancellation | ✅ Pass | Execute methods take context |
| Use urfave/cli for CLI | ✅ Pass | Already using urfave/cli/v2 |
| No panic in non-test code | ✅ Pass | No must functions used |
| **III. Testing Discipline** | | |
| Use testify/assert/suite | ✅ Pass | Tests use testify |
| Test-first mindset | ⚠️ Follow | Tests must be written before profile changes |
| **IV. Database Standards** | N/A | No database changes in this feature |

**Gate Status**: ✅ PASS - No violations. Proceed to Phase 0.

### Post-Design Re-Check (Phase 1 Complete)

| Principle | Status | Notes |
|-----------|--------|-------|
| No feature creep | ✅ Pass | Changes limited to profile content and documented contracts |
| Architecture simplicity | ✅ Pass | No new abstractions; using existing Step/StepResponse types |
| Composition over inheritance | ✅ Pass | Profiles compose via `profiles` field |
| Test-first | ⚠️ Reminder | Implementation must write tests before profile changes |

**Post-Design Gate Status**: ✅ PASS

## Project Structure

### Documentation (this feature)

```text
specs/024-profile-mcp-integration/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (MCP tool contracts)
│   ├── initiative-tool.md
│   └── step-tool.md
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── step/
│   ├── types.go              # Step, StepResponse, Phase types
│   ├── service.go            # Step execution service
│   ├── feature.go            # buildWorkflowPhases()
│   ├── loader.go             # Step loading from embedded/global/local
│   └── *_test.go             # Tests
├── mcp/
│   ├── server.go             # MCP server registration
│   └── tools/
│       ├── step/tool.go      # Step MCP tool
│       └── initiative/tool.go # Initiative MCP tool
└── initiative/
    ├── service.go            # Initiative lifecycle
    ├── state.go              # State management
    └── cycle.go              # Cycle management

templates/
└── steps/                    # Embedded step profiles (PRIMARY CHANGE AREA)
    ├── feature.md            # Multi-phase: research→create→audit→highlight
    ├── bug.md                # Multi-phase workflow for bugs
    ├── refactor.md           # Multi-phase workflow for refactors
    ├── plan.md               # Single-phase: create implementation plan
    ├── tasks.md              # Single-phase: generate task list
    ├── eat.md                # Single-phase: execute tasks
    ├── audit.md              # Single-phase: cross-artifact audit
    └── clarify.md            # Single-phase: ambiguity detection
```

**Structure Decision**: Single project layout. Changes focus on `templates/steps/*.md` profiles with supporting updates to `internal/step/` types and MCP tools.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. No complexity justifications required.

## Implementation Phases

### Phase 0: Profile Content Updates (P0)

Update profiles with major gaps:

1. **plan.md** - Add structured directive with constitution check, phases, output sections
2. **tasks.md** - Add structured directive with dependency format, parallel markers, TDD guidance

### Phase 1: Profile Enhancements (P1)

Update profiles with moderate gaps:

3. **eat.md** - Add `next_task` response handling, task-by-task implementation guidance
4. **audit.md** - Add severity classification (CRITICAL/MAJOR/MINOR/INFO), cross-artifact alignment
5. **clarify.md** - Add ambiguity taxonomy, question format, integration rules

### Phase 2: Profile Polish (P2)

Update profiles with minor gaps:

6. **feature.md** - Add "Response Handling" section, verify phase structure matches Go code
7. **bug.md** - Mirror feature.md improvements, ensure classification workflow clear
8. **refactor.md** - Mirror feature.md improvements, ensure before/after format clear

### Phase 3: Testing (P3)

9. Add profile loading tests to `internal/step/service_test.go`
10. Add response validation tests to `internal/mcp/tools/step/tool_test.go`

## Artifacts Generated

| Artifact | Location | Status |
|----------|----------|--------|
| research.md | specs/024-profile-mcp-integration/ | ✅ Complete |
| data-model.md | specs/024-profile-mcp-integration/ | ✅ Complete |
| contracts/initiative-tool.md | specs/024-profile-mcp-integration/contracts/ | ✅ Complete |
| contracts/step-tool.md | specs/024-profile-mcp-integration/contracts/ | ✅ Complete |
| quickstart.md | specs/024-profile-mcp-integration/ | ✅ Complete |

## Next Step

Run `/speckit.tasks` to generate the detailed task breakdown.
