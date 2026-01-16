# Data Model: Profile-MCP Integration

**Feature**: 024-profile-mcp-integration
**Date**: 2025-12-24

## Overview

This document describes the data entities involved in the profile-MCP integration. The primary change area is profile content (markdown), not Go types—existing types are sufficient.

## 1. Core Entities

### 1.1 Step

Represents a workflow step definition loaded from markdown with YAML frontmatter.

**Location**: `internal/step/types.go:32-50`

```go
type Step struct {
    Name        string     `json:"name"`
    Description string     `json:"description,omitempty"`
    Profiles    []string   `json:"profiles,omitempty"`
    Files       []string   `json:"files,omitempty"`
    Directive   string     `json:"directive"`
    Type        string     `json:"type,omitempty"`
    Source      StepSource `json:"-"`
    Path        string     `json:"-"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Step identifier (e.g., "feature", "plan") |
| Description | string | Human-readable description |
| Profiles | []string | Profile names to compose |
| Files | []string | Glob patterns for files_to_read |
| Directive | string | Markdown body—the agent instructions |
| Type | string | Always "step" for step definitions |
| Source | StepSource | embedded/global/local (not serialized) |
| Path | string | Filesystem path if loaded from file |

**Lifecycle**:
1. Loaded from embedded FS (`templates/steps/`)
2. Optionally overridden from global (`~/.brains/steps/`)
3. Optionally overridden from local (`.brains/steps/`)
4. Parsed via `frontmatter.Parse()`
5. Directive executed by agent

### 1.2 StepResponse

Structured output from step execution via MCP.

**Location**: `internal/step/types.go:66-87`

```go
type StepResponse struct {
    Directive        string           `json:"directive"`
    HistoryFolder    string           `json:"history_folder"`
    FilesToRead      []string         `json:"files_to_read"`
    ComposedPrompt   string           `json:"composed_prompt"`
    InitiativeFolder string           `json:"initiative_folder,omitempty"`
    CycleFolder      string           `json:"cycle_folder,omitempty"`
    WorkflowPhases   []Phase          `json:"workflow_phases,omitempty"`
    NextTask         *TaskInfo        `json:"next_task,omitempty"`
    Prerequisites    PrerequisiteInfo `json:"prerequisites,omitempty"`
}
```

| Field | Type | When Present | Description |
|-------|------|--------------|-------------|
| Directive | string | Always | Step instructions from profile |
| HistoryFolder | string | Always | Deprecated, use InitiativeFolder |
| FilesToRead | []string | Always | Resolved absolute paths |
| ComposedPrompt | string | When profiles defined | Concatenated profile content |
| InitiativeFolder | string | Always | Path to initiative folder |
| CycleFolder | string | Always | Path to active cycle folder |
| WorkflowPhases | []Phase | feature/bug/refactor | Phase definitions |
| NextTask | *TaskInfo | eat step | Next incomplete task |
| Prerequisites | PrerequisiteInfo | Always | Prerequisite status |

### 1.3 Phase

Represents a workflow phase in multi-phase steps.

**Location**: `internal/step/types.go:109-121`

```go
type Phase struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Agents      []string `json:"agents"`
    Outputs     []string `json:"outputs"`
    Parallel    bool     `json:"parallel"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Phase identifier (research, create, audit, highlight) |
| Description | string | What the phase does |
| Agents | []string | Suggested agent types to spawn |
| Outputs | []string | Expected artifacts |
| Parallel | bool | Whether agents can run in parallel |

**Current Phases** (feature step):
1. `research` - Parallel research agents → research.md
2. `create` - Single spec writer → spec.md
3. `audit` - Parallel audit agents → audit/{date}.md
4. `highlight` - Single highlighter → user approval

### 1.4 TaskInfo

Information about a task from tasks.md.

**Location**: `internal/step/types.go:89-97`

