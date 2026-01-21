---
name: status
description: Display current initiative status, work items, and suggested next steps.
type: skill
handoffs:
  - label: Continue Feature
    skill: brains.feature
    prompt: Add another feature to the initiative
  - label: Start Implementation
    skill: brains.implement
    prompt: Begin implementing the current work item
  - label: Mark Complete
    skill: brains.complete
    prompt: Mark the initiative as complete
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Provide clear visibility into current initiative state and guide next actions.

Execution steps:

1. **Load Active State**
   - Read `.brains/active.json`
   - If no active initiative: Report and suggest `/brains.init` or `/brains.feature`

2. **Initiative Summary**
   - Initiative name and start date
   - Original goal from INITIATIVE.md
   - Time since last activity

3. **Work Items Inventory**
   For each work item (features, refactors, bugs):
   - Status: pending, in-progress, complete, blocked
   - Current phase: spec, plan, tasks, implement
   - Blocker reason (if blocked)

4. **Progress Metrics**
   - Total work items
   - Completed count
   - Blocked count
   - Estimated remaining

5. **Current Focus**
   - Which work item is active
   - Current phase
   - Next expected action

6. **Suggested Next Steps**
   - Based on current state, suggest commands
   - Prioritize unblocking if blocked
   - Suggest completion if all done

## Output Format

```markdown
# Initiative: {name}
Started: {date} | Last activity: {relative time}

## Goal
{From INITIATIVE.md}

## Work Items

| Type | Name | Status | Phase |
|------|------|--------|-------|
| Feature | auth-api | Complete | - |
| Feature | session-mgmt | In Progress | planning |
| Refactor | middleware | Pending | - |

## Current Focus
**features/002-session-mgmt** - Planning phase

Last action: Research completed
Next: Run `/brains.plan` to create implementation plan

## Suggested Commands
1. `/brains.plan` - Continue with current work item
2. `/brains.feature "..."` - Add another feature
3. `/brains.complete` - Mark initiative done (when ready)
```

## Behavior Rules

- Always show actionable next steps
- Highlight blockers prominently
- Show time since last activity as reminder
- Never modify state, read-only command
