# Feature Specification: Workflow Step Tracking

**Feature Branch**: `feat/workflow-step-tracking`
**Created**: 2026-01-31
**Status**: Draft
**Linear Ticket**: [DEV-103](https://linear.app/heinsight/issue/DEV-103/we-should-have-the-workflow-define-the-steps)

## Problem Statement

Currently, workflow steps are implicit - derived from which artifacts exist. This creates issues:

1. **Steps aren't declared** - The workflow progression is scattered across code logic
2. **State isn't self-contained** - `active.json` mixes pointer duties with state tracking
3. **No dynamic modification** - Can't adapt workflow mid-flight based on discoveries
4. **Single workflow assumption** - No clear model for feature → refactor → bugfix in one initiative

## Goals

1. **Workflows define default steps** - Each workflow declares its step sequence in frontmatter
2. **Initiative state is self-contained** - All step progress lives in `INITIATIVE.md`
3. **Agent-driven modification** - Steps can alter the workflow table (add/skip/modify)
4. **Multiple cycles per initiative** - Track sequential workflow runs (feat, then ref, then fix)
5. **`active.json` is a minimal pointer** - Only tracks which initiative is active

## Non-Goals

- Backwards compatibility with old active.json format
- Parallel cycle execution (sequential only for now)
- Explicit DAG DSL in frontmatter (agent-driven instead)
- Per-step metadata beyond status and timestamp

## Architecture

### Separation of Concerns

| Location | Responsibility |
|----------|---------------|
| `active.json` | Pointer: which initiative, overall status |
| Workflow `.md` frontmatter | Default steps: starting template for the workflow |
| `INITIATIVE.md` | Mutable state: cycles, steps, status - source of truth |

### active.json (Minimal Pointer)

```json
{
  "initiative": "history/697e8bb3-feature-foo",
  "status": "in_progress",
  "started": "2026-01-31T15:09:39Z"
}
```

No step tracking. No cycle info. Minimal pointer with start timestamp.

### Workflow Default Steps (Frontmatter)

Workflows declare their **default** step sequence. This is a template, not a contract.

```yaml
---
name: feature
description: Feature development workflow
steps:
  - name: spec
    profile: feature
  - name: plan
    profile: plan
  - name: tasks
    profile: tasks
  - name: implement
    profile: implement
---
```

When a cycle starts, these steps are copied into INITIATIVE.md. The agent can then modify them.

### INITIATIVE.md (Mutable Source of Truth)

```markdown
# Initiative: user-auth

**Type**: feature
**Status**: in_progress
**Created**: 2026-01-31

## Cycles

### 1. feat/user-auth (completed)

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-01-31 10:30 |
| plan | skipped | 2026-01-31 11:00 |
| implement | completed | 2026-01-31 14:00 |

**Note**: Plan skipped - research showed this was a small change.

### 2. ref/user-auth (active)

| Step | Status | Updated |
|------|--------|---------|
| analyze | in_progress | 2026-01-31 15:00 |
| plan | pending | - |
| implement | pending | - |

## Description
...
```

Key properties:
- **Human-readable** - Markdown tables, clear structure
- **Machine-parseable** - Consistent format for tooling
- **Mutable** - Agent can edit steps (add rows, change status, add notes)
- **Self-contained** - No external dependencies for state

### Folder Structure

```
history/697e8bb3-feature-user-auth/
  INITIATIVE.md                    # All cycles tracked here
  697e8bb3-feat-user-auth/         # Cycle 1 artifacts
    spec.md
    research.md
  697e8bb4-ref-user-auth/          # Cycle 2 artifacts
    analysis.md
    plan.md
```

## Agent-Driven Workflow Modification

Profiles include instructions for when to modify the workflow. Examples:

### Skip Steps (Small Change)

In `feature.md` profile:
```markdown
### Small Change Detection

After research, if this is a trivial change (< 50 lines, single file, obvious fix):
1. Update INITIATIVE.md: mark `plan` and `tasks` as `skipped`
2. Add note explaining why
3. Proceed directly to `implement`
```

### Add Steps (Unexpected Complexity)

In `plan.md` profile:
```markdown
### Complexity Discovery

If planning reveals this needs architectural review:
1. Update INITIATIVE.md: insert `architecture-review` step before `implement`
2. Set its status to `pending`
3. Complete current step, agent will pick up new step on `/brains.next`
```

### Convert Workflow (Pivot)

In `spec.md` profile:
```markdown
### Research-Only Outcome

If research determines no code change needed (docs only, or "won't fix"):
1. Update INITIATIVE.md: mark remaining steps as `skipped`
2. Add note: "Converted to research-only task"
3. Suggest `/brains.complete`
```

## User Scenarios

### Scenario 1: Start New Initiative

**When** I run `/brains.new "add user auth"`
**Then**:
1. `active.json` → `{"initiative": "history/XXX", "status": "in_progress"}`
2. Workflow type detected (feature)
3. `INITIATIVE.md` created with Cycle 1, default steps from `feature` workflow
4. First step marked `in_progress`

### Scenario 2: Complete Step and Advance

**Given** Cycle 1 shows `spec | in_progress`
**When** I run `/brains.next`
**Then**:
1. `spec` marked `completed` with timestamp
2. `plan` marked `in_progress`
3. Plan profile loaded

### Scenario 3: Skip Steps (Agent Decision)

**Given** Agent in spec phase determines this is trivial
**When** Agent updates INITIATIVE.md
**Then**:
1. `plan` and `tasks` marked `skipped`
2. Note added explaining rationale
3. Next `/brains.next` jumps to `implement`

### Scenario 4: Add New Step

**Given** Agent in plan phase discovers need for security review
**When** Agent updates INITIATIVE.md
**Then**:
1. New row `security-review | pending` inserted after `plan`
2. Workflow now has 5 steps instead of 4
3. `/brains.next` will hit the new step

### Scenario 5: Start Second Cycle

**Given** Cycle 1 (feat) is completed
**When** I run `/brains.new refactor auth module`
**Then**:
1. New Cycle 2 section added to INITIATIVE.md
2. Default steps from `refactor` workflow
3. Cycle 2 marked `active`, Cycle 1 remains `completed`

### Scenario 6: Check Status

**When** I query `initiative status`
**Then** Response includes:
- Active cycle: "2. ref/user-auth"
- Current step: "analyze"
- Step status: "in_progress"
- Cycles completed: 1
- Total cycles: 2

Derived from parsing INITIATIVE.md.

## Data Model

### active.json

```go
type ActiveState struct {
    Initiative string    `json:"initiative"` // path to initiative folder
    Status     string    `json:"status"`     // "in_progress" | "complete"
    Started    time.Time `json:"started"`    // when initiative became active
}
```

### Workflow Frontmatter

```go
type WorkflowStep struct {
    Name    string `yaml:"name"`
    Profile string `yaml:"profile"`
}

type WorkflowMeta struct {
    Name  string         `yaml:"name"`
    Steps []WorkflowStep `yaml:"steps"`
}
```

### Parsed from INITIATIVE.md

```go
type CycleState struct {
    Number int         // 1, 2, 3...
    Type   string      // "feat", "ref", "fix"
    Name   string      // "user-auth"
    Status string      // "active", "completed"
    Steps  []StepState
}

type StepState struct {
    Name    string // "spec", "plan", custom steps
    Status  string // "pending", "in_progress", "completed", "skipped"
    Updated string // timestamp or "-"
}
```

## Implementation Components

| Component | Change |
|-----------|--------|
| `active.json` schema | Simplify to `initiative` + `status` only |
| Workflow `.md` files | Add `steps:` frontmatter to feature/bug/refactor |
| `initiative create` | Parse workflow steps, generate INITIATIVE.md with Cycle 1 |
| INITIATIVE.md parser | New: parse cycles and step tables from markdown |
| `initiative status` | Read from INITIATIVE.md, not active.json |
| `step` MCP tool | Update step status in INITIATIVE.md |
| `next.md` workflow | Find current step in INITIATIVE.md, advance to next non-skipped |
| Profile instructions | Add guidance for when to modify workflow table |

## INITIATIVE.md Parsing

The parser must handle:

1. **Cycle sections**: `### N. type/name (status)`
2. **Step tables**: Standard markdown table with Step, Status, Updated columns
3. **Notes**: Freeform text after tables (preserved but not parsed)
4. **Dynamic rows**: Tables may have more/fewer rows than workflow default

Parsing is read-heavy (every status check) so should be efficient.

## Workflow Default Steps

| Workflow | Default Steps |
|----------|---------------|
| feature | spec → plan → tasks → implement |
| bug | investigate → fix → verify |
| refactor | analyze → plan → implement → verify |

These are starting points. Agent can modify freely.

## Success Criteria

1. `active.json` contains only `initiative` and `status`
2. Workflow files have `steps:` in frontmatter
3. `INITIATIVE.md` tracks cycles with step tables
4. Agent can add/skip/modify steps by editing INITIATIVE.md
5. `/brains.next` reads step state from INITIATIVE.md
6. `initiative status` derives all info from INITIATIVE.md
7. Multiple cycles tracked sequentially in one initiative
