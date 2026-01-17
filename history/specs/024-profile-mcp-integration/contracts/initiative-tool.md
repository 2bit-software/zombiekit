# MCP Tool Contract: initiative

**Version**: 1.0.0
**Date**: 2025-12-24

## Overview

The `initiative` tool manages workflow initiative lifecycle. It handles creation, status checking, completion, and listing of initiatives.

## Tool Definition

```json
{
  "name": "initiative",
  "description": "Manage workflow initiative lifecycle. Actions: create (start new initiative), status (check current initiative), complete (finish initiative), list (show all initiatives).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "action": {
        "type": "string",
        "enum": ["create", "status", "complete", "list"],
        "description": "The lifecycle action to perform"
      },
      "dir": {
        "type": "string",
        "description": "Working directory containing the .brains folder"
      },
      "type": {
        "type": "string",
        "enum": ["feature", "bug", "refactor"],
        "description": "Required for create: Type of initiative"
      },
      "name": {
        "type": "string",
        "description": "Required for create: Name/slug for the initiative (e.g., 'user-auth')"
      },
      "description": {
        "type": "string",
        "description": "Optional for create: Description of the initiative"
      }
    },
    "required": ["action", "dir"]
  }
}
```

## Actions

### create

Creates a new initiative and returns context for the first step.

**Required Parameters**:
- `action`: "create"
- `dir`: Working directory path
- `type`: "feature" | "bug" | "refactor"
- `name`: Initiative name/slug

**Response**:
```json
{
  "action": "create",
  "initiative_id": "675d8a3f-feature-user-auth",
  "initiative_path": "history/675d8a3f-feature-user-auth",
  "cycle_id": "2025-12-24-feat-user-auth",
  "cycle_path": "history/675d8a3f-feature-user-auth/2025-12-24-feat-user-auth",
  "branch": "675d8a3f-feature-user-auth",
  "type": "feature",
  "name": "user-auth",
  "next_step": "feature"
}
```

**Side Effects**:
1. Creates initiative folder in `history/`
2. Creates cycle folder within initiative
3. Copies spec.md and research.md templates to cycle
4. Creates git branch (if in git repo, errors are warnings not failures)
5. Updates `.brains/active.json` state

### status

Returns current initiative status and suggested next action.

**Required Parameters**:
- `action`: "status"
- `dir`: Working directory path

**Response (active)**:
```json
{
  "action": "status",
  "active": true,
  "initiative_id": "675d8a3f-feature-user-auth",
  "initiative_type": "feature",
  "current_step": "feature",
  "cycle_id": "2025-12-24-feat-user-auth",
  "available_docs": ["research.md", "spec.md"],
  "suggested_next": "plan"
}
```

**Response (inactive)**:
```json
{
  "action": "status",
  "active": false
}
```

### complete

Marks the current initiative as complete and clears active state.

**Required Parameters**:
- `action`: "complete"
- `dir`: Working directory path

**Response**:
```json
{
  "action": "complete",
  "initiative_id": "675d8a3f-feature-user-auth",
  "completed_at": "2025-12-24T15:30:00Z"
}
```

**Side Effects**:
1. Clears `.brains/active.json`
2. Initiative folder remains in `history/` for reference

### list

Lists all initiatives in the history folder.

**Required Parameters**:
- `action`: "list"
- `dir`: Working directory path

**Response**:
```json
{
  "action": "list",
  "initiatives": [
    {
      "id": "675d8a3f-feature-user-auth",
      "type": "feature",
      "name": "user-auth",
      "status": "completed",
      "path": "history/675d8a3f-feature-user-auth"
    },
    {
      "id": "675d9b4e-bug-login-crash",
      "type": "bug",
      "name": "login-crash",
      "status": "active",
      "path": "history/675d9b4e-bug-login-crash"
    }
  ]
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "hint": "Suggestion to fix"
  }
}
```

**Error Codes**:

| Code | When | Hint |
|------|------|------|
| `NOT_INITIALIZED` | .brains folder missing | Run 'brains init' first |
| `MISSING_REQUIRED_PARAM` | Required parameter missing | Provide {param} |
| `INVALID_ACTION` | Unknown action | Valid actions: create, status, complete, list |
| `INITIATIVE_ALREADY_ACTIVE` | create when active exists | Complete current initiative first |
| `NO_ACTIVE_INITIATIVE` | complete/step when none active | Create an initiative first |

## Git Integration

The `initiative create` action automatically creates a git branch:

- **Branch naming**: `{type-prefix}/{name}` where prefix is:
  - feature → `feat/`
  - bug → `fix/`
  - refactor → `ref/`

- **Failure handling**: Git failures are warnings, not errors. The initiative is created even if git operations fail. The response includes `branch` field (may be empty) and optionally `git_warning` field.

## State File

The tool manages `.brains/active.json`:

```json
{
  "initiative": "history/675d8a3f-feature-user-auth",
  "cycle": "history/675d8a3f-feature-user-auth/2025-12-24-feat-user-auth",
  "current_step": "feature"
}
```

This file is:
- Created/updated on `create`
- Read on `status`
- Cleared on `complete`
