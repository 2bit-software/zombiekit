# Data Model: Update Step Types & MCP Tool Interface

**Date**: 2025-12-24
**Feature**: 023-update-step-types

## Entities

### Step (existing, simplified)

A workflow step definition loaded from templates. Now focused purely on workflow execution, not creation.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Step identifier: feature, bug, refactor, plan, tasks, eat, audit, clarify |
| Description | string | Human-readable description |
| Profiles | []string | Profile names to compose for context |
| Files | []string | Glob patterns for files to read |
| Directive | string | Instruction markdown for the step |
| Type | string | Always "step" for step definitions |
| Source | StepSource | Where loaded from (embedded, global, local) |
| Path | string | Absolute path if loaded from file |

**Removed Values for Name**: init, specify, implement, complete (complete is now an initiative action)

**Added Values for Name**: bug, refactor, eat (replaces implement)

### StepPrerequisite (existing)

Defines requirements that must be met before a step can execute.

| Field | Type | Description |
|-------|------|-------------|
| RequiredArtifact | string | File that must exist (e.g., "spec.md") |
| RequiredStatus | string | Optional status in frontmatter (e.g., "approved"). Detected via YAML frontmatter `status` field. |
| Hint | string | Guidance shown when prerequisite not met |
| BlockingStep | string | Name of step that produces the artifact |

**Approval Detection**: The prerequisite checker reads YAML frontmatter from the artifact file and validates that `status: approved` is present. Example frontmatter:

```yaml
---
status: approved
approved_by: user
approved_date: 2025-12-24
---
```

### Initiative (existing, unchanged)

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique hex-prefixed identifier (e.g., `abc123-user-auth`) - ensures name uniqueness |
| Type | InitiativeType | feature, bug, or refactor |
| Name | string | Human-readable name slug (duplicates allowed due to hex prefix) |
| Path | string | Absolute path to initiative folder |
| Status | InitiativeStatus | active or completed |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last activity timestamp |

### InitiativeState (existing, unchanged)

| Field | Type | Description |
|-------|------|-------------|
| Initiative | string | Relative path to active initiative |
| Cycle | string | Relative path to active cycle |
| Started | time.Time | When initiative became active |
| LastActivity | time.Time | Last step execution time |
| CurrentStep | string | Last executed step |

## MCP Tool Types (NEW)

### Initiative Tool Request/Response

```go
// internal/mcp/tools/initiative/types.go

type InitiativeRequest struct {
    Action      string `json:"action"`      // create | status | complete | list
    Dir         string `json:"dir"`         // Working directory
    Type        string `json:"type"`        // For create: feature | bug | refactor
    Name        string `json:"name"`        // For create: initiative name
    Description string `json:"description"` // For create: optional description
}

type InitiativeCreateResponse struct {
    Action         string `json:"action"`
    InitiativeID   string `json:"initiative_id"`
    InitiativePath string `json:"initiative_path"`
    CycleID        string `json:"cycle_id"`
    CyclePath      string `json:"cycle_path"`
    Branch         string `json:"branch"`
    Type           string `json:"type"`
    Name           string `json:"name"`
    NextStep       string `json:"next_step"` // "feature", "bug", or "refactor"
}

type InitiativeStatusResponse struct {
    Action         string   `json:"action"`
    Active         bool     `json:"active"`
    InitiativeID   string   `json:"initiative_id,omitempty"`
    InitiativeType string   `json:"initiative_type,omitempty"`
    CurrentStep    string   `json:"current_step,omitempty"`
    CycleID        string   `json:"cycle_id,omitempty"`
    AvailableDocs  []string `json:"available_docs,omitempty"`
    SuggestedNext  string   `json:"suggested_next,omitempty"`
}

type InitiativeCompleteResponse struct {
    Action       string    `json:"action"`
    InitiativeID string    `json:"initiative_id"`
    CompletedAt  time.Time `json:"completed_at"`
}

type InitiativeListResponse struct {
    Action      string              `json:"action"`
    Initiatives []InitiativeSummary `json:"initiatives"`
}

type InitiativeSummary struct {
    ID     string `json:"id"`
    Type   string `json:"type"`
    Name   string `json:"name"`
    Status string `json:"status"`
    Path   string `json:"path"`
}
```

### Step Tool Request/Response (Simplified)

