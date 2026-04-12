---
name: refactor
description: Refactoring workflow — creates a branch, analyzes dependencies, assesses safety nets, and produces an atomic refactor plan. /brains.next advances to implement then audit.
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

## Refactor Workflow

Goal: Create a safe refactoring specification that restructures code while preserving behavior. Implementation and verification happen in subsequent steps via `/brains.next`.

### Execution Steps

1. **Initiative Check**
   - Read `.brains/active.json`
   - If no active initiative: Create one with an auto-generated name derived from the user input
   - If active and `--new` flag present: Complete current, create new
   - If active: Add this refactor to the current initiative
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
     - Derive a branch name: `refactor/{initiative-slug}/{refactor-slug}`
       (e.g., `refactor/auth-api/extract-session-middleware`)
     - Create and check out the branch via `mcp__zombiekit__git`:
       - Use Bash `git checkout -b <name>` if the MCP tool does not support branch creation
     - If already on a non-main branch that matches the initiative: Skip, use current branch
     - If the branch already exists remotely: Check it out without creating

4. **Write INITIATIVE.md Step Table**
   - Create or update INITIATIVE.md in the initiative directory with a cycle entry:
     ```markdown
     ### {n}. refactor/{refactor-slug} (active)

     | Step | Status | Updated |
     |------|--------|---------|
     | analyze | in_progress | {now} |
     | plan | pending | - |
     | tasks | pending | - |
     | implement | pending | - |
     ```

5. **Goal Definition**
   - Document improvement goal in `goal.md`
   - Identify what "better" means (readability, performance, maintainability)
   - Define success criteria

6. **Constraint Identification**
   - Document behavior that MUST NOT change in `constraints.md`
   - List all public interfaces that must remain stable
   - Identify external dependencies

7. **Dependency Analysis** (research agents)
   - Spawn a research-codebase agent: Map affected code
   - Use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to follow execution paths through affected modules
   - Identify all files, modules, tests impacted
   - Document in `dependency-analysis.md`

8. **Safety Net Assessment**
   - Evaluate existing test coverage in `safety-net.md`
   - Identify gaps in coverage
   - Recommend additional tests before refactoring

9. **Refactor Planning**
   - Create `refactor-plan.md` with atomic steps
   - Each step should be independently committable
   - Include rollback strategy

10. **Report Completion**
    - Scope of changes
    - Test coverage assessment
    - Risk areas identified
    - List of artifacts created
    - Remind user: `/brains.next` to begin implementation

### Artifact Structure

```
history/{id}-refactor-{slug}/
  INITIATIVE.md
  goal.md
  constraints.md
  dependency-analysis.md
  refactor-plan.md
  safety-net.md
  progress.md
```

### Behavior Rules

- Refactors NEVER change external behavior
- Require test coverage before major restructuring
- Each step must be atomic and reversible
- Flag any behavior changes for promotion to feature
- Never assume safety — evidence required before proceeding
- Do not implement the refactor in this step — that is `/brains.next`
