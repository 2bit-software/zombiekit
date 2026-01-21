---
name: complete
description: Mark the current initiative as complete and clear active state.
type: skill
handoffs:
  - label: Start New
    skill: brains.feature
    prompt: Start a new feature...
  - label: Review History
    skill: brains.status
    prompt: Show what was accomplished
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Properly close out an initiative, archive artifacts, and clear active state.

Execution steps:

1. **Load Active Initiative**
   - Read `.brains/active.json`
   - If no active initiative: Report nothing to complete

2. **Completion Check**
   - Review all work items
   - Identify incomplete items
   - If incomplete items exist:
     - List them with status
     - Confirm user wants to complete anyway
     - OR suggest completing work items first

3. **Generate Summary**
   - Create completion summary in INITIATIVE.md
   - List all work items and outcomes
   - Note any incomplete items (if proceeding anyway)
   - Record completion timestamp

4. **Update INITIATIVE.md**
   ```markdown
   ## Completion

   **Completed**: {timestamp}
   **Duration**: {start to end}

   ### Outcomes
   - Feature: auth-api - Complete
   - Feature: session-mgmt - Complete
   - Refactor: middleware - Skipped (deprioritized)

   ### Notes
   {Any closing notes}
   ```

5. **Clear Active State**
   - Remove or clear `.brains/active.json`
   - Initiative remains in history (never deleted)

6. **Report Completion**
   - Initiative name
   - Work items completed vs skipped
   - Total duration
   - History location
   - Suggested next command

## Output Format

```markdown
# Initiative Completed

**{initiative-name}**

Duration: {X days}
Location: history/{date}-{name}/

## Summary
- Features completed: 2
- Refactors completed: 0
- Bugs fixed: 1
- Items skipped: 1

## Next Steps
- Start new initiative with `/brains.feature "..."`
- View history with `/brains.status`
```

## Behavior Rules

- Never delete initiative, only mark complete
- Warn about incomplete items
- Require confirmation for partial completion
- Always update INITIATIVE.md with summary
