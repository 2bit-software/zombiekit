# Research: Update Step Types & MCP Tool Interface

**Date**: 2025-12-24 (Updated)
**Feature**: 023-update-step-types

## Executive Summary

The feature has evolved from a simple step renaming to a more significant architectural change: splitting the `step` MCP tool into two focused tools:

1. **`initiative` tool** - Handles lifecycle (create, status, complete, list)
2. **`step` tool** - Handles workflow execution (simplified, no creation params)

This separation addresses the interface overloading problem where creation steps required different parameters than execution steps.

## Findings by Category

### MCP Tool Interface Problem

**Problem**: The current `step` tool is overloaded:
- Creation steps (feature/bug/refactor) need: type, name, description
- Execution steps (plan/tasks/eat/audit/clarify) need: active initiative context
- Termination (complete) needs: active initiative

**Solution**: Split into two tools with clear responsibilities:

| Tool | Purpose | Parameters |
|------|---------|------------|
| `initiative` | Lifecycle CRUD | action, dir, type, name, description |
| `step` | Workflow execution | step, dir, initiative (optional override) |

**Rationale**: Clear separation of concerns, better parameter validation, simpler mental model for LLM callers.

### Initiative Tool Design

**Decision**: Action-based interface (like REST verbs)

```go
// Request
type InitiativeRequest struct {
    Action      string // create | status | complete | list
    Dir         string
    Type        string // For create: feature | bug | refactor
    Name        string // For create
    Description string // For create (optional)
}

// Responses vary by action
```

**Rationale**: Standard pattern, LLMs handle it well (similar to TodoWrite's status enum).

### Step Tool Simplification

**Decision**: Remove all creation-related parameters

**Current params** (before):
- step, dir, initiative, type, name, description, new_initiative, phase

**New params** (after):
- step, dir, initiative (optional override)

**Rationale**: Steps no longer create initiatives - they only execute within existing ones.

### Creation Logic Migration

**Decision**: Move from `internal/step/feature.go` to `internal/initiative/service.go`

**Operations to move**:
- Initiative folder creation
- Git branch creation
- Cycle folder creation
- Template copying

**Operations that stay in step service**:
- Step directive loading
- Profile composition
- File resolution
- Prerequisite checking

### Step Type Changes

**Final list** (8 steps):
1. `feature` - Feature specification workflow
2. `bug` - Bug investigation workflow
3. `refactor` - Refactor planning workflow
4. `plan` - Implementation planning
5. `tasks` - Task generation
6. `eat` - Implementation execution
7. `audit` - Cross-artifact alignment
8. `clarify` - Underspecification identification

**Removed**:
- `complete` → Now `initiative(action="complete")`
- `init` → Legacy, removed entirely
- `specify` → Merged into feature step
- `implement` → Renamed to `eat`

### Prerequisite Enforcement

**Decision**: Keep in step service, require active initiative for ALL steps

```go
func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
    // First: check active initiative
    state, err := s.stateManager.Load()
    if state.IsEmpty() {
        return nil, &StepError{Code: "NO_ACTIVE_INITIATIVE", ...}
    }

    // Second: check step-specific prerequisites
    if err := s.checkPrerequisite(stepName, cyclePath); err != nil {
        return nil, err
    }

    // Then: execute step
}
```

### Files to Modify/Create

**New files**:
- `internal/mcp/tools/initiative/tool.go` - Initiative MCP tool
- `internal/mcp/tools/initiative/tool_test.go` - Tests
- `internal/mcp/tools/initiative/types.go` - Request/response types

**Modified files**:
- `internal/mcp/tools/step/tool.go` - Remove creation params
- `internal/step/service.go` - Remove creation logic, add initiative check
- `internal/step/feature.go` - Remove initiative/cycle creation
- `internal/step/types.go` - Simplify ExecuteOptions
- `internal/initiative/service.go` - Add Create, CreateCycle methods
- `internal/mcp/server.go` - Register initiative tool

**Deleted**:
- `templates/steps/init.md`
- `templates/steps/specify.md`
- `templates/steps/complete.md` (now initiative action)

**Renamed**:
- `templates/steps/implement.md` → `templates/steps/eat.md`

## Alternatives Considered

### Alternative 1: Keep single step tool with conditional validation
**Rejected**: Makes the interface confusing. LLM has to know which params are valid for which steps. Error messages are unclear.

### Alternative 2: Three tools (initiative-create, step, initiative-complete)
**Rejected**: Over-splitting. The action-based `initiative` tool handles all lifecycle operations cleanly.

### Alternative 3: Resource-oriented design with separate context tool
**Rejected**: Over-engineered. Would require three tools and more round-trips.

### Initiative Naming & Uniqueness

**Decision**: Initiatives are prefixed with a unique hex ID (e.g., `abc123-user-auth`)

**Rationale**: Name collisions cannot occur - duplicate names are allowed because the hex ID ensures folder uniqueness. No need for collision detection or auto-suffixing.

### Force Flag for Active Initiative

**Decision**: No force flag - user must complete or abandon current initiative first

**Rationale**: Keep interface simple. The explicit workflow (complete/abandon before create) prevents accidental data loss and makes the state machine clear.

## Open Questions

None - all clarifications resolved in spec/clarify phases.

## Sources

- Audit session 2025-12-24 - Tool interface design discussion
- Systems architect agent analysis
- `internal/step/service.go` - Current step execution logic
- `internal/step/feature.go` - Current creation logic to migrate
- `internal/mcp/tools/step/tool.go` - Current MCP interface to simplify
- SpecKit shell scripts - Reference for creation workflow
