# Data Model: Profile Source Abstraction

**Feature**: 004-source-interface
**Date**: 2025-12-22

## Entities

### SourceType

Enum-like type representing available profile sources.

| Value | Description |
|-------|-------------|
| `brains` | Default source, uses `.brains/profiles/` directories |
| `claude` | Claude Code agents, uses `.claude/agents/` directories |

**Validation**: Must be one of the defined values. Unknown values produce an error.

### ProfileSource (Interface)

Abstraction for reading and writing profiles from different backends.

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `FindProfileDirs` | - | `[]ResolvedDirectory, error` | Discover available profile directories |
| `LoadProfiles` | `dirs []ResolvedDirectory` | `map[string]*Profile, error` | Load profiles with shadowing |
| `LoadAllProfiles` | `dirs []ResolvedDirectory` | `map[string][]*Profile, error` | Load all profiles including shadowed |
| `GetInheritanceChain` | `name string` | `[]*Profile, error` | Get inheritance chain for a profile |
| `CreateProfile` | `name string, global bool` | `string, error` | Create new profile, return path |
| `GetInitDir` | `global bool` | `string, error` | Get directory path for init |
| `DefaultInherits` | - | `bool` | Default value for inherits field |
| `SourceName` | - | `string` | Human-readable source name |

### BrainsSource

Implementation of ProfileSource for brains profiles.

| Field | Type | Description |
|-------|------|-------------|
| `resolver` | `*Resolver` | Existing resolver for directory discovery |
| `workingDir` | `string` | Current working directory |
| `homeDir` | `string` | User home directory |

**Directory Pattern**: `.brains/profiles/`
**Resolution**: local > parent (up to git root) > global
**Default Inherits**: `true`

### ClaudeSource

Implementation of ProfileSource for Claude agents.

| Field | Type | Description |
|-------|------|-------------|
| `workingDir` | `string` | Current working directory |
| `homeDir` | `string` | User home directory |

**Directory Pattern**: `.claude/agents/`
**Resolution**: local > global (no parent traversal)
**Default Inherits**: `false`

### Profile (Extended)

Existing Profile struct extended with optional Claude-specific fields.

| Field | Type | Required | Source | Description |
|-------|------|----------|--------|-------------|
| `Name` | `string` | Yes | Both | Profile/agent name |
| `Path` | `string` | Yes | Both | Absolute file path |
| `Source` | `ProfileSource` | Yes | Both | Which level (local/parent/global) |
| `Description` | `string` | No | Both | Human-readable description |
| `Includes` | `[]string` | No | Both | Profiles to include before this one |
| `Inherits` | `bool` | No | Both | Whether to inherit from parent |
| `Body` | `string` | Yes | Both | Markdown content after frontmatter |
| `RawContent` | `[]byte` | Yes | Both | Original file content |
| `Model` | `string` | No | Claude | Claude model (e.g., "opus", "sonnet") |
| `Color` | `string` | No | Claude | UI color for Claude Code display |

### ClaudeFrontmatter

YAML frontmatter structure for Claude agents.

```yaml
---
name: agent-name           # Optional, defaults to filename
description: What it does  # Optional
model: opus                # Optional, Claude model
color: purple              # Optional, UI color
includes: []               # Optional, other agents to include
inherits: false            # Optional, defaults to false
---
```

### ResolvedDirectory

Existing structure representing a discovered profile directory.

| Field | Type | Description |
|-------|------|-------------|
| `Path` | `string` | Absolute path to profiles directory |
| `Source` | `ProfileSource` | Source level (local/parent/global) |

## Relationships

```
SourceType (string enum)
    │
    ▼
ProfileSource (interface)
    │
    ├──▶ BrainsSource (implementation)
    │       └── uses Resolver
    │
    └──▶ ClaudeSource (implementation)
            └── standalone

Profile
    │
    ├── belongs to ResolvedDirectory
    ├── may include other Profiles
    └── may inherit from parent Profiles
```

## State Transitions

### Profile Lifecycle

```
[Non-existent] ──create──▶ [Created] ──edit──▶ [Modified]
                                │
                                ▼
                          [Validated] ──compose──▶ [Composed Output]
```

### Source Selection

```
[CLI Invocation]
    │
    ├── --source brains (or default) ──▶ BrainsSource
    │
    └── --source claude ──▶ ClaudeSource
```

## Validation Rules

| Rule | Scope | Description |
|------|-------|-------------|
| Name format | Both | Lowercase, hyphens, alphanumeric only |
| Name length | Both | 1-64 characters |
| Unique name per source | Both | No duplicate names in same source |
| Valid includes | Both | All included profiles must exist in same source |
| No cycles | Both | Include graph must be acyclic |
| Valid model | Claude | If specified, should be known Claude model |
| Valid color | Claude | If specified, should be valid color name |

## JSON Output Schema

### ListEntry (extended for Claude)

```json
{
  "name": "string",
  "source": "local|parent|global",
  "path": "/absolute/path/to/profile.md",
  "description": "string",
  "includes": ["string"],
  "inherits": true,
  "shadowed": false,
  "model": "opus",      // Claude only
  "color": "purple"     // Claude only
}
```

### ShowResult (extended for Claude)

```json
{
  "name": "string",
  "source": "local|parent|global",
  "path": "/absolute/path/to/profile.md",
  "description": "string",
  "includes": ["string"],
  "inherits": true,
  "content": "composed content",
  "raw_content": "original file content",
  "inherited_from": [...],
  "model": "opus",      // Claude only
  "color": "purple"     // Claude only
}
```
