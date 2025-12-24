# Implementation Plan: Update Step Types & MCP Tool Interface

**Branch**: `023-update-step-types` | **Date**: 2025-12-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/023-update-step-types/spec.md`
**Updated**: 2025-12-24 (post-analysis resolution)

## Summary

Split the overloaded `step` MCP tool into two focused tools:
1. **`initiative` tool** - Lifecycle management (create, status, complete, list)
2. **`step` tool** - Workflow execution (feature, bug, refactor, plan, tasks, eat, audit, clarify)

This separation provides clearer interfaces, better parameter validation, and aligns with the principle that initiative lifecycle and step execution are orthogonal concerns.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2, mark3labs/mcp-go, gopkg.in/yaml.v3, adrg/frontmatter
**Storage**: File-based (YAML frontmatter in markdown, JSON for state in `.brains/active.json`)
**Testing**: testify/assert, testify/suite
**Target Platform**: CLI tool, cross-platform (macOS, Linux, Windows)
**Project Type**: Single CLI application
**Performance Goals**: N/A (interactive CLI tool)
**Constraints**: Must maintain compatibility with existing initiative folder structure in `history/`
**Scale/Scope**: Single-user CLI tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. General Best Practices | PASS | Code follows existing patterns, small focused functions |
| II. Go Development Standards | PASS | Using urfave/cli, proper error context, no panic |
| III. Testing Discipline | PASS | Tests exist for step package, will add tests for new initiative tool |
| IV. Database Standards | N/A | No database changes in this feature |
| Technology Stack | PASS | Using approved stack (Go, urfave/cli, mcp-go, testify) |
| Development Workflow | PASS | Constitution check done, tests before implementation |

**Gate Result**: PASS - No violations, proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/023-update-step-types/
├── plan.md              # This file
├── research.md          # Phase 0 output (complete)
├── data-model.md        # Phase 1 output (complete, updated with NextTask)
├── quickstart.md        # Phase 1 output (complete)
├── contracts/           # Phase 1 output (complete, updated with eat step response)
│   ├── initiative-tool.md
│   └── step-tool.md
└── tasks.md             # Phase 2 output (complete, 72 tasks)
```

### Source Code (repository root)

```text
internal/
├── initiative/
│   ├── types.go         # Initiative, Cycle, State types (EXISTS)
│   ├── service.go       # Initiative CRUD operations (EXISTS, extend)
│   └── state.go         # State management (EXISTS)
├── step/
│   ├── types.go         # Step, StepResponse types (EXISTS, simplify)
│   ├── service.go       # Step execution logic (EXISTS, simplify)
│   ├── loader.go        # Step definition loading (EXISTS)
│   ├── prereq.go        # NEW: YAML frontmatter status parser
│   ├── feature.go       # Feature step handler (REMOVE creation logic)
│   └── *_test.go        # Tests
└── mcp/
    └── tools/
        ├── initiative/
        │   ├── tool.go      # NEW: Initiative MCP tool
        │   ├── types.go     # NEW: Request/response types
        │   └── tool_test.go # NEW: Tests
        └── step/
            ├── tool.go      # EXISTS: Simplify (remove creation params)
            └── tool_test.go # EXISTS: Update tests

templates/steps/
├── feature.md           # Feature step directive (EXISTS)
├── bug.md               # Bug step directive (NEW)
├── refactor.md          # Refactor step directive (NEW)
├── plan.md              # Plan step directive (EXISTS)
├── tasks.md             # Tasks step directive (EXISTS)
├── eat.md               # Eat step directive (RENAMED from implement.md)
├── audit.md             # Audit step directive (EXISTS)
└── clarify.md           # Clarify step directive (EXISTS)
```

**Structure Decision**: Single project layout. Create new `internal/mcp/tools/initiative/` package and `internal/step/prereq.go`. Simplify existing `internal/mcp/tools/step/` and `internal/step/` packages by moving creation logic to initiative tool.

## Key Design Decisions

### 1. Tool Separation

The current `step` tool conflates two concerns:
- **Initiative lifecycle** (create container, check status, complete)
- **Step execution** (run workflow phases within container)

Splitting provides:
- Clear parameter validation per tool
- Better error messages
- Simpler mental model for LLM callers
- Easier extensibility (new initiative types vs new steps are separate)

