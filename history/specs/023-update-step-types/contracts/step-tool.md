# MCP Tool Contract: step

**Tool Name**: `step`
**Purpose**: Execute workflow steps within an active initiative

## Input Schema (Simplified)

```json
{
  "type": "object",
  "properties": {
    "step": {
      "type": "string",
      "description": "Step name to execute: feature, bug, refactor, plan, tasks, eat, audit, clarify"
    },
    "dir": {
      "type": "string",
      "description": "Working directory containing the .brains folder"
    },
    "initiative": {
      "type": "string",
      "description": "Optional: Override active initiative path (relative to history/)"
    }
  },
  "required": ["step", "dir"]
}
```

**Removed parameters** (compared to previous version):
- `type` - Now handled by `initiative create`
- `name` - Now handled by `initiative create`
- `description` - Now handled by `initiative create`
- `new_initiative` - Replaced by `initiative create`
- `phase` - No longer exposed

## Response Schema

```json
{
  "step": "feature",
  "directive": "# Feature Specification Workflow\n\n...",
  "initiative_folder": "/path/to/history/abc123-user-auth",
  "cycle_folder": "/path/to/history/abc123-user-auth/abc124-feat-user-auth",
  "files_to_read": [
    "spec.md",
    "research.md"
  ],
  "composed_prompt": "You are working on a feature specification...",
  "prerequisites": {
    "met": true
  },
  "workflow_phases": [
    {
      "name": "research",
      "description": "Gather context and requirements",
      "agents": ["Explore"],
      "outputs": ["research.md"],
      "parallel": false
    }
  ],
  "next_task": null
}
```

### Eat Step Response (with NextTask)

For the `eat` step, the response includes `next_task` identifying the next incomplete task:

```json
{
  "step": "eat",
  "directive": "# Implementation Guidance\n\n...",
  "initiative_folder": "/path/to/history/abc123-user-auth",
  "cycle_folder": "/path/to/history/abc123-user-auth/abc124-feat-user-auth",
  "files_to_read": ["tasks.md", "plan.md"],
  "composed_prompt": "Implement the next task...",
  "prerequisites": {
    "met": true
  },
  "next_task": {
    "id": "T005",
    "description": "Add Create(type, name, dir) method to internal/initiative/service.go",
    "phase": "Phase 2: Foundational"
  }
}
```

When all tasks are complete:

```json
{
  "step": "eat",
  "directive": "All tasks complete! Run 'initiative complete' to finish.",
  "next_task": null
}
```

### Prerequisites Not Met Response

```json
{
  "step": "plan",
  "directive": "",
  "initiative_folder": "/path/to/history/abc123-user-auth",
  "cycle_folder": "/path/to/history/abc123-user-auth/abc124-feat-user-auth",
  "files_to_read": [],
  "composed_prompt": "",
  "prerequisites": {
    "met": false,
    "required": "spec.md with status: approved",
    "hint": "Complete the feature step and get spec approved before planning"
  }
}
```

## Error Responses

### NO_ACTIVE_INITIATIVE
```json
{
  "error": {
    "code": "NO_ACTIVE_INITIATIVE",
    "message": "No active initiative",
    "hint": "Use 'initiative create' to start a new initiative first"
  }
}
```

### UNKNOWN_STEP
```json
{
  "error": {
    "code": "UNKNOWN_STEP",
    "message": "Unknown step: 'init'",
    "hint": "Valid steps: feature, bug, refactor, plan, tasks, eat, audit, clarify"
  }
}
```

### PREREQUISITE_NOT_MET
```json
{
  "error": {
    "code": "PREREQUISITE_NOT_MET",
    "message": "Step 'plan' requires spec.md with status: approved",
    "hint": "Complete the feature step and get spec approved first"
  }
}
```

## Step Prerequisites

| Step | Requires | Required Status |
|------|----------|-----------------|
| feature | Active initiative | - |
| bug | Active initiative | - |
| refactor | Active initiative | - |
| plan | spec.md | approved |
| tasks | plan.md | approved |
| eat | tasks.md | exists |
| audit | Active initiative | - |
| clarify | Active initiative | - |

## Valid Step Names

- `feature` - Feature specification workflow (research, create, audit, highlight phases)
- `bug` - Bug investigation workflow (reproduction, root cause, fix spec)
- `refactor` - Refactor planning workflow (before/after, behavior preservation)
- `plan` - Implementation planning (architecture, components, approach)
- `tasks` - Task generation (ordered, dependency-tracked)
- `eat` - Implementation execution (task-by-task guidance)
- `audit` - Cross-artifact alignment check
- `clarify` - Underspecification identification

## Removed Step Names

These step names are no longer valid and will return `UNKNOWN_STEP`:
- `init` - Use `initiative create` instead
- `specify` - Use `feature` instead
- `implement` - Use `eat` instead
- `complete` - Use `initiative complete` instead

## Side Effects

| Step | Creates/Modifies |
|------|------------------|
| feature | Updates spec.md (agent fills content) |
| bug | Updates spec.md (agent fills content) |
| refactor | Updates spec.md (agent fills content) |
| plan | Creates/updates plan.md |
| tasks | Creates/updates tasks.md |
| eat | Modifies source code files |
| audit | Creates audit/{date}.md |
| clarify | Updates existing artifacts with clarifications |
