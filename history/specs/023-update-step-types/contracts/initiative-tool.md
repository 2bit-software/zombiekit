# MCP Tool Contract: initiative

**Tool Name**: `initiative`
**Purpose**: Manage workflow initiative lifecycle (create, status, complete, list)

## Input Schema

```json
{
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
```

## Response Schemas

### action=create

```json
{
  "action": "create",
  "initiative_id": "abc123-user-auth",
  "initiative_path": "/path/to/history/abc123-user-auth",
  "cycle_id": "abc124-feat-user-auth",
  "cycle_path": "/path/to/history/abc123-user-auth/abc124-feat-user-auth",
  "branch": "abc123-user-auth",
  "type": "feature",
  "name": "user-auth",
  "next_step": "feature"
}
```

### action=status

```json
{
  "action": "status",
  "active": true,
  "initiative_id": "abc123-user-auth",
  "initiative_type": "feature",
  "current_step": "feature",
  "cycle_id": "abc124-feat-user-auth",
  "available_docs": ["spec.md", "research.md"],
  "suggested_next": "plan"
}
```

When no active initiative:
```json
{
  "action": "status",
  "active": false,
  "suggested_next": "initiative create"
}
```

### action=complete

```json
{
  "action": "complete",
  "initiative_id": "abc123-user-auth",
  "completed_at": "2025-12-24T12:00:00Z"
}
```

### action=list

```json
{
  "action": "list",
  "initiatives": [
    {
      "id": "abc123-user-auth",
      "type": "feature",
      "name": "user-auth",
      "status": "completed",
      "path": "/path/to/history/abc123-user-auth"
    },
    {
      "id": "def456-fix-timeout",
      "type": "bug",
      "name": "fix-timeout",
      "status": "active",
      "path": "/path/to/history/def456-fix-timeout"
    }
  ]
}
```

## Error Responses

### NO_ACTIVE_INITIATIVE (for complete)
```json
{
  "error": {
    "code": "NO_ACTIVE_INITIATIVE",
    "message": "No active initiative to complete",
    "hint": "Use 'initiative create' to start a new initiative first"
  }
}
```

### INITIATIVE_ALREADY_ACTIVE (for create)
```json
{
  "error": {
    "code": "INITIATIVE_ALREADY_ACTIVE",
    "message": "An initiative is already active: abc123-user-auth",
    "hint": "Complete or abandon the current initiative first with 'initiative complete'"
  }
}
```

### MISSING_REQUIRED_PARAM (for create)
```json
{
  "error": {
    "code": "MISSING_REQUIRED_PARAM",
    "message": "Missing required parameter: type",
    "hint": "Provide type (feature|bug|refactor) for create action"
  }
}
```

### INVALID_ACTION
```json
{
  "error": {
    "code": "INVALID_ACTION",
    "message": "Invalid action: 'foo'",
    "hint": "Valid actions: create, status, complete, list"
  }
}
```

## Validation Rules

| Action | Required Params | Preconditions |
|--------|-----------------|---------------|
| create | action, dir, type, name | No active initiative |
| status | action, dir | None |
| complete | action, dir | Active initiative exists |
| list | action, dir | None |

## Side Effects

| Action | Creates/Modifies |
|--------|------------------|
| create | Creates initiative folder, cycle folder, git branch, copies templates, sets active state |
| status | None (read-only) |
| complete | Updates INITIATIVE.md status, clears active state |
| list | None (read-only) |