```go
// internal/step/types.go (simplified from existing)

type StepRequest struct {
    Step       string `json:"step"`       // Required: step name
    Dir        string `json:"dir"`        // Required: working directory
    Initiative string `json:"initiative"` // Optional: override active initiative
}

type StepResponse struct {
    Step             string           `json:"step"`
    Directive        string           `json:"directive"`
    InitiativeFolder string           `json:"initiative_folder"`
    CycleFolder      string           `json:"cycle_folder"`
    FilesToRead      []string         `json:"files_to_read"`
    ComposedPrompt   string           `json:"composed_prompt"`
    Prerequisites    PrerequisiteInfo `json:"prerequisites"`
    WorkflowPhases   []Phase          `json:"workflow_phases,omitempty"` // Optional: step-specific phases
    NextTask         *TaskInfo        `json:"next_task,omitempty"`       // For eat step: next incomplete task
}

type PrerequisiteInfo struct {
    Met      bool   `json:"met"`
    Required string `json:"required,omitempty"`
    Hint     string `json:"hint,omitempty"`
}

type Phase struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Agents      []string `json:"agents,omitempty"`
    Outputs     []string `json:"outputs,omitempty"`
    Parallel    bool     `json:"parallel"`
}

type TaskInfo struct {
    ID          string `json:"id"`          // Task ID (e.g., "T005")
    Description string `json:"description"` // Task description
    Phase       string `json:"phase"`       // Phase the task belongs to
}
```

## State Transitions

### Initiative Lifecycle (NEW - via initiative tool)

```
[No Active Initiative]
        │
        │ initiative(action="create", type="feature", name="...")
        ▼
[Active Initiative]
        │
        │ initiative(action="complete")
        ▼
[Completed Initiative]
```

### Step Execution Flow (Simplified - no creation)

```
[Active Initiative]
        │
        ▼
    step feature/bug/refactor (specification guidance)
        │
        ▼
    step plan (requires approved spec)
        │
        ▼
    step tasks (requires approved plan)
        │
        ▼
    step eat (requires tasks.md)
        │
        ▼
    step audit/clarify (anytime)
```

### Prerequisite Validation States

| Current State | Step Requested | Prerequisite Check | Result |
|---------------|----------------|-------------------|--------|
| No initiative | Any step | No active initiative | BLOCK: "Run initiative create first" |
| Active initiative | feature | Type matches | ALLOW |
| Has spec (draft) | plan | spec.md not approved | BLOCK: "Approve spec first" |
| Has spec (approved) | plan | spec.md approved | ALLOW |
| Has plan (draft) | tasks | plan.md not approved | BLOCK: "Approve plan first" |
| Has plan (approved) | tasks | plan.md approved | ALLOW |
| No tasks | eat | No tasks.md | BLOCK: "Run tasks first" |
| Has tasks | eat | tasks.md exists | ALLOW |

## Validation Rules

### Initiative Tool

| Action | Required Params | Validation |
|--------|-----------------|------------|
| create | action, dir, type, name | type in (feature, bug, refactor); no active initiative |
| status | action, dir | None |
| complete | action, dir | Must have active initiative |
| list | action, dir | None |

### Step Tool

| Validation | Rule |
|------------|------|
| Step name | Must be: feature, bug, refactor, plan, tasks, eat, audit, clarify |
| Active initiative | Required for all steps |
| Prerequisites | Enforced per step (see table above) |

## File Artifacts

| Action/Step | Creates | Location |
|-------------|---------|----------|
| initiative create | INITIATIVE.md, spec.md, research.md | initiative folder, cycle folder |
| step feature | (fills spec.md) | cycle folder |
| step bug | (fills spec.md) | cycle folder |
| step refactor | (fills spec.md) | cycle folder |
| step plan | plan.md | cycle folder |
| step tasks | tasks.md | cycle folder |
| step eat | (code changes) | source tree |
| step audit | audit/{date}.md | cycle folder |
| step clarify | (updates existing artifacts) | cycle folder |
| initiative complete | (updates INITIATIVE.md status) | initiative folder |

## Relationships

```
Initiative 1---* Cycle
Cycle 1---* Artifact (spec.md, plan.md, tasks.md, etc.)
InitiativeState 1---1 Initiative (points to active)
Step *---* Profile (via profiles[] field)
MCP Tool --uses--> Initiative Service
MCP Tool --uses--> Step Service
```
