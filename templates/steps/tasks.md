---
name: tasks
description: Generate actionable, dependency-ordered tasks from the implementation plan
profiles: []
files:
  - "spec.md"
  - "plan.md"
  - "tasks.md"
  - "data-model.md"
  - "contracts/**/*.md"
type: step
---
# Task Generation Workflow

## Context

You are generating a detailed task breakdown from an approved implementation plan. Tasks guide the eat (implementation) step with specific, actionable work items organized by dependency order.

### Your Responsibilities

- Load and analyze the approved plan
- Break phases into specific, actionable tasks
- Define dependencies and parallel opportunities
- Organize by user story or component
- Apply TDD ordering (tests before implementation)

### System Responsibilities (handled by MCP tool)

- Prerequisite verification (plan.md with approved status)
- File path resolution
- State management

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load plan.md, spec.md, data-model.md, contracts/
3. **Follow `directive`**: Execute according to this document
4. **Output to `cycle_folder`**: Save tasks.md here
5. **Reference previous tasks.md**: If it exists, update rather than replace

---

## Prerequisites

| Required | Status | Blocking Step |
|----------|--------|---------------|
| plan.md | approved | plan |

If prerequisites are not met, the MCP response will contain:
- `prerequisites.met: false`
- `prerequisites.required`: What's needed
- `prerequisites.hint`: How to satisfy the requirement

---

## Workflow

### Step 1: Analyze Plan

From the implementation plan, extract:

1. **Phases**: Ordered implementation phases
2. **Components**: Modules, services, files to create/modify
3. **Dependencies**: What depends on what
4. **Test Requirements**: Contract tests, integration tests, unit tests

### Step 2: Define Task Format

Each task follows this format:

```markdown
- [ ] T{NNN} [P?] {Description}
```

| Element | Required | Description |
|---------|----------|-------------|
| `[ ]` | Yes | Checkbox for completion status |
| `T{NNN}` | Yes | Task ID (T001, T002, ...) |
| `[P]` | No | Parallel marker - can run with adjacent [P] tasks |
| `{Description}` | Yes | Clear, actionable description with file paths |

### Step 3: Organize by Phase

Group tasks by implementation phase:

```markdown
## Phase 1: Setup

- [ ] T001 Initialize project structure
- [ ] T002 [P] Configure dependencies
- [ ] T003 [P] Set up testing framework

**Checkpoint**: {what should be true after this phase}
```

### Step 4: Apply TDD Ordering

Within each phase:

1. **Test tasks first**: Write tests before implementation
2. **Implementation follows**: Code to make tests pass
3. **Integration last**: Connect components

Example:
```markdown
- [ ] T010 Write contract tests for UserService per contracts/user-service.md
- [ ] T011 Implement UserService to satisfy contract tests
- [ ] T012 Write integration test for UserService with database
```

### Step 5: Mark Dependencies

Use comments for complex dependencies:

```markdown
- [ ] T015 Implement authentication middleware
  <!-- Depends: T010, T011 -->
- [ ] T016 [P] Implement user routes
- [ ] T017 [P] Implement admin routes
  <!-- T016 and T017 can run in parallel after T015 -->
```

### Step 6: Include Validation Tasks

End each phase with validation:

```markdown
- [ ] T020 Run `go test ./...` to verify all tests pass
- [ ] T021 Run `go build` to verify compilation
```

---

## Output

Create `tasks.md` in `cycle_folder` with this structure:

```markdown
# Tasks: {Feature Name}

**Input**: Design documents from `{cycle_folder}`
**Prerequisites**: {list available docs: spec.md, plan.md, data-model.md, contracts/}

**Tests**: {Requested | Not requested - implementation only}

**Organization**: Tasks organized by {phase | user story | component}

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: {what this phase accomplishes}

- [ ] T001 {Description with file path}
- [ ] T002 [P] {Description}
- [ ] T003 [P] {Description}

**Checkpoint**: {verification criteria}

---

## Phase 2: Core Implementation

**Purpose**: {what this phase accomplishes}

- [ ] T004 {Test task first}
- [ ] T005 {Implementation task}
- [ ] T006 [P] {Parallel task}
- [ ] T007 [P] {Parallel task}

**Checkpoint**: {verification criteria}

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies
- **Phase 2**: Depends on Phase 1 completion
- ...

### Parallel Opportunities

Within Phase 2:
- T006 and T007 can run in parallel

---

## Notes

{Any special considerations, external dependencies, or warnings}
```

---

## Success Criteria

- [ ] All plan phases translated to tasks
- [ ] Task IDs are sequential and unique
- [ ] Parallel markers [P] correctly identify parallelizable work
- [ ] Test tasks precede implementation tasks (TDD)
- [ ] Each phase has a checkpoint
- [ ] File paths included where applicable
- [ ] Dependencies documented

---

## Behavior Rules

1. **One Task, One Outcome**: Each task produces a specific, verifiable result
2. **TDD by Default**: Test tasks before implementation tasks
3. **Parallel When Possible**: Mark [P] for tasks touching different files
4. **Sequential When Required**: Omit [P] for dependent tasks
5. **Include File Paths**: Tasks should reference specific files
6. **Checkpoints**: End phases with verification steps
7. **No Ambiguity**: Task descriptions should be clear enough to execute without clarification
8. **Respect Contracts**: Reference contract documents for test requirements
