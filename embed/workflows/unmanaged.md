---
name: unmanaged
description: Minimal workflow — creates a branch and initiative files, then gets out of the way. The user implements independently. First /brains.next goes straight to /brains.complete.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Unmanaged Workflow

Goal: Scaffold the branch and initiative tracking, then hand off. No spec, no plan, no implement steps — you're on your own until you call `/brains.next`.

### Execution Steps

1. **Initiative Check**
   - Read `.brains/active.json`
   - If no active initiative: Create one with an auto-generated name derived from the user input
   - If active and `--new` flag present: Complete current, create new
   - If active: Confirm with the user before proceeding (an active initiative already exists)

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
   - Infer branch type prefix from the user input:

     | Inferred type | Prefix | Signal words |
     |---------------|--------|--------------|
     | feature | `feat/` | "add", "implement", "create", "new", "build", "support" |
     | bug fix | `fix/` | "fix", "bug", "broken", "error", "crash", "failing", "wrong", "incorrect" |
     | refactor | `refactor/` | "refactor", "cleanup", "reorganize", "simplify", "restructure", "rename", "move" |
     | chore | `chore/` | "chore", "bump", "upgrade", "update deps", "ci", "release", "version" |

   - If the type is clear from the input: use the corresponding prefix without asking.
   - If the type cannot be confidently inferred: ask the user once with `AskUserQuestion`:
     ```json
     {
       "questions": [{
         "question": "What type of change is this?",
         "header": "Branch type",
         "multiSelect": false,
         "options": [
           {"label": "feat", "description": "New feature or addition"},
           {"label": "fix", "description": "Bug fix"},
           {"label": "refactor", "description": "Restructuring without behaviour change"},
           {"label": "chore", "description": "Tooling, deps, CI, or housekeeping"}
         ]
       }]
     }
     ```
   - Construct branch name: `{prefix}{initiative-slug}/{description-slug}`
     (e.g., `fix/auth-api/session-token-expiry`, `chore/deps/bump-go-1-23`)
   - Create and check out the branch via `mcp__zombiekit__git`:
     - Use Bash `git checkout -b <name>` if the MCP tool does not support branch creation
   - If already on a non-main branch that matches the initiative: Skip, use current branch
   - If the branch already exists remotely: Check it out without creating

4. **Write INITIATIVE.md**
   - Create INITIATIVE.md in the initiative directory with an empty step table:
     ```markdown
     ### 1. unmanaged/{description-slug} (active)

     | Step | Status | Updated |
     |------|--------|---------|
     ```
   - No rows — the table is intentionally empty so `/brains.next` routes directly to complete

5. **Hand Off**
   - Print a brief status message:
     ```
     Branch: {branch-name}
     Initiative: {initiative-name}

     You're on your own from here. When you're done, run `/brains.next` to go straight to complete.
     ```
   - Do nothing else — no spec, no plan, no profiles loaded

### Behavior Rules

- Do not load any work profiles (implement, plan, spec, etc.)
- Do not ask clarifying questions about the work itself
- One question max if the input is genuinely ambiguous for naming purposes
- If an active initiative exists, always confirm before overwriting active state
