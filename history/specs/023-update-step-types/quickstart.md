# Quickstart: Update Step Types & MCP Tool Interface

## Overview

This feature splits the overloaded `step` MCP tool into two focused tools:

1. **`initiative` tool** - Lifecycle management (create, status, complete, list)
2. **`step` tool** - Workflow execution (feature, bug, refactor, plan, tasks, eat, audit, clarify)

## Key Changes

### MCP Tools

**New tool: `initiative`**
- `action=create` - Creates initiative with folder, git branch, templates
- `action=status` - Returns active initiative info
- `action=complete` - Marks initiative done, clears state
- `action=list` - Returns all initiatives

**Simplified tool: `step`**
- Removed: `type`, `name`, `description`, `new_initiative` params
- Now only accepts: `step`, `dir`, optional `initiative` override

### Step Types

- **8 steps** (down from 9): feature, bug, refactor, plan, tasks, eat, audit, clarify
- **Removed**: `complete` (now `initiative complete`)
- **Also removed**: `init`, `specify`, `implement` (legacy)

## Typical Workflow

```bash
# 1. Create initiative (new tool)
initiative(action="create", dir=".", type="feature", name="user-auth")
# Returns: initiative_id, paths, branch, next_step="feature"

# 2. Run specification step
step(step="feature", dir=".")
# Returns: directive, files, composed_prompt

# 3. Continue with planning
step(step="plan", dir=".")
# Returns: directive for planning

# 4. Generate tasks
step(step="tasks", dir=".")
# Returns: directive for task generation

# 5. Implement
step(step="eat", dir=".")
# Returns: directive for implementation

# 6. Complete initiative (new tool)
initiative(action="complete", dir=".")
# Returns: completed confirmation
```

## Implementation Tasks

### 1. Create Initiative MCP Tool

**New file: `internal/mcp/tools/initiative/tool.go`**

```go
func (t *Tool) Definition() ToolDefinition {
    return ToolDefinition{
        Name:        "initiative",
        Description: "Manage workflow initiatives. Actions: create, status, complete, list",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "action": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"create", "status", "complete", "list"},
                },
                "dir": map[string]interface{}{
                    "type": "string",
                },
                "type": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"feature", "bug", "refactor"},
                },
                "name": map[string]interface{}{
                    "type": "string",
                },
                "description": map[string]interface{}{
                    "type": "string",
                },
            },
            "required": []string{"action", "dir"},
        },
    }
}
```

### 2. Simplify Step MCP Tool

**Modify: `internal/mcp/tools/step/tool.go`**

```go
// Remove these properties from InputSchema:
// - type
// - name
// - description
// - new_initiative
// - phase

// Remove these from ExecuteOptions:
// - Type
// - Name
// - Description
// - NewInitiative
```

### 3. Move Creation Logic

**From: `internal/step/feature.go`**
**To: `internal/initiative/service.go`**

Move these operations to initiative service:
- `Create(type, name)` - folder creation, git branch
- `CreateCycle(initiative, type, name)` - cycle folder, templates
- Keep existing `Complete()` logic

### 4. Update Step Service

**Modify: `internal/step/service.go`**

```go
// Remove special handling for feature/bug/refactor creation
// They now just return directive + context like other steps

func (s *Service) Execute(stepName string, opts *ExecuteOptions) (*StepResponse, error) {
    // All steps now require active initiative
    state, err := s.stateManager.Load()
    if err != nil || state.IsEmpty() {
        return nil, &StepError{
            Code:    "NO_ACTIVE_INITIATIVE",
            Message: "no active initiative",
            Hint:    "Use 'initiative create' to start a new initiative",
        }
    }

    // Check prerequisites
    if err := s.checkPrerequisite(stepName, cyclePath); err != nil {
        return nil, err
    }

    // Load step and return context
    step, err := s.loader.Get(stepName)
    if err != nil {
        return nil, err
    }

    return &StepResponse{
        Step:             stepName,
        Directive:        step.Directive,
        InitiativeFolder: initiativeFolder,
        CycleFolder:      cyclePath,
        FilesToRead:      s.resolveFiles(step.Files, cyclePath),
        ComposedPrompt:   s.composeProfiles(step.Profiles),
        Prerequisites:    PrerequisiteInfo{Met: true},
    }, nil
}
```

### 5. Register Both Tools in MCP Server

**Modify: `internal/mcp/server.go`**

```go
import (
    initiativeTool "github.com/2bit-software/zombiekit/internal/mcp/tools/initiative"
    stepTool "github.com/2bit-software/zombiekit/internal/mcp/tools/step"
)

func (s *Server) registerTools() {
    // Initiative tool (NEW)
    s.RegisterTool(initiativeTool.NewTool())

    // Step tool (simplified)
    s.RegisterTool(stepTool.NewTool())

    // ... other tools
}
```

## Testing Checklist

### Initiative Tool
- [ ] `initiative create` creates folder, git branch, templates
- [ ] `initiative create` when one exists returns error
- [ ] `initiative status` returns active initiative info
- [ ] `initiative status` with no initiative returns empty
- [ ] `initiative complete` clears active state
- [ ] `initiative list` returns all initiatives

### Step Tool (Simplified)
- [ ] `step feature` requires active initiative
- [ ] `step plan` blocks if spec.md not approved
- [ ] `step tasks` blocks if plan.md not approved
- [ ] `step eat` blocks if tasks.md doesn't exist
- [ ] All 8 steps work with active initiative
- [ ] Old params (type, name, etc.) are ignored/error

### Integration
- [ ] Full workflow: create → feature → plan → tasks → eat → complete
- [ ] Skills (brains.feature, etc.) work with new tools

## File Changes Summary

| Action | File |
|--------|------|
| Create | internal/mcp/tools/initiative/tool.go |
| Create | internal/mcp/tools/initiative/tool_test.go |
| Create | internal/mcp/tools/initiative/types.go |
| Modify | internal/mcp/tools/step/tool.go |
| Modify | internal/mcp/tools/step/tool_test.go |
| Modify | internal/step/service.go |
| Modify | internal/step/feature.go |
| Modify | internal/step/types.go |
| Modify | internal/initiative/service.go |
| Modify | internal/mcp/server.go |
