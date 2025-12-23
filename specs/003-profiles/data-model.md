# Data Model: Profile Composition System

**Date**: 2025-12-22
**Feature**: 003-profiles

## Entities

### Profile

A composable unit of prompt content with optional YAML frontmatter and markdown body.

```go
// Profile represents a loaded profile with parsed frontmatter and content.
type Profile struct {
    // Identity
    Name        string   // Derived from filename if not in frontmatter
    Path        string   // Absolute path to the profile file
    Source      ProfileSource

    // Frontmatter fields (all optional)
    Description string
    Includes    []string // Names of profiles to include before this one
    Inherits    bool     // Whether to prepend parent directory versions (default: true)

    // Content
    Body        string   // Markdown content after frontmatter
    RawContent  []byte   // Original file content for --raw mode
}
```

**Storage**: `.md` files in `.brains/profiles/` directories

**Validation Rules**:
- Name: lowercase alphanumeric with hyphens, derived from filename if not specified
- Includes: Must reference existing profile names (validated during DAG build)
- Inherits: Defaults to `true` when not specified

### ProfileSource

Enumeration indicating where a profile was loaded from.

```go
type ProfileSource int

const (
    SourceLocal  ProfileSource = iota // Project .brains/profiles/
    SourceParent                       // Intermediate .brains/profiles/ (between CWD and git root)
    SourceGlobal                       // ~/.brains/profiles/
)

func (s ProfileSource) String() string {
    switch s {
    case SourceLocal:
        return "local"
    case SourceParent:
        return "parent"
    case SourceGlobal:
        return "global"
    default:
        return "unknown"
    }
}
```

**Precedence** (highest to lowest):
1. `SourceLocal` - CWD `.brains/profiles/`
2. `SourceParent` - Intermediate directories up to git root
3. `SourceGlobal` - `~/.brains/profiles/`

### ProfileFrontmatter

YAML frontmatter structure for parsing.

```go
// ProfileFrontmatter represents the optional YAML frontmatter in a profile file.
type ProfileFrontmatter struct {
    Name        string   `yaml:"name"`
    Description string   `yaml:"description"`
    Includes    []string `yaml:"includes"`
    Inherits    *bool    `yaml:"inherits"` // Pointer to detect unset vs explicit false
}

// GetInherits returns the inherits value, defaulting to true if not set.
func (f ProfileFrontmatter) GetInherits() bool {
    if f.Inherits == nil {
        return true
    }
    return *f.Inherits
}
```

### CompositionResult

The output of composing one or more profiles.

```go
// CompositionResult contains the merged output and metadata from profile composition.
type CompositionResult struct {
    // Output
    Content       string   // Raw concatenated content (no separators)

    // Metadata
    ProfilesUsed  []string // Names of profiles included (in order)
    CharacterCount int
    EstimatedTokens int    // Rough estimate: CharacterCount / 4

    // Diagnostics
    Warnings      []string // Non-fatal issues encountered
    ResolutionLog []ResolutionEntry
}

// ResolutionEntry records how a profile was resolved.
type ResolutionEntry struct {
    Name       string
    Source     ProfileSource
    Path       string
    Inherited  bool          // Whether content was inherited from parent
    IncludedBy string        // Which profile included this one (empty for top-level)
}
```

### Registry

Persistent list of known `.brains/` directories for cross-project discovery.

```go
// Registry stores known .brains/ directories across projects.
type Registry struct {
    Directories []RegistryEntry `json:"directories"`
    UpdatedAt   time.Time       `json:"updated_at"`
}

// RegistryEntry represents a single known .brains/ directory.
type RegistryEntry struct {
    Path       string    `json:"path"`       // Absolute path to .brains/ directory
    AddedAt    time.Time `json:"added_at"`
    LastSeenAt time.Time `json:"last_seen_at"`
}
```

**Storage**: `~/.brains/registry.json`

**Concurrency**: Protected by OS-level file lock (flock) via separate `.lock` file

## Relationships

```
┌─────────────┐     includes      ┌─────────────┐
│   Profile   │───────────────────│   Profile   │
└─────────────┘                   └─────────────┘
       │                                 │
       │ has source                      │
       ▼                                 │
┌─────────────────┐                      │
│  ProfileSource  │◄─────────────────────┘
└─────────────────┘

┌─────────────────┐     produces    ┌───────────────────┐
│    Composer     │────────────────►│ CompositionResult │
└─────────────────┘                 └───────────────────┘
       │
       │ builds
       ▼
┌─────────────┐
│     DAG     │
└─────────────┘

┌─────────────────┐     stores      ┌─────────────────┐
│    Registry     │────────────────►│  RegistryEntry  │
└─────────────────┘                 └─────────────────┘
```

## State Transitions

### Profile Loading States

```
┌──────────┐   parse file   ┌──────────┐   validate DAG   ┌──────────┐
│  Unread  │───────────────►│  Parsed  │─────────────────►│  Valid   │
└──────────┘                └──────────┘                  └──────────┘
     │                           │                             │
     │ file error                │ parse error                 │ cycle detected
     ▼                           ▼                             ▼
┌──────────┐               ┌──────────┐                  ┌──────────┐
│  Error   │               │  Error   │                  │  Error   │
└──────────┘               └──────────┘                  └──────────┘
```

## File Format Examples

### Minimal Profile (no frontmatter)

```markdown
You are a helpful assistant specializing in database design.
```

- Name: derived from filename (e.g., `database.md` → `database`)
- Description: empty
- Includes: empty
- Inherits: true (default)

### Full Profile (all fields)

```markdown
---
name: database-expert
description: Expert guidance for SQL and schema design
includes:
  - sql-basics
  - best-practices
inherits: false
---

You are an expert database architect with deep knowledge of:
- Schema design and normalization
- Query optimization
- Index strategies
```

### Registry File Format

```json
{
  "directories": [
    {
      "path": "/Users/dev/project-a/.brains",
      "added_at": "2025-12-22T10:00:00Z",
      "last_seen_at": "2025-12-22T15:30:00Z"
    },
    {
      "path": "/Users/dev/project-b/.brains",
      "added_at": "2025-12-20T09:00:00Z",
      "last_seen_at": "2025-12-21T14:00:00Z"
    }
  ],
  "updated_at": "2025-12-22T15:30:00Z"
}
```
