---
name: next
description: Advance to the next workflow step based on current state
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Next Step Workflow

Goal: Determine and advance to the next logical step in the workflow.

### Execution Steps

1. **Load Active State**
   - Read `.brains/active.json`
   - If no active initiative: Report error and suggest `/brains.new`

2. **Determine Current Phase**
   - Check artifacts in current work item directory
   - Identify most recent completed phase
   - Determine work item type (feature, bug, refactor)

3. **Calculate Next Step**
   - Based on current phase and work item type
   - See progression tables below

4. **Load Next Profile**
   - Use `mcp__zombiekit__profile-compose` to load the next profile
   - Pass through any arguments

### Workflow Progressions

**Feature Workflow**
```
[start] -> spec -> plan -> tasks -> implement -> [complete]
```

| Current State | Next Step | Profile |
|---------------|-----------|---------|
| No artifacts | spec | `feature` |
| `business-spec.md` exists | plan | `plan` |
| `implementation-plan.md` exists | tasks | `tasks` |
| `tasks.md` exists | implement | `implement` |
| All tasks complete | complete | suggest `/brains.complete` |

**Bug Workflow**
```
[start] -> report -> investigate -> fix-plan -> implement -> [complete]
```

| Current State | Next Step | Profile |
|---------------|-----------|---------|
| No artifacts | report | `bug` |
| `report.md` exists | investigate | `bug` (investigation phase) |
| `investigation.md` exists | fix-plan | `plan` |
| `fix-plan.md` exists | implement | `implement` |
| Fix verified | complete | suggest `/brains.complete` |

**Refactor Workflow**
```
[start] -> goal -> analysis -> plan -> tasks -> implement -> [complete]
```

| Current State | Next Step | Profile |
|---------------|-----------|---------|
| No artifacts | goal | `refactor` |
| `goal.md` exists | analysis | `refactor` (analysis phase) |
| `dependency-analysis.md` exists | plan | `plan` |
| `refactor-plan.md` exists | tasks | `tasks` |
| `tasks.md` exists | implement | `implement` |
| All tasks complete | complete | suggest `/brains.complete` |

### Alternate Path Handling

If arguments include an alternate directive:
- `next audit` - Run audit instead of normal progression
- `next clarify` - Run clarification instead
- `next research` - Do more research before proceeding

These overrides let users take detours without losing their place.

### Output Format

```markdown
## Current Status
Work item: {type}/{name}
Current phase: {phase}
Artifacts: {list of existing artifacts}

## Next Step
Advancing to: **{next-phase}**

Loading {profile} profile...
```

### Error Conditions

**No active work item:**
```
No active work item found.

Use `/brains.new <description>` to start a new initiative.
```

**Already complete:**
```
All phases complete for {work-item}.

Options:
- `/brains.complete` - Mark initiative done
- `/brains.new <description>` - Start new work
```
