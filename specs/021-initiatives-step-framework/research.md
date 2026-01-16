# Research: Initiatives Step Framework

**Date**: 2025-12-23
**Feature**: 021-initiatives-step-framework

## Research Topics

### 1. Step Definition Format

**Decision**: Reuse existing profile frontmatter pattern with YAML frontmatter in markdown files.

**Rationale**:
- Consistent with existing `internal/profile` implementation
- Already have `adrg/frontmatter` dependency for parsing
- Markdown body can contain the step directive text
- YAML frontmatter can specify profiles to compose, files to read, etc.

**Alternatives Considered**:
- Pure YAML files: Rejected - less human-readable for directive text
- TOML frontmatter: Rejected - already using YAML across codebase
- JSON configuration: Rejected - not as readable for long directive text

**Step Definition Schema**:
```yaml
---
name: specify
description: Create or update the feature specification
profiles:               # Profiles to compose for this step
  - research
  - spec-creator
files:                  # Glob patterns for files to read
  - "spec.md"
  - "research.md"
  - "**/*.md"
type: step              # Profile type marker
---

# Directive text here as markdown body
Your task is to create or update the feature specification...
```

### 2. State File Format and Locking

**Decision**: JSON file at `.brains/active.json` with file-based locking using `gofrs/flock`.

**Rationale**:
- JSON is simple, human-readable, and easy to parse
- Already have `gofrs/flock` dependency in go.mod for file locking
- Single-user tool, so simple advisory locking is sufficient
- Gitignored to avoid conflicts between developers

**State File Schema**:
```json
{
  "initiative": "history/2024-01-15-feature-user-auth",
  "type": "feature",
  "name": "user-auth",
  "started": "2024-01-15T10:30:00Z",
  "last_activity": "2024-01-15T14:22:00Z",
  "status": "active",
  "current_step": "specify"
}
```

**Alternatives Considered**:
- SQLite: Rejected - overkill for simple state tracking
- TOML: Considered - JSON chosen for easier programmatic access
- No locking: Rejected - could lead to corrupted state with concurrent access

### 3. Initiative Folder Naming Convention

**Decision**: `{hex-timestamp}-{type}-{name}` pattern in `./history/` directory.

**Rationale**:
- Matches existing spec naming convention (hex timestamp ensures uniqueness and chronological sorting)
- Type prefix (feature/bug/refactor) makes purpose clear at a glance
- Name suffix provides human-readable identifier

**Format**: `{8-char-hex}-{type}-{slug}`
- Example: `675d8a3f-feature-user-auth`
- Example: `675d8b12-bug-login-crash`
- Example: `675d8c01-refactor-extract-middleware`

**Alternatives Considered**:
- Date-based prefix (2024-01-15-): Less precise, potential conflicts
- UUID prefix: Not chronologically sortable
- No type in name: Harder to filter by initiative type

### 4. Integration with Profile Composition

**Decision**: Step service calls existing `internal/profile.Service.Compose()` method.

**Rationale**:
- Reuses existing profile composition logic
- Maintains single source of truth for profile resolution
- Step definitions just specify which profiles to compose

**Integration Pattern**:
```go
// In step service
func (s *StepService) Execute(stepName string, workDir string) (*StepResponse, error) {
    step := s.GetStep(stepName)

    // Use existing profile service
    profileSvc, _ := profile.NewService(workDir)
    composition, _ := profileSvc.Compose(step.Profiles)

    return &StepResponse{
        Directive:       step.Directive,
        HistoryFolder:   s.getActiveInitiativePath(),
        FilesToRead:     s.resolveFiles(step.Files),
        ComposedPrompt:  composition.Content,
    }
}
```

### 5. Default Built-in Steps

**Decision**: Embed default step definitions in binary, allow project-level overrides.

**Rationale**:
- Zero-config experience for new users
- Projects can customize by placing files in `.brains/steps/`
- Follows existing pattern of embedded profiles (see `internal/profile/embedded.go`)

**Default Steps** (based on design docs):
| Step | Profiles | Purpose |
|------|----------|---------|
| `init` | - | Create new initiative |
| `specify` | research, spec-creator | Create/update spec |
| `plan` | research, plan-creator | Create implementation plan |
| `tasks` | task-creator | Break down into tasks |
| `implement` | - | Execute implementation |
| `audit` | auditor | Check artifact alignment |
| `clarify` | highlighter | Surface ambiguities |
| `complete` | - | Mark initiative done |

### 6. MCP Tool Parameters

**Decision**: Two required parameters (`step`, `dir`) plus optional override parameter (`initiative`).

**Schema**:
```json
{
  "name": "step",
  "description": "Execute a workflow step within an initiative",
  "inputSchema": {
    "type": "object",
    "properties": {
      "step": {
        "type": "string",
        "description": "Step name to execute (e.g., 'specify', 'plan', 'implement')"
      },
      "dir": {
        "type": "string",
        "description": "Working directory containing .brains folder"
      },
      "initiative": {
        "type": "string",
        "description": "Optional: override current initiative (path relative to history/)"
      }
    },
    "required": ["step", "dir"]
  }
}
```

**Response Format**:
```json
{
  "directive": "Your task is to...",
  "history_folder": "/path/to/history/675d8a3f-feature-user-auth",
  "files_to_read": ["spec.md", "research.md"],
  "composed_prompt": "# Research Methodology\n\n..."
}
```

## Summary of Decisions

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| Step format | YAML frontmatter + markdown | Consistent with profiles |
| State file | JSON + flock locking | Simple, human-readable |
| Naming | `{hex}-{type}-{name}` | Sortable, readable |
| Profile integration | Call existing Service.Compose() | Reuse, single source of truth |
| Default steps | Embedded, with overrides | Zero-config + customizable |
| MCP params | step, dir, initiative? | Minimal required, flexible |
