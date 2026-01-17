# Research: Profile Source Abstraction

**Feature**: 004-source-interface
**Date**: 2025-12-22

## Research Topics

### 1. Go Interface Design for Profile Sources

**Decision**: Define a `ProfileSource` interface with methods for all profile operations.

**Rationale**:
- Go interfaces are implicitly satisfied, enabling clean separation
- Interface should be defined where it's used (in the service layer), not in implementation packages
- Small, focused interface following Go idiom "accept interfaces, return structs"

**Alternatives Considered**:
- **Function-based approach**: Pass individual functions instead of interface. Rejected because multiple related operations benefit from shared state (working directory, home directory).
- **Embed source in Profile struct**: Let profiles know their source. Rejected because it couples data to source mechanism.

**Interface Design**:
```go
type ProfileSource interface {
    // Directory discovery
    FindProfileDirs() ([]ResolvedDirectory, error)

    // Loading
    LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error)
    LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error)

    // Inheritance
    GetInheritanceChain(name string) ([]*Profile, error)

    // Write operations
    CreateProfile(name string, global bool) (string, error)
    GetInitDir(global bool) (string, error)

    // Metadata
    DefaultInherits() bool
    SourceName() string
}
```

### 2. Claude Agent Frontmatter Format

**Decision**: Parse Claude agent frontmatter with extended fields: `name`, `description`, `model`, `color`.

**Rationale**:
- Examined actual Claude agent file (`.claude/agents/systems-architect.md`)
- Claude uses same YAML frontmatter pattern but with different/additional fields
- `model` specifies which Claude model (e.g., "opus")
- `color` is UI metadata for Claude Code display

**Claude Frontmatter Structure**:
```yaml
---
name: systems-architect
description: Use this agent when...
model: opus
color: purple
---
```

**Comparison with Brains Frontmatter**:
| Field | Brains | Claude | Notes |
|-------|--------|--------|-------|
| name | ✓ | ✓ | Both support |
| description | ✓ | ✓ | Both support |
| includes | ✓ | ✓ | Claude can use for agent composition |
| inherits | ✓ (default: true) | ✓ (default: false) | Different defaults |
| model | ✗ | ✓ | Claude-specific |
| color | ✗ | ✓ | Claude-specific |

### 3. Directory Resolution Strategy

**Decision**: Different resolution strategies per source type.

**Brains Source**:
- Local: `{CWD}/.brains/profiles/`
- Parent: Walk up to git root, check each `.brains/profiles/`
- Global: `~/.brains/profiles/`
- Precedence: local > parent > global

**Claude Source** (per clarification):
- Local: `{CWD}/.claude/agents/`
- Global: `~/.claude/agents/`
- No parent traversal
- Precedence: local > global

**Rationale**: Claude Code's actual behavior only checks project-local and user-global directories. No intermediate parent scanning like brains.

### 4. Refactoring Approach

**Decision**: Extract interface, wrap existing Resolver as BrainsSource, create new ClaudeSource.

**Refactoring Steps**:
1. Define `ProfileSource` interface in new `source.go`
2. Create `BrainsSource` struct that embeds/wraps existing `Resolver`
3. Create `ClaudeSource` struct with similar structure but different:
   - Directory paths (`.claude/agents/` instead of `.brains/profiles/`)
   - Resolution strategy (two-level only)
   - Frontmatter parsing (extended fields)
   - Default values (inherits=false)
4. Modify `Service` to accept `ProfileSource` instead of `Resolver`
5. Add factory function `NewSource(sourceType string) (ProfileSource, error)`

**Rationale**: Minimal changes to existing code. BrainsSource delegates to Resolver, maintaining backward compatibility.

### 5. CLI Flag Integration

**Decision**: Add `--source` flag as a global flag on the `profile` command group.

**Rationale**:
- Single flag definition shared by all subcommands
- Consistent with CLI patterns in urfave/cli
- Default value "brains" ensures backward compatibility

**Implementation**:
```go
&cli.StringFlag{
    Name:    "source",
    Aliases: []string{"s"},
    Value:   "brains",
    Usage:   "Profile source: brains (default) or claude",
}
```

### 6. Error Handling Consistency

**Decision**: Reuse existing error types, add source context.

**Error Types** (already exist):
- `ProfileNotFoundError` - profile doesn't exist
- `NotInitializedError` - directory not initialized
- `ProfileExistsError` - profile already exists (on create)
- `CycleError` - circular dependency detected

**Enhancement**: Each error type should include source information for clear messaging.

### 7. Testing Strategy

**Decision**: Use test fixtures with temporary directories per source type.

**Test Structure**:
- Unit tests for each source implementation
- Integration tests for Service with both sources
- CLI tests for flag handling and source selection

**Fixture Strategy**:
- Create temp directories with sample profiles/agents
- Test precedence rules for each source type
- Test cross-source isolation (sources don't affect each other)

## Summary

No unresolved clarifications. All technical decisions documented above. Ready for Phase 1 design.
