---
name: feature-light
description: Lightweight feature workflow — creates a branch, writes spec and plan in one pass, then implements. No research agents, no audit cycle. Use for small, well-understood work.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

### AutoMode Detection

If the user input contains the keyword **automode** (case-insensitive):

- Load the automode profile via `mcp__zombiekit__profile-compose` with `profiles: ["automode"]` and follow its "At each step" instructions.
- These instructions override the interactive confirmation prompts below.

## Feature-Light Workflow

Goal: Go from intent to working code quickly — one spec+plan pass, then straight to implement and complete.

Skip research agents, skip audit cycles. If the work turns out to be more complex than expected,
stop and escalate to `/brains.new` instead of silently expanding scope.

### Execution Steps

1. **Initiative Check**
   - Read `.brains/active.json`
   - If no active initiative: Create one with an auto-generated name derived from the user input
   - If active and `--new` flag present: Complete current, create new
   - If active: Add this feature to the current initiative
   - Check if user input contains `USE_GRAPHITE: true` metadata block — if present, pass `use_graphite: true` when calling `mcp__zombiekit__initiative` create

2. **Source Section** (if Linear ticket metadata present)
   - Check if user input contains `LINEAR_TICKET:` metadata block
   - If not present: Skip to step 3
   - If present:
     a. Extract LINEAR_TICKET, LINEAR_URL, LINEAR_TITLE from metadata
     b. Read the initiative's INITIATIVE.md
     c. Insert a Source section before "## Description":
        ```markdown
        ## Source

        **Linear Ticket**: [LINEAR_TICKET](LINEAR_URL)
        **Title**: LINEAR_TITLE

        ```
     d. Proceed to step 3

3. **Create Branch**
   - If the initiative was just created in step 1 (i.e., `initiative create` was called and returned a `branch` field): the branch is already created and checked out. **Skip branch creation entirely.**
   - If joining an existing initiative (step 1 found an active initiative and did not call `create`):
     - Derive a branch name from the initiative and feature name:
       `feat/{initiative-slug}/{feature-slug}` (e.g., `feat/auth-api/add-refresh-endpoint`)
     - Create and check out the branch via `mcp__zombiekit__git`:
       - action: `branch` (or equivalent — use Bash `git checkout -b <name>` if the MCP
         tool does not support branch creation)
     - If already on a non-main branch that matches the initiative: Skip, use current branch
     - If the branch already exists remotely: Check it out without creating

4. **Spec + Plan** (single pass — no agents, no delegation)
   - In one focused step, produce two artifacts in the feature directory:

   **`notes.md`** — the spec:
   - What needs to be built or changed (2-5 sentences)
   - Any constraints or preferences from the user input
   - Acceptance criteria (bulleted list — what "done" looks like)
   - Known risks or open questions (note inline, don't block)

   **`tasks.md`** — the plan as an executable task list:
   - Break the spec into atomic, independently-executable tasks
   - Each task should map to a specific file or function change
   - Mark parallelizable tasks with `[P]`
   - Keep total task count under ~10; if it exceeds that, consider escalating to
     `/brains.new` instead
   - Format:
     ```markdown
     ## Tasks

     - [ ] T001: {description} — `path/to/file.go`
     - [ ] T002: {description} [P] — `path/to/other.go`
     - [ ] T003: {description} [P] — `path/to/other.go`
     ```

   Do NOT:
   - Spawn research or audit agents
   - Produce `business-spec.md`, `technical-requirements-research.md`, or
     `implementation-plan.md` — `notes.md` + `tasks.md` is sufficient
   - Ask clarifying questions unless the input is genuinely ambiguous (one question max)

5. **Implement**
   - Load the implement profile:
     ```
     mcp__zombiekit__profile-compose with profiles: ["implement"]
     ```
   - Follow its execution steps against `notes.md` and `tasks.md`
   - The implement profile handles task execution, progress tracking, and blockers

6. **Complete**
   - When all tasks are done, load the complete workflow:
     ```
     mcp__zombiekit__workflow-compose with name: "complete"
     ```
   - Follow its steps (commit/push/PR offer, flimsy update, risk assessment, report)

### Artifact Structure

```
history/{id}-feature-{slug}/
  INITIATIVE.md
  notes.md
  tasks.md
```

### Behavior Rules

- No research agents, no audit agents — that's the point of this workflow
- One clarifying question max before writing the spec; never more
- If a task reveals unexpected complexity (unfamiliar code, risky data changes,
  non-obvious side effects): surface it immediately, don't silently expand scope
- Never proceed past step 4 without at least one task in `tasks.md`
- If the task count exceeds ~10, stop and suggest `/brains.new` for the fuller workflow
