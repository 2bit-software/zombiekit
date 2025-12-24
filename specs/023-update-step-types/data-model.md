# Data Model: Update Step Types

**Date**: 2025-12-23
**Feature**: 023-update-step-types

## Entities

### Step (existing, modified)

A workflow step definition loaded from templates.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Step identifier: feature, bug, refactor, plan, tasks, eat, audit, clarify, complete |
| Description | string | Human-readable description |
| Profiles | []string | Profile names to compose for context |
| Files | []string | Glob patterns for files to read |
| Directive | string | Instruction markdown for the step |
| Type | string | Always "step" for step definitions |
| Source | StepSource | Where loaded from (embedded, global, local) |
| Path | string | Absolute path if loaded from file |

**Removed Values for Name**: init, specify, implement

**Added Values for Name**: bug, refactor, eat (replaces implement)

### StepPrerequisite (NEW)

Defines requirements that must be met before a step can execute.

| Field | Type | Description |
|-------|------|-------------|
| RequiredArtifact | string | File that must exist (e.g., "spec.md") |
| RequiredStatus | string | Optional status in frontmatter (e.g., "approved") |
| Hint | string | Guidance shown when prerequisite not met |
| BlockingStep | string | Name of step that produces the artifact |

**Prerequisite Relationships**:

```
feature/bug/refactor → (creates spec.md)
         ↓
        plan (requires spec.md approved)
         ↓ (creates plan.md)
       tasks (requires plan.md approved)
         ↓ (creates tasks.md)
        eat (requires tasks.md exists)
         ↓
      complete
```

### Initiative (existing, unchanged)

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique identifier |
| Type | InitiativeType | feature, bug, or refactor |
| Name | string | Human-readable name slug |
| Path | string | Absolute path to initiative folder |
| Status | InitiativeStatus | active or completed |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last activity timestamp |

### InitiativeType (existing, unchanged)

Enum values: `feature`, `bug`, `refactor`

### Cycle (existing, unchanged)

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique identifier |
| Type | CycleType | feat, ref, or fix |
| Name | string | Name slug |
| Path | string | Absolute path |
| Status | CycleStatus | template, in_progress, audited, approved |
| InitiativeID | string | Parent initiative ID |
| Number | int | Cycle number (1, 2, 3...) |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last update timestamp |

## State Transitions

### Step Execution Flow

```
[No Active Initiative]
        │
        ▼
    feature/bug/refactor
        │ (creates initiative + cycle)
        ▼
    [Active Initiative]
        │
        ▼
       plan (requires approved spec)
        │
        ▼
      tasks (requires approved plan)
        │
        ▼
       eat (requires tasks.md)
        │
        ▼
     complete
        │
        ▼
[No Active Initiative]
```

### Prerequisite Validation States

| Current State | Step Requested | Prerequisite Check | Result |
|---------------|----------------|-------------------|--------|
| No initiative | plan | No spec.md | BLOCK: "Run feature/bug/refactor first" |
| Has spec (draft) | plan | spec.md not approved | BLOCK: "Approve spec first" |
| Has spec (approved) | plan | spec.md approved | ALLOW |
| Has plan (draft) | tasks | plan.md not approved | BLOCK: "Approve plan first" |
| Has plan (approved) | tasks | plan.md approved | ALLOW |
| No tasks | eat | No tasks.md | BLOCK: "Run tasks first" |
| Has tasks | eat | tasks.md exists | ALLOW |

## Validation Rules

1. **Step name validation**: Only allowed values are feature, bug, refactor, plan, tasks, eat, audit, clarify, complete
2. **Prerequisite enforcement**: Hard block with error code "PREREQUISITE_NOT_MET"
3. **Initiative type validation**: feature, bug, refactor only
4. **One active initiative**: Only one initiative can be active at a time

## File Artifacts

| Step | Creates | Location |
|------|---------|----------|
| feature | spec.md, research.md, INITIATIVE.md | cycle folder, initiative folder |
| bug | spec.md, research.md, INITIATIVE.md | cycle folder, initiative folder |
| refactor | spec.md, research.md, INITIATIVE.md | cycle folder, initiative folder |
| plan | plan.md | cycle folder |
| tasks | tasks.md | cycle folder |
| eat | (code changes) | source tree |
| audit | audit/{date}.md | cycle folder |
| clarify | (updates existing artifacts) | cycle folder |
| complete | (updates status) | initiative folder |
