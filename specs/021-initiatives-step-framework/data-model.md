# Data Model: Initiatives Step Framework

**Date**: 2025-12-23
**Feature**: 021-initiatives-step-framework

## Entities

### Initiative

Represents a unit of work (feature, bug, refactor) being tracked.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Unique identifier (e.g., "675d8a3f-feature-user-auth") |
| `type` | enum | yes | One of: "feature", "bug", "refactor" |
| `name` | string | yes | Human-readable name slug (e.g., "user-auth") |
| `path` | string | yes | Absolute path to initiative folder |
| `status` | enum | yes | One of: "active", "completed" |
| `created_at` | timestamp | yes | When the initiative was created |
| `updated_at` | timestamp | yes | Last activity timestamp |

**Validation Rules**:
- `type` must be one of the defined enum values
- `name` must be lowercase alphanumeric with hyphens only
- `path` must exist and be a directory

**State Transitions**:
```
[none] --create--> active --complete--> completed
```

### InitiativeState

Tracks the currently active initiative for a project. Stored in `.brains/active.json`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `initiative` | string | no | Relative path to active initiative (from project root) |
| `type` | enum | no | Type of active initiative |
| `name` | string | no | Name of active initiative |
| `started` | timestamp | no | When this initiative became active |
| `last_activity` | timestamp | no | Last step execution time |
| `status` | enum | no | Current status |
| `current_step` | string | no | Last executed step |

**Notes**:
- All fields are optional to support "no active initiative" state (empty file or `{}`)
- File is gitignored to avoid developer conflicts

### Step

A workflow step definition.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Step identifier (e.g., "specify", "plan") |
| `description` | string | no | Human-readable description |
| `profiles` | []string | no | Profile names to compose for this step |
| `files` | []string | no | Glob patterns for files to read |
| `directive` | string | yes | The instruction text for this step |
| `type` | string | no | Always "step" for step definitions |
| `source` | enum | yes | One of: "embedded", "global", "local" |
| `path` | string | no | Absolute path if loaded from file |

**Resolution Order** (first match wins):
1. Local: `.brains/steps/{name}.md`
2. Global: `~/.brains/steps/{name}.md`
3. Embedded: Built-in defaults

### StepResponse

The structured output from executing a step via MCP.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `directive` | string | yes | The step directive/instruction text |
| `history_folder` | string | yes | Absolute path to the initiative's history folder |
| `files_to_read` | []string | yes | List of files the agent should read |
| `composed_prompt` | string | yes | Pre-composed profile prompt for this step |

## Relationships

```
┌─────────────────┐
│ InitiativeState │──────references────┐
└─────────────────┘                    │
                                       ▼
┌─────────────────┐         ┌─────────────────┐
│      Step       │         │   Initiative    │
│                 │         │                 │
│  profiles[] ────┼────────▶│  (folder on     │
│  files[]        │         │   filesystem)   │
│  directive      │         │                 │
└─────────────────┘         └─────────────────┘
        │
        │ compose()
        ▼
┌─────────────────┐
│ profile.Service │
│                 │
│  Compose()      │
└─────────────────┘
```

## File System Layout

```
project/
├── .brains/
│   ├── active.json          # InitiativeState (gitignored)
│   ├── profiles/            # Project-specific profiles
│   └── steps/               # Project-specific step overrides
│       └── specify.md       # Custom specify step
│
├── history/
│   ├── 675d8a3f-feature-user-auth/
│   │   ├── INITIATIVE.md    # Initiative metadata
│   │   ├── spec.md
│   │   ├── plan.md
│   │   └── tasks.md
│   │
│   └── 675d8b12-bug-login-crash/
│       └── ...
│
└── (source code)

~/.brains/
├── profiles/                # Global profiles
└── steps/                   # Global step definitions
```

## Go Type Definitions

```go
// internal/initiative/types.go

type InitiativeType string

const (
    TypeFeature  InitiativeType = "feature"
    TypeBug      InitiativeType = "bug"
    TypeRefactor InitiativeType = "refactor"
)

type InitiativeStatus string

const (
    StatusActive    InitiativeStatus = "active"
    StatusCompleted InitiativeStatus = "completed"
)

type Initiative struct {
    ID        string           `json:"id"`
    Type      InitiativeType   `json:"type"`
    Name      string           `json:"name"`
    Path      string           `json:"path"`
    Status    InitiativeStatus `json:"status"`
    CreatedAt time.Time        `json:"created_at"`
    UpdatedAt time.Time        `json:"updated_at"`
}

type InitiativeState struct {
    Initiative   string           `json:"initiative,omitempty"`
    Type         InitiativeType   `json:"type,omitempty"`
    Name         string           `json:"name,omitempty"`
    Started      time.Time        `json:"started,omitempty"`
    LastActivity time.Time        `json:"last_activity,omitempty"`
    Status       InitiativeStatus `json:"status,omitempty"`
    CurrentStep  string           `json:"current_step,omitempty"`
}

// internal/step/types.go

type StepSource int

const (
    SourceEmbedded StepSource = iota
    SourceGlobal
    SourceLocal
)

type Step struct {
    Name        string     `json:"name"`
    Description string     `json:"description,omitempty"`
    Profiles    []string   `json:"profiles,omitempty"`
    Files       []string   `json:"files,omitempty"`
    Directive   string     `json:"directive"`
    Type        string     `json:"type,omitempty"`
    Source      StepSource `json:"-"`
    Path        string     `json:"-"`
}

type StepResponse struct {
    Directive      string   `json:"directive"`
    HistoryFolder  string   `json:"history_folder"`
    FilesToRead    []string `json:"files_to_read"`
    ComposedPrompt string   `json:"composed_prompt"`
}
```

## Validation Rules Summary

| Entity | Rule | Error Code |
|--------|------|------------|
| Initiative.type | Must be feature/bug/refactor | INVALID_TYPE |
| Initiative.name | Must be slug format (lowercase, hyphens) | INVALID_NAME |
| Initiative.path | Directory must exist | PATH_NOT_FOUND |
| Step.name | Must match step definition | UNKNOWN_STEP |
| Step execution | .brains folder must exist in dir | NOT_INITIALIZED |
| Step execution | Active initiative required (unless init step) | NO_ACTIVE_INITIATIVE |
