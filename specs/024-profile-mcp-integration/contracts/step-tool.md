# MCP Tool Contract: step

**Version**: 1.0.0
**Date**: 2025-12-24

## Overview

The `step` tool executes workflow steps within an active initiative. It returns structured responses containing directives, file paths, composed prompts, and step-specific data.

## Tool Definition

```json
{
  "name": "step",
  "description": "Execute a workflow step within an active initiative. Returns directive text, file paths, and composed profile prompt. Requires an active initiative (created via 'initiative' tool).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "step": {
        "type": "string",
        "description": "Step name to execute: feature, bug, refactor, plan, tasks, eat, audit, clarify"
      },
      "dir": {
        "type": "string",
        "description": "Working directory containing the .brains folder"
      },
      "initiative": {
        "type": "string",
        "description": "Optional: Override the current active initiative. Path relative to history/ folder (e.g., '675d8a3f-feature-user-auth')"
      }
    },
    "required": ["step", "dir"]
  }
}
```

## Available Steps

| Step | Type | Prerequisite | Description |
|------|------|--------------|-------------|
| feature | Multi-phase | None | Research→Create→Audit→Highlight workflow |
| bug | Multi-phase | None | Bug investigation and fix specification |
| refactor | Multi-phase | None | Refactoring with behavior preservation |
| plan | Single-phase | spec.md (approved) | Create implementation plan |
| tasks | Single-phase | plan.md (approved) | Generate task breakdown |
| eat | Single-phase | tasks.md (exists) | Execute implementation tasks |
| audit | Single-phase | None | Cross-artifact alignment check |
| clarify | Single-phase | None | Ambiguity detection and resolution |

## Response Schema

### Base Response

All steps return this base structure:

```json
{
  "directive": "# Step Directive\n\nMarkdown content...",
  "history_folder": "/abs/path/to/history/initiative",
  "initiative_folder": "/abs/path/to/history/initiative",
  "cycle_folder": "/abs/path/to/history/initiative/cycle",
  "files_to_read": [
    "/abs/path/to/research.md",
    "/abs/path/to/spec.md"
  ],
  "composed_prompt": "Concatenated profile content...",
  "prerequisites": {
    "met": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| directive | string | Markdown instructions from step profile |
| history_folder | string | Deprecated. Use initiative_folder |
| initiative_folder | string | Absolute path to initiative folder |
| cycle_folder | string | Absolute path to active cycle folder |
| files_to_read | string[] | Resolved absolute paths to context files |
| composed_prompt | string | Concatenated content from listed profiles |
| prerequisites | object | Prerequisite check result |

### Multi-Phase Steps (feature, bug, refactor)

Additional fields for multi-phase workflows:

```json
{
  "workflow_phases": [
    {
      "name": "research",
      "description": "Gather context and domain knowledge through parallel research agents",
      "agents": ["research-codebase", "research-domain"],
      "outputs": ["research.md"],
      "parallel": true
    },
    {
      "name": "create",
      "description": "Synthesize specification from research findings",
      "agents": ["spec-writer"],
      "outputs": ["spec.md"],
      "parallel": false
    },
    {
      "name": "audit",
      "description": "Check specification quality and completeness with severity classification",
      "agents": ["audit-completeness", "audit-ai-readiness"],
      "outputs": ["audit/{date}.md"],
      "parallel": true
    },
    {
      "name": "highlight",
      "description": "Present key decisions for user approval before proceeding",
      "agents": ["highlighter"],
      "outputs": [],
      "parallel": false
    }
  ]
}
```

### Eat Step

Additional field for task tracking:

```json
{
  "next_task": {
    "id": "T005",
    "description": "Implement user authentication endpoint",
    "phase": "Phase 2: Core Implementation"
  }
}
```

If all tasks are complete, `next_task` is null and `directive` indicates completion.

### Prerequisites Not Met

When prerequisites fail:

```json
{
  "prerequisites": {
    "met": false,
    "required": "spec.md with status: approved",
    "hint": "Run feature, bug, or refactor first and approve the spec"
  }
}
```

## Error Responses

Errors are thrown (not returned in response):

| Code | When | Hint |
|------|------|------|
| `NOT_INITIALIZED` | .brains folder missing | Run 'brains init' first |
| `NO_ACTIVE_INITIATIVE` | No active initiative | Create an initiative first |
| `INITIATIVE_NOT_FOUND` | Override initiative doesn't exist | Check path |
| `STEP_NOT_FOUND` | Unknown step name | Valid: feature, bug, refactor, plan, tasks, eat, audit, clarify |
| `PREREQUISITE_NOT_MET` | Required artifact missing/unapproved | (specific hint provided) |

## Response Field Usage

Agents should process response fields in this order:

1. **Check prerequisites.met** - If false, follow hint to unblock
2. **Read files_to_read** - Load context files before proceeding
3. **Parse workflow_phases** - (Multi-phase only) Understand phase structure
4. **Check next_task** - (Eat step only) Know which task to implement
5. **Follow directive** - Execute the step according to instructions
6. **Use cycle_folder** - Output artifacts to this directory
7. **Reference composed_prompt** - Additional context from profiles

## Step Profiles

Steps load their directive from profiles with priority:

1. **Local**: `.brains/steps/{step}.md`
2. **Global**: `~/.brains/steps/{step}.md`
3. **Embedded**: `templates/steps/{step}.md` (in binary)

Local overrides global, which overrides embedded.

## File Pattern Resolution

The `files` field in step frontmatter contains glob patterns:

```yaml
files:
  - "research.md"
  - "spec.md"
  - "audit/**/*.md"
  - "../**/research.md"
```

Patterns are resolved relative to `cycle_folder`. Only existing files are included in `files_to_read`.

## Profile Composition

The `profiles` field in step frontmatter lists profiles to compose:

```yaml
profiles:
  - research
  - create
  - audit
```

Each profile's content is loaded and concatenated into `composed_prompt`.

## State Updates

After successful step execution, the tool updates `.brains/active.json`:

```json
{
  "current_step": "feature"
}
```

This tracks the last executed step for status reporting.
