---
status: complete
updated: 2026-01-15
---

# Research: Profile Creator Feature

## Executive Summary

The profile creator feature needs to integrate with ZombieKit's existing profile system which uses a three-tier resolution hierarchy (local → global → embedded). New profiles should be written to either `.brains/profiles/` (local) or `~/.brains/profiles/` (global) based on user preference, and become immediately discoverable without restart due to the dynamic discovery mechanism.

## Findings

### Codebase Context

**Profile Resolution Hierarchy** (highest to lowest precedence):
1. **Local**: `{project}/.brains/profiles/`
2. **Parent**: Intermediate `.brains/profiles/` directories (up to git root)
3. **Global**: `~/.brains/profiles/`
4. **Embedded**: Binary-compiled profiles (fallback)

**Key Insight**: Profiles are discovered on each request via `FindProfileDirs()` in `internal/profile/resolver.go:55-103`, meaning new files are immediately available without restart.

**Profile File Structure** (from `internal/profile/types.go:35-54`):
```yaml
---
name: profile-name              # Optional; derived from filename if omitted
description: Human description  # Optional; displayed in lists
includes: [profile-a, ...]      # Optional; array of profiles to include
inherits: true/false            # Optional; defaults to true for brains profiles
type: action/domain/step        # Optional; classification
---
Markdown content body...
```

**Existing Validation** (from `internal/profile/service.go`):
- Missing includes check: validates all included profile names exist
- Circular dependency detection: uses DFS with path tracking
- Name normalization: lowercase, spaces/underscores → hyphens, alphanumeric only

**MCP Profile Tools** (from `internal/mcp/server.go:229-258`):
- `profile-compose` - registered, merges profiles
- `profile-list` - registered, lists available profiles
- `HandleShow` - implemented but not registered
- `HandleValidate` - implemented but not registered

### Domain Knowledge

**Workflow Pattern** (from `templates/steps/feature.md`):
Existing workflows follow a consistent multi-phase structure:
1. Phase 0: Initialize metadata
2. Phase I: Research (parallel agents)
3. Phase II: Create (single agent)
4. Phase III: Audit (parallel agents)
5. Phase IV: Highlight (user approval)

**Step Profile Pattern** (from `internal/step/loader.go`):
Steps are loaded from three locations with same precedence as profiles:
1. `.brains/steps/{name}.md` (local)
2. `~/.brains/steps/{name}.md` (global)
3. Embedded (fallback)

**Profile Types in Use**:
- `skill` - callable agent actions (brains.feature, brains.plan)
- `action` - operational profiles
- `domain` - domain-specific knowledge
- `step` - workflow step definitions

## Decision Points

- [x] **D1**: Where to implement? → New step profile (`profile-create.md`) to dogfood existing infrastructure
- [x] **D2**: Storage location? → Ask user at creation time (local vs global)
- [x] **D3**: Content generation? → AI-generated via research → create cycle (dogfooding)
- [x] **D4**: Validation timing? → Validate before writing to prevent invalid profiles

## Recommendations

1. **Create a `profile-create` step** in `templates/steps/profile-create.md` following the multi-phase pattern

2. **Add profile write MCP tool** (`profile-write`) to enable writing validated profiles to disk

3. **Reuse existing validation** via `Service.Validate()` before finalizing content

4. **Add Claude Code command** (`.claude/commands/brains.profile.new.md`) as entry point

5. **Register `profile-validate`** MCP tool to expose validation for the workflow

## Sources

- `internal/profile/resolver.go:55-103` - Profile discovery mechanism
- `internal/profile/service.go` - Profile operations and validation
- `internal/profile/types.go:35-54` - Profile data structures
- `internal/mcp/tools/profile/tool.go` - MCP tool implementations
- `internal/mcp/server.go:229-258` - Tool registration
- `templates/steps/feature.md` - Reference workflow structure
- `internal/step/loader.go` - Step loading mechanism
