---
name: tasks
description: Break down the plan into actionable tasks
profiles: []
files:
  - "spec.md"
  - "plan.md"
  - "tasks.md"
type: step
---
# Generate Task Breakdown

Your task is to create a detailed task list (`tasks.md`) from the implementation plan.

## Guidelines
1. Create specific, actionable tasks with clear completion criteria
2. Identify dependencies between tasks
3. Mark tasks that can run in parallel with [P]
4. Group tasks by phase or component
5. Include test tasks before implementation tasks (TDD)

## Output
Create `tasks.md` with:
- Task ID format: T001, T002, etc.
- Task status: [ ] pending, [x] completed
- Dependencies and parallel markers
- Estimated complexity (optional)
