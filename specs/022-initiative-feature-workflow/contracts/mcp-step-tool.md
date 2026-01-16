# MCP Step Tool Contract (Extended)

**Version**: 1.1.0
**Status**: Draft
**Date**: 2025-12-23

## Overview

This contract defines the extended MCP step tool interface for the feature workflow. It extends the existing step tool to support cycle creation and returns enhanced response data.

## Tool Definition

### Name
`step`

### Description
Execute a workflow step within an initiative. Creates cycles, copies templates, manages git branches, and returns directive text with orchestration instructions.

### Input Schema

```json
{
  "type": "object",
  "properties": {
    "step": {
      "type": "string",
      "description": "Step name to execute. Built-in steps: init, feature, specify, plan, tasks, implement, audit, clarify, complete"
    },
    "dir": {
      "type": "string",
      "description": "Working directory containing the .brains folder. Used for profile resolution and initiative state."
    },
    "initiative": {
      "type": "string",
      "description": "Optional: Override the current active initiative. Path relative to history/ folder (e.g., '675d8a3f-feature-user-auth')"
    },
    "type": {
      "type": "string",
      "enum": ["feature", "bug", "refactor"],
      "description": "Initiative type. Required for 'init' and 'feature' steps when creating new initiative."
    },
    "name": {
      "type": "string",
      "description": "Name/slug for the new initiative or cycle (e.g., 'user-auth'). Required for 'init' and 'feature' steps."
    },
    "description": {
      "type": "string",
      "description": "Optional: Description of the feature or initiative."
    },
    "new_initiative": {
      "type": "boolean",
      "description": "Optional: Force creation of a new initiative even if one is active. Default false."
    }
  },
  "required": ["step", "dir"]
}
```

## Response Schema

### Success Response

```json
{
  "type": "object",
  "properties": {
    "directive": {
      "type": "string",
      "description": "Multi-phase instruction text for the LLM to follow."
    },
    "history_folder": {
      "type": "string",
      "description": "Absolute path to the initiative folder (deprecated, use initiative_folder)."
    },
    "initiative_folder": {
      "type": "string",
      "description": "Absolute path to the initiative folder."
    },
    "cycle_folder": {
      "type": "string",
      "description": "Absolute path to the active cycle folder."
    },
    "files_to_read": {
      "type": "array",
      "items": { "type": "string" },
      "description": "List of absolute file paths the LLM should read for context."
    },
    "composed_prompt": {
      "type": "string",
      "description": "Merged content from profiles specified in the step definition."
    },
    "workflow_phases": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "description": { "type": "string" },
          "agents": { "type": "array", "items": { "type": "string" } },
          "outputs": { "type": "array", "items": { "type": "string" } },
          "parallel": { "type": "boolean" }
        }
      },
      "description": "Structured description of each workflow phase."
    }
  },
  "required": ["directive", "initiative_folder", "cycle_folder", "files_to_read"]
}
```

### Error Response

```json
{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "description": "Machine-readable error code."
    },
    "message": {
      "type": "string",
      "description": "Human-readable error message."
    },
    "suggestion": {
      "type": "string",
      "description": "Suggested action to resolve the error."
    }
  },
  "required": ["code", "message"]
}
```

## Error Codes

| Code | Message | Suggestion |
|------|---------|------------|
| `NOT_INITIALIZED` | Directory does not contain a .brains folder | Run 'brains init' in the project directory first |
| `MISSING_STEP` | Missing required parameter: step | Provide the step name |
| `MISSING_DIR` | Missing required parameter: dir | Provide the working directory path |
| `MISSING_TYPE` | Type parameter is required for feature step | Provide type: feature, bug, or refactor |
| `MISSING_NAME` | Name parameter is required for feature step | Provide a name for the feature (e.g., 'user-auth') |
| `INVALID_TYPE` | Invalid initiative type '{type}' | Type must be one of: feature, bug, refactor |
| `INITIATIVE_NOT_FOUND` | Initiative '{id}' not found in history/ | Check the initiative path or use 'init' to create a new one |
| `NO_ACTIVE_INITIATIVE` | No active initiative | Run step='feature' with name parameter to create a new initiative |
| `STEP_NOT_FOUND` | Step '{name}' not found | Available steps: init, feature, specify, plan, tasks, implement, audit, clarify, complete |