```go
type TaskInfo struct {
    ID          string `json:"id"`
    Description string `json:"description"`
    Phase       string `json:"phase"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Task identifier (e.g., "T005") |
| Description | string | Task description text |
| Phase | string | Phase/section from tasks.md headings |

**Extraction Logic**: First `- [ ]` checkbox in tasks.md, parsed from line content.

### 1.5 PrerequisiteInfo

Prerequisite check results.

**Location**: `internal/step/types.go:99-107`

```go
type PrerequisiteInfo struct {
    Met      bool   `json:"met"`
    Required string `json:"required,omitempty"`
    Hint     string `json:"hint,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| Met | bool | Whether all prerequisites satisfied |
| Required | string | What's required if not met |
| Hint | string | Guidance to satisfy prerequisite |

### 1.6 StepPrerequisite

Internal prerequisite definition.

**Location**: `internal/step/types.go:123-134`

```go
type StepPrerequisite struct {
    RequiredArtifact string
    RequiredStatus   string
    Hint             string
    BlockingStep     string
}
```

| Step | Artifact | Status | Blocking Step |
|------|----------|--------|---------------|
| plan | spec.md | approved | feature/bug/refactor |
| tasks | plan.md | approved | plan |
| eat | tasks.md | (exists) | tasks |

## 2. Profile Content Structure

### 2.1 Frontmatter Schema

All step profiles use this YAML frontmatter:

```yaml
---
name: string          # Required: step identifier
description: string   # Required: human-readable description
profiles: [string]    # Optional: profiles to compose
files: [string]       # Optional: glob patterns for files
type: step           # Required: always "step"
---
```

**Validation Rules**:
- `name` must match filename (e.g., `feature.md` → `name: feature`)
- `profiles` references must exist in profile system
- `files` patterns are relative to cycle folder
- `type` must be "step"

### 2.2 Directive Body Schema

The markdown body follows this structure for consistency:

```markdown
# {Step Name} Workflow

## Context
What this step does and agent responsibilities.

## Response Handling (NEW)
How to interpret MCP response fields:
- `files_to_read`: Read these first for context
- `composed_prompt`: Reusable context from profiles
- `workflow_phases`: Phase definitions (for multi-phase)
- `next_task`: Current task (for eat step)
- `cycle_folder`: Output directory for artifacts

## Prerequisites
What must exist before this step runs.

## Workflow / Phases
Phase I: ...
Phase II: ...
(or single-phase instructions)

## Output
What artifacts to create/update.

## Success Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Behavior Rules
1. Rule 1
2. Rule 2
```

## 3. State Management

### 3.1 Active State

**Location**: `.brains/active.json`

```json
{
  "initiative": "history/675d8a3f-feature-user-auth",
  "cycle": "history/675d8a3f-feature-user-auth/2025-12-24-feat-user-auth",
  "current_step": "feature"
}
```

| Field | Description |
|-------|-------------|
| initiative | Path to active initiative folder |
| cycle | Path to active cycle folder |
| current_step | Last executed step name |

### 3.2 Initiative State Transitions

```
EMPTY → initiative create → ACTIVE
ACTIVE → step execute → ACTIVE (current_step updated)
ACTIVE → initiative complete → EMPTY
```

## 4. Relationships

```
Initiative 1───* Cycle
    │
    └──> State (active.json)
         │
         └──> Step
              │
              ├──> Profile (composed)
              │
              └──> StepResponse
                   │
                   ├──> Phase[] (multi-phase only)
                   │
                   └──> TaskInfo (eat step only)
```

## 5. No Schema Changes Required

The existing data model is sufficient for this feature:

| Entity | Change Needed |
|--------|---------------|
| Step | No |
| StepResponse | No |
| Phase | No |
| TaskInfo | No |
| PrerequisiteInfo | No |
| StepPrerequisite | No |
| StepFrontmatter | No |

**Reason**: The feature updates profile *content* (markdown), not the structures that parse and return that content.
