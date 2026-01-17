# Quickstart: Initiatives Step Framework

## Overview

The Initiatives Step Framework provides MCP tool support for structured development workflows. It enables Claude Code to execute workflow steps (specify, plan, implement, etc.) within the context of an initiative (feature, bug, refactor).

## Prerequisites

- Go 1.24+ installed
- Project initialized with `brains init`
- MCP server running (`brains serve`)

## Basic Usage

### 1. Create a New Initiative

Call the MCP `step` tool with `step="init"`:

```json
{
  "step": "init",
  "dir": "/path/to/project",
  "type": "feature",
  "name": "user-auth"
}
```

This creates:
- `history/675d8a3f-feature-user-auth/` folder
- `INITIATIVE.md` with metadata
- Updates `.brains/active.json` to track this as active

### 2. Execute a Workflow Step

```json
{
  "step": "specify",
  "dir": "/path/to/project"
}
```

Response:
```json
{
  "directive": "Your task is to create or update the feature specification...",
  "history_folder": "/path/to/project/history/675d8a3f-feature-user-auth",
  "files_to_read": ["spec.md", "research.md"],
  "composed_prompt": "# Research Methodology\n\n..."
}
```

### 3. Available Steps

| Step | Purpose | Requires Active Initiative |
|------|---------|---------------------------|
| `init` | Create new initiative | No (creates one) |
| `specify` | Create/update spec | Yes |
| `plan` | Create implementation plan | Yes |
| `tasks` | Break down into tasks | Yes |
| `implement` | Execute implementation | Yes |
| `audit` | Check artifact alignment | Yes |
| `clarify` | Surface ambiguities | Yes |
| `complete` | Mark initiative done | Yes |

### 4. Override Active Initiative

To work on a different initiative temporarily:

```json
{
  "step": "specify",
  "dir": "/path/to/project",
  "initiative": "675d8b12-bug-login-crash"
}
```

## Custom Step Definitions

Create `.brains/steps/{step-name}.md` to override defaults:

```markdown
---
name: specify
description: Custom specification step for our team
profiles:
  - research
  - our-spec-template
files:
  - "spec.md"
  - "requirements/*.md"
---

# Custom Directive

Your task is to create a specification following our team's guidelines...
```

## File Structure After Usage

```
project/
├── .brains/
│   ├── active.json              # Current initiative state
│   ├── profiles/
│   └── steps/                   # Custom step overrides
│
├── history/
│   ├── 675d8a3f-feature-user-auth/
│   │   ├── INITIATIVE.md
│   │   ├── spec.md
│   │   ├── plan.md
│   │   └── tasks.md
│   │
│   └── 675d8b12-bug-login-crash/
│       └── ...
│
└── (source code)
```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| `NOT_INITIALIZED` | No .brains folder | Run `brains init` |
| `UNKNOWN_STEP` | Invalid step name | Check available steps |
| `NO_ACTIVE_INITIATIVE` | No active initiative | Run `init` step first |
| `INVALID_TYPE` | Bad initiative type | Use feature/bug/refactor |

## Integration with Profile System

Steps compose profiles using the existing profile system:

1. Step definition specifies `profiles: [research, spec-creator]`
2. Step service calls `profile.Service.Compose(["research", "spec-creator"])`
3. Composed content returned in `composed_prompt` field

This means project-level profile overrides work automatically with steps.
