---
name: bug
description: Bug investigation workflow — creates a branch, investigates root cause, classifies the failure, and produces a fix plan. /brains.next advances to implement then audit.
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

## Bug Workflow

Goal: Investigate a bug, determine root cause, classify the failure, and produce a fix plan. Implementation and verification happen in subsequent steps via `/brains.next`.

### Execution Steps

1. **Initiative Check**
   - Read `.brains/active.json`
   - If no active initiative: Create one with an auto-generated name derived from the user input
   - If active and `--new` flag present: Complete current, create new
   - If active: Add this bug to the current initiative
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
   - Derive a branch name: `fix/{initiative-slug}/{bug-slug}`
     (e.g., `fix/auth-api/session-token-not-refreshing`)
   - Create and check out the branch via `mcp__zombiekit__git`:
     - Use Bash `git checkout -b <name>` if the MCP tool does not support branch creation
   - If already on a non-main branch that matches the initiative: Skip, use current branch
   - If the branch already exists remotely: Check it out without creating

4. **Write INITIATIVE.md Step Table**
   - Create or update INITIATIVE.md in the initiative directory with a cycle entry:
     ```markdown
     ### {n}. bug/{bug-slug} (active)

     | Step | Status | Updated |
     |------|--------|---------|
     | investigate | in_progress | {now} |
     | plan | pending | - |
     | tasks | pending | - |
     | fix | pending | - |
     | verify | pending | - |
     ```

5. **Report Phase**
   - Capture the original bug report in `report.md`:
     - Symptoms and observable behavior
     - Error messages, stack traces, or log output (verbatim)
     - Environment details (OS, version, config)
     - Steps that trigger the issue

6. **Reproduction Phase**
   - Create `reproduction.md` with:
     - Minimal reproduction steps
     - Prerequisites and environment setup
     - Expected vs actual behavior
   - Identify or create a failing test case that demonstrates the bug
   - **Do not proceed to investigation until a reliable reproduction exists**

7. **Investigation Phase**
   - Spawn a research-codebase agent to trace the relevant code paths
   - Use `codebase-memory-mcp` tools (`search_code`, `trace_call_path`) to follow execution from entry point to failure
   - Verify actual code behavior — do not infer from method names alone
   - Document findings in `investigation.md`:
     - Relevant files and functions
     - Execution flow to the failure point
     - The specific line or condition that produces the wrong behavior

8. **Classification Phase**
   - Determine failure type and document in `classification.md`:
     - **Implementation Error**: Code does not match the intended design
     - **Spec Gap**: Behavior is undefined or ambiguous — no correct implementation exists
   - Include supporting evidence from investigation

9. **Fix Planning**
   - Create `fix-plan.md` with the required changes:
     - Specific files and functions to modify
     - The change to make at each location
     - Tests to add or update for verification
   - If spec gap: also create `spec-update.md` with proposed spec changes before the fix

10. **Report Completion**
    - Root cause summary (one paragraph)
    - Classification and evidence
    - List of artifacts created
    - Remind user: `/brains.next` to begin implementation

### Artifact Structure

```
history/{id}-bug-{slug}/
  INITIATIVE.md
  report.md
  reproduction.md
  investigation.md
  classification.md
  fix-plan.md
  spec-update.md  (if spec gap)
```

### Behavior Rules

- Never assume root cause — evidence required before classification
- A reliable reproduction must exist before investigation begins
- Spec gaps require `spec-update.md` before `fix-plan.md`
- Do not implement the fix in this step — that is `/brains.next`
