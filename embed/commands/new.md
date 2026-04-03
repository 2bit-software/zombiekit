---
name: new
description: Unified workflow entrypoint that detects work type and routes to feature/bug/refactor
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding.

## Pre-Classification: Active Initiative Check

Before classifying the user's input, check for an active initiative:

1. Call `mcp__zombiekit__initiative` with `action: "status"` and `dir` set to the working directory
2. If `active: false` — skip to classification below
3. If `active: true` — display the active initiative details and ask the user how to proceed:

> **Active initiative detected:**
> - **ID**: {initiative_id}
> - **Type**: {initiative_type}
> - **Current step**: {current_step} ({steps_completed}/{steps_total} steps)
>
> How would you like to proceed?
> 1. **Close out early** — Mark the current initiative as complete (keeps history) and start new work
> 2. **Delete history** — Remove the current initiative entirely and start fresh
> 3. **Cancel** — Keep working on the current initiative

Use `AskUserQuestion` to present these options. Then:
- **Option 1**: Call `mcp__zombiekit__initiative` with `action: "complete"` and `dir`, then proceed to classification
- **Option 2**: Call `mcp__zombiekit__initiative` with `action: "abandon"` and `dir`, then proceed to classification
- **Option 3**: Stop execution. Tell the user: "Continuing with the current initiative. Use `/brains.next` to advance."

## Pre-Classification: Branch Check

Before classifying, check if the current branch might cause stacked PRs:

1. Get the current branch name via `mcp__zombiekit__git` with `action: "status"` (or use Bash: `git branch --show-current`)
2. If the branch is `main`, `master`, or `develop` — skip to Classification below
3. If on any other branch, warn the user and offer options via `AskUserQuestion`:

> **You're currently on branch `{branch_name}`.**
> Starting new work here will stack changes on top of this branch.

Present these options:
- **Switch to `main`** — Check out main and pull latest
- **Switch to `develop`** — Check out develop and pull latest
- **Type a branch name** — Switch to a custom base branch (for non-standard main branches)
- **Stay on `{branch_name}`** — Continue on the current branch (stack intentionally)

Then:
- **Switch to main/develop**: Run `git checkout {branch} && git pull` via Bash. If checkout fails (branch doesn't exist), inform the user and re-prompt.
- **Type a branch name**: The user provides a branch name via the "Other" free-text option. Run `git checkout {input} && git pull`. If it fails, inform the user and re-prompt.
- **Stay**: Proceed to Classification without switching.

## Classification Task

Analyze the user's input and determine which workflow type best matches their intent.

### Available Workflows

| Workflow | Use When | Examples |
|----------|----------|----------|
| **feature-light** | Small, well-understood feature — no research or audit needed | "quick feature", "feature-light", "fl: add X" |
| **feature** | Adding NEW functionality with unknown scope | "add notifications", "implement search", "create dashboard" |
| **bug** | Fixing BROKEN behavior | "fix login", "error when clicking", "doesn't work" |
| **refactor** | Restructuring WITHOUT changing behavior | "cleanup auth code", "reorganize modules", "improve performance" |
| **unmanaged** | User handles implementation independently — just scaffold the branch and bookkeeping | "unmanaged", "manual", "self-managed" |

### Classification Rules

1. **unmanaged**: User explicitly requests it ("unmanaged", "manual", "self-managed"). Never infer this — only route here on explicit request.
2. **feature-light**: User explicitly requests it ("feature-light", "fl:", "quick feature") OR the work is clearly small and well-understood (single file change, trivial addition). When in doubt between feature and feature-light, prefer **feature**.
3. **feature**: User wants functionality that doesn't currently exist and scope is uncertain
4. **bug**: User reports something that should work but doesn't
5. **refactor**: User wants to improve code structure/quality without changing what it does

### Decision Process

1. Look for explicit keywords:
   - "unmanaged", "manual", "self-managed" -> **unmanaged** (explicit only)
   - "feature-light", "fl:", "quick feature", "small feature" -> **feature-light**
   - "add", "create", "implement", "new", "build" -> likely **feature**
   - "fix", "bug", "broken", "error", "failing", "doesn't work" -> likely **bug**
   - "refactor", "cleanup", "reorganize", "simplify", "improve" -> likely **refactor**

2. If ambiguous, consider intent:
   - "Make it faster" -> Is this adding caching (feature) or optimizing existing code (refactor)?
   - "Improve error handling" -> Adding new handling (feature) or fixing broken handling (bug)?

3. If still unclear, ask one clarifying question before proceeding.

### AutoMode Detection

Before classification, check if the user input contains the keyword **automode** (case-insensitive).

- If detected: Strip "automode" from the input text and set `AUTOMODE = true` for this session.
- If not detected: Set `AUTOMODE = false`.

When `AUTOMODE = true`, every `mcp__zombiekit__workflow-load` call for `feature`/`bug`/`refactor` must also load the automode profile via `mcp__zombiekit__profile-compose` with `profiles: ["automode"]` immediately after.

### After Classification

Once you've determined the type:

1. State your classification and brief rationale
2. Check for Linear ticket reference (see below)
3. Dispatch based on type — all use `mcp__zombiekit__workflow-load` with `type: "workflow"`:
   - **unmanaged**: `name: "unmanaged"`. No automode support.
   - **feature-light**: `name: "feature-light"`. AutoMode is handled inside the workflow — do not add it here.
   - **feature / bug / refactor**: `name: "{detected_type}"`. After loading, if AUTOMODE is true also load `mcp__zombiekit__profile-compose` with `profiles: ["automode"]`.

Example output:

```
Detected: **bug**
Rationale: User reports login is failing, which indicates broken existing functionality.
```

### Linear Ticket Detection

Before loading the profile, check if the user input references a Linear ticket:

1. **Pattern match**: Look for `[A-Z]+-[0-9]+` pattern (case-insensitive) in user input
   - Examples: "DEV-101", "proj-42", "TEAM-1234"

2. **If ticket found**:
   - Extract and uppercase the identifier (e.g., "dev-101" → "DEV-101")
   - Fetch ticket details via `mcp__linear-server__get_issue` with the identifier
   - If successful, append metadata to the user input before passing to profile:
     ```
     ---
     LINEAR_TICKET: DEV-101
     LINEAR_URL: https://linear.app/...
     LINEAR_TITLE: Ticket title here
     ```
   - If fetch fails (404, MCP unavailable): Display brief warning, proceed without metadata

3. **If no ticket found**: Proceed normally without metadata

**Example flow**:
```
User input: "work on DEV-101 add commit offer"

1. Classification: feature
2. Ticket detected: DEV-101
3. Fetch ticket → success
4. Pass to profile:
   "work on DEV-101 add commit offer

   ---
   LINEAR_TICKET: DEV-101
   LINEAR_URL: https://linear.app/heinsight/issue/DEV-101/...
   LINEAR_TITLE: Have the /brains.complete command also offer to write a commit"
```

Then dispatch using `mcp__zombiekit__workflow-load` with `type: "workflow"` for all types, passing the enriched arguments. If AUTOMODE is active and the type is `feature`/`bug`/`refactor`, also call `mcp__zombiekit__profile-compose` with `profiles: ["automode"]`.
