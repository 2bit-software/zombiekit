---
name: implement
description: Execute tasks from the task list, implementing the feature/fix/refactor.
type: skill
handoffs:
  - label: Revise Spec
    skill: brains.revise
    prompt: Implementation revealed spec issues...
  - label: Mark Complete
    skill: brains.complete
    prompt: All tasks are done, mark initiative complete
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Execute tasks from the task list, implementing working code.

Execution steps:

1. **Load Context**
   - Load `tasks.md` for current work item
   - Load `technical-spec.md` for implementation details
   - Load `business-spec.md` for acceptance criteria

2. **Progress Tracking**
   - Initialize or load `progress.md`
   - Track completed/pending/blocked tasks
   - Note decisions and blockers

3. **Task Execution Loop**
   For each task in dependency order:

   a. **Load Task Context**
      - Extract relevant spec sections
      - Load any dependent task outputs
      - Start with fresh context (memory-efficient)

   b. **Implement**
      - Execute task according to spec
      - Follow technical-spec patterns
      - Write tests where specified

   c. **Verify**
      - Run task-specific acceptance criteria
      - Check against spec requirements
      - Update progress tracking

   d. **Handle Blockers**
      - If blocked: Document reason
      - If spec issue: Suggest `/brains.revise`
      - If task unclear: Clarify and continue

4. **Parallel Execution** (where marked)
   - Tasks marked `[P]` can run concurrently
   - Maintain isolation between parallel tasks
   - Merge results after completion

5. **Final Verification**
   - All tasks complete
   - All acceptance criteria pass
   - Integration tests pass

6. **Report Completion**
   - Tasks completed
   - Tests added/modified
   - Files changed
   - Blockers encountered (if any)
   - Suggested next command

## Progress Format

```markdown
## Progress Log

### T001 - Create User model
- Status: Complete
- Files: src/models/user.go
- Notes: Added validation for email

### T002 - Implement auth endpoint
- Status: Blocked
- Reason: Spec unclear on token format
- Action: Needs /brains.revise
```

## Behavior Rules

- One task at a time unless parallelizable
- Document every decision in progress.md
- Stop and escalate if spec issues discovered
- Never guess at unclear requirements
