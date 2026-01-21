---
name: eat
description: Execute implementation tasks one-by-one (BRAAAAINS!)
profiles: []
files:
  - "spec.md"
  - "plan.md"
  - "tasks.md"
  - "data-model.md"
  - "contracts/**/*.md"
type: step
---
# Task Execution Workflow (BRAAAAINS!)

## Context

You are executing implementation tasks from tasks.md. Work through tasks one at a time, following TDD principles and respecting dependencies. The MCP tool provides your next task via the `next_task` field.

### Your Responsibilities

- Execute the current task completely before moving to the next
- Follow TDD: write tests first, then implementation
- Mark tasks complete in tasks.md as you finish them
- Respect dependencies and parallel markers
- Report blockers immediately

### System Responsibilities (handled by MCP tool)

- Prerequisite verification (tasks.md must exist)
- Next task identification from tasks.md
- State management

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load spec.md, plan.md, tasks.md for context
3. **Check `next_task`**: This contains your current task to execute
   - If `next_task` is null: All tasks are complete!
   - If `next_task` has a value: Execute this task
4. **Follow `directive`**: Execute according to this document
5. **Output to `cycle_folder`**: Update tasks.md with completion status

### Understanding `next_task`

The `next_task` field contains:

```json
{
  "id": "T005",
  "description": "Implement user authentication endpoint",
  "phase": "Phase 2: Core Implementation"
}
```

| Field | Description |
|-------|-------------|
| `id` | Task identifier (e.g., T005) |
| `description` | What needs to be done |
| `phase` | Which phase this task belongs to |

**Important**: If `next_task` is null, all tasks are complete. Report success and stop execution.

---

## Prerequisites

| Required | Status | Blocking Step |
|----------|--------|---------------|
| tasks.md | exists | tasks |

---

## Workflow

### Step 1: Load Context

Read and understand:
1. **spec.md**: What we're building
2. **plan.md**: Technical approach
3. **tasks.md**: All tasks and their status
4. **data-model.md**: Entities and relationships (if exists)
5. **contracts/**: API contracts and test requirements (if exists)

### Step 2: Execute Current Task

For the task in `next_task`:

```
IF task is a test task (contains "test", "verify", "check"):
    1. Write the test according to contracts/spec
    2. Verify the test fails (nothing to test yet)
    3. Mark task complete in tasks.md
ELSE IF task is an implementation task:
    1. Review related tests
    2. Implement the code
    3. Run tests to verify
    4. Mark task complete in tasks.md
ELSE IF task is a validation task (contains "run", "build", "lint"):
    1. Execute the command
    2. Fix any issues
    3. Mark task complete in tasks.md
```

### Step 3: Mark Task Complete

Update tasks.md:

```markdown
# Before
- [ ] T005 Implement user authentication endpoint

# After
- [x] T005 Implement user authentication endpoint
```

### Step 4: Request Next Task

After completing the current task:
1. Call the `step` MCP tool again with `step: "eat"`
2. The tool will provide the next incomplete task
3. Repeat until `next_task` is null

### Step 5: Handle Completion

When `next_task` is null:

```
All tasks complete!

Summary:
- Total tasks: {count}
- Completed: {count}
- Phase: All phases complete

Next steps:
- Run full test suite
- Review implementation
- Consider calling the `step` MCP tool with `step: "audit"` for alignment check
```

---

## Output

Update `tasks.md` as you complete tasks:

```markdown
## Phase 2: Core Implementation

- [x] T004 Write contract tests for UserService
- [x] T005 Implement UserService to satisfy tests
- [ ] T006 Write integration tests  ← currently here
- [ ] T007 Implement database layer
```

---

## Success Criteria

- [ ] Current task executed completely
- [ ] Tests pass (for implementation tasks)
- [ ] Task marked complete in tasks.md
- [ ] No regressions in existing tests
- [ ] Dependencies respected

---

## Behavior Rules

1. **One Task at a Time**: Complete current task before requesting next
2. **TDD Always**: Tests first, implementation second
3. **Mark Immediately**: Update tasks.md as soon as task completes
4. **Respect Dependencies**: Don't skip ahead even if you could
5. **Report Blockers**: If stuck, report and request guidance
6. **Follow Existing Patterns**: Match codebase conventions
7. **Small Commits**: Commit after each task or logical group
8. **No Feature Creep**: Implement exactly what the task describes

---

## Handling Blockers

If you cannot complete a task:

```
## Blocker Report

**Task**: T005 - Implement user authentication endpoint
**Phase**: Phase 2: Core Implementation

**Issue**: Missing API key configuration

**Attempted**:
1. Checked .env file - not present
2. Checked environment variables - not set
3. Checked documentation - no setup instructions

**Need**:
- Environment configuration instructions
- OR: Skip this task and continue with non-blocking tasks

**Suggested Resolution**:
[Your recommendation]
```

---

## Parallel Task Handling

When multiple [P] tasks are available:

```markdown
- [x] T010 Setup complete
- [ ] T011 [P] Implement user routes
- [ ] T012 [P] Implement admin routes
- [ ] T013 Requires T011 and T012
```

The MCP tool returns tasks sequentially. For parallel tasks:
1. Complete T011
2. Request next task (get T012)
3. Complete T012
4. Request next task (get T013)

If you can work on multiple files simultaneously, you may execute [P] tasks in parallel, but mark them complete individually.
