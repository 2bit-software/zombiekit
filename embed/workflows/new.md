---
name: new
description: Unified workflow entrypoint that detects work type and routes to feature/bug/refactor
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding.

## Classification Task

Analyze the user's input and determine which workflow type best matches their intent.

### Available Workflows

| Workflow | Use When | Examples |
|----------|----------|----------|
| **feature** | Adding NEW functionality | "add notifications", "implement search", "create dashboard" |
| **bug** | Fixing BROKEN behavior | "fix login", "error when clicking", "doesn't work" |
| **refactor** | Restructuring WITHOUT changing behavior | "cleanup auth code", "reorganize modules", "improve performance" |

### Classification Rules

1. **feature**: User wants functionality that doesn't currently exist
2. **bug**: User reports something that should work but doesn't
3. **refactor**: User wants to improve code structure/quality without changing what it does

### Decision Process

1. Look for explicit keywords:
   - "add", "create", "implement", "new", "build" -> likely **feature**
   - "fix", "bug", "broken", "error", "failing", "doesn't work" -> likely **bug**
   - "refactor", "cleanup", "reorganize", "simplify", "improve" -> likely **refactor**

2. If ambiguous, consider intent:
   - "Make it faster" -> Is this adding caching (feature) or optimizing existing code (refactor)?
   - "Improve error handling" -> Adding new handling (feature) or fixing broken handling (bug)?

3. If still unclear, ask one clarifying question before proceeding.

### After Classification

Once you've determined the type:

1. State your classification and brief rationale
2. Check for Linear ticket reference (see below)
3. Load the corresponding profile using `mcp__zombiekit__profile-compose` with the detected profile name ("feature", "bug", or "refactor")

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

Then call `mcp__zombiekit__profile-compose` with the detected profile name and enriched arguments.
