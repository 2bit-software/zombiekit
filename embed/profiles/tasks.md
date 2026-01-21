---
name: tasks
description: Generate an actionable, dependency-ordered task list from the implementation plan.
type: skill
handoffs:
  - label: Start Implementation
    skill: brains.implement
    prompt: Execute the tasks in order
  - label: Analyze Consistency
    skill: brains.audit
    prompt: Check alignment between spec, plan, and tasks
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Break implementation plan into independent, parallelizable tasks.

Execution steps:

1. **Load Context**
   - Load `implementation-plan.md`
   - Load `technical-spec.md`
   - Load original `business-spec.md` for traceability

2. **Complexity Analysis**
   - Count files affected
   - Estimate lines of change
   - Count cross-module dependencies
   - Classify: Simple (<5 files), Medium (5-15), Complex (>15)

3. **Task Generation**
   - Break plan into atomic tasks
   - Each task should be independently executable
   - Each task can start from fresh context
   - Include acceptance criteria per task

4. **Dependency Graph**
   - Map task dependencies
   - Identify parallel execution opportunities
   - Ensure no circular dependencies
   - Calculate critical path

5. **Splitting (if Complex)**
   - If complexity is high, split into multiple task lists
   - Each list is independently implementable
   - Document dependency order between lists

6. **Validation**
   - Every spec requirement maps to at least one task
   - Every task traces to plan step
   - No orphan tasks

7. **Report Completion**
   - Total task count
   - Parallel opportunities
   - Estimated complexity
   - Suggested execution order
   - Suggested next command (`/brains.implement`)

## Task Format

```markdown
- [ ] T001 [P] [US1] Description with file path
```

- `T001`: Sequential ID
- `[P]`: Parallelizable marker (optional)
- `[US1]`: User story reference (optional)
- Description with exact file paths

## Artifact Structure

```
{work-item}/
  tasks.md
  tasks-02.md (if split due to complexity)
```

## Behavior Rules

- Each task must be completable in isolation
- Include file paths for every task
- Maximum 20 tasks per list (split if more)
- Parallel tasks must not depend on each other
