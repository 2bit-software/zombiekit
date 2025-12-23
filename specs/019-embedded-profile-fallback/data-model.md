# Data Model: Embedded Profile Fallback

**Feature**: 019-embedded-profile-fallback
**Date**: 2025-12-23

## Entity Changes

### ProfileSource (types.go)

Extended enumeration for profile source locations.

| Value | Constant | String | Description |
|-------|----------|--------|-------------|
| 0 | SourceLocal | "local" | Project's .brains/profiles/ directory |
| 1 | SourceParent | "parent" | Intermediate .brains/profiles/ directories |
| 2 | SourceGlobal | "global" | ~/.brains/profiles/ directory |
| **3** | **SourceEmbedded** | **"embedded"** | **Binary-embedded profiles (NEW)** |

### ResolvedDirectory (resolver.go)

No structural changes. Existing fields accommodate embedded profiles:

| Field | Type | Embedded Behavior |
|-------|------|-------------------|
| Path | string | Set to `[embedded]` as virtual marker |
| Source | ProfileSource | Set to `SourceEmbedded` |

### Profile (types.go)

No structural changes. Existing fields accommodate embedded profiles:

| Field | Embedded Behavior |
|-------|-------------------|
| Name | Derived from embedded filename |
| Path | Format: `[embedded]/<name>.md` |
| Source | Set to `SourceEmbedded` |
| RawContent | Read from embed.FS |
| Body | Parsed from RawContent |

## New Components

### EmbeddedFS Registry (embedded.go)

Global registry for embedded filesystem.

```text
Package-level variable:
  globalEmbeddedFS embed.FS

Functions:
  SetEmbeddedFS(fs embed.FS)           - Register embedded profiles
  GetEmbeddedFS() embed.FS             - Retrieve registered embed.FS
  HasEmbeddedProfiles() bool           - Check if any embedded profiles available
  loadEmbeddedProfiles() []*Profile    - Load all profiles from embed.FS
```

## Resolution Order

The profile resolution hierarchy (highest to lowest precedence):

```text
1. Local      (.brains/profiles/ in current directory)
2. Parent     (.brains/profiles/ in parent directories up to git root)
3. Global     (~/.brains/profiles/)
4. Embedded   (binary-embedded from profiles/)
```

## State Transitions

No state transitions apply - embedded profiles are immutable and read-only.

## Data Constraints

| Constraint | Rule |
|------------|------|
| Immutability | Embedded profiles cannot be modified at runtime |
| Naming | Embedded profile names must be valid (lowercase, hyphens, alphanumeric) |
| Shadowing | Any filesystem profile with same name shadows embedded version |
| Inclusion | Embedded profiles can be included by filesystem profiles |
| Inheritance | Filesystem profiles with `inherits: true` can inherit from embedded base |

## Validation Rules

| Rule | Validation |
|------|------------|
| Parse errors | Skip invalid embedded profiles, log warning |
| Missing includes | Allow includes referencing embedded profiles |
| Circular deps | Check cycles across all sources including embedded |
| Empty embed.FS | Proceed without embedded profiles (no error) |

## JSON Output Examples

### profile list (embedded profile)

```json
{
  "name": "init",
  "source": "embedded",
  "path": "[embedded]/init.md",
  "description": "Start using ZombieKit",
  "includes": [],
  "inherits": true,
  "type": "action"
}
```

### profile show (embedded profile)

```json
{
  "name": "research",
  "source": "embedded",
  "path": "[embedded]/research.md",
  "description": "Research aggregation agent",
  "includes": [],
  "inherits": true,
  "content": "...",
  "raw_content": "..."
}
```

### profile compose resolution log

```json
{
  "resolution": [
    {
      "name": "research",
      "source": "embedded",
      "path": "[embedded]/research.md"
    }
  ]
}
```
