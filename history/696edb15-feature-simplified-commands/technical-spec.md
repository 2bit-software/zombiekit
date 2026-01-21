# Technical Specification: Workflow Entrypoints

## Overview

Add a unified `/brains.new` command that detects work type (feature, bug, refactor) and routes to the appropriate profile. No code changes needed - just new profile and command files.

## New Profile Type Value

The existing `type` field in profile frontmatter gains a new valid value:

| Type | Purpose |
|------|---------|
| `skill` | Instructional profiles with workflow logic |
| `step` | Orchestration definitions (compose multiple profiles) |
| `workflow` | Entrypoint profiles with classification/routing |

No struct changes needed - `type` is already a string field.

## MCP Tool Change: profile-compose

Add a `workflow` boolean parameter to disambiguate when profiles share names.

### Input Schema Update

```json
{
  "type": "object",
  "properties": {
    "profiles": {
      "type": "array",
      "items": { "type": "string" },
      "description": "List of profile names to compose"
    },
    "working_directory": {
      "type": "string",
      "description": "Working directory for profile resolution"
    },
    "workflow": {
      "type": "boolean",
      "description": "If true, filter to type:workflow profiles only. If false/omitted, filter to non-workflow profiles."
    }
  },
  "required": ["profiles"]
}
```

### Behavior

| `workflow` value | Profiles loaded |
|------------------|-----------------|
| `true` | Only `type: workflow` profiles |
| `false` or omitted | Only non-workflow profiles (`type: skill`, `type: step`, or no type) |

### Implementation

In `internal/mcp/tools/profile/tool.go`:

1. Add `Workflow *bool` to input struct
2. Pass filter to profile service
3. Profile service filters by type before composition

In `internal/profile/service.go`:

1. Add `workflowOnly bool` parameter to `Compose` method (or use options pattern)
2. Filter loaded profiles by type before building composition graph

## File: `profiles/new.md`

```markdown
---
name: new
description: Unified workflow entrypoint that detects work type and routes to feature/bug/refactor
type: workflow
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
   - "add", "create", "implement", "new", "build" → likely **feature**
   - "fix", "bug", "broken", "error", "failing", "doesn't work" → likely **bug**
   - "refactor", "cleanup", "reorganize", "simplify", "improve" → likely **refactor**

2. If ambiguous, consider intent:
   - "Make it faster" → Is this adding caching (feature) or optimizing existing code (refactor)?
   - "Improve error handling" → Adding new handling (feature) or fixing broken handling (bug)?

3. If still unclear, ask one clarifying question before proceeding.

### After Classification

Once you've determined the type:

1. State your classification and brief rationale
2. Immediately load the corresponding profile:

```
Detected: **bug**
Rationale: User reports login is failing, which indicates broken existing functionality.
```

Then call `mcp__zombiekit__profile-compose` with the detected profile name ("feature", "bug", or "refactor") and continue with that workflow.
```

## File: `integrations/claude/commands/brains.new.md`

```markdown
---
description: Start new work - automatically detects if this is a feature, bug, or refactor
---

Use the mcp__zombiekit__profile-compose tool with `workflow: true` to load the "new" workflow profile. Use this as your system prompt for the query.

ARGUMENTS: $ARGUMENTS
```

## Commands to Delete

Remove after implementing `brains.new`:

- `integrations/claude/commands/brains.feature.md`
- `integrations/claude/commands/brains.bug.md`
- `integrations/claude/commands/brains.refactor.md`

## Init Command Update

Update the command list in `internal/cli/init.go` to:
1. Include `brains.new.md`
2. Exclude deleted feature/bug/refactor commands

## Testing

### Unit Tests (MCP tool change)

| Test | Input | Expected |
|------|-------|----------|
| Workflow filter true | `profiles: ["new"], workflow: true` | Loads only workflow-type profiles |
| Workflow filter false | `profiles: ["feature"], workflow: false` | Loads only non-workflow profiles |
| Workflow filter omitted | `profiles: ["feature"]` | Same as false - loads non-workflow |
| Name collision | Two profiles named "x" (one workflow, one skill) | Filter selects correct one |

### Manual Tests (end-to-end)

| Input | Expected Detection |
|-------|-------------------|
| `/brains.new fix the login bug` | bug |
| `/brains.new add user notifications` | feature |
| `/brains.new cleanup the auth module` | refactor |
| `/brains.new make it faster` | asks clarifying question |

## Migration

- Existing `.claude/commands/brains.feature.md` etc. will continue to work
- Users can manually delete old commands or wait for `brains init --force`
- No breaking changes - old commands just become redundant
