# Data Model: Profile Type Classification

**Feature**: 018-profile-type
**Date**: 2025-12-23

## Entity Changes

### ProfileFrontmatter (extended)

Represents the optional YAML frontmatter in a profile file.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | string | No | Profile name override |
| Description | string | No | Human-readable description |
| Includes | []string | No | Names of profiles to include |
| Inherits | *bool | No | Whether to prepend parent versions (default: true) |
| **Type** | **string** | **No** | **Profile type: "action", "domain", or "step"** |

### Profile (extended)

Represents a loaded profile with parsed frontmatter and content.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | string | Yes | Derived from filename if not in frontmatter |
| Path | string | Yes | Absolute path to the profile file |
| Source | ProfileSource | Yes | Where this profile was loaded from |
| Description | string | No | Human-readable description |
| Includes | []string | No | Names of profiles to include |
| Inherits | bool | Yes | Whether to prepend parent versions |
| Model | string | No | Claude model (e.g., "opus", "sonnet") |
| Color | string | No | UI color for Claude Code display |
| **Type** | **string** | **No** | **Profile type classification** |
| Body | string | Yes | Markdown content after frontmatter |
| RawContent | []byte | Yes | Original file content |

### ListEntry (extended)

Represents a profile in the list output (API/CLI).

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | string | `name` | Profile name |
| Source | ProfileSource | `-` | Internal source enum |
| SourceStr | string | `source` | String representation of source |
| Path | string | `path` | Absolute path |
| Description | string | `description` | Human-readable description |
| Includes | []string | `includes` | Included profiles |
| Inherits | bool | `inherits` | Inheritance setting |
| Shadowed | bool | `shadowed,omitempty` | True if shadowed |
| Model | string | `model,omitempty` | Claude model |
| Color | string | `color,omitempty` | UI color |
| **Type** | **string** | **`type,omitempty`** | **Profile type** |

### ShowResult (extended)

Contains the result of showing a single profile.

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | string | `name` | Profile name |
| Source | ProfileSource | `-` | Internal source enum |
| SourceStr | string | `source` | String representation |
| Path | string | `path` | Absolute path |
| Description | string | `description` | Human-readable description |
| Includes | []string | `includes` | Included profiles |
| Inherits | bool | `inherits` | Inheritance setting |
| Content | string | `content` | Profile body content |
| RawContent | string | `raw_content` | Original file content |
| InheritedFrom | []InheritedFrom | `inherited_from,omitempty` | Parent sources |
| Model | string | `model,omitempty` | Claude model |
| Color | string | `color,omitempty` | UI color |
| **Type** | **string** | **`type,omitempty`** | **Profile type** |

## Type Values

| Value | Description | Use Case |
|-------|-------------|----------|
| `action` | Prompts for doing work within steps | Task execution, code generation |
| `domain` | Domain knowledge prompts | Context, expertise, guidelines |
| `step` | Workflow step profiles | "specify", "research", "clarify" phases |
| (empty) | No type specified | Legacy/unclassified profiles |
| (other) | Unknown/custom type | Forward compatibility |

## Validation Rules

1. **Type is optional**: Empty string is valid (no type specified)
2. **Case insensitive matching**: "Action", "action", "ACTION" all match the "action" type
3. **Original casing preserved**: The value as written in YAML is stored and displayed
4. **Unknown values accepted**: Any string is valid; only known values get special UI treatment

## State Transitions

N/A - Type is a static classification, not a stateful field.

## Relationships

- **Profile ← ProfileFrontmatter**: Type parsed from frontmatter during profile loading
- **ListEntry ← Profile**: Type copied when building list entries
- **ShowResult ← Profile**: Type copied when building show results

## Migration

None required. Existing profiles without `type` field will have empty Type value (backwards compatible).
