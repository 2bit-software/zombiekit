# Implementation Plan: Profile Creator

**Branch**: `6969926a-feature-profile-creator` | **Date**: 2026-01-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `history/6969926a-feature-profile-creator/6969926a-feat-profile-creator/spec.md`

## Summary

Add a `/brains.profile.new` workflow command that guides users through creating new profiles using the research → create → audit → highlight cycle. Implement a `profile-write` MCP tool for persisting validated profiles to local (`.brains/profiles/`) or global (`~/.brains/profiles/`) directories.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: urfave/cli/v2, mark3labs/mcp-go, gopkg.in/yaml.v3, adrg/frontmatter
**Storage**: File-based (markdown with YAML frontmatter)
**Testing**: go test with testify/assert, testify/suite
**Target Platform**: CLI tool (macOS, Linux)
**Project Type**: Single project with MCP server
**Performance Goals**: N/A (developer CLI tool)
**Constraints**: Profiles must be immediately discoverable after write (no restart)
**Scale/Scope**: ~5-8 files modified, 1 new MCP tool, 1 new command profile

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Status | Notes |
|-----------|--------|-------|
| Meaningful variable names | ✅ Pass | Follow existing conventions |
| Single responsibility | ✅ Pass | profile-write does one thing |
| Error handling with context | ✅ Pass | Use ProfileError type |
| No panic in non-test code | ✅ Pass | Return errors |
| Test-first mindset | ⚠️ Follow | Tests before implementation |

**Gate Status**: ✅ PASS

## Project Structure

### Documentation (this feature)

```text
history/6969926a-feature-profile-creator/
├── INITIATIVE.md        # Initiative metadata
└── 6969926a-feat-profile-creator/
    ├── spec.md          # Feature specification (approved)
    ├── research.md      # Research findings
    ├── plan.md          # This file
    └── tasks.md         # Task breakdown (next step)
```

### Source Code Changes

```text
internal/
├── profile/
│   ├── service.go       # Add Write() method
│   └── service_test.go  # Add write tests
└── mcp/
    ├── server.go        # Register profile-write tool
    └── tools/
        └── profile/
            ├── tool.go      # Add HandleWrite handler
            ├── types.go     # Add WriteRequest/WriteResponse
            └── tool_test.go # Add write tests

profiles/
└── profile-new.md       # New workflow profile (Claude Code command backing)
```

## Implementation Phases

### Phase 1: MCP Tool Implementation

Add `profile-write` MCP tool to enable writing profiles to disk.

**Files**:
- `internal/profile/service.go` - Add `Write(name, content, location string, overwrite bool) (string, error)`
- `internal/mcp/tools/profile/types.go` - Add `WriteRequest`, `WriteResponse`
- `internal/mcp/tools/profile/tool.go` - Add `HandleWrite` handler
- `internal/mcp/server.go` - Register `profile-write` tool

**Acceptance**:
- `profile-write` with `location: "local"` creates file in `.brains/profiles/`
- `profile-write` with `location: "global"` creates file in `~/.brains/profiles/`
- Returns error if file exists and `overwrite: false`
- Creates directory if it doesn't exist
- Returns absolute path on success

### Phase 2: Workflow Profile

Create the `/brains.profile.new` workflow profile.

**Files**:
- `profiles/profile-new.md` - Workflow profile with phases:
  1. Gather inputs (name, description, location)
  2. Research existing profiles
  3. Generate content
  4. Validate and audit
  5. Present for approval
  6. Write via `profile-write` tool

**Acceptance**:
- Profile discoverable via `profile-list`
- Can be invoked as skill `/brains.profile.new`
- Workflow produces valid profile content

### Phase 3: Testing

Add tests for new functionality.

**Files**:
- `internal/profile/service_test.go` - Test Write() method
- `internal/mcp/tools/profile/tool_test.go` - Test HandleWrite

**Acceptance**:
- Tests cover success path (local, global)
- Tests cover error paths (exists, invalid location)
- Tests cover directory creation

## Dependencies

```
Phase 1 (MCP Tool) ← Phase 2 (Workflow Profile)
                  ← Phase 3 (Testing)
```

Phase 1 must complete first. Phases 2 and 3 can run in parallel after.

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Profile write permissions | Low | Medium | Check directory writability before write |
| Atomic write failure | Low | Medium | Use temp file + rename pattern |
| Name collision | Medium | Low | Clear error message with suggestions |

## Next Step

Run `/brains.tasks` to generate the detailed task breakdown.
