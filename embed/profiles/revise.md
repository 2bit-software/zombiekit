---
name: revise
description: Re-enter the workflow cycle to revise specifications when significant changes are needed.
type: skill
handoffs:
  - label: Continue Planning
    skill: brains.plan
    prompt: Create a new plan based on revised spec
  - label: Audit Changes
    skill: brains.audit
    prompt: Verify revised artifacts are consistent
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Properly revise artifacts when implementation reveals issues, preserving history.

Execution steps:

1. **Load Current Artifacts**
   - Load all artifacts for current work item
   - Identify what needs revision
   - Parse reason for revision from user input

2. **Version Archival**
   - Archive current versions with suffix (e.g., `spec.v1.md`)
   - Preserve complete history
   - Update revision-log.md

3. **Revision Log Entry**
   ```markdown
   ## v{N} - {date}
   **Reason**: {user-provided reason}
   **Trigger**: {implementation/audit/user request}
   **Changed**: {list of artifacts}
   **By**: /brains.revise
   ```

4. **Re-enter Workflow**
   - Determine starting point based on revision scope:
     - Spec change: Re-run research + create cycle
     - Plan change: Re-run plan cycle only
     - Tasks change: Re-generate tasks only
   - Apply same research-create-audit-highlight cycle

5. **Differential Analysis**
   - Compare new version to archived version
   - Highlight what changed
   - Identify downstream impacts

6. **Cascade Updates**
   - If spec changed: Plan may need revision
   - If plan changed: Tasks may need regeneration
   - Flag cascading updates needed

7. **Report Completion**
   - Version created
   - Changes from previous version
   - Downstream impacts
   - Suggested next command

## Version Structure

```
{work-item}/
  spec.md              # Current version
  spec.v1.md           # First version
  spec.v2.md           # Second version
  plan.md              # Current version
  plan.v1.md           # If plan also revised
  revision-log.md      # History of all revisions
```

## When to Use

| Scenario | Command |
|----------|---------|
| Typo/minor fix | `/brains.update` |
| New requirement | `/brains.revise` |
| Implementation blocked | `/brains.revise` |
| Major audit finding | `/brains.revise` |
| Scope change | `/brains.revise` |

## Behavior Rules

- Always archive before modifying
- Maintain complete revision history
- Never lose previous versions
- Flag all cascade impacts
- User must approve scope of revision