## Feature Step Behavior

### Creating New Initiative + Cycle

When `step="feature"` is called:

1. **If no active initiative OR `new_initiative=true`**:
   - Create initiative folder: `./history/{hex}-{name}/`
   - Create INITIATIVE.md with metadata
   - Create cycle folder: `./history/{hex}-{name}/{hex}-feat-{name}/`
   - Copy templates to cycle folder
   - Create/switch git branch: `feat/{name}`
   - Update `.brains/active.json`

2. **If active initiative exists**:
   - Reuse existing initiative folder
   - Create new cycle folder: `./history/{existing-init}/{hex}-feat-{name}/`
   - Copy templates to cycle folder
   - Do NOT change git branch
   - Update `.brains/active.json` with new cycle

### Template Copying

Templates are copied from embedded filesystem or local `.brains/templates/`:

| Source | Destination | Purpose |
|--------|-------------|---------|
| `spec-template.md` | `{cycle}/spec.md` | Feature specification |
| `research-template.md` | `{cycle}/research.md` | Research findings |
| N/A | `{cycle}/audit/` | Audit reports directory |

### Git Branch Handling

| Initiative Type | Branch Prefix |
|-----------------|---------------|
| feature | `feat/` |
| bug | `fix/` |
| refactor | `ref/` |

Branch operations fail gracefully if git is not available or directory is not a git repository.

## Example Usage

### Create New Feature

**Request:**
```json
{
  "step": "feature",
  "dir": "/path/to/project",
  "type": "feature",
  "name": "user-auth",
  "description": "Add user authentication with OAuth2"
}
```

**Response:**
```json
{
  "directive": "# Feature Specification Workflow\n\n## Phase I: Research...",
  "initiative_folder": "/path/to/project/history/675d8a3f-feature-user-auth",
  "cycle_folder": "/path/to/project/history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth",
  "files_to_read": [
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth/spec.md",
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth/research.md"
  ],
  "composed_prompt": "# Research Guidelines\n...\n# Create Guidelines\n...",
  "workflow_phases": [
    {
      "name": "research",
      "description": "Gather context and domain knowledge",
      "agents": ["research-codebase", "research-domain"],
      "outputs": ["research.md"],
      "parallel": true
    },
    {
      "name": "create",
      "description": "Synthesize specification from research",
      "agents": ["spec-writer"],
      "outputs": ["spec.md"],
      "parallel": false
    },
    {
      "name": "audit",
      "description": "Check specification quality and completeness",
      "agents": ["audit-completeness", "audit-ai-readiness"],
      "outputs": ["audit/{date}.md"],
      "parallel": true
    },
    {
      "name": "highlight",
      "description": "Present key decisions for user approval",
      "agents": ["highlighter"],
      "outputs": [],
      "parallel": false
    }
  ]
}
```

### Add Refactor Cycle to Existing Initiative

**Request:**
```json
{
  "step": "feature",
  "dir": "/path/to/project",
  "type": "refactor",
  "name": "user-auth"
}
```

**Response:**
```json
{
  "directive": "# Refactor Specification Workflow\n\n## Phase I: Research...",
  "initiative_folder": "/path/to/project/history/675d8a3f-feature-user-auth",
  "cycle_folder": "/path/to/project/history/675d8a3f-feature-user-auth/675d8a41-ref-user-auth",
  "files_to_read": [
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a41-ref-user-auth/spec.md",
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a41-ref-user-auth/research.md",
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth/spec.md",
    "/path/to/project/history/675d8a3f-feature-user-auth/675d8a40-feat-user-auth/research.md"
  ],
  "composed_prompt": "...",
  "workflow_phases": [...]
}
```

Note: `files_to_read` includes artifacts from previous cycles for context.

## Backward Compatibility

- `history_folder` is preserved for backward compatibility but deprecated
- New clients should use `initiative_folder` and `cycle_folder`
- Existing step names continue to work unchanged
- The `feature` step is additive, not replacing existing steps
