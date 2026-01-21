# Implementation Plan: Workflow Entrypoints

## Executive Summary

Introduce a "workflow" concept as a unified entrypoint that detects the type of work (feature, bug, refactor) and routes to the appropriate profile. Workflows are profiles with `type: workflow` containing classification instructions in the body.

## Design Decision: Profile-Based Workflows

**Chosen approach**: Workflows as profiles with `type: workflow` frontmatter, classification logic in body.

**Rationale**:
- Leverages existing profile composition infrastructure unchanged
- No new types, structs, or parsing logic needed
- Classification/routing handled by Claude reading the prompt, not code
- Consistent with existing `type: skill` and `type: step` patterns
- Embedded like current profiles via `EmbeddedProfiles`

**What we're NOT doing**:
- No `routes` field in frontmatter - classification is prose in the body
- No `WorkflowRoute` struct - Claude handles routing decisions
- No special composition logic - works like any other profile

## Frontmatter Schema

### Current Types
- `type: skill` - Instructional profiles (feature.md, bug.md, plan.md, etc.)
- `type: step` - Orchestration definitions (templates/steps/*.md)

### New Type
- `type: workflow` - Entrypoint profiles with classification/routing instructions

```yaml
---
name: new
description: Unified workflow entrypoint that routes to feature/bug/refactor
type: workflow
---

[Classification instructions in body - Claude reads this and decides which profile to load next]
```

## Implementation Phases

### Phase 1: Add Workflow Filter to MCP Tool
**Files**: `internal/mcp/tools/profile/tool.go`

1. Add `workflow` boolean parameter to profile-compose input schema
2. When `workflow: true`, filter to only `type: workflow` profiles
3. When `workflow: false` or omitted, filter to non-workflow profiles (default behavior)
4. Update tool description to document the parameter

### Phase 2: Create Embedded Workflow Profile
**Files**: `profiles/new.md` (new)

1. Create `profiles/new.md` with `type: workflow`
2. Body contains:
   - User input placeholder (`$ARGUMENTS`)
   - Classification instructions (feature vs bug vs refactor)
   - Instructions to load detected profile via `profile-compose`

### Phase 3: Create Claude Command
**Files**: `integrations/claude/commands/brains.new.md` (new)

1. Create `brains.new.md` that loads the `new` workflow profile
2. Passes user arguments to the classification prompt

### Phase 4: Remove Legacy Commands
**Files**: Delete old entrypoint commands

1. Delete `integrations/claude/commands/brains.feature.md`
2. Delete `integrations/claude/commands/brains.bug.md`
3. Delete `integrations/claude/commands/brains.refactor.md`
4. Delete corresponding `.claude/commands/` copies
5. Update `brains init` to not copy deleted commands

## Data Flow

```
User: "/brains.new fix the login bug"
           │
           ▼
┌─────────────────────────────────┐
│  brains.new command             │
│  profile-compose "new"          │
│  workflow: true                 │
└───────────┬─────────────────────┘
            │ loads workflow profile
            ▼
┌─────────────────────────────────┐
│  new.md (type: workflow)        │
│  - body: classification         │
│    prompt with $ARGUMENTS       │
└───────────┬─────────────────────┘
            │ Claude reads prompt,
            │ classifies as "bug"
            ▼
┌─────────────────────────────────┐
│  Claude calls                   │
│  profile-compose "bug"          │
│  workflow: false (default)      │
└───────────┬─────────────────────┘
            │ loads skill profile
            ▼
┌─────────────────────────────────┐
│  bug.md (type: skill)           │
│  - full bug workflow            │
└─────────────────────────────────┘
```

## File Changes Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/mcp/tools/profile/tool.go` | Modify | Add `workflow` boolean parameter |
| `internal/profile/service.go` | Modify | Add type filter to composition |
| `profiles/new.md` | New | Embedded workflow entrypoint |
| `integrations/claude/commands/brains.new.md` | New | Claude command for unified entry |
| `integrations/claude/commands/brains.feature.md` | Delete | Replaced by brains.new |
| `integrations/claude/commands/brains.bug.md` | Delete | Replaced by brains.new |
| `integrations/claude/commands/brains.refactor.md` | Delete | Replaced by brains.new |

## Testing Strategy

1. **Unit tests**: Workflow filter in profile service
   - `workflow: true` returns only workflow-type profiles
   - `workflow: false` returns only non-workflow profiles
   - Name collision resolved by filter

2. **Manual tests**: End-to-end workflow detection via Claude
   - `/brains.new fix the login bug` → detects bug
   - `/brains.new add user notifications` → detects feature
   - `/brains.new cleanup the auth module` → detects refactor

## Design Decisions

1. **Single entrypoint via `/brains.new`**
   - Remove separate brains.feature/bug/refactor commands
   - All new work starts through unified workflow detection
   - Simplifies command surface, reduces user confusion

2. **AI-driven classification**
   - Classification logic is prose in the profile body
   - Claude reads instructions and decides which profile to load
   - No code-level routing or structured metadata needed

3. **Minimal code changes**
   - `type: workflow` is just a new valid value for existing field
   - One new boolean parameter on MCP tool for disambiguation
   - Composition logic unchanged, just adds type filter

## Dependency Order

```
Phase 1 (MCP tool) → Phase 2 (new.md) → Phase 3 (brains.new command) → Phase 4 (remove legacy)
```

Each phase is independently testable and committable.
