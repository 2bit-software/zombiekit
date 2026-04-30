---
name: feature
description: Feature specification workflow — creates a branch, researches and writes a business spec, audits it for completeness. /brains.next advances through plan, tasks, and implement.
steps:
  - name: spec
    profiles: [feature]
  - name: plan
    profiles: [plan]
  - name: tasks
    profiles: [tasks]
  - name: implement
    profiles: [implement]
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

## Feature Workflow

Goal: Produce a complete, audited feature spec through the research→create→audit cycle. Planning, tasking, and implementation happen in subsequent steps via `/brains.next`.

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
     - Derive a branch name: `feat/{initiative-slug}/{feature-slug}`
       (e.g., `feat/auth-api/add-refresh-endpoint`)
     - Create and check out the branch via `mcp__zombiekit__git`:
       - Use Bash `git checkout -b <name>` if the MCP tool does not support branch creation
     - If already on a non-main branch that matches the initiative: Skip, use current branch
     - If the branch already exists remotely: Check it out without creating

4. **Write INITIATIVE.md Step Table**
   - Create or update INITIATIVE.md in the initiative directory with a cycle entry:
     ```markdown
     ### {n}. feat/{feature-slug} (active)

     | Step | Status | Updated |
     |------|--------|---------|
     | spec | in_progress | {now} |
     | plan | pending | - |
     | tasks | pending | - |
     | implement | pending | - |
     ```

5. **Separation Phase**
   - Split the user input into two concerns:
     - **Business requirements** (user-visible behavior, outcomes, constraints) → `business-spec.md` stub
     - **Technical preferences** (implementation hints, technology choices) → `technical-requirements-research.md`
   - `business-spec.md` must describe *what*, not *how*

6. **Research Phase** (parallel agents)
   - Spawn a research-codebase agent: explore existing patterns, interfaces, and constraints relevant to this feature
   - Spawn a research-domain agent: gather domain knowledge, prior art, and edge cases
   - Spawn additional domain-specific agents if the input calls for it
   - When the feature depends on existing component behavior: use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to verify actual state transitions and data flow — do not infer from method names alone
   - Collate and deduplicate findings into `research-summary.md`

7. **Create Phase**
   - Synthesize research into a complete `business-spec.md`:
     - Functional requirements (user-visible behavior)
     - Acceptance criteria (bulleted, testable)
     - Out of scope (explicit exclusions)
     - Open questions (unresolved decisions)
   - Update `technical-requirements-research.md` with research findings

8. **Audit Phase** (parallel agents)
   - Run audit-completeness: check for missing requirements, untested edge cases, ambiguous language
   - Run audit-ai-consumer: check the spec is concrete enough for an AI to implement without guessing
   - Classify findings: CRITICAL (blocks implementation), MAJOR (significant gap), MINOR (polish)

9. **Loop or Highlight**
   - If CRITICAL or MAJOR findings: Return to research/create with feedback, maximum 3 iterations
   - If MINOR or none: Present key decisions and open questions to the user for approval before proceeding
   - Never proceed past this step without explicit user sign-off

10. **Report Completion**
    - Path to created artifacts
    - Summary of key decisions and open questions resolved
    - Remind user: `/brains.next` to begin planning

### Artifact Structure

```
history/{id}-feature-{slug}/
  INITIATIVE.md
  business-spec.md
  technical-requirements-research.md
  research-summary.md
  audit-reports/
```

### Behavior Rules

- Maximum 3 audit iterations before escalating to the user
- Never proceed past the highlight step without user approval
- `business-spec.md` describes behavior only — no implementation details
- Do not begin planning or implementation in this step — that is `/brains.next`