### 2. Initiative Tool Interface

```go
InputSchema: {
    "action": enum["create", "status", "complete", "list"],
    "dir": string,           // Required - working directory
    "type": enum["feature", "bug", "refactor"],  // Required for create
    "name": string,          // Required for create
    "description": string,   // Optional for create
}
```

### 3. Simplified Step Tool Interface

```go
InputSchema: {
    "step": string,          // Required - step name
    "dir": string,           // Required - working directory
    "initiative": string,    // Optional - override active initiative
}
```

Removed parameters: `type`, `name`, `description`, `new_initiative`, `phase`

### 4. Approval Detection Mechanism

Prerequisites requiring "approved" status are validated by reading YAML frontmatter from artifact files:

```yaml
---
status: approved
approved_by: user
approved_date: 2025-12-24
---
```

The new `internal/step/prereq.go` implements this parser.

### 5. Task Progress Tracking (eat step)

The eat step identifies the next incomplete task by:
1. Reading tasks.md
2. Finding the first unchecked checkbox (`- [ ]`)
3. Returning task info in `NextTask` field of response

The eat step does NOT mutate tasks.md - the agent marks tasks complete as it works.

### 6. Clarification Encoding

The clarify step appends Q/A pairs to the Clarifications section of artifacts with session date headers (e.g., "### Session 2025-12-24").

### 7. Response Types

**Initiative responses** (new):
```go
type InitiativeCreateResponse struct {
    Action           string `json:"action"`
    InitiativeID     string `json:"initiative_id"`
    InitiativePath   string `json:"initiative_path"`
    CycleID          string `json:"cycle_id"`
    CyclePath        string `json:"cycle_path"`
    Branch           string `json:"branch"`
    Type             string `json:"type"`
    Name             string `json:"name"`
    NextStep         string `json:"next_step"`
}

type InitiativeStatusResponse struct {
    Action           string   `json:"action"`
    Active           bool     `json:"active"`
    InitiativeID     string   `json:"initiative_id,omitempty"`
    InitiativeType   string   `json:"initiative_type,omitempty"`
    CurrentStep      string   `json:"current_step,omitempty"`
    CycleID          string   `json:"cycle_id,omitempty"`
    AvailableDocs    []string `json:"available_docs,omitempty"`
    SuggestedNext    string   `json:"suggested_next,omitempty"`
}
```

**Step response** (updated with NextTask):
```go
type StepResponse struct {
    Step             string           `json:"step"`
    Directive        string           `json:"directive"`
    InitiativeFolder string           `json:"initiative_folder"`
    CycleFolder      string           `json:"cycle_folder"`
    FilesToRead      []string         `json:"files_to_read"`
    ComposedPrompt   string           `json:"composed_prompt"`
    Prerequisites    PrerequisiteInfo `json:"prerequisites"`
    WorkflowPhases   []Phase          `json:"workflow_phases,omitempty"`
    NextTask         *TaskInfo        `json:"next_task,omitempty"`  // For eat step
}

type TaskInfo struct {
    ID          string `json:"id"`
    Description string `json:"description"`
    Phase       string `json:"phase"`
}
```

## Complexity Tracking

No constitution violations to justify.

## Migration Path

1. Add `initiative` tool alongside existing `step` tool
2. Create `internal/step/prereq.go` for frontmatter status parsing
3. Extract creation logic from `step/feature.go` to `initiative/service.go`
4. Simplify `step` tool schema (remove creation params)
5. Update MCP server registration to include both tools
6. Add next task detection to eat step handler
7. Update tests
8. Remove deprecated code paths from step tool

## Phase 0 Artifacts

- [x] `research.md` - Complete, all clarifications resolved

## Phase 1 Artifacts

- [x] `data-model.md` - Complete, updated with Phase/TaskInfo/NextTask types
- [x] `contracts/initiative-tool.md` - Complete
- [x] `contracts/step-tool.md` - Complete, updated with eat step NextTask response
- [x] `quickstart.md` - Complete

## Ready for Implementation

All design artifacts complete. Run `/speckit.implement` to begin Phase 2 task execution.

**Task Count**: 72 tasks across 15 phases
**Critical Path**: Setup → Foundational → US1 → US4 → US5 → US6 → US9 → Polish
