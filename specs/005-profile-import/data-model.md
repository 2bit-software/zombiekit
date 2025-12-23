# Data Model: Profile Import Subcommand

**Feature**: 005-profile-import
**Date**: 2025-12-22

## Entities

### ImportResult

Summary of an import operation. Added to `internal/profile/types.go`.

```go
// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
    // Counters
    Created    int // Number of new profiles created
    Overwritten int // Number of existing profiles overwritten
    Failed     int // Number of agents that failed to import

    // Details
    CreatedPaths     []string // Paths to created profiles
    OverwrittenPaths []string // Paths to overwritten profiles
    FailedAgents     []ImportFailure // Details of failed imports

    // Operation mode
    DryRun bool // True if this was a dry run (no actual writes)
}

// ImportFailure describes why an agent failed to import.
type ImportFailure struct {
    AgentName string // Name of the agent that failed
    AgentPath string // Source path
    Error     string // Error message
}
```

### Importer

Service that performs the import operation. New file `internal/profile/importer.go`.

```go
// Importer handles importing profiles from external sources to brains format.
type Importer struct {
    claudeSource *ClaudeSource
    workingDir   string
    homeDir      string
}

// NewImporter creates an Importer for the given working directory.
func NewImporter(workingDir string) (*Importer, error)

// Import imports all Claude agents to brains profiles.
// If dryRun is true, no files are written.
func (i *Importer) Import(dryRun bool) (*ImportResult, error)
```

## Field Mappings

### Claude → Brains Frontmatter

| Source (Claude) | Target (Brains) | Transformation |
|-----------------|-----------------|----------------|
| name | name | Copy |
| description | description | Copy |
| includes | includes | Copy |
| inherits | inherits | Force to `false` |
| model | - | Discard |
| color | - | Discard |

### Body Content

Body content is copied unchanged from Claude agent to brains profile.

## Directory Mappings

| Source Directory | Target Directory |
|------------------|------------------|
| `.claude/agents/` | `.brains/profiles/` |
| `~/.claude/agents/` | `~/.brains/profiles/` |

## State Transitions

### Profile File States

```
Non-existent → Created (new profile from import)
Existing → Overwritten (profile replaced by import)
```

### Import Operation Flow

```
Start
  ↓
Read all Claude agents (local + global)
  ↓
For each agent:
  ├─ Parse frontmatter → Success: continue
  │                    → Failure: record in FailedAgents, continue
  ↓
  Convert to brains format
  ↓
  Check target exists?
  ├─ Yes: record as Overwritten
  └─ No: record as Created
  ↓
  [If not dry-run] Write file
  ↓
Next agent
  ↓
Return ImportResult
```

## Validation Rules

1. **Agent name**: Derived from filename (no extension), must be valid filename characters
2. **Frontmatter**: Invalid YAML causes import failure for that agent only
3. **Target directory**: Created if missing (with 0o755 permissions)
4. **File write**: Uses 0o644 permissions (readable by all, writable by owner)

## File Format

### Input (Claude Agent)

```yaml
---
name: my-agent
description: Does something useful
model: opus
color: blue
includes: [other-agent]
inherits: false
---

Agent body content here.
```

### Output (Brains Profile)

```yaml
---
name: my-agent
description: Does something useful
includes: [other-agent]
inherits: false
---

Agent body content here.
```
